package platform

import (
	"context"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/jfrog/terraform-provider-shared/util"
	utilfw "github.com/jfrog/terraform-provider-shared/util/fw"
)

const odicIdentityMappingEndpoint = "/access/api/v1/oidc/{provider_name}/identity_mappings"

var _ resource.Resource = (*odicIdentityMappingResource)(nil)

type odicIdentityMappingResource struct {
	ProviderData util.ProvderMetadata
}

func NewOIDCIdentityMappingResource() resource.Resource {
	return &odicIdentityMappingResource{}
}

func (r *odicIdentityMappingResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_oidc_identity_mapping"
}
func (r *odicIdentityMappingResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Required: true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 255),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Description: "Name of the OIDC identity mapping",
			},
			"description": schema.StringAttribute{
				Optional:    true,
				Description: "Description of the OIDC mapping",
			},
			"provider_name": schema.StringAttribute{
				Required:    true,
				Description: "Name of the OIDC configuration",
			},
			"priority": schema.Int64Attribute{
				Optional:    true,
				Description: "Priority of the identity mapping. The priority should be a number. The higher priority is set for the lower number. If you do not enter a value, the identity mapping is assigned the lowest priority. We recommend that you assign the highest priority (1) to the strongest permission gate. Set the lowest priority to the weakest permission for a logical and effective access control setup.",
			},
			"claims": schema.SingleNestedAttribute{
				Required: true,
				Attributes: map[string]schema.Attribute{
					"sub": schema.StringAttribute{
						Required: true,
					},
					"workflow_ref": schema.StringAttribute{
						Required: true,
					},
				},
				Description: "Claims information from the OIDC provider.",
			},
			"token_spec": schema.SingleNestedAttribute{
				Required: true,
				Attributes: map[string]schema.Attribute{
					"username": schema.StringAttribute{
						Required:    true,
						Description: "User name of the OIDC user.",
					},
					"scope": schema.StringAttribute{
						Required: true,
						Validators: []validator.String{
							stringvalidator.OneOf([]string{
								"applied-permissions/user",
								"applied-permissions/admin",
								"applied-permissions/group",
							}...),
						},
						MarkdownDescription: "Scope of the token. You can use `applied-permissions/user`, `applied-permissions/admin`, or `applied-permissions/group`.",
					},
					"audience": schema.StringAttribute{
						Optional:            true,
						Computed:            true,
						Default:             stringdefault.StaticString("*@*"),
						MarkdownDescription: "Sets of (space separated) the JFrog services to which the mapping applies. Default value is `*@*`, which applies to all services.",
					},
					"expires_in": schema.Int64Attribute{
						Optional:            true,
						Computed:            true,
						Default:             int64default.StaticInt64(60),
						MarkdownDescription: "Token expiry time in seconds. Default value is 60.",
					},
				},
				Description: "Specifications of the token.",
			},
		},
		MarkdownDescription: "Resource of identity mapping for an OIDC configuration.  JFrog [OIDC identity mappings documentation](https://jfrog.com/help/r/jfrog-platform-administration-documentation/configure-identity-mappings).",
	}
}

type odicIdentityMappingResourceModel struct {
	Name         types.String `tfsdk:"name"`
	Description  types.String `tfsdk:"description"`
	ProviderName types.String `tfsdk:"provider_name"`
	Priority     types.Int64  `tfsdk:"priority"`
	Claims       types.Object `tfsdk:"claims"`
	TokenSpec    types.Object `tfsdk:"token_spec"`
}

type odicIdentityMappingClaimsResourceModel struct {
	Sub         types.String `tfsdk:"sub"`
	WorkflowRef types.String `tfsdk:"workflow_ref"`
}

var odicIdentityMappingClaimsResourceModelAttributeType map[string]attr.Type = map[string]attr.Type{
	"sub":          types.StringType,
	"workflow_ref": types.StringType,
}

type odicIdentityMappingTokenSpecResourceModel struct {
	Username  types.String `tfsdk:"username"`
	Scope     types.String `tfsdk:"scope"`
	Audience  types.String `tfsdk:"audience"`
	ExpiresIn types.Int64  `tfsdk:"expires_in"`
}

var odicIdentityMappingTokenSpecResourceModelAttributeType map[string]attr.Type = map[string]attr.Type{
	"username":   types.StringType,
	"scope":      types.StringType,
	"audience":   types.StringType,
	"expires_in": types.Int64Type,
}

func (r *odicIdentityMappingResourceModel) toAPIModel(ctx context.Context, apiModel *odicIdentityMappingAPIModel) (ds diag.Diagnostics) {

	var claims odicIdentityMappingClaimsResourceModel
	ds.Append(r.Claims.As(ctx, &claims, basetypes.ObjectAsOptions{})...)
	if ds.HasError() {
		return
	}

	var tokenSpec odicIdentityMappingTokenSpecResourceModel
	ds.Append(r.Claims.As(ctx, &tokenSpec, basetypes.ObjectAsOptions{})...)
	if ds.HasError() {
		return
	}

	*apiModel = odicIdentityMappingAPIModel{
		Name:         r.Name.ValueString(),
		Description:  r.Description.ValueString(),
		ProviderName: r.ProviderName.ValueString(),
		Priority:     r.Priority.ValueInt64(),
		Claims: odicIdentityMappingClaimsAPIModel{
			Sub:         claims.Sub.ValueString(),
			WorkflowRef: claims.WorkflowRef.ValueString(),
		},
		TokenSpec: odicIdentityMappingTokenSpecAPIModel{
			Username:  tokenSpec.Username.ValueString(),
			Scope:     tokenSpec.Scope.ValueString(),
			Audience:  tokenSpec.Audience.ValueString(),
			ExpiresIn: tokenSpec.ExpiresIn.ValueInt64(),
		},
	}

	return
}

