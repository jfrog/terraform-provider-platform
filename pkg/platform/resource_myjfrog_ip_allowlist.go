package platform

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	utilfw "github.com/jfrog/terraform-provider-shared/util/fw"
	"github.com/samber/lo"
	"github.com/sethvargo/go-retry"
)

const ipAllowlistEndpoint = "/api/jmis/v1/allowlist/{serverName}"

type MyJFrogAllowlistError struct {
	Message string   `json:"message"`
	Details []string `json:"details"`
}

type MyJFrogAllowlistErrorResponse struct {
	Status  string                  `json:"status"`
	Message string                  `json:"message"`
	Errors  []MyJFrogAllowlistError `json:"errors"`
}

func (r MyJFrogAllowlistErrorResponse) Error() string {
	messages := lo.Reduce(r.Errors, func(msgs []string, err MyJFrogAllowlistError, _ int) []string {
		msg := fmt.Sprintf("%s - %s", err.Message, strings.Join(err.Details, ", "))
		return append(msgs, msg)
	}, []string{})
	return strings.Join(messages, ", ")
}

type MyJFrogAllowlistConflictErrorResponse struct {
	Status string   `json:"status"`
	Errors []string `json:"errors"`
}

func (r MyJFrogAllowlistConflictErrorResponse) Error() string {
	return strings.Join(r.Errors, ", ")
}

var _ resource.Resource = (*ipAllowListResource)(nil)

type ipAllowListResource struct {
	ProviderData PlatformProviderMetadata
}

func NewMyJFrogIPAllowListResource() resource.Resource {
	return &ipAllowListResource{}
}

func (r *ipAllowListResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_myjfrog_ip_allowlist"
}

func (r *ipAllowListResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}
	r.ProviderData = req.ProviderData.(PlatformProviderMetadata)

	if r.ProviderData.MyJFrogClient == nil {
		resp.Diagnostics.AddError(
			"MyJFrogClient not configured in provider",
			"MyJFrog Resty client is not configured due to missing `myjfrog_api_token` attribute in provider configuration, or missing `JFROG_MYJFROG_API_TOKEN` env var.",
		)
	}
}

func (r *ipAllowListResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"server_name": schema.StringAttribute{
				Required: true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				MarkdownDescription: "Name of the server. If your JFrog URL is `myserver.jfrog.io`, the `server_name` is `myserver`.",
			},
			"ips": schema.SetAttribute{
				ElementType: types.StringType,
				Required:    true,
				Description: "List of IPs for the JPD allowlist",
				Validators: []validator.Set{
					setvalidator.ValueStringsAre(
						IPCIDR(),
					),
				},
			},
		},
		MarkdownDescription: "Provides a MyJFrog [IP allowlist](https://jfrog.com/help/r/jfrog-hosting-models-documentation/configure-the-ip/cidr-allowlist) resource to manage list of allow IP/CIDR addresses. " +
			"To use this resource, you need an access token. Only a Primary Admin can generate MyJFrog tokens. For more information, see [Generate a Token in MyJFrog](https://jfrog.com/help/r/jfrog-hosting-models-documentation/generate-a-token-in-myjfrog).\n\n" +
			"->This resource is supported only on the Cloud (SaaS) platform.\n\n" +
			"~>The rate limit is **5 times per hour** for actions that result in a successful outcome (for Create, Update, and Delete actions). See [Allowlist REST API](https://jfrog.com/help/r/jfrog-rest-apis/allowlist-rest-api) for full list of limitations.",
	}
}

type ipAllowListResourceModel struct {
	ServerName types.String `tfsdk:"server_name"`
	IPs        types.Set    `tfsdk:"ips"`
}

func (r *ipAllowListResourceModel) fromIPs(ctx context.Context, ips []string) (ds diag.Diagnostics) {
	ipsSet, d := types.SetValueFrom(ctx, types.StringType, ips)
	if d != nil {
		ds.Append(d...)
	}
	r.IPs = ipsSet

	return
}

type ipAllowListAPIPostRequestModel struct {
	IPs []string `json:"ips"`
}

type ipAllowListAPIResponseModel struct {
	Status string `json:"status"`
}

type ipAllowListAPIPostDeleteResponseModel struct {
	ipAllowListAPIResponseModel
	Message string `json:"message"`
}

