package platform

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	utilfw "github.com/jfrog/terraform-provider-shared/util/fw"
	"github.com/samber/lo"
)

const ipAllowlistEndpoint = "/access/api/v1/allowlist/{serverName}"

var _ resource.Resource = (*ipAllowListResource)(nil)

type ipAllowListResource struct {
	ProviderData PlatformProviderMetadata
}

func NewIPAllowListResource() resource.Resource {
	return &ipAllowListResource{}
}

func (r *ipAllowListResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_ip_allowlist"
}

func (r *ipAllowListResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}
	r.ProviderData = req.ProviderData.(PlatformProviderMetadata)

	if r.ProviderData.MyJFrogClient == nil {
		resp.Diagnostics.AddError(
			"MyJFrogClient not configured",
			"MyJFrog Resty client is not configured due to missing `myjfrog_api_token` attribute or `JFROG_MYJFROG_API_TOKEN` env var.",
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
			},
		},
		MarkdownDescription: "Provides a JFrog [allowlist](https://jfrog.com/help/r/jfrog-hosting-models-documentation/configure-the-ip/cidr-allowlist) resource to manage list of allow IP/CIDR addresses. To use this resource, you need an access token. Only a Primary Admin can generate MyJFrog tokens. For more information, see [Generate a Token in MyJFrog](https://jfrog.com/help/r/jfrog-hosting-models-documentation/generate-a-token-in-myjfrog).\n\n" +
			"~>See [Allowlist REST API](https://jfrog.com/help/r/jfrog-rest-apis/allowlist-rest-api) for limitations.",
	}
}

type ipAllowListResourceModel struct {
	ServerName types.String `tfsdk:"server_name"`
	IPs        types.Set    `tfsdk:"ips"`
}

type ipAllowListAPIPostRequestModel struct {
	IPs []string `json:"ips"`
}

type ipAllowListIPAPIGetResponseModel struct {
	IP string `json:"ip"`
}

type ipAllowListAPIGetResponseModel struct {
	IPs []ipAllowListIPAPIGetResponseModel `json:"ips"`
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

	allowList := ipAllowListAPIPostRequestModel{
		IPs: planIPs,
	}

	response, err := r.ProviderData.MyJFrogClient.R().
		SetPathParam("serverName", plan.ServerName.ValueString()).
		SetBody(&allowList).
		Post(ipAllowlistEndpoint)
	if err != nil {
		utilfw.UnableToCreateResourceError(resp, err.Error())
		return
	}

	if response.IsError() {
		utilfw.UnableToCreateResourceError(resp, response.String())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ipAllowListResource) getAllowList(serverName string) ([]string, error) {
	var allowList ipAllowListAPIGetResponseModel

	response, err := r.ProviderData.MyJFrogClient.R().
		SetPathParam("serverName", serverName).
		SetResult(&allowList).
		Get(ipAllowlistEndpoint)

	if err != nil {
		return nil, err
	}

	if response.IsError() {
		return nil, fmt.Errorf("%s", response.String())
	}

	ips := lo.Map(allowList.IPs, func(list ipAllowListIPAPIGetResponseModel, _ int) string {
		return list.IP
	})
	return ips, nil
}

func (r *ipAllowListResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ipAllowListResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ips, err := r.getAllowList(state.ServerName.ValueString())
	if err != nil {
		utilfw.UnableToRefreshResourceError(resp, err.Error())
		return
	}

	// Convert from the API data model to the Terraform data model
	// and refresh any attribute values.
	ipsSet, d := types.SetValueFrom(ctx, types.StringType, ips)
	if d != nil {
		resp.Diagnostics.Append(d...)
	}
	if resp.Diagnostics.HasError() {
		return
	}
	state.IPs = ipsSet

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *ipAllowListResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan ipAllowListResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get the current list of IPs
	jpdIPs, err := r.getAllowList(plan.ServerName.ValueString())
	if err != nil {
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
		allowList := ipAllowListAPIPostRequestModel{
			IPs: ipsToAdd,
		}

		response, err := r.ProviderData.MyJFrogClient.R().
			SetPathParam("serverName", plan.ServerName.ValueString()).
			SetBody(&allowList).
			Post(ipAllowlistEndpoint)
		if err != nil {
			utilfw.UnableToUpdateResourceError(resp, err.Error())
			return
		}

		if response.IsError() {
			utilfw.UnableToUpdateResourceError(resp, response.String())
			return
		}
	}

	if len(ipsToRemove) > 0 {
		allowList := ipAllowListAPIPostRequestModel{
			IPs: ipsToRemove,
		}

		response, err := r.ProviderData.MyJFrogClient.R().
			SetPathParam("serverName", plan.ServerName.ValueString()).
			SetBody(&allowList).
			Delete(ipAllowlistEndpoint)
		if err != nil {
			utilfw.UnableToUpdateResourceError(resp, err.Error())
			return
		}

		if response.IsError() {
			utilfw.UnableToUpdateResourceError(resp, response.String())
			return
		}
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
	resp.Diagnostics.Append(state.IPs.ElementsAs(ctx, ipsToRemove, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	allowList := ipAllowListAPIPostRequestModel{
		IPs: ipsToRemove,
	}

	response, err := r.ProviderData.MyJFrogClient.R().
		SetPathParam("serverName", state.ServerName.ValueString()).
		SetBody(&allowList).
		Delete(ipAllowlistEndpoint)
	if err != nil {
		utilfw.UnableToDeleteResourceError(resp, err.Error())
		return
	}

	if response.IsError() {
		utilfw.UnableToDeleteResourceError(resp, response.String())
		return
	}

	// If the logic reaches here, it implicitly succeeded and will remove
	// the resource from state if there are no other errors.
}

func (r *ipAllowListResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("server_name"), req, resp)
}
