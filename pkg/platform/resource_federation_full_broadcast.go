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
	FullBroadcastEndpoint = "access/api/v1/system/federation/{federation_target_servername}/full_broadcast"
	FullBroadcastEndpoint  = "access/api/v1/system/federation/{federation_target_servername}/full_broadcast"
)

func NewFederationFullBroadcastResource() resource.Resource {
	return &FederationFullBroadcastResource{
		TypeName: "platform_full_broadcast",
	}
}

type FederationFullBroadcastResource struct {
	ProviderData util.ProviderMetadata
	TypeName     string
}

type FederationFullBroadcastResourceModel struct {
	FederationTargetServername types.String `tfsdk:"federation_target_servername"`
}

type FullBroadcastRequestAPIModel struct {
	FederationTargetServername string `json:"federation_target_servername"`
}

type FullBroadcastAPIModel struct {
	FederationTargetServername string `json:"federation_target_servername"`
}

func (r *FederationFullBroadcastResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = r.TypeName
}

func (r *FederationFullBroadcastResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"federation_target_servername": schema.StringAttribute{
				Required: true,
				Description: "The federation_target_servername of the resource.",
			},
		},
		MarkdownDescription: "Manages full_broadcast in JFrog Platform.",
	}
}

func (r *FederationFullBroadcastResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.ProviderData = req.ProviderData.(util.ProviderMetadata)
}



func (r *FederationFullBroadcastResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	go util.SendUsageResourceRead(ctx, r.ProviderData.Client.R(), r.ProviderData.ProductId, r.TypeName)

	var state FederationFullBroadcastResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var result FederationFullBroadcastAPIModel

	response, err := r.ProviderData.Client.R().
		SetPathParams(map[string]string{
			"federation_target_servername": state.FederationTargetServername.ValueString(),
		}).
		SetResult(&result).
		Get(FullBroadcastEndpoint)
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


func (r *FederationFullBroadcastResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	go util.SendUsageResourceUpdate(ctx, r.ProviderData.Client.R(), r.ProviderData.ProductId, r.TypeName)

	var plan FederationFullBroadcastResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	requestBody := FederationFullBroadcastRequestAPIModel{
		FederationTargetServername: plan.FederationTargetServername.ValueString(),
	}

	response, err := r.ProviderData.Client.R().
		SetPathParams(map[string]string{
			"federation_target_servername": plan.FederationTargetServername.ValueString(),
		}).
		SetBody(requestBody).
		Put(FullBroadcastEndpoint)
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