type ipAllowListIPAPIGetResponseModel struct {
	IP string `json:"ip"`
}

type ipAllowListAPIGetResponseModel struct {
	ipAllowListAPIResponseModel
	IPs []ipAllowListIPAPIGetResponseModel `json:"ips"`
}

// waitForCompletion makes GET request to MyJFrog API to get the current IP allowlist status.
// The response includes the current list of IPs so we return that as well when status changes to "COMPLETED".
func (r ipAllowListResource) waitForCompletion(ctx context.Context, serverName string) ([]string, error) {
	tflog.Info(ctx, "waiting for ip allowlist process to complete")

	ips := []string{}

	backoff := retry.NewConstant(15 * time.Second)
	backoff = retry.WithJitterPercent(10, backoff)
	backoff = retry.WithCappedDuration(1*time.Minute, backoff)
	backoff = retry.WithMaxDuration(15*time.Minute, backoff)

	var retryFunc = func(_ context.Context) error {
		tflog.Info(ctx, "checking ip allowlist process status")
		i, status, err := r.getIPs(serverName)
		if err != nil {
			return err
		}
		if status != "COMPLETED" {
			tflog.Info(ctx, "ip allowlist process still in progress")
			return retry.RetryableError(fmt.Errorf(status))
		}

		ips = i
		tflog.Info(ctx, "ip allowlist process completed")
		return nil
	}

	if err := retry.Do(ctx, backoff, retryFunc); err != nil {
		return nil, err
	}

	return ips, nil
}

func (r *ipAllowListResource) getIPs(serverName string) ([]string, string, error) {
	var result ipAllowListAPIGetResponseModel

	response, err := r.ProviderData.MyJFrogClient.R().
		SetPathParam("serverName", serverName).
		SetResult(&result).
		Get(ipAllowlistEndpoint)

	if err != nil {
		return nil, "", err
	}

	if response.IsError() {
		return nil, "", fmt.Errorf("%s", response.String())
	}

	ips := lo.Map(result.IPs, func(list ipAllowListIPAPIGetResponseModel, _ int) string {
		return list.IP
	})
	return ips, result.Status, nil
}

func (r *ipAllowListResource) addIPs(ctx context.Context, serverName string, ips []string) ([]string, error) {
	return r.mutateIPs(ctx, serverName, ips, func(req *resty.Request) (*resty.Response, error) {
		return req.Post(ipAllowlistEndpoint)
	})
}

func (r *ipAllowListResource) removeIPs(ctx context.Context, serverName string, ips []string) ([]string, error) {
	return r.mutateIPs(ctx, serverName, ips, func(req *resty.Request) (*resty.Response, error) {
		return req.Delete(ipAllowlistEndpoint)
	})
}

func (r *ipAllowListResource) mutateIPs(ctx context.Context, serverName string, ips []string, requestCallback func(req *resty.Request) (*resty.Response, error)) ([]string, error) {
	if requestCallback == nil {
		return nil, fmt.Errorf("requestCallback cannot be nil")
	}

	allowList := ipAllowListAPIPostRequestModel{
		IPs: ips,
	}

	var result ipAllowListAPIPostDeleteResponseModel
	var apiErr MyJFrogAllowlistErrorResponse
	req := r.ProviderData.MyJFrogClient.R().
		SetPathParam("serverName", serverName).
		SetBody(&allowList).
		SetResult(&result).
		SetError(&apiErr)

	resp, err := requestCallback(req)

	if err != nil {
		return nil, err
	}

	if resp.IsError() {
		if slices.Contains([]int{http.StatusConflict, http.StatusTooManyRequests}, resp.StatusCode()) {
			var conflictErr MyJFrogAllowlistConflictErrorResponse
			err := json.Unmarshal(resp.Body(), &conflictErr)
			if err != nil {
				return nil, err
			}
			return nil, conflictErr
		}
		return nil, apiErr
	}

	updatedIPS, err := r.waitForCompletion(ctx, serverName)
	if err != nil {
		return nil, err
	}

	return updatedIPS, nil
}

