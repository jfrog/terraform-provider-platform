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

const FullBroadcastEndpoint = "access/api/v1/system/federation/{federation_target_servername}/full_broadcast"

func NewFederationFullBroadcastResource() resource.Resource {
	return &FederationFullBroadcastResource{
		TypeName: "platform_federation_full_broadcast",
	}
}

type FederationFullBroadcastResource struct {
	ProviderData util.ProviderMetadata
	TypeName     string
}

type FederationFullBroadcastResourceModel struct {
	FederationTargetServername types.String `tfsdk:"federation_target_servername"`
}

func (r *FederationFullBroadcastResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = r.TypeName
}

func (r *FederationFullBroadcastResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"federation_target_servername": schema.StringAttribute{
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Description: "The server name of the federation target to invoke full broadcast on.",
			},
		},
		MarkdownDescription: "Invokes [Access Federation full broadcast](https://jfrog.com/help/r/jfrog-rest-apis/full-broadcast-access-federation) " +
			"from a single federation target. This triggers a full synchronization of all federated entities " +
			"to the specified target.\n\n" +
			"~>This resource is an action trigger. Creating it invokes the full broadcast. " +
			"Destroying it only removes the resource from state without any API call.",
	}
}

func (r *FederationFullBroadcastResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.ProviderData = req.ProviderData.(util.ProviderMetadata)
}

func (r *FederationFullBroadcastResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	go util.SendUsageResourceCreate(ctx, r.ProviderData.Client.R(), r.ProviderData.ProductId, r.TypeName)

	var plan FederationFullBroadcastResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	response, err := r.ProviderData.Client.R().
		SetPathParams(map[string]string{
			"federation_target_servername": plan.FederationTargetServername.ValueString(),
		}).
		Put(FullBroadcastEndpoint)
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

func (r *FederationFullBroadcastResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	go util.SendUsageResourceRead(ctx, r.ProviderData.Client.R(), r.ProviderData.ProductId, r.TypeName)

	var state FederationFullBroadcastResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Full broadcast is an action endpoint with no dedicated GET.
	// Verify the federation target still exists using the federation endpoint.
	response, err := r.ProviderData.Client.R().
		SetPathParams(map[string]string{
			"server_name": state.FederationTargetServername.ValueString(),
		}).
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

func (r *FederationFullBroadcastResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// federation_target_servername has RequiresReplace, so Update is never called.
}

func (r *FederationFullBroadcastResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	go util.SendUsageResourceDelete(ctx, r.ProviderData.Client.R(), r.ProviderData.ProductId, r.TypeName)
	// Action-only resource: nothing to delete on the server side.
	// Removing from state is handled automatically by the framework.
}
