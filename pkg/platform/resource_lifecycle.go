// Copyright (c) JFrog Ltd. (2025)
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package platform

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/jfrog/terraform-provider-shared/util"
	utilfw "github.com/jfrog/terraform-provider-shared/util/fw"
)

var _ resource.Resource = (*lifecycleResource)(nil)

// minAccessVersionLifecycle is the minimum Access version required for lifecycle and lifecycle_stage resources.
const minAccessVersionLifecycle = "7.155.0"

// lifecycleEndpoint is used for both collection and document; the lifecycle API uses the same path for both.
const lifecycleEndpoint = "access/api/v2/lifecycle"

type lifecycleResource struct {
	util.JFrogResource
	ProviderData util.ProviderMetadata
}

func NewLifecycleResource() resource.Resource {
	return &lifecycleResource{
		JFrogResource: util.JFrogResource{
			TypeName:           "platform_lifecycle",
			CollectionEndpoint: lifecycleEndpoint,
		},
	}
}

func (r *lifecycleResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}
	r.ProviderData = req.ProviderData.(util.ProviderMetadata)
}

func (r *lifecycleResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	if r.ProviderData.AccessVersion == "" {
		return
	}
	ok, err := util.CheckVersion(r.ProviderData.AccessVersion, minAccessVersionLifecycle)
	if err != nil {
		resp.Diagnostics.AddError("Failed to verify Access version", err.Error())
		return
	}
	if !ok {
		resp.Diagnostics.AddError(
			"Incompatible Access version",
			fmt.Sprintf("This resource is only supported by Access version %s or later.", minAccessVersionLifecycle),
		)
		return
	}
}

// globalStageValidator validates that global stages (PR, COMMIT, PROD) are not included in promote_stages
type globalStageValidator struct{}

func (v globalStageValidator) Description(ctx context.Context) string {
	return "Validates that global stages (PR, COMMIT, PROD) are not included in promote_stages"
}

func (v globalStageValidator) MarkdownDescription(ctx context.Context) string {
	return "Validates that global stages (PR, COMMIT, PROD) are not included in promote_stages. These stages are system-managed and cannot be assigned to the 'promote' category."
}

func (v globalStageValidator) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	// Skip validation if value is null or unknown
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}

	stageName := req.ConfigValue.ValueString()
	stageNameUpper := strings.ToUpper(stageName)

	// Check if this is a global stage
	globalStages := []string{"PR", "COMMIT", "PROD"}
	for _, globalStage := range globalStages {
		if stageNameUpper == globalStage {
			resp.Diagnostics.AddAttributeError(
				req.Path,
				"Invalid Stage for Promote Category",
				fmt.Sprintf(
					"Stage '%s' is a global system-managed stage and cannot be included in 'promote_stages'. "+
						"Global stages (PR, COMMIT, PROD) are automatically managed by the system. "+
						"PROD is always present as the release stage and cannot be assigned to the 'promote' category.",
					stageName,
				),
			)
			return
		}
	}
}

func (r *lifecycleResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"project_key": schema.StringAttribute{
				Optional: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				MarkdownDescription: "The project key for which to manage the lifecycle. If not set, manages the global lifecycle.",
			},
			"promote_stages": schema.ListAttribute{
				ElementType: types.StringType,
				Required:    true,
				PlanModifiers: []planmodifier.List{
					listplanmodifier.UseStateForUnknown(),
				},
				Validators: []validator.List{
					listvalidator.ValueStringsAre(
						globalStageValidator{},
					),
				},
				MarkdownDescription: "The new, ordered list of stage names that comprise the lifecycle. Global stages, such as PR, COMMIT, and PROD, cannot be modified and should not be included in the request.",
			},
			"release_stage": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Name of the release stage (for example, PROD).",
			},
			"categories": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"category": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "The category name (code or promote).",
						},
						"stages": schema.ListNestedAttribute{
							Computed: true,
							NestedObject: schema.NestedAttributeObject{
								Attributes: map[string]schema.Attribute{
									"name": schema.StringAttribute{
										Computed:            true,
										MarkdownDescription: "The stage name (for example, DEV or QA).",
									},
									"scope": schema.StringAttribute{
										Computed:            true,
										MarkdownDescription: "The scope at which the stage exists (global or project).",
									},
								},
							},
							MarkdownDescription: "An ordered list of stages within a particular category.",
						},
					},
				},
				MarkdownDescription: "An ordered list of lifecycle categories and stages.",
			},
		},
		MarkdownDescription: "Provides a lifecycle resource to manage the lifecycle configuration for a project or globally. The lifecycle defines the ordered stages through which software progresses. See [JFrog documentation](https://jfrog.com/help/r/jfrog-platform-administration-documentation/stages-lifecycle) for more details.",
	}
}

type lifecycleResourceModel struct {
	ProjectKey    types.String `tfsdk:"project_key"`
	PromoteStages types.List   `tfsdk:"promote_stages"`
	ReleaseStage  types.String `tfsdk:"release_stage"`
	Categories    types.List   `tfsdk:"categories"`
}

