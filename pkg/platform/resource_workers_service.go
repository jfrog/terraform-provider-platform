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

	"github.com/hashicorp/terraform-plugin-framework-validators/objectvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/jfrog/terraform-provider-shared/util"
	utilfw "github.com/jfrog/terraform-provider-shared/util/fw"
	"github.com/samber/lo"
)

const WorkersServiceEndpoint = "worker/api/v1/workers"

var validActions = []string{
	"BEFORE_DOWNLOAD",
	"AFTER_DOWNLOAD",
	"BEFORE_UPLOAD",
	"AFTER_CREATE",
	"AFTER_BUILD_INFO_SAVE",
	"AFTER_MOVE",
	"BEFORE_PROPERTY_CREATE",
	"BEFORE_PROPERTY_DELETE",
	"AFTER_PROPERTY_CREATE",
	"AFTER_PROPERTY_DELETE",
	"SCHEDULED_EVENT",
	"GENERIC_EVENT",
}

// actionFilterRequirement is what `filter_criteria` an action expects.
type actionFilterRequirement int

const (
	// filterRejected: the action does not accept a filter at all.
	filterRejected actionFilterRequirement = iota
	// filterArtifactRequired: the action needs `filter_criteria.artifact_filter_criteria`.
	filterArtifactRequired
	// filterScheduleRequired: the action needs `filter_criteria.schedule`.
	filterScheduleRequired
)

// Derived from `GET /worker/api/v2/actions`, which reports a `mandatoryFilter` flag
// and a `filterType` for every action: an action with no `mandatoryFilter` rejects a
// filter, `FILTER_REPO` requires an artifact filter, and `SCHEDULE` requires a
// schedule. The JFrog platform only enforces any of it while the worker is enabled.
// Keep in sync with validActions.
var actionFilterRequirements = map[string]actionFilterRequirement{
	"BEFORE_DOWNLOAD":        filterArtifactRequired,
	"AFTER_DOWNLOAD":         filterArtifactRequired,
	"BEFORE_UPLOAD":          filterArtifactRequired,
	"AFTER_CREATE":           filterArtifactRequired,
	"AFTER_BUILD_INFO_SAVE":  filterRejected,
	"AFTER_MOVE":             filterArtifactRequired,
	"BEFORE_PROPERTY_CREATE": filterArtifactRequired,
	"BEFORE_PROPERTY_DELETE": filterArtifactRequired,
	"AFTER_PROPERTY_CREATE":  filterArtifactRequired,
	"AFTER_PROPERTY_DELETE":  filterArtifactRequired,
	"SCHEDULED_EVENT":        filterScheduleRequired,
	"GENERIC_EVENT":          filterRejected,
}

var _ resource.Resource = (*workersServiceResource)(nil)
var _ resource.ResourceWithValidateConfig = (*workersServiceResource)(nil)

type workersServiceResource struct {
	ProviderData util.ProviderMetadata
	TypeName     string
}

func NewWorkerServiceResource() resource.Resource {
	return &workersServiceResource{
		TypeName: "platform_workers_service",
	}
}

func (r *workersServiceResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = r.TypeName
}

