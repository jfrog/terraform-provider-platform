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
}

var _ resource.Resource = (*workersServiceResource)(nil)

type workersServiceResource struct {
	ProviderData PlatformProviderMetadata
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
				Required:    true,
				Description: "Defines the repositories to be used or excluded.",
				Attributes: map[string]schema.Attribute{
					"artifact_filter_criteria": schema.SingleNestedAttribute{
						Required: true,
						Attributes: map[string]schema.Attribute{
							"repo_keys": schema.SetAttribute{
								ElementType: types.StringType,
								Required:    true,
								Description: "Defines which repositories are used when an action event occurs to trigger the worker.",
							},
							"include_patterns": schema.SetAttribute{
								ElementType: types.StringType,
								Optional:    true,
								Description: "Define patterns to match all repository paths for repositories identified in the repoKeys. Defines those repositories that trigger the worker.",
							},
							"exclude_patterns": schema.SetAttribute{
								ElementType: types.StringType,
								Optional:    true,
								Description: "Define patterns to for all repository paths for repositories to be excluded in the repoKeys. Defines those repositories that do not trigger the worker.",
							},
						},
					},
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
		Description: "Provides a JFrog [Workers Service](https://jfrog.com/help/r/jfrog-platform-administration-documentation/workers-service) resource. This can be used to create and manage Workers Service.\n\n!>JFrog Workers Service is only available for JFrog Cloud customers to use free of charge during the beta period. The API may not be backward compatible after the beta period is over. Be aware of this caveat when you create workers during this period.",
	}
}

type workersServiceResourceModel struct {
	Key            types.String `tfsdk:"key"`
	Description    types.String `tfsdk:"description"`
	SourceCode     types.String `tfsdk:"source_code"`
	Action         types.String `tfsdk:"action"`
	FilterCriteria types.Object `tfsdk:"filter_criteria"`
	Enabled        types.Bool   `tfsdk:"enabled"`
	Secrets        types.Set    `tfsdk:"secrets"`
}

type filterCriteriaResourceModel struct {
	ArtifactFilterCriteria types.Object `tfsdk:"artifact_filter_criteria"`
}

type artifactFilterCriteriaResourceModel struct {
	RepoKeys        types.Set `tfsdk:"repo_keys"`
	IncludePatterns types.Set `tfsdk:"include_patterns"`
	ExcludePatterns types.Set `tfsdk:"exclude_patterns"`
}

func (r *workersServiceResourceModel) toAPIModel(ctx context.Context, apiModel *WorkersServiceAPIModel, secretKeysToBeRemoved []string) (ds diag.Diagnostics) {
	var filterCriteria filterCriteriaResourceModel
	ds.Append(r.FilterCriteria.As(ctx, &filterCriteria, basetypes.ObjectAsOptions{})...)
	if ds.HasError() {
		return
	}

	var artifactFilterCriteria artifactFilterCriteriaResourceModel
	ds.Append(filterCriteria.ArtifactFilterCriteria.As(ctx, &artifactFilterCriteria, basetypes.ObjectAsOptions{})...)
	if ds.HasError() {
		return
	}

	var repoKeys []string
	artifactFilterCriteria.RepoKeys.ElementsAs(ctx, &repoKeys, false)

	var includePatterns []string
	artifactFilterCriteria.IncludePatterns.ElementsAs(ctx, &includePatterns, false)

	var excludePatterns []string
	artifactFilterCriteria.ExcludePatterns.ElementsAs(ctx, &excludePatterns, false)

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
			ArtifactFilterCriteria: artifactFilterCriteriaAPIModel{
				RepoKeys:        repoKeys,
				IncludePatterns: includePatterns,
				ExcludePatterns: excludePatterns,
			},
		},
		Enabled: r.Enabled.ValueBool(),
		Secrets: secrets,
	}

	return nil
}

var filterCriteriaResourceModelAttributeTypes map[string]attr.Type = map[string]attr.Type{
	"artifact_filter_criteria": types.ObjectType{
		AttrTypes: artifactFilterCriteriaResourceModelAttributeTypes,
	},
}