type lifecycleCategoryModel struct {
	Category types.String `tfsdk:"category"`
	Stages   types.List   `tfsdk:"stages"`
}

type lifecycleStageModel struct {
	Name  types.String `tfsdk:"name"`
	Scope types.String `tfsdk:"scope"`
}

func (r *lifecycleResourceModel) toAPIModel(ctx context.Context, apiModel *lifecycleAPIModel) (ds diag.Diagnostics) {
	var promoteStages []string
	ds.Append(r.PromoteStages.ElementsAs(ctx, &promoteStages, false)...)
	if ds.HasError() {
		return
	}

	*apiModel = lifecycleAPIModel{
		ProjectKey:    r.ProjectKey.ValueStringPointer(),
		PromoteStages: promoteStages,
	}

	return nil
}

var lifecycleStageObjectType = types.ObjectType{
	AttrTypes: map[string]attr.Type{
		"name":  types.StringType,
		"scope": types.StringType,
	},
}

var lifecycleCategoryObjectType = types.ObjectType{
	AttrTypes: map[string]attr.Type{
		"category": types.StringType,
		"stages": types.ListType{
			ElemType: lifecycleStageObjectType,
		},
	},
}

func (r *lifecycleResourceModel) fromAPIModel(ctx context.Context, apiModel lifecycleAPIModel) diag.Diagnostics {
	diags := diag.Diagnostics{}

	r.ReleaseStage = types.StringValue(apiModel.ReleaseStage)

	// Extract promote_stages from categories or use PromoteStages if available
	var promoteStages []string
	if len(apiModel.PromoteStages) > 0 {
		// Use PromoteStages if directly provided (e.g., from API response)
		promoteStages = apiModel.PromoteStages
	} else if len(apiModel.Lifecycle) > 0 {
		// Extract promote stages from categories
		for _, cat := range apiModel.Lifecycle {
			if cat.Category == "promote" {
				for _, stage := range cat.Stages {
					promoteStages = append(promoteStages, stage.Name)
				}
				break // Only one promote category expected
			}
		}
	}

	// Set promote_stages in the model
	if len(promoteStages) > 0 {
		promoteStagesList, d := types.ListValueFrom(ctx, types.StringType, promoteStages)
		if d != nil {
			diags.Append(d...)
			return diags
		}
		r.PromoteStages = promoteStagesList
	} else {
		// Empty list if no promote stages
		emptyList, d := types.ListValue(types.StringType, []attr.Value{})
		if d != nil {
			diags.Append(d...)
			return diags
		}
		r.PromoteStages = emptyList
	}

	// Convert lifecycle categories
	if len(apiModel.Lifecycle) > 0 {
		categoryValues := make([]attr.Value, 0, len(apiModel.Lifecycle))
		for _, cat := range apiModel.Lifecycle {
			// Convert stages
			stageValues := make([]attr.Value, 0, len(cat.Stages))
			for _, stage := range cat.Stages {
				stageObj, d := types.ObjectValue(
					lifecycleStageObjectType.AttrTypes,
					map[string]attr.Value{
						"name":  types.StringValue(stage.Name),
						"scope": types.StringValue(stage.Scope),
					},
				)
				if d != nil {
					diags.Append(d...)
					return diags
				}
				stageValues = append(stageValues, stageObj)
			}

			stagesList := types.ListNull(lifecycleStageObjectType)
			if len(stageValues) > 0 {
				var d diag.Diagnostics
				stagesList, d = types.ListValue(lifecycleStageObjectType, stageValues)
				if d != nil {
					diags.Append(d...)
					return diags
				}
			}

			categoryObj, d := types.ObjectValue(
				lifecycleCategoryObjectType.AttrTypes,
				map[string]attr.Value{
					"category": types.StringValue(cat.Category),
					"stages":   stagesList,
				},
			)
			if d != nil {
				diags.Append(d...)
				return diags
			}
			categoryValues = append(categoryValues, categoryObj)
		}

		lifecycleList, d := types.ListValue(lifecycleCategoryObjectType, categoryValues)
		if d != nil {
			diags.Append(d...)
			return diags
		}
		r.Categories = lifecycleList
	} else {
		r.Categories = types.ListNull(lifecycleCategoryObjectType)
	}

	return diags
}

type lifecycleAPIModel struct {
	ProjectKey    *string                `json:"project_key,omitempty"`
	PromoteStages []string               `json:"promote_stages"`
	ReleaseStage  string                 `json:"release_stage,omitempty"`
	Lifecycle     []lifecycleCategoryAPI `json:"categories,omitempty"`
}

type lifecycleCategoryAPI struct {
	Category string              `json:"category"`
	Stages   []lifecycleStageAPI `json:"stages"`
}

type lifecycleStageAPI struct {
	Name  string `json:"name"`
	Scope string `json:"scope"`
}

