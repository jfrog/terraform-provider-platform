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

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/jfrog/terraform-provider-shared/util"
	utilfw "github.com/jfrog/terraform-provider-shared/util/fw"
)

var _ resource.Resource = (*lifecycleStageResource)(nil)

// stagesEndpoint is the base path for lifecycle stages API; document URLs are built as stagesEndpoint + "/" + name.
const stagesEndpoint = "access/api/v2/stages"

type lifecycleStageResource struct {
	util.JFrogResource
	ProviderData util.ProviderMetadata
}

func NewLifecycleStageResource() resource.Resource {
	return &lifecycleStageResource{
		JFrogResource: util.JFrogResource{
			TypeName:           "platform_lifecycle_stage",
			CollectionEndpoint: stagesEndpoint,
		},
	}
}

func (r *lifecycleStageResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}
	r.ProviderData = req.ProviderData.(util.ProviderMetadata)
}

func (r *lifecycleStageResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
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

// validateProjectScopedStageName checks that when project_key is set, name is prefixed with project_key (e.g. "bookverse-deploy" when project_key is "bookverse").
// Returns diagnostics to append when validation fails; call from schema validators or ValidateConfig.
func validateProjectScopedStageName(stageName string, projectKey types.String, attrPath path.Path) diag.Diagnostics {
	if projectKey.IsNull() || projectKey.ValueString() == "" {
		return nil
	}
	expectedPrefix := projectKey.ValueString() + "-"
	if strings.HasPrefix(stageName, expectedPrefix) {
		return nil
	}
	stageNameOnly := stageName
	if strings.Contains(stageName, "-") {
		parts := strings.SplitN(stageName, "-", 2)
		if len(parts) > 1 {
			stageNameOnly = parts[len(parts)-1]
		}
	}
	return diag.Diagnostics{
		diag.NewAttributeErrorDiagnostic(
			attrPath,
			"Invalid Stage Name for Project Scope",
			fmt.Sprintf(
				"Stage name '%s' must be prefixed with project_key '%s'. "+
					"Expected format: '%s-%s'. "+
					"For example, if project_key is '%s' and you want a stage named '%s', use '%s-%s'.",
				stageName,
				projectKey.ValueString(),
				projectKey.ValueString(),
				stageNameOnly,
				projectKey.ValueString(),
				stageNameOnly,
				projectKey.ValueString(),
				stageNameOnly,
			),
		),
	}
}

// projectStageNameValidator validates that when project_key is set, name is prefixed with project_key.
type projectStageNameValidator struct{}

func (v projectStageNameValidator) Description(ctx context.Context) string {
	return "When project_key is set, stage name must be prefixed with project_key (e.g. 'bookverse-deploy' if project_key is 'bookverse')"
}

func (v projectStageNameValidator) MarkdownDescription(ctx context.Context) string {
	return "When project_key is set, stage name must be prefixed with project_key (e.g. 'bookverse-deploy' if project_key is 'bookverse')."
}

func (v projectStageNameValidator) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	// This validator runs on project_key: when project_key is set, name must be prefixed with it
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	projectKey := types.StringValue(req.ConfigValue.ValueString())
	var name types.String
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("name"), &name)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if name.IsNull() || name.IsUnknown() {
		return
	}
	resp.Diagnostics.Append(validateProjectScopedStageName(name.ValueString(), projectKey, path.Root("name"))...)
}