func (r *workersServiceResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"key": schema.StringAttribute{
				Required:    true,
				Description: "The unique ID of the worker.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
					stringplanmodifier.RequiresReplace(),
				},
			},
			"description": schema.StringAttribute{
				Optional:    true,
				Description: "Description of the worker.",
			},
			"enabled": schema.BoolAttribute{
				Required:    true,
				Description: "Whether to enable the worker immediately after creation.",
			},
			"source_code": schema.StringAttribute{
				Required:    true,
				Description: "The worker script in TypeScript or JavaScript.",
			},
			"action": schema.StringAttribute{
				Required:    true,
				Description: fmt.Sprintf("The worker action with which the worker is associated. Valid values: %s", strings.Join(validActions, ", ")),
				Validators:  []validator.String{stringvalidator.OneOf(validActions...)},
			},
			"filter_criteria": schema.SingleNestedAttribute{
				Optional:    true,
				Description: "Defines the criteria for triggering the worker, either by specifying repositories and path patterns for artifact-based filtering or by defining a schedule using a Cron expression. Most actions require a filter once the worker is enabled: every artifact action requires `artifact_filter_criteria`, and `SCHEDULED_EVENT` requires `schedule`. `AFTER_BUILD_INFO_SAVE` and `GENERIC_EVENT` reject a filter, so omit this attribute for them.",
				Attributes: map[string]schema.Attribute{
					"artifact_filter_criteria": schema.SingleNestedAttribute{
						Optional: true,
						Validators: []validator.Object{
							objectvalidator.ExactlyOneOf(
								// self (required workaround for now)
								path.MatchRelative(),
								// sibling
								path.MatchRelative().AtParent().AtName("schedule"),
							),
						},
						Attributes: map[string]schema.Attribute{
							"repo_keys": schema.SetAttribute{
								ElementType: types.StringType,
								Optional:    true,
								Description: "Defines which repositories are used when an action event occurs to trigger the worker. Can be omitted when at least one of `any_local`, `any_remote`, or `any_federated` is set. An explicit empty set (`repo_keys = []`) is transmitted to the platform and round-trips as an empty set; omit the attribute entirely when no repository list is intended.",
								Validators: []validator.Set{
									setvalidator.AtLeastOneOf(
										path.MatchRelative().AtParent().AtName("any_local"),
										path.MatchRelative().AtParent().AtName("any_remote"),
										path.MatchRelative().AtParent().AtName("any_federated"),
									),
								},
							},
							"any_local": schema.BoolAttribute{
								Optional:    true,
								Description: "Trigger the worker for every local repository, in addition to any repository listed in `repo_keys`.",
							},
							"any_remote": schema.BoolAttribute{
								Optional:    true,
								Description: "Trigger the worker for every remote repository, in addition to any repository listed in `repo_keys`.",
							},
							"any_federated": schema.BoolAttribute{
								Optional:    true,
								Description: "Trigger the worker for every federated repository, in addition to any repository listed in `repo_keys`.",
							},
							"include_patterns": schema.SetAttribute{
								ElementType: types.StringType,
								Optional:    true,
								Description: "Define patterns to match all repository paths for repositories identified in the repoKeys. Defines those repositories that trigger the worker. An explicit empty set is transmitted and round-trips as an empty set; omit the attribute when no include patterns are intended.",
							},
							"exclude_patterns": schema.SetAttribute{
								ElementType: types.StringType,
								Optional:    true,
								Description: "Define patterns to for all repository paths for repositories to be excluded in the repoKeys. Defines those repositories that do not trigger the worker. An explicit empty set is transmitted and round-trips as an empty set; omit the attribute when no exclude patterns are intended.",
							},
						},
					},
					"schedule": schema.SingleNestedAttribute{
						Optional: true,
						Attributes: map[string]schema.Attribute{
							"cron": schema.StringAttribute{
								Required:    true,
								Description: "Defines the Cron expression to schedule the worker.",
							},
							"timezone": schema.StringAttribute{
								Optional:    true,
								Computed:    true,
								Default:     stringdefault.StaticString("UTC"),
								Description: "Define which timezone the schedule applies to if provided.",
							},
						},
					},
				},
			},
			"shared": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "When true, allows other users to execute the worker (UI: 'Allow other users to execute the worker').",
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"secrets": schema.SetNestedAttribute{
				Optional:    true,
				Description: "The secrets to be added to the worker.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"key": schema.StringAttribute{
							Required:    true,
							Description: "The name of the secret.",
						},
						"value": schema.StringAttribute{
							Required:    true,
							Description: "The name of the secret.",
						},
					},
				},
			},
		},
		Description: "Provides a JFrog [Workers Service](https://jfrog.com/help/r/jfrog-platform-administration-documentation/workers-service) resource. This can be used to create and manage Workers Service.\n\n" +
			"->From Artifactory 7.94 the Workers service will be available in a general availability release to Enterprise X and Enterprise+ licenses.",
	}
}

