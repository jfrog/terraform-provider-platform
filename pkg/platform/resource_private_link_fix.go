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
	FixsEndpoint = "platform/api/jmis/v1/private-link/fix"
	FixsEndpoint  = "platform/api/jmis/v1/private-link/fix"
)

func NewPrivateLinkFixResource() resource.Resource {
	return &PrivateLinkFixResource{
		TypeName: "platform_fix",
	}
}

type PrivateLinkFixResource struct {
	ProviderData util.ProviderMetadata
	TypeName     string
}

type PrivateLinkFixResourceModel struct {
	ServerName types.String `tfsdk:"serverName"`
	Results types.String `tfsdk:"results"`
	FromPrivateLinkId types.String `tfsdk:"fromPrivateLinkId"`
	ToPrivateLinkId types.String `tfsdk:"toPrivateLinkId"`
	Status types.String `tfsdk:"status"`
}

type FixRequestAPIModel struct {
	ServerName string `json:"serverName"`
	Results string `json:"results"`
	FromPrivateLinkId string `json:"fromPrivateLinkId"`
	ToPrivateLinkId string `json:"toPrivateLinkId"`
	Status string `json:"status"`
}

type FixAPIModel struct {
	ServerName string `json:"serverName"`
	Results string `json:"results"`
	FromPrivateLinkId string `json:"fromPrivateLinkId"`
	ToPrivateLinkId string `json:"toPrivateLinkId"`
	Status string `json:"status"`
}

func (r *PrivateLinkFixResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = r.TypeName
}

func (r *PrivateLinkFixResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"serverName": schema.StringAttribute{
				Optional: true,
				Description: "The serverName of the resource.",
			},
			"results": schema.StringAttribute{
				Optional: true,
				Description: "The results of the resource.",
			},
			"fromPrivateLinkId": schema.StringAttribute{
				Optional: true,
				Description: "The fromPrivateLinkId of the resource.",
			},
			"toPrivateLinkId": schema.StringAttribute{
				Optional: true,
				Description: "The toPrivateLinkId of the resource.",
			},
			"status": schema.StringAttribute{
				Optional: true,
				Description: "The status of the resource.",
			},
		},
		MarkdownDescription: "Manages fix in JFrog Platform.",
	}
}

func (r *PrivateLinkFixResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.ProviderData = req.ProviderData.(util.ProviderMetadata)
}


func (r *PrivateLinkFixResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	go util.SendUsageResourceCreate(ctx, r.ProviderData.Client.R(), r.ProviderData.ProductId, r.TypeName)

	var plan PrivateLinkFixResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	requestBody := PrivateLinkFixRequestAPIModel{
		ServerName: plan.ServerName.ValueString(),
		Results: plan.Results.ValueString(),
		FromPrivateLinkId: plan.FromPrivateLinkId.ValueString(),
		ToPrivateLinkId: plan.ToPrivateLinkId.ValueString(),
		Status: plan.Status.ValueString(),
	}

	var result PrivateLinkFixAPIModel

	response, err := r.ProviderData.Client.R().
		SetBody(requestBody).
		SetResult(&result).
		Post(FixsEndpoint)
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


func (r *PrivateLinkFixResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	go util.SendUsageResourceRead(ctx, r.ProviderData.Client.R(), r.ProviderData.ProductId, r.TypeName)

	var state PrivateLinkFixResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var result PrivateLinkFixAPIModel

	response, err := r.ProviderData.Client.R().
		SetPathParams(map[string]string{
		}).
		SetResult(&result).
		Get(FixsEndpoint)
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