func (r *lifecycleStageResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Required: true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				MarkdownDescription: "The unique name of the stage (for example, DEV, QA, PROD). For project-scoped stages, the name must be prefixed with the project_key (e.g., 'bookverse-deploy' if project_key is 'bookverse'). **Important:** Stage names are case-sensitive.",
			},
			"scope": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
				MarkdownDescription: "The scope of the stage: GLOBAL or PROJECT. This is determined by the API based on whether `project_key` is provided. Read-only.",
			},
			"project_key": schema.StringAttribute{
				Optional: true,
				Validators: []validator.String{
					projectStageNameValidator{},
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				MarkdownDescription: "[For project-level stages only] The project key associated with the stage. When set, the stage name must be prefixed with this value (e.g. 'bookverse-deploy' if project_key is 'bookverse').",
			},
			"category": schema.StringAttribute{
				Optional: true,
				Computed: true,
				Default:  stringdefault.StaticString("promote"),
				Validators: []validator.String{
					stringvalidator.OneOf("none", "code", "promote"),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
				MarkdownDescription: "The category of the stage: `none`, `code`, or `promote` (default: `promote`).",
			},
			"repositories": schema.SetAttribute{
				ElementType:         types.StringType,
				Computed:            true,
				MarkdownDescription: "A list of repository keys assigned to this stage. This is relevant only when the category is `promote`. Repositories are managed by the API and returned in the response. Read-only.",
			},
			"used_in_lifecycles": schema.ListAttribute{
				ElementType:         types.StringType,
				Computed:            true,
				MarkdownDescription: "Lists the project keys that use this stage as part of its lifecycle.",
			},
			"created": schema.Int64Attribute{
				Computed:            true,
				MarkdownDescription: "The timestamp when the stage was created (milliseconds since epoch).",
			},
			"modified": schema.Int64Attribute{
				Computed:            true,
				MarkdownDescription: "The timestamp when the stage was last modified (milliseconds since epoch).",
			},
			"total_repository_count": schema.Int64Attribute{
				Computed:            true,
				MarkdownDescription: "The total number of repositories assigned to this stage. Read-only.",
			},
		},
		MarkdownDescription: "Provides a lifecycle stage resource to create and manage lifecycle stages. A lifecycle stage represents a step in the software development lifecycle. See [JFrog documentation](https://jfrog.com/help/r/jfrog-platform-administration-documentation/stages-lifecycle) for more details.",
	}
}

type lifecycleStageResourceModel struct {
	Name                 types.String `tfsdk:"name"`
	Scope                types.String `tfsdk:"scope"`
	ProjectKey           types.String `tfsdk:"project_key"`
	Category             types.String `tfsdk:"category"`
	Repositories         types.Set    `tfsdk:"repositories"`
	UsedInLifecycles     types.List   `tfsdk:"used_in_lifecycles"`
	Created              types.Int64  `tfsdk:"created"`
	Modified             types.Int64  `tfsdk:"modified"`
	TotalRepositoryCount types.Int64  `tfsdk:"total_repository_count"`
}

func (r *lifecycleStageResourceModel) toAPIModel(ctx context.Context, apiModel *lifecycleStageAPIModel) (ds diag.Diagnostics) {
	category := r.Category.ValueString()
	if category == "" {
		category = "promote"
	}

	*apiModel = lifecycleStageAPIModel{
		Name:       r.Name.ValueString(),
		ProjectKey: r.ProjectKey.ValueStringPointer(),
		Category:   category,
		// Scope and Repositories are not set here - they're only read from API responses
	}

	return nil
}

func (r *lifecycleStageResourceModel) fromAPIModel(ctx context.Context, apiModel lifecycleStageAPIModel) diag.Diagnostics {
	diags := diag.Diagnostics{}

	r.Name = types.StringValue(apiModel.Name)

	// Read scope from API response (computed, read-only)
	scope := strings.ToUpper(apiModel.Scope)
	r.Scope = types.StringValue(scope)

	r.ProjectKey = types.StringPointerValue(apiModel.ProjectKey)
	r.Category = types.StringValue(apiModel.Category)
	r.Created = types.Int64Value(apiModel.Created)
	r.Modified = types.Int64Value(apiModel.Modified)
	r.TotalRepositoryCount = types.Int64Value(apiModel.TotalRepositoryCount)

	// API returns empty array [] for repositories, not null
	// Use empty set to match API response behavior
	if len(apiModel.Repositories) > 0 {
		repositories, d := types.SetValueFrom(ctx, types.StringType, apiModel.Repositories)
		if d != nil {
			diags.Append(d...)
			return diags
		}
		r.Repositories = repositories
	} else {
		// API returns empty array [], so use empty set instead of null
		emptySet, d := types.SetValue(types.StringType, []attr.Value{})
		if d != nil {
			diags.Append(d...)
			return diags
		}
		r.Repositories = emptySet
	}

	// API returns empty array [] for used_in_lifecycles, not null
	// Use empty list to match API response behavior
	if len(apiModel.UsedInLifecycles) > 0 {
		usedInLifecycles, d := types.ListValueFrom(ctx, types.StringType, apiModel.UsedInLifecycles)
		if d != nil {
			diags.Append(d...)
			return diags
		}
		r.UsedInLifecycles = usedInLifecycles
	} else {
		// API returns empty array [], so use empty list instead of null
		emptyList, d := types.ListValue(types.StringType, []attr.Value{})
		if d != nil {
			diags.Append(d...)
			return diags
		}
		r.UsedInLifecycles = emptyList
	}

	return diags
}