type workersServiceResourceModel struct {
	Key            types.String `tfsdk:"key"`
	Description    types.String `tfsdk:"description"`
	SourceCode     types.String `tfsdk:"source_code"`
	Action         types.String `tfsdk:"action"`
	FilterCriteria types.Object `tfsdk:"filter_criteria"`
	Enabled        types.Bool   `tfsdk:"enabled"`
	Shared         types.Bool   `tfsdk:"shared"`
	Secrets        types.Set    `tfsdk:"secrets"`
}

type filterCriteriaResourceModel struct {
	ArtifactFilterCriteria types.Object `tfsdk:"artifact_filter_criteria"`
	Schedule               types.Object `tfsdk:"schedule"`
}

type artifactFilterCriteriaResourceModel struct {
	RepoKeys        types.Set  `tfsdk:"repo_keys"`
	AnyLocal        types.Bool `tfsdk:"any_local"`
	AnyRemote       types.Bool `tfsdk:"any_remote"`
	AnyFederated    types.Bool `tfsdk:"any_federated"`
	IncludePatterns types.Set  `tfsdk:"include_patterns"`
	ExcludePatterns types.Set  `tfsdk:"exclude_patterns"`
}

type scheduleResourceModel struct {
	Cron     types.String `tfsdk:"cron"`
	Timezone types.String `tfsdk:"timezone"`
}

func (r *workersServiceResourceModel) toAPIModel(ctx context.Context, apiModel *WorkersServiceAPIModel, secretKeysToBeRemoved []string) (ds diag.Diagnostics) {
	// filter_criteria is optional: actions such as AFTER_BUILD_INFO_SAVE reject a
	// filter entirely. When it is absent both nested attributes stay nil and the
	// request carries an empty filterCriteria object, which the API accepts.
	var filterCriteria filterCriteriaResourceModel
	ds.Append(r.FilterCriteria.As(ctx, &filterCriteria, basetypes.ObjectAsOptions{
		UnhandledNullAsEmpty: true,
	})...)
	if ds.HasError() {
		return
	}

	var artifactFilterCriteriaObject *artifactFilterCriteriaAPIModel
	if !filterCriteria.ArtifactFilterCriteria.IsNull() {
		var artifactFilterCriteria artifactFilterCriteriaResourceModel
		ds.Append(filterCriteria.ArtifactFilterCriteria.As(ctx, &artifactFilterCriteria, basetypes.ObjectAsOptions{})...)
		if ds.HasError() {
			return
		}

		repoKeys, d := stringSetToOptionalSlicePtr(ctx, artifactFilterCriteria.RepoKeys)
		ds.Append(d...)
		if ds.HasError() {
			return
		}

		includePatterns, d := stringSetToOptionalSlicePtr(ctx, artifactFilterCriteria.IncludePatterns)
		ds.Append(d...)
		if ds.HasError() {
			return
		}

		excludePatterns, d := stringSetToOptionalSlicePtr(ctx, artifactFilterCriteria.ExcludePatterns)
		ds.Append(d...)
		if ds.HasError() {
			return
		}

		artifactFilterCriteriaObject = &artifactFilterCriteriaAPIModel{
			RepoKeys:        repoKeys,
			AnyLocal:        artifactFilterCriteria.AnyLocal.ValueBoolPointer(),
			AnyRemote:       artifactFilterCriteria.AnyRemote.ValueBoolPointer(),
			AnyFederated:    artifactFilterCriteria.AnyFederated.ValueBoolPointer(),
			IncludePatterns: includePatterns,
			ExcludePatterns: excludePatterns,
		}
	}

	var scheduleObject *scheduleAPIModel
	if !filterCriteria.Schedule.IsNull() && !filterCriteria.Schedule.IsUnknown() {
		var schedule scheduleResourceModel
		ds.Append(filterCriteria.Schedule.As(ctx, &schedule, basetypes.ObjectAsOptions{})...)
		if ds.HasError() {
			return
		}

		scheduleObject = &scheduleAPIModel{
			Cron:     schedule.Cron.ValueString(),
			Timezone: schedule.Timezone.ValueString(),
		}
	}

	secrets := lo.Map[attr.Value](
		r.Secrets.Elements(),
		func(elem attr.Value, index int) secretAPIModel {
			attr := elem.(types.Object).Attributes()

			return secretAPIModel{
				Key:   attr["key"].(types.String).ValueString(),
				Value: attr["value"].(types.String).ValueString(),
			}
		},
	)

	for _, secretKeyToBeRemoved := range secretKeysToBeRemoved {
		s := secretAPIModel{
			Key:              secretKeyToBeRemoved,
			MarkedForRemoval: true,
		}

		secrets = append(secrets, s)
	}

	*apiModel = WorkersServiceAPIModel{
		Key:         r.Key.ValueString(),
		Description: r.Description.ValueString(),
		SourceCode:  r.SourceCode.ValueString(),
		Action:      r.Action.ValueString(),
		FilterCriteria: filterCriteriaAPIModel{
			ArtifactFilterCriteria: artifactFilterCriteriaObject,
			Schedule:               scheduleObject,
		},
		Enabled: r.Enabled.ValueBool(),
		Shared:  r.Shared.ValueBool(),
		Secrets: secrets,
	}

	return nil
}