func (r *odicIdentityMappingResourceModel) fromAPIModel(ctx context.Context, apiModel *odicIdentityMappingAPIModel) (ds diag.Diagnostics) {
	r.Name = types.StringValue(apiModel.Name)
	r.Description = types.StringValue(apiModel.Description)
	r.ProviderName = types.StringValue(apiModel.ProviderName)
	r.Priority = types.Int64Value(apiModel.Priority)

	claims, d := types.ObjectValueFrom(
		ctx,
		odicIdentityMappingClaimsResourceModelAttributeType,
		apiModel.Claims,
	)
	if d != nil {
		ds = append(ds, d...)
	}
	if ds.HasError() {
		return
	}
	r.Claims = claims

	tokenSpec, d := types.ObjectValueFrom(
		ctx,
		odicIdentityMappingTokenSpecResourceModelAttributeType,
		apiModel.TokenSpec,
	)
	if d != nil {
		ds = append(ds, d...)
	}
	if ds.HasError() {
		return
	}
	r.TokenSpec = tokenSpec

	return
}

type odicIdentityMappingAPIModel struct {
	Name         string                               `json:"name"`
	Description  string                               `json:"description"`
	ProviderName string                               `json:"provider_name"`
	Priority     int64                                `json:"priority"`
	Claims       odicIdentityMappingClaimsAPIModel    `json:"claims"`
	TokenSpec    odicIdentityMappingTokenSpecAPIModel `json:"token_spec"`
}

type odicIdentityMappingClaimsAPIModel struct {
	Sub         string `json:"sub"`
	WorkflowRef string `json:"workflow_ref"`
}

type odicIdentityMappingTokenSpecAPIModel struct {
	Username  string `json:"username"`
	Scope     string `json:"scope"`
	Audience  string `json:"audience"`
	ExpiresIn int64  `json:"expires_in"`
}

func (r *odicIdentityMappingResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan odicIdentityMappingResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var odicIdentityMapping odicIdentityMappingAPIModel
	resp.Diagnostics.Append(plan.toAPIModel(ctx, &odicIdentityMapping)...)
	if resp.Diagnostics.HasError() {
		return
	}

	response, err := r.ProviderData.Client.R().
		SetPathParam("provider_name", plan.ProviderName.ValueString()).
		SetBody(&odicIdentityMapping).
		Post(odicIdentityMappingEndpoint)
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

func (r *odicIdentityMappingResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state odicIdentityMappingResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var odicIdentityMapping odicIdentityMappingAPIModel

	response, err := r.ProviderData.Client.R().
		SetPathParams(map[string]string{
			"provider_name": state.ProviderName.ValueString(),
			"name":          state.Name.ValueString(),
		}).
		SetResult(&odicIdentityMapping).
		Get(odicIdentityMappingEndpoint + "/{name}")

	if err != nil {
		utilfw.UnableToRefreshResourceError(resp, err.Error())
		return
	}

	// Treat HTTP 404 Not Found status as a signal to recreate resource
	// and return early
	if response.StatusCode() == http.StatusNotFound {
		resp.State.RemoveResource(ctx)
		return
	}

	if response.IsError() {
		utilfw.UnableToRefreshResourceError(resp, response.String())
		return
	}

	// Convert from the API data model to the Terraform data model
	// and refresh any attribute values.
	resp.Diagnostics.Append(state.fromAPIModel(ctx, &odicIdentityMapping)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *odicIdentityMappingResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan odicIdentityMappingResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var odicIdentityMapping odicIdentityMappingAPIModel
	resp.Diagnostics.Append(plan.toAPIModel(ctx, &odicIdentityMapping)...)
	if resp.Diagnostics.HasError() {
		return
	}

	response, err := r.ProviderData.Client.R().
		SetPathParams(map[string]string{
			"provider_name": plan.ProviderName.ValueString(),
			"name":          plan.Name.ValueString(),
		}).
		SetBody(&odicIdentityMapping).
		Put(odicIdentityMappingEndpoint + "{name}")
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

func (r *odicIdentityMappingResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state odicIdentityMappingResourceModel

	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	response, err := r.ProviderData.Client.R().
		SetPathParams(map[string]string{
			"provider_name": state.ProviderName.ValueString(),
			"name":          state.Name.ValueString(),
		}).
		Delete(odicIdentityMappingEndpoint + "/{name}")
	if err != nil {
		utilfw.UnableToDeleteResourceError(resp, err.Error())
		return
	}

	if response.StatusCode() != http.StatusNoContent {
		utilfw.UnableToDeleteResourceError(resp, response.String())
		return
	}

	// If the logic reaches here, it implicitly succeeded and will remove
	// the resource from state if there are no other errors.
}

func (r *odicIdentityMappingResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("name"), req, resp)
}
