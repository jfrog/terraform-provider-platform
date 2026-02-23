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
	ServiceProviderConfigssEndpoint = "access/api/v1/scim/v2/ServiceProviderConfigs"
	ServiceProviderConfigssEndpoint  = "access/api/v1/scim/v2/ServiceProviderConfigs"
)

func NewScimServiceProviderConfigsResource() resource.Resource {
	return &ScimServiceProviderConfigsResource{
		TypeName: "platform_service_provider_configs",
	}
}

type ScimServiceProviderConfigsResource struct {
	ProviderData util.ProviderMetadata
	TypeName     string
}

type ScimServiceProviderConfigsResourceModel struct {
}

type ServiceProviderConfigsAPIModel struct {
}

func (r *ScimServiceProviderConfigsResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = r.TypeName
}

func (r *ScimServiceProviderConfigsResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
		},
		MarkdownDescription: "Manages ServiceProviderConfigs in JFrog Platform.",
	}
}

func (r *ScimServiceProviderConfigsResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.ProviderData = req.ProviderData.(util.ProviderMetadata)
}



func (r *ScimServiceProviderConfigsResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	go util.SendUsageResourceRead(ctx, r.ProviderData.Client.R(), r.ProviderData.ProductId, r.TypeName)

	var state ScimServiceProviderConfigsResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var result ScimServiceProviderConfigsAPIModel

	response, err := r.ProviderData.Client.R().
		SetPathParams(map[string]string{
		}).
		SetResult(&result).
		Get(ServiceProviderConfigssEndpoint)
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