var filterCriteriaResourceModelAttributeTypes map[string]attr.Type = map[string]attr.Type{
	"artifact_filter_criteria": types.ObjectType{
		AttrTypes: artifactFilterCriteriaResourceModelAttributeTypes,
	},
	"schedule": types.ObjectType{
		AttrTypes: scheduleResourceModelAttributeTypes,
	},
}

var artifactFilterCriteriaResourceModelAttributeTypes map[string]attr.Type = map[string]attr.Type{
	"repo_keys":        types.SetType{ElemType: types.StringType},
	"any_local":        types.BoolType,
	"any_remote":       types.BoolType,
	"any_federated":    types.BoolType,
	"include_patterns": types.SetType{ElemType: types.StringType},
	"exclude_patterns": types.SetType{ElemType: types.StringType},
}

var scheduleResourceModelAttributeTypes map[string]attr.Type = map[string]attr.Type{
	"cron":     types.StringType,
	"timezone": types.StringType,
}

func (r *workersServiceResourceModel) fromAPIModel(ctx context.Context, apiModel *WorkersServiceAPIModel) (ds diag.Diagnostics) {
	r.Key = types.StringValue(apiModel.Key)
	r.Description = types.StringValue(apiModel.Description)
	r.SourceCode = types.StringValue(apiModel.SourceCode)
	r.Action = types.StringValue(apiModel.Action)

	artifactFilterCriteriaObject := types.ObjectNull(artifactFilterCriteriaResourceModelAttributeTypes)
	if apiModel.FilterCriteria.ArtifactFilterCriteria != nil {
		repoKeys, d := optionalSlicePtrToStringSet(ctx, apiModel.FilterCriteria.ArtifactFilterCriteria.RepoKeys)
		ds.Append(d...)
		if ds.HasError() {
			return
		}
		includePatterns, d := optionalSlicePtrToStringSet(ctx, apiModel.FilterCriteria.ArtifactFilterCriteria.IncludePatterns)
		ds.Append(d...)
		if ds.HasError() {
			return
		}
		excludePatterns, d := optionalSlicePtrToStringSet(ctx, apiModel.FilterCriteria.ArtifactFilterCriteria.ExcludePatterns)
		ds.Append(d...)
		if ds.HasError() {
			return
		}

		artifactFilterCriteriaValue := artifactFilterCriteriaResourceModel{
			RepoKeys:        repoKeys,
			AnyLocal:        types.BoolPointerValue(apiModel.FilterCriteria.ArtifactFilterCriteria.AnyLocal),
			AnyRemote:       types.BoolPointerValue(apiModel.FilterCriteria.ArtifactFilterCriteria.AnyRemote),
			AnyFederated:    types.BoolPointerValue(apiModel.FilterCriteria.ArtifactFilterCriteria.AnyFederated),
			IncludePatterns: includePatterns,
			ExcludePatterns: excludePatterns,
		}

		atrifactFilterCriteria, d := types.ObjectValueFrom(
			ctx,
			artifactFilterCriteriaResourceModelAttributeTypes,
			artifactFilterCriteriaValue,
		)
		if d != nil {
			ds = append(ds, d...)
		}
		if ds.HasError() {
			return
		}

		artifactFilterCriteriaObject = atrifactFilterCriteria
	}

	scheduleObject := types.ObjectNull(scheduleResourceModelAttributeTypes)
	if apiModel.FilterCriteria.Schedule != nil {
		scheduleValue := scheduleResourceModel{
			Cron:     types.StringValue(apiModel.FilterCriteria.Schedule.Cron),
			Timezone: types.StringValue(apiModel.FilterCriteria.Schedule.Timezone),
		}

		schedule, d := types.ObjectValueFrom(
			ctx,
			scheduleResourceModelAttributeTypes,
			scheduleValue,
		)
		if d != nil {
			ds = append(ds, d...)
		}
		if ds.HasError() {
			return
		}

		scheduleObject = schedule
	}

	// A worker created without a filter comes back as an empty filterCriteria object.
	// Materialising it here would put an object holding two null attributes into state,
	// which drifts forever against a configuration that omits filter_criteria entirely.
	filterCriteriaObject := types.ObjectNull(filterCriteriaResourceModelAttributeTypes)
	if apiModel.FilterCriteria.ArtifactFilterCriteria != nil || apiModel.FilterCriteria.Schedule != nil {
		filterCriteria, d := types.ObjectValue(
			filterCriteriaResourceModelAttributeTypes,
			map[string]attr.Value{
				"artifact_filter_criteria": artifactFilterCriteriaObject,
				"schedule":                 scheduleObject,
			},
		)
		if d != nil {
			ds = append(ds, d...)
		}
		if ds.HasError() {
			return
		}

		filterCriteriaObject = filterCriteria
	}

	r.FilterCriteria = filterCriteriaObject
	r.Enabled = types.BoolValue(apiModel.Enabled)
	r.Shared = types.BoolValue(apiModel.Shared)

	return
}

