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
	AddsEndpoint = "platform/api/jmis/v1/private-link/add"
	AddsEndpoint  = "platform/api/jmis/v1/private-link/add"
)

func NewPrivateLinkAddResource() resource.Resource {
	return &PrivateLinkAddResource{
		TypeName: "platform_add",
	}
}

type PrivateLinkAddResource struct {
	ProviderData util.ProviderMetadata
	TypeName     string
}

type PrivateLinkAddResourceModel struct {
	PrivateLinkId types.String `tfsdk:"privateLinkId"`
	Results types.String `tfsdk:"results"`
	ServerName types.String `tfsdk:"serverName"`
	PrivateLinkStatus types.String `tfsdk:"privateLinkStatus"`
}

type AddRequestAPIModel struct {
	PrivateLinkId string `json:"privateLinkId"`
	Results string `json:"results"`
	ServerName string `json:"serverName"`
	PrivateLinkStatus string `json:"privateLinkStatus"`
}

type AddAPIModel struct {
	PrivateLinkId string `json:"privateLinkId"`
	Results string `json:"results"`
	ServerName string `json:"serverName"`
	PrivateLinkStatus string `json:"privateLinkStatus"`
}

func (r *PrivateLinkAddResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = r.TypeName
}

func (r *PrivateLinkAddResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"privateLinkId": schema.StringAttribute{
				Optional: true,
				Description: "The privateLinkId of the resource.",
			},
			"results": schema.StringAttribute{
				Optional: true,
				Description: "The results of the resource.",
			},
			"serverName": schema.StringAttribute{
				Optional: true,
				Description: "The serverName of the resource.",
			},
			"privateLinkStatus": schema.StringAttribute{
				Optional: true,
				Description: "The privateLinkStatus of the resource.",
			},
		},
		MarkdownDescription: "Manages add in JFrog Platform.",
	}
}

func (r *PrivateLinkAddResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.ProviderData = req.ProviderData.(util.ProviderMetadata)
}


func (r *PrivateLinkAddResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	go util.SendUsageResourceCreate(ctx, r.ProviderData.Client.R(), r.ProviderData.ProductId, r.TypeName)

	var plan PrivateLinkAddResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	requestBody := PrivateLinkAddRequestAPIModel{
		PrivateLinkId: plan.PrivateLinkId.ValueString(),
		Results: plan.Results.ValueString(),
		ServerName: plan.ServerName.ValueString(),
		PrivateLinkStatus: plan.PrivateLinkStatus.ValueString(),
	}

	var result PrivateLinkAddAPIModel

	response, err := r.ProviderData.Client.R().
		SetBody(requestBody).
		SetResult(&result).
		Post(AddsEndpoint)
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


func (r *PrivateLinkAddResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	go util.SendUsageResourceRead(ctx, r.ProviderData.Client.R(), r.ProviderData.ProductId, r.TypeName)

	var state PrivateLinkAddResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var result PrivateLinkAddAPIModel

	response, err := r.ProviderData.Client.R().
		SetPathParams(map[string]string{
		}).
		SetResult(&result).
		Get(AddsEndpoint)
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




