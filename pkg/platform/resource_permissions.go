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
	PermissionsEndpoint = "access/api/v2/permissions/{permission_name}/{resource_type}"
	PermissionsEndpoint  = "access/api/v2/permissions/{permission_name}/{resource_type}"
)

func NewPermissionsResource() resource.Resource {
	return &PermissionsResource{
		TypeName: "platform_permissions",
	}
}

type PermissionsResource struct {
	ProviderData util.ProviderMetadata
	TypeName     string
}

type PermissionsResourceModel struct {
	PermissionName types.String `tfsdk:"permission_name"`
	ResourceType types.String `tfsdk:"resource_type"`
}

type PermissionsAPIModel struct {
	PermissionName string `json:"permission_name"`
	ResourceType string `json:"resource_type"`
}

func (r *PermissionsResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = r.TypeName
}

func (r *PermissionsResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"permission_name": schema.StringAttribute{
				Required: true,
				Description: "The permission_name of the resource.",
			},
			"resource_type": schema.StringAttribute{
				Required: true,
				Description: "The resource_type of the resource.",
			},
		},
		MarkdownDescription: "Manages permissions in JFrog Platform.",
	}
}

func (r *PermissionsResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.ProviderData = req.ProviderData.(util.ProviderMetadata)
}



func (r *PermissionsResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	go util.SendUsageResourceRead(ctx, r.ProviderData.Client.R(), r.ProviderData.ProductId, r.TypeName)

	var state PermissionsResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var result PermissionsAPIModel

	response, err := r.ProviderData.Client.R().
		SetPathParams(map[string]string{
			"permission_name": state.PermissionName.ValueString(),
			"resource_type": state.ResourceType.ValueString(),
		}).
		SetResult(&result).
		Get(PermissionsEndpoint)
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


func (r *PermissionsResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	go util.SendUsageResourceUpdate(ctx, r.ProviderData.Client.R(), r.ProviderData.ProductId, r.TypeName)

	var plan PermissionsResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	requestBody := PermissionsRequestAPIModel{
		PermissionName: plan.PermissionName.ValueString(),
		ResourceType: plan.ResourceType.ValueString(),
	}

	response, err := r.ProviderData.Client.R().
		SetPathParams(map[string]string{
			"permission_name": plan.PermissionName.ValueString(),
			"resource_type": plan.ResourceType.ValueString(),
		}).
		SetBody(requestBody).
		Patch(PermissionsEndpoint)
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