type WorkersServiceAPIModel struct {
	Key            string                 `json:"key"`
	Description    string                 `json:"description"`
	SourceCode     string                 `json:"sourceCode"`
	Action         string                 `json:"action"`
	FilterCriteria filterCriteriaAPIModel `json:"filterCriteria"`
	Enabled        bool                   `json:"enabled"`
	Shared         bool                   `json:"shared"`
	Secrets        []secretAPIModel       `json:"secrets"`
}

type filterCriteriaAPIModel struct {
	ArtifactFilterCriteria *artifactFilterCriteriaAPIModel `json:"artifactFilterCriteria,omitempty"`
	Schedule               *scheduleAPIModel               `json:"schedule,omitempty"`
}

type artifactFilterCriteriaAPIModel struct {
	RepoKeys        *[]string `json:"repoKeys,omitempty"`
	AnyLocal        *bool     `json:"anyLocal,omitempty"`
	AnyRemote       *bool     `json:"anyRemote,omitempty"`
	AnyFederated    *bool     `json:"anyFederated,omitempty"`
	IncludePatterns *[]string `json:"includePatterns,omitempty"`
	ExcludePatterns *[]string `json:"excludePatterns,omitempty"`
}

// stringSetToOptionalSlicePtr maps a Terraform set to a JSON slice pointer. A null
// set becomes a nil pointer so omitempty drops the key; an empty set becomes a
// pointer to an empty slice so the API receives [] rather than an omitted key.
func stringSetToOptionalSlicePtr(ctx context.Context, set types.Set) (*[]string, diag.Diagnostics) {
	if set.IsNull() {
		return nil, nil
	}

	var values []string
	ds := set.ElementsAs(ctx, &values, false)
	if ds.HasError() {
		return nil, ds
	}

	return &values, ds
}

// optionalSlicePtrToStringSet maps a JSON slice pointer back to a Terraform set. A
// nil pointer becomes a null set; a pointer to an empty slice becomes an empty set.
func optionalSlicePtrToStringSet(ctx context.Context, values *[]string) (types.Set, diag.Diagnostics) {
	if values == nil {
		return types.SetNull(types.StringType), nil
	}

	set, d := types.SetValueFrom(ctx, types.StringType, *values)
	if d != nil {
		return types.SetNull(types.StringType), d
	}

	return set, nil
}