type lifecycleStageAPIModel struct {
	Name                 string   `json:"name"`
	Scope                string   `json:"scope"`
	ProjectKey           *string  `json:"project_key,omitempty"`
	Category             string   `json:"category"`
	Repositories         []string `json:"repositories,omitempty"`
	UsedInLifecycles     []string `json:"used_in_lifecycles,omitempty"`
	Created              int64    `json:"created"`
	Modified             int64    `json:"modified"`
	TotalRepositoryCount int64    `json:"total_repository_count"`
}

func (r *lifecycleStageResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	go util.SendUsageResourceCreate(ctx, r.ProviderData.Client.R(), r.ProviderData.ProductId, r.TypeName)

	var plan lifecycleStageResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Category defaults to "promote"
	if plan.Category.IsNull() {
		plan.Category = types.StringValue("promote")
	}

	var apiModel lifecycleStageAPIModel
	resp.Diagnostics.Append(plan.toAPIModel(ctx, &apiModel)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Create: POST /access/api/v2/stages/ — request body has name, project_key (optional), category (optional); no query param per docs
	var newStage lifecycleStageAPIModel
	var apiErrs util.JFrogErrors
	response, err := r.ProviderData.Client.R().
		SetBody(&apiModel).
		SetResult(&newStage).
		SetError(&apiErrs).
		Post(r.JFrogResource.CollectionEndpoint)

	if err != nil {
		utilfw.UnableToCreateResourceError(resp, err.Error())
		return
	}

	if response.StatusCode() != http.StatusCreated {
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

	resp.Diagnostics.Append(plan.fromAPIModel(ctx, newStage)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *lifecycleStageResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	go util.SendUsageResourceRead(ctx, r.ProviderData.Client.R(), r.ProviderData.ProductId, r.TypeName)

	var state lifecycleStageResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var stage lifecycleStageAPIModel
	var apiErrs util.JFrogErrors
	request := r.ProviderData.Client.R().
		SetPathParam("name", state.Name.ValueString()).
		SetResult(&stage).
		SetError(&apiErrs)

	// Add project_key as query parameter if set
	if !state.ProjectKey.IsNull() && state.ProjectKey.ValueString() != "" {
		request = request.SetQueryParam("project_key", state.ProjectKey.ValueString())
	}

	response, err := request.Get(r.JFrogResource.CollectionEndpoint + "/" + state.Name.ValueString())

	if err != nil {
		utilfw.UnableToRefreshResourceError(resp, err.Error())
		return
	}

	if response.StatusCode() == http.StatusNotFound {
		resp.State.RemoveResource(ctx)
		return
	}

	if response.IsError() {
		errorMsg := apiErrs.String()
		if errorMsg == "" {
			errorMsg = response.String()
		}
		if errorMsg == "" {
			errorMsg = fmt.Sprintf("unexpected status code: %d", response.StatusCode())
		}
		utilfw.UnableToRefreshResourceError(resp, errorMsg)
		return
	}

	resp.Diagnostics.Append(state.fromAPIModel(ctx, stage)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *lifecycleStageResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	go util.SendUsageResourceUpdate(ctx, r.ProviderData.Client.R(), r.ProviderData.ProductId, r.TypeName)

	var plan lifecycleStageResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state lifecycleStageResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// First, verify the stage exists in the expected scope before updating
	// If project_key is provided, check if stage exists in that project scope
	// If project_key is not provided, check if stage exists in global scope
	var checkStage lifecycleStageAPIModel
	var checkApiErrs util.JFrogErrors
	checkRequest := r.ProviderData.Client.R().
		SetPathParam("name", plan.Name.ValueString()).
		SetResult(&checkStage).
		SetError(&checkApiErrs)

	// Add project_key as query parameter if set
	if !plan.ProjectKey.IsNull() && plan.ProjectKey.ValueString() != "" {
		checkRequest = checkRequest.SetQueryParam("project_key", plan.ProjectKey.ValueString())
	}

	checkResponse, err := checkRequest.Get(r.JFrogResource.CollectionEndpoint + "/" + plan.Name.ValueString())
	if err != nil {
		utilfw.UnableToUpdateResourceError(resp, fmt.Sprintf("failed to verify stage exists: %s", err.Error()))
		return
	}

	if checkResponse.StatusCode() == http.StatusNotFound {
		scope := "global"
		if !plan.ProjectKey.IsNull() && plan.ProjectKey.ValueString() != "" {
			scope = fmt.Sprintf("project '%s'", plan.ProjectKey.ValueString())
		}
		utilfw.UnableToUpdateResourceError(resp, fmt.Sprintf("stage '%s' does not exist in %s scope", plan.Name.ValueString(), scope))
		return
	}

	if checkResponse.IsError() {
		errorMsg := checkApiErrs.String()
		if errorMsg == "" {
			errorMsg = checkResponse.String()
		}
		if errorMsg == "" {
			errorMsg = fmt.Sprintf("unexpected status code: %d", checkResponse.StatusCode())
		}
		utilfw.UnableToUpdateResourceError(resp, fmt.Sprintf("failed to verify stage exists: %s", errorMsg))
		return
	}

	// Stage exists in the expected scope, proceed with update
	// Build update request body with only fields that are actually being updated
	// Only include fields that have changed from state to plan
	updateStage := map[string]interface{}{}

	// Only include name if it has changed
	if plan.Name.ValueString() != state.Name.ValueString() {
		updateStage["name"] = plan.Name.ValueString()
	}

	// Only include category if it has changed and has a value
	if !plan.Category.IsNull() && plan.Category.ValueString() != "" {
		stateCategory := "promote" // default
		if !state.Category.IsNull() && state.Category.ValueString() != "" {
			stateCategory = state.Category.ValueString()
		}
		if plan.Category.ValueString() != stateCategory {
			updateStage["category"] = plan.Category.ValueString()
		}
	}

	// If nothing is being updated, skip API call and just update state
	if len(updateStage) == 0 {
		resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
		return
	}

	var updatedStage lifecycleStageAPIModel
	var apiErrs util.JFrogErrors
	request := r.ProviderData.Client.R().
		SetPathParam("name", plan.Name.ValueString()).
		SetBody(updateStage).
		SetResult(&updatedStage).
		SetError(&apiErrs)

	// Add project_key as query parameter if set
	// If project_key is provided, updates the stage in that project scope
	// If not provided, updates the stage in global scope (if exists)
	if !plan.ProjectKey.IsNull() && plan.ProjectKey.ValueString() != "" {
		request = request.SetQueryParam("project_key", plan.ProjectKey.ValueString())
	}

	response, err := request.Patch(r.JFrogResource.CollectionEndpoint + "/" + plan.Name.ValueString())
	if err != nil {
		utilfw.UnableToUpdateResourceError(resp, err.Error())
		return
	}

	if response.IsError() {
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

	resp.Diagnostics.Append(plan.fromAPIModel(ctx, updatedStage)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *lifecycleStageResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	go util.SendUsageResourceDelete(ctx, r.ProviderData.Client.R(), r.ProviderData.ProductId, r.TypeName)

	var state lifecycleStageResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var apiErrs util.JFrogErrors
	request := r.ProviderData.Client.R().
		SetPathParam("name", state.Name.ValueString()).
		SetError(&apiErrs)

	// Add project_key as query parameter if set
	if !state.ProjectKey.IsNull() && state.ProjectKey.ValueString() != "" {
		request = request.SetQueryParam("project_key", state.ProjectKey.ValueString())
	}

	response, err := request.Delete(r.JFrogResource.CollectionEndpoint + "/" + state.Name.ValueString())

	if err != nil {
		utilfw.UnableToDeleteResourceError(resp, err.Error())
		return
	}

	if response.StatusCode() != http.StatusNotFound && response.StatusCode() != http.StatusNoContent {
		errorMsg := apiErrs.String()
		if errorMsg == "" {
			errorMsg = response.String()
		}
		if errorMsg == "" {
			errorMsg = fmt.Sprintf("unexpected status code: %d", response.StatusCode())
		}
		utilfw.UnableToDeleteResourceError(resp, errorMsg)
		return
	}
}

func (r *lifecycleStageResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import ID format: "name" or "name:project_key"
	parts := strings.SplitN(req.ID, ":", 2)

	if len(parts) > 0 && parts[0] != "" {
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("name"), parts[0])...)
	}

	if len(parts) == 2 && parts[1] != "" {
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("project_key"), parts[1])...)
	}
}
