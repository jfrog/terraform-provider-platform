package platform

import (
	"context"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/jfrog/terraform-provider-shared/util"
	utilfw "github.com/jfrog/terraform-provider-shared/util/fw"
)

const (
	AllowlistEndpoint = "platform/api/jmis/v1/allowlist/{serverName}"
	AllowlistEndpoint  = "platform/api/jmis/v1/allowlist/{serverName}"
)

func NewJmisAllowlistResource() resource.Resource {
	return &JmisAllowlistResource{
		TypeName: "platform_allowlist",
	}
}

type JmisAllowlistResource struct {
	ProviderData util.ProviderMetadata
	TypeName     string
}

type JmisAllowlistResourceModel struct {
	ServerName types.String `tfsdk:"serverName"`
	Ips types.String `tfsdk:"ips"`
	Status types.String `tfsdk:"status"`
	Message types.String `tfsdk:"message"`
	Errors types.String `tfsdk:"errors"`
	Details types.String `tfsdk:"details"`
	Ip types.String `tfsdk:"ip"`
}

type AllowlistRequestAPIModel struct {
	ServerName string `json:"serverName"`
	Ips string `json:"ips"`
	Status string `json:"status"`
	Message string `json:"message"`
	Errors string `json:"errors"`
	Details string `json:"details"`
	Ip string `json:"ip"`
}

type AllowlistAPIModel struct {
	ServerName string `json:"serverName"`
	Ips string `json:"ips"`
	Status string `json:"status"`
	Message string `json:"message"`
	Errors string `json:"errors"`
	Details string `json:"details"`
	Ip string `json:"ip"`
}

func (r *JmisAllowlistResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = r.TypeName
}

func (r *JmisAllowlistResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"serverName": schema.StringAttribute{
				Required: true,
				Description: "The serverName of the resource.",
			},
			"ips": schema.StringAttribute{
				Optional: true,
				Description: "The ips of the resource.",
			},
			"status": schema.StringAttribute{
				Optional: true,
				Description: "The status of the resource.",
			},
			"message": schema.StringAttribute{
				Optional: true,
				Description: "The message of the resource.",
			},
			"errors": schema.StringAttribute{
				Optional: true,
				Description: "The errors of the resource.",
			},
			"details": schema.StringAttribute{
				Optional: true,
				Description: "The details of the resource.",
			},
			"ip": schema.StringAttribute{
				Optional: true,
				Description: "The ip of the resource.",
			},
		},
		MarkdownDescription: "Manages allowlist in JFrog Platform.",
	}
}

func (r *JmisAllowlistResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.ProviderData = req.ProviderData.(util.ProviderMetadata)
}


func (r *JmisAllowlistResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	go util.SendUsageResourceCreate(ctx, r.ProviderData.Client.R(), r.ProviderData.ProductId, r.TypeName)

	var plan JmisAllowlistResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	requestBody := JmisAllowlistRequestAPIModel{
		ServerName: plan.ServerName.ValueString(),
		Ips: plan.Ips.ValueString(),
		Status: plan.Status.ValueString(),
		Message: plan.Message.ValueString(),
		Errors: plan.Errors.ValueString(),
		Details: plan.Details.ValueString(),
		Ip: plan.Ip.ValueString(),
	}

	var result JmisAllowlistAPIModel

	response, err := r.ProviderData.Client.R().
		SetBody(requestBody).
		SetResult(&result).
		Post(AllowlistEndpoint)
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


func (r *JmisAllowlistResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	go util.SendUsageResourceRead(ctx, r.ProviderData.Client.R(), r.ProviderData.ProductId, r.TypeName)

	var state JmisAllowlistResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var result JmisAllowlistAPIModel

	response, err := r.ProviderData.Client.R().
		SetPathParams(map[string]string{
			"serverName": state.ServerName.ValueString(),
		}).
		SetResult(&result).
		Get(AllowlistEndpoint)
	if err != nil {
		utilfw.UnableToRefreshResourceError(resp, err.Error())
		return
	}

	if response.StatusCode() == http.StatusNotFound {
		resp.State.RemoveResource(ctx)
		return
	}

	if response.IsError() {
		utilfw.UnableToRefreshResourceError(resp, response.String())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}




func (r *JmisAllowlistResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	go util.SendUsageResourceDelete(ctx, r.ProviderData.Client.R(), r.ProviderData.ProductId, r.TypeName)

	var state JmisAllowlistResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	response, err := r.ProviderData.Client.R().
		SetPathParams(map[string]string{
			"serverName": state.ServerName.ValueString(),
		}).
		Delete(AllowlistEndpoint)
	if err != nil {
		utilfw.UnableToDeleteResourceError(resp, err.Error())
		return
	}

	if response.StatusCode() == http.StatusNotFound {
		return
	}

	if response.IsError() {
		utilfw.UnableToDeleteResourceError(resp, response.String())
		return
	}
}