type scheduleAPIModel struct {
	Cron     string `json:"cron"`
	Timezone string `json:"timezone,omitempty"`
}

type secretAPIModel struct {
	Key              string `json:"key"`
	Value            string `json:"value"`
	MarkedForRemoval bool   `json:"markedForRemoval,omitempty"`
}

// ValidateConfig reports at plan time the filter combinations the JFrog platform
// rejects at apply time. The rules span `action`, `enabled` and `filter_criteria`
// together, so they cannot be expressed as validators on a single attribute.
func (r *workersServiceResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var data workersServiceResourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(validateFilterCriteria(ctx, data.Action, data.Enabled, data.FilterCriteria)...)
}

// validateFilterCriteria holds the whole rule set, as a pure function of the three
// attributes the rules depend on, so it can be exercised directly by a unit test
// without a Terraform configuration, the acceptance harness, or a live platform. It
// returns no diagnostics for a combination the JFrog platform accepts.
func validateFilterCriteria(ctx context.Context, action types.String, enabled types.Bool, filterCriteria types.Object) diag.Diagnostics {
	var ds diag.Diagnostics

	// Every rule keys off the action, and `filter_criteria` as a whole has to be
	// resolved before its nested attributes can be inspected. An interpolation that
	// is still unknown leaves nothing to check until apply time.
	if action.IsNull() || action.IsUnknown() || filterCriteria.IsUnknown() {
		return ds
	}

	actionName := action.ValueString()
	requirement, ok := actionFilterRequirements[actionName]
	if !ok {
		// stringvalidator.OneOf on `action` already reports an unrecognised value.
		return ds
	}

	// The JFrog platform only enforces these rules for an enabled worker, so a hard
	// error would refuse configurations it genuinely accepts. An unknown `enabled` is
	// treated as enabled, that being the stricter reading of a value that may well
	// resolve to true.
	addDiagnostic := func(summary, message, disabledNote string) {
		if enabled.IsNull() || enabled.IsUnknown() || enabled.ValueBool() {
			ds.Append(diag.NewAttributeErrorDiagnostic(path.Root("filter_criteria"), summary, message))
			return
		}
		ds.Append(diag.NewAttributeWarningDiagnostic(
			path.Root("filter_criteria"),
			summary,
			fmt.Sprintf("%s %s", message, disabledNote),
		))
	}

	filter := filterCriteriaResourceModel{
		ArtifactFilterCriteria: types.ObjectNull(artifactFilterCriteriaResourceModelAttributeTypes),
		Schedule:               types.ObjectNull(scheduleResourceModelAttributeTypes),
	}
	if !filterCriteria.IsNull() {
		ds.Append(filterCriteria.As(ctx, &filter, basetypes.ObjectAsOptions{})...)
		if ds.HasError() {
			return ds
		}
	}

	switch requirement {
	case filterArtifactRequired:
		if filter.ArtifactFilterCriteria.IsUnknown() {
			return ds
		}
		if filter.ArtifactFilterCriteria.IsNull() {
			addDiagnostic(
				"Missing Attribute Configuration",
				fmt.Sprintf("filter_criteria.artifact_filter_criteria must be configured when action is set to '%s'. This action is triggered by repository events, so the JFrog platform requires a repository filter and rejects an enabled worker without one with \"Filter must be set if worker is enabled\".", actionName),
				"The JFrog platform does not enforce this while enabled is false, but it will reject the worker as soon as enabled is set to true.",
			)
		}
	case filterScheduleRequired:
		if filter.Schedule.IsUnknown() {
			return ds
		}
		if filter.Schedule.IsNull() {
			addDiagnostic(
				"Missing Attribute Configuration",
				fmt.Sprintf("filter_criteria.schedule must be configured when action is set to '%s'. This action is triggered by a Cron schedule rather than by repository events, so the JFrog platform requires a schedule and rejects an enabled worker without one with \"Filter must be set if worker is enabled\".", actionName),
				"The JFrog platform does not enforce this while enabled is false, but it will reject the worker as soon as enabled is set to true.",
			)
		}
	case filterRejected:
		if !filterCriteria.IsNull() {
			addDiagnostic(
				"Invalid Attribute Configuration",
				fmt.Sprintf("filter_criteria must be omitted when action is set to '%s'. This worker action does not accept a filter, and the JFrog platform rejects an enabled worker that supplies one with \"Filter must not be set for this action\".", actionName),
				"The JFrog platform does not reject this while enabled is false: it accepts the worker and silently discards the filter. The refresh that follows the apply then reads filter_criteria back as null, so every subsequent terraform plan reports a difference that no apply can resolve. Remove filter_criteria from the configuration.",
			)
		}
	}

	return ds
}