var artifactFilterCriteriaResourceModelAttributeTypes map[string]attr.Type = map[string]attr.Type{
	"repo_keys":        types.SetType{ElemType: types.StringType},
	"include_patterns": types.SetType{ElemType: types.StringType},
	"exclude_patterns": types.SetType{ElemType: types.StringType},
}

func (r *workersServiceResourceModel) fromAPIModel(ctx context.Context, apiModel *WorkersServiceAPIModel) (ds diag.Diagnostics) {
	r.Key = types.StringValue(apiModel.Key)
	r.Description = types.StringValue(apiModel.Description)
	r.SourceCode = types.StringValue(apiModel.SourceCode)
	r.Action = types.StringValue(apiModel.Action)

	repoKeys, d := types.SetValueFrom(
		ctx,
		types.StringType,
		apiModel.FilterCriteria.ArtifactFilterCriteria.RepoKeys,
	)
	if d != nil {
		ds = append(ds, d...)
	}
	if ds.HasError() {
		return
	}
	includePatterns, d := types.SetValueFrom(
		ctx,
		types.StringType,
		apiModel.FilterCriteria.ArtifactFilterCriteria.IncludePatterns,
	)
	if d != nil {
		ds = append(ds, d...)
	}
	if ds.HasError() {
		return
	}
	excludePatterns, d := types.SetValueFrom(
		ctx,
		types.StringType,
		apiModel.FilterCriteria.ArtifactFilterCriteria.ExcludePatterns,
	)
	if d != nil {
		ds = append(ds, d...)
	}
	if ds.HasError() {
		return
	}

	artifactFilterCriteriaValue := artifactFilterCriteriaResourceModel{
		RepoKeys:        repoKeys,
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

	filterCriteria, d := types.ObjectValue(
		filterCriteriaResourceModelAttributeTypes,
		map[string]attr.Value{
			"artifact_filter_criteria": atrifactFilterCriteria,
		},
	)
	if d != nil {
		ds = append(ds, d...)
	}
	if ds.HasError() {
		return
	}

	r.FilterCriteria = filterCriteria
	r.Enabled = types.BoolValue(apiModel.Enabled)

	return
}

type WorkersServiceAPIModel struct {
	Key            string                 `json:"key"`
	Description    string                 `json:"description"`
	SourceCode     string                 `json:"sourceCode"`
	Action         string                 `json:"action"`
	FilterCriteria filterCriteriaAPIModel `json:"filterCriteria"`
	Enabled        bool                   `json:"enabled"`
	Secrets        []secretAPIModel       `json:"secrets"`
}

type filterCriteriaAPIModel struct {
	ArtifactFilterCriteria artifactFilterCriteriaAPIModel `json:"artifactFilterCriteria"`
}

type artifactFilterCriteriaAPIModel struct {
	RepoKeys        []string `json:"repoKeys"`
	IncludePatterns []string `json:"includePatterns,omitempty"`
	ExcludePatterns []string `json:"excludePatterns,omitempty"`
}

type secretAPIModel struct {
	Key              string `json:"key"`
	Value            string `json:"value"`
	MarkedForRemoval bool   `json:"markedForRemoval,omitempty"`
}

func (r *workersServiceResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}
	r.ProviderData = req.ProviderData.(PlatformProviderMetadata)
}

func (r *workersServiceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	go util.SendUsageResourceCreate(ctx, r.ProviderData.Client.R(), r.ProviderData.ProductId, r.TypeName)

	var plan workersServiceResourceModel

	diags := req.Config.Get(ctx, &plan)
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

	resp.Diagnostics.Append(req.Config.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	planSecrets := lo.Map[attr.Value](
		plan.Secrets.Elements(),
		func(elem attr.Value, index int) secretAPIModel {
			attrs := elem.(types.Object).Attributes()
			return secretAPIModel{
				Key: attrs["key"].(types.String).ValueString(),
			}
		},
	)

	stateSecrets := lo.Map[attr.Value](state.Secrets.Elements(), func(elem attr.Value, index int) secretAPIModel {
		attrs := elem.(types.Object).Attributes()
		return secretAPIModel{
			Key: attrs["key"].(types.String).ValueString(),
		}
	})

	_, secretsToBeRemoved := lo.Difference[secretAPIModel](planSecrets, stateSecrets)
	secretKeysToBeRemovedKeys := lo.Map[secretAPIModel](
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