func (r *lifecycleResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	go util.SendUsageResourceCreate(ctx, r.ProviderData.Client.R(), r.ProviderData.ProductId, r.TypeName)

	var plan lifecycleResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var lifecycle lifecycleAPIModel
	resp.Diagnostics.Append(plan.toAPIModel(ctx, &lifecycle)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var apiErrs util.JFrogErrors
	request := r.ProviderData.Client.R().
		SetBody(lifecycle).
		SetError(&apiErrs)

	// Add project_key as query parameter if set
	if !plan.ProjectKey.IsNull() && plan.ProjectKey.ValueString() != "" {
		request = request.SetQueryParam("project_key", plan.ProjectKey.ValueString())
	}

	response, err := request.Patch(r.JFrogResource.CollectionEndpoint)

	if err != nil {
		utilfw.UnableToCreateResourceError(resp, err.Error())
		return
	}

	if response.StatusCode() != http.StatusOK {
		errorMsg := apiErrs.String()
		if errorMsg == "" {
			errorMsg = response.String()
		}
		if errorMsg == "" {
			errorMsg = fmt.Sprintf("unexpected status code: %d", response.StatusCode())
		}
		utilfw.UnableToCreateResourceError(resp, errorMsg)
		return
	}

	// Read back the lifecycle to get computed values
	resp.Diagnostics.Append(r.readLifecycle(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *lifecycleResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	go util.SendUsageResourceRead(ctx, r.ProviderData.Client.R(), r.ProviderData.ProductId, r.TypeName)

	var state lifecycleResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(r.readLifecycle(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *lifecycleResource) readLifecycle(ctx context.Context, model *lifecycleResourceModel) diag.Diagnostics {
	var lifecycle lifecycleAPIModel
	var apiErrs util.JFrogErrors
	request := r.ProviderData.Client.R().
		SetResult(&lifecycle).
		SetError(&apiErrs)

	// Add project_key as query parameter if set
	if !model.ProjectKey.IsNull() && model.ProjectKey.ValueString() != "" {
		request = request.SetQueryParam("project_key", model.ProjectKey.ValueString())
	}

	response, err := request.Get(r.JFrogResource.CollectionEndpoint)

	if err != nil {
		return diag.Diagnostics{
			diag.NewErrorDiagnostic(
				"Unable to Read Lifecycle",
				err.Error(),
			),
		}
	}

	if response.StatusCode() == http.StatusNotFound {
		// According to the API, lifecycle must exist - 404 means it truly doesn't exist
		// This is an error for both global and project lifecycles
		return diag.Diagnostics{
			diag.NewErrorDiagnostic(
				"Lifecycle Not Found",
				"The lifecycle was not found. Lifecycles are created automatically when stages are added.",
			),
		}
	}

	if response.IsError() {
		errorMsg := apiErrs.String()
		if errorMsg == "" {
			errorMsg = response.String()
		}
		if errorMsg == "" {
			errorMsg = fmt.Sprintf("unexpected status code: %d", response.StatusCode())
		}
		return diag.Diagnostics{
			diag.NewErrorDiagnostic(
				"Unable to Read Lifecycle",
				errorMsg,
			),
		}
	}

	return model.fromAPIModel(ctx, lifecycle)
}

func (r *lifecycleResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	go util.SendUsageResourceUpdate(ctx, r.ProviderData.Client.R(), r.ProviderData.ProductId, r.TypeName)

	var plan lifecycleResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var lifecycle lifecycleAPIModel
	resp.Diagnostics.Append(plan.toAPIModel(ctx, &lifecycle)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var apiErrs util.JFrogErrors
	request := r.ProviderData.Client.R().
		SetBody(lifecycle).
		SetError(&apiErrs)

	// Add project_key as query parameter if set
	if !plan.ProjectKey.IsNull() && plan.ProjectKey.ValueString() != "" {
		request = request.SetQueryParam("project_key", plan.ProjectKey.ValueString())
	}

	response, err := request.Patch(r.JFrogResource.CollectionEndpoint)
	if err != nil {
		utilfw.UnableToUpdateResourceError(resp, err.Error())
		return
	}

	if response.StatusCode() != http.StatusOK {
		errorMsg := apiErrs.String()
		if errorMsg == "" {
			errorMsg = response.String()
		}
		if errorMsg == "" {
			errorMsg = fmt.Sprintf("unexpected status code: %d", response.StatusCode())
		}
		utilfw.UnableToUpdateResourceError(resp, errorMsg)
		return
	}

	// Read back the lifecycle to get computed values
	resp.Diagnostics.Append(r.readLifecycle(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// Delete is not supported by the Access API; the resource is removed from state only.
func (r *lifecycleResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	go util.SendUsageResourceDelete(ctx, r.ProviderData.Client.R(), r.ProviderData.ProductId, r.TypeName)
	resp.Diagnostics.AddWarning(
		"Unable to Delete Resource",
		"Lifecycle cannot be deleted via the Access API. The resource has been removed from Terraform state only.",
	)
}

func (r *lifecycleResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import ID format: "project_key" or empty for global
	if req.ID != "" {
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("project_key"), req.ID)...)
	}
}
