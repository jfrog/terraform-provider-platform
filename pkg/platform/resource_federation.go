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
	FederationsEndpoint = "mc/api/v1/federation?includeNonConfiguredJPDs=false"
	FederationEndpoint  = "mc/api/v1/federation/{JPD"
)

func NewFederationResource() resource.Resource {
	return &FederationResource{
		TypeName: "platform_federation",
	}
}

type FederationResource struct {
	ProviderData util.ProviderMetadata
	TypeName     string
}

type FederationResourceModel struct {
	Entities types.String `tfsdk:"entities"`
	Targets types.String `tfsdk:"targets"`
	Name types.String `tfsdk:"name"`
	Code types.String `tfsdk:"code"`
	Url types.String `tfsdk:"url"`
	Source types.String `tfsdk:"source"`
	Label types.String `tfsdk:"label"`
	Status types.String `tfsdk:"status"`
}

type FederationRequestAPIModel struct {
	Entities string `json:"entities"`
	Targets string `json:"targets"`
	Name string `json:"name"`
	Code string `json:"code"`
	Url string `json:"url"`
	Source string `json:"source"`
	Label string `json:"label"`
	Status string `json:"status"`
}

type FederationAPIModel struct {
	Entities string `json:"entities"`
	Targets string `json:"targets"`
	Name string `json:"name"`
	Code string `json:"code"`
	Url string `json:"url"`
	Source string `json:"source"`
	Label string `json:"label"`
	Status string `json:"status"`
}

func (r *FederationResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = r.TypeName
}

func (r *FederationResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"entities": schema.StringAttribute{
				Optional: true,
				Description: "The entities of the resource.",
			},
			"targets": schema.StringAttribute{
				Optional: true,
				Description: "The targets of the resource.",
			},
			"name": schema.StringAttribute{
				Optional: true,
				Description: "The name of the resource.",
			},
			"code": schema.StringAttribute{
				Optional: true,
				Description: "The code of the resource.",
			},
			"url": schema.StringAttribute{
				Optional: true,
				Description: "The url of the resource.",
			},
			"source": schema.StringAttribute{
				Optional: true,
				Description: "The source of the resource.",
			},
			"label": schema.StringAttribute{
				Optional: true,
				Description: "The label of the resource.",
			},
			"status": schema.StringAttribute{
				Optional: true,
				Description: "The status of the resource.",
			},
		},
		MarkdownDescription: "Manages federation in JFrog Platform.",
	}
}

func (r *FederationResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.ProviderData = req.ProviderData.(util.ProviderMetadata)
}



func (r *FederationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	go util.SendUsageResourceRead(ctx, r.ProviderData.Client.R(), r.ProviderData.ProductId, r.TypeName)

	var state FederationResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var result FederationAPIModel

	response, err := r.ProviderData.Client.R().
		SetPathParams(map[string]string{
		}).
		SetResult(&result).
		Get(FederationEndpoint)
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


func (r *FederationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	go util.SendUsageResourceUpdate(ctx, r.ProviderData.Client.R(), r.ProviderData.ProductId, r.TypeName)

	var plan FederationResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	requestBody := FederationRequestAPIModel{
		Entities: plan.Entities.ValueString(),
		Targets: plan.Targets.ValueString(),
		Name: plan.Name.ValueString(),
		Code: plan.Code.ValueString(),
		Url: plan.Url.ValueString(),
		Source: plan.Source.ValueString(),
		Label: plan.Label.ValueString(),
		Status: plan.Status.ValueString(),
	}

	response, err := r.ProviderData.Client.R().
		SetPathParams(map[string]string{
		}).
		SetBody(requestBody).
		Put(FederationEndpoint)
	if err != nil {
		utilfw.UnableToUpdateResourceError(resp, err.Error())
		return
	}

	if response.IsError() {
		utilfw.UnableToUpdateResourceError(resp, response.String())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}



