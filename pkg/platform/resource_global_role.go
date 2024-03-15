package platform

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/jfrog/terraform-provider-shared/util"
	utilfw "github.com/jfrog/terraform-provider-shared/util/fw"
)

const (
	globalRolePostEndpoint = "/access/api/v1/roles"
	globalRoleGetEndpoint  = "/access/api/v1/roles/{name}"
)

var globalRoleTypes []string = []string{"ADMIN", "CUSTOM_GLOBAL", "PREDEFINED"}
var globalRoleActions []string = []string{
	"READ_REPOSITORY",
	"ANNOTATE_REPOSITORY",
	"DEPLOY_CACHE_REPOSITORY",
	"DELETE_OVERWRITE_REPOSITORY",
	"MANAGE_XRAY_MD_REPOSITORY",
	"READ_RELEASE_BUNDLE",
	"ANNOTATE_RELEASE_BUNDLE",
	"CREATE_RELEASE_BUNDLE",
	"DISTRIBUTE_RELEASE_BUNDLE",
	"DELETE_RELEASE_BUNDLE",
	"MANAGE_XRAY_MD_RELEASE_BUNDLE",
	"READ_BUILD",
	"ANNOTATE_BUILD",
	"DEPLOY_BUILD",
	"DELETE_BUILD",
	"MANAGE_XRAY_MD_BUILD",
	"READ_SOURCES_PIPELINE",
	"TRIGGER_PIPELINE",
	"READ_INTEGRATIONS_PIPELINE",
	"READ_POOLS_PIPELINE",
	"REPORTS_SECURITY",
	"WATCHES_SECURITY",
	"POLICIES_SECURITY",
	"RULES_SECURITY",
	"READ_POLICIES_SECURITY",
}

var _ resource.Resource = (*globalRoleResource)(nil)

type globalRoleResource struct {
	ProviderData util.ProvderMetadata
}

func NewGlobalRoleResource() resource.Resource {
	return &globalRoleResource{}
}

func (r *globalRoleResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_global_role"
}

func (r *globalRoleResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Required: true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
				Description: "Name of the role",
			},
			"description": schema.StringAttribute{
				Optional:    true,
				Description: "Description of the role",
			},
			"type": schema.StringAttribute{
				Required: true,
				Validators: []validator.String{
					stringvalidator.OneOf(globalRoleTypes...),
				},
				Description: fmt.Sprintf("Type of the role. Allowed values: %s", strings.Join(globalRoleTypes, ", ")),
			},
			"environments": schema.SetAttribute{
				ElementType: types.StringType,
				Required:    true,
				Validators: []validator.Set{
					setvalidator.SizeAtLeast(1),
				},
				PlanModifiers: []planmodifier.Set{
					setplanmodifier.UseStateForUnknown(),
				},
				Description: "List of global or custom environments. A repository can be available in different environments. Members with roles defined in the set environment will have access to the repository.",
			},
			"actions": schema.SetAttribute{
				ElementType: types.StringType,
				Required:    true,
				Validators: []validator.Set{
					setvalidator.SizeAtLeast(1),
					setvalidator.ValueStringsAre(stringvalidator.OneOf(globalRoleActions...)),
				},
				PlanModifiers: []planmodifier.Set{
					setplanmodifier.UseStateForUnknown(),
				},
				Description: fmt.Sprintf("List of actions. Allowed values: %s", strings.Join(globalRoleActions, ", ")),
			},
		},
		MarkdownDescription: "Provides a JFrog [global role](https://jfrog.com/help/r/jfrog-platform-administration-documentation/global-and-project-role-types) resource to manage custom global roles.",
	}
}

type globalRoleResourceModel struct {
	Name         types.String `tfsdk:"name"`
	Description  types.String `tfsdk:"description"`
	Type         types.String `tfsdk:"type"`
	Environments types.Set    `tfsdk:"environments"`
	Actions      types.Set    `tfsdk:"actions"`
}

func (r *globalRoleResourceModel) toAPIModel(ctx context.Context, apiModel *globalRoleAPIModel) (ds diag.Diagnostics) {
	var environments []string
	ds.Append(r.Environments.ElementsAs(ctx, &environments, false)...)
	if ds.HasError() {
		return
	}

	var actions []string
	ds.Append(r.Actions.ElementsAs(ctx, &actions, false)...)
	if ds.HasError() {
		return
	}

	*apiModel = globalRoleAPIModel{
		Name:         r.Name.ValueString(),
		Description:  r.Description.ValueString(),
		Type:         r.Type.ValueString(),
		Environments: environments,
		Actions:      actions,
	}

	return nil
}

func (r *globalRoleResourceModel) fromAPIModel(ctx context.Context, apiModel *globalRoleAPIModel) (ds diag.Diagnostics) {
	r.Name = types.StringValue(apiModel.Name)
	r.Description = types.StringValue(apiModel.Description)
	r.Type = types.StringValue(apiModel.Type)

	environments, d := types.SetValueFrom(ctx, types.StringType, apiModel.Environments)
	if d != nil {
		ds = append(ds, d...)
	}
	if ds.HasError() {
		return
	}
	r.Environments = environments

	actions, d := types.SetValueFrom(ctx, types.StringType, apiModel.Actions)
	if d != nil {
		ds = append(ds, d...)
	}
	if ds.HasError() {
		return
	}
	r.Actions = actions

	return
}

type globalRoleAPIModel struct {
	Name         string   `json:"name"`
	Description  string   `json:"description"`
	Type         string   `json:"type"`
	Environments []string `json:"environments"`
	Actions      []string `json:"actions"`
}

func (r *globalRoleResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}
	r.ProviderData = req.ProviderData.(util.ProvderMetadata)
}

func (r *globalRoleResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan globalRoleResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var role globalRoleAPIModel
	resp.Diagnostics.Append(plan.toAPIModel(ctx, &role)...)
	if resp.Diagnostics.HasError() {
		return
	}

	response, err := r.ProviderData.Client.R().
		SetBody(&role).
		Post(globalRolePostEndpoint)
	if err != nil {
		utilfw.UnableToCreateResourceError(resp, err.Error())
		return
	}

	// Return error if the HTTP status code is not 201 Created
	if response.StatusCode() != http.StatusCreated {
		utilfw.UnableToCreateResourceError(resp, response.String())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *globalRoleResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state globalRoleResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var role globalRoleAPIModel

	response, err := r.ProviderData.Client.R().
		SetPathParam("name", state.Name.ValueString()).
		SetResult(&role).
		Get(globalRoleGetEndpoint)

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
	resp.Diagnostics.Append(state.fromAPIModel(ctx, &role)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *globalRoleResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan globalRoleResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var role globalRoleAPIModel
	resp.Diagnostics.Append(plan.toAPIModel(ctx, &role)...)
	if resp.Diagnostics.HasError() {
		return
	}

	response, err := r.ProviderData.Client.R().
		SetPathParam("name", plan.Name.ValueString()).
		SetBody(&role).
		Put(globalRoleGetEndpoint)
	if err != nil {
		utilfw.UnableToUpdateResourceError(resp, err.Error())
		return
	}

	// Return error if the HTTP status code is not 200 OK
	if response.StatusCode() != http.StatusOK {
		utilfw.UnableToUpdateResourceError(resp, response.String())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *globalRoleResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state globalRoleResourceModel

	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	response, err := r.ProviderData.Client.R().
		SetPathParam("name", state.Name.ValueString()).
		Delete(globalRoleGetEndpoint)
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

func (r *globalRoleResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("name"), req, resp)
}