func (r *ipAllowListResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ipAllowListResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var planIPs []string
	resp.Diagnostics.Append(plan.IPs.ElementsAs(ctx, &planIPs, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if len(planIPs) > 0 {
		ips, err := r.addIPs(ctx, plan.ServerName.ValueString(), planIPs)
		if err != nil {
			utilfw.UnableToCreateResourceError(resp, err.Error())
			return
		}

		resp.Diagnostics.Append(plan.fromIPs(ctx, ips)...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ipAllowListResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ipAllowListResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// waitForCompletion fetches list of ips from server so no need for extra GET request
	ips, err := r.waitForCompletion(ctx, state.ServerName.ValueString())
	if err != nil {
		utilfw.UnableToRefreshResourceError(resp, err.Error())
		return
	}

	// Convert from the API data model to the Terraform data model
	// and refresh any attribute values.
	resp.Diagnostics.Append(state.fromIPs(ctx, ips)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *ipAllowListResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan ipAllowListResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	serverName := plan.ServerName.ValueString()

	// Get the current list of IPs
	jpdIPs, _, err := r.getIPs(serverName)
	if err != nil {
		utilfw.UnableToUpdateResourceError(resp, err.Error())
		return
	}

	if _, err := r.waitForCompletion(ctx, serverName); err != nil {
		utilfw.UnableToUpdateResourceError(resp, err.Error())
		return
	}

	var planIPs []string
	resp.Diagnostics.Append(plan.IPs.ElementsAs(ctx, &planIPs, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ipsToAdd, ipsToRemove := lo.Difference(planIPs, jpdIPs)

	if len(ipsToAdd) > 0 {
		_, e := r.addIPs(ctx, serverName, ipsToAdd)
		if e != nil {
			resp.Diagnostics.AddError(
				"failed to add IPs",
				e.Error(),
			)
		}
	}
	if len(ipsToRemove) > 0 {
		_, e := r.removeIPs(ctx, serverName, ipsToRemove)
		if e != nil {
			resp.Diagnostics.AddError(
				"failed to add IPs",
				e.Error(),
			)
		}
	}

	if resp.Diagnostics.HasError() {
		return
	}

	// waitForCompletion fetches list of ips from server so no need for extra GET request
	ips, err := r.waitForCompletion(ctx, serverName)
	if err != nil {
		utilfw.UnableToUpdateResourceError(resp, err.Error())
		return
	}

	resp.Diagnostics.Append(plan.fromIPs(ctx, ips)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ipAllowListResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state ipAllowListResourceModel

	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var ipsToRemove []string
	resp.Diagnostics.Append(state.IPs.ElementsAs(ctx, &ipsToRemove, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if len(ipsToRemove) > 0 {
		_, err := r.removeIPs(ctx, state.ServerName.ValueString(), ipsToRemove)
		if err != nil {
			utilfw.UnableToDeleteResourceError(resp, err.Error())
		}
	}

	// If the logic reaches here, it implicitly succeeded and will remove
	// the resource from state if there are no other errors.
}

func (r *ipAllowListResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("server_name"), req, resp)
}

type ipCIDRValidator struct{}

// Description returns a plain text description of the validator's behavior, suitable for a practitioner to understand its impact.
func (v ipCIDRValidator) Description(ctx context.Context) string {
	return `IP/CIDR must be in format like "192.0.2.0/24" or "2001:db8::/32", as defined in RFC 4632 and RFC 4291.`
}

// MarkdownDescription returns a markdown formatted description of the validator's behavior, suitable for a practitioner to understand its impact.
func (v ipCIDRValidator) MarkdownDescription(ctx context.Context) string {
	return `IP/CIDR must be in format like "192.0.2.0/24" or "2001:db8::/32", as defined in RFC 4632 and RFC 4291.`
}

func (v ipCIDRValidator) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	// If the value is unknown or null, there is nothing to validate.
	if req.ConfigValue.IsUnknown() || req.ConfigValue.IsNull() {
		return
	}

	isValidIP := true
	isValidCIDR := true
	var err error

	ip := net.ParseIP(req.ConfigValue.ValueString())
	if ip == nil {
		isValidIP = false
		err = fmt.Errorf("invalid IP address: %s", req.ConfigValue.ValueString())
	}

	_, _, e := net.ParseCIDR(req.ConfigValue.ValueString())
	if e != nil {
		isValidCIDR = false
		err = e
	}

	if !isValidIP && !isValidCIDR {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Invalid IP/CIDR format",
			err.Error(),
		)
	}
}

func IPCIDR() ipCIDRValidator {
	return ipCIDRValidator{}
}