func (r *workersServiceResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}
	r.ProviderData = req.ProviderData.(util.ProviderMetadata)
}

func (r *workersServiceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	go util.SendUsageResourceCreate(ctx, r.ProviderData.Client.R(), r.ProviderData.ProductId, r.TypeName)

	var plan workersServiceResourceModel

	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var workersService WorkersServiceAPIModel
	resp.Diagnostics.Append(plan.toAPIModel(ctx, &workersService, []string{})...)
	if resp.Diagnostics.HasError() {
		return
	}

	response, err := r.ProviderData.Client.R().
		SetBody(&workersService).
		Post(WorkersServiceEndpoint)
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

func (r *workersServiceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	go util.SendUsageResourceRead(ctx, r.ProviderData.Client.R(), r.ProviderData.ProductId, r.TypeName)

	var state workersServiceResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var workersService WorkersServiceAPIModel

	response, err := r.ProviderData.Client.R().
		SetPathParam("key", state.Key.ValueString()).
		SetResult(&workersService).
		Get(WorkersServiceEndpoint + "/{key}")
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
	resp.Diagnostics.Append(state.fromAPIModel(ctx, &workersService)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *workersServiceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	go util.SendUsageResourceUpdate(ctx, r.ProviderData.Client.R(), r.ProviderData.ProductId, r.TypeName)

	var plan workersServiceResourceModel
	var state workersServiceResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	planSecrets := lo.Map(
		plan.Secrets.Elements(),
		func(elem attr.Value, index int) secretAPIModel {
			attrs := elem.(types.Object).Attributes()
			return secretAPIModel{
				Key: attrs["key"].(types.String).ValueString(),
			}
		},
	)

	stateSecrets := lo.Map(state.Secrets.Elements(), func(elem attr.Value, index int) secretAPIModel {
		attrs := elem.(types.Object).Attributes()
		return secretAPIModel{
			Key: attrs["key"].(types.String).ValueString(),
		}
	})

	_, secretsToBeRemoved := lo.Difference(planSecrets, stateSecrets)
	secretKeysToBeRemovedKeys := lo.Map(
		secretsToBeRemoved,
		func(x secretAPIModel, index int) string {
			return x.Key
		},
	)

	var workersService WorkersServiceAPIModel
	resp.Diagnostics.Append(plan.toAPIModel(ctx, &workersService, secretKeysToBeRemovedKeys)...)
	if resp.Diagnostics.HasError() {
		return
	}

	response, err := r.ProviderData.Client.R().
		SetBody(&workersService).
		Put(WorkersServiceEndpoint)
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

func (r *workersServiceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	go util.SendUsageResourceDelete(ctx, r.ProviderData.Client.R(), r.ProviderData.ProductId, r.TypeName)

	var data workersServiceResourceModel

	diags := req.State.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	key := data.Key.ValueString()

	response, err := r.ProviderData.Client.R().
		SetPathParam("key", key).
		Delete(WorkersServiceEndpoint + "/{key}")
	if err != nil {
		utilfw.UnableToDeleteResourceError(resp, err.Error())
		return
	}

	if response.IsError() {
		utilfw.UnableToDeleteResourceError(resp, response.String())
		return
	}

	// If the logic reaches here, it implicitly succeeded and will remove
	// the resource from state if there are no other errors.
}

func (r *workersServiceResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("key"), req, resp)
}
