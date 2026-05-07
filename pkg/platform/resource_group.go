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

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/jfrog/terraform-provider-shared/util"
	utilfw "github.com/jfrog/terraform-provider-shared/util/fw"
	validatorfw "github.com/jfrog/terraform-provider-shared/validator/fw"
	"github.com/samber/lo"
)

var _ resource.Resource = (*groupResource)(nil)
var _ resource.ResourceWithValidateConfig = (*groupResource)(nil)

const groupRolesArtifactoryVersion = "7.128.0"

type groupResource struct {
	util.JFrogResource
}

func NewGroupResource() resource.Resource {
	return &groupResource{
		JFrogResource: util.JFrogResource{
			TypeName:                "platform_group",
			ValidArtifactoryVersion: "7.49.3",
			CollectionEndpoint:      "access/api/v2/groups",
			DocumentEndpoint:        "access/api/v2/groups/{name}",
		},
	}
}

var groupSchemaV0 = schema.Schema{
	Version: 0,
	Attributes: map[string]schema.Attribute{
		"name": schema.StringAttribute{
			Required: true,
			Validators: []validator.String{
				stringvalidator.LengthBetween(1, 64),
			},
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.RequiresReplace(),
			},
			MarkdownDescription: "Name of the group.",
		},
		"description": schema.StringAttribute{
			Optional: true,
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
			},
			Validators: []validator.String{
				stringvalidator.LengthAtLeast(1),
			},
			MarkdownDescription: "A description for the group. Must be non-empty when set; omit the attribute to leave the group with no description. The Access service normalizes empty strings to null on read, so allowing `\"\"` here would cause perpetual plan drift.",
		},
		"external_id": schema.StringAttribute{
			Optional: true,
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
			},
			MarkdownDescription: "New external group ID used to configure the corresponding group in Azure AD.",
		},
		"auto_join": schema.BoolAttribute{
			Optional: true,
			Computed: true,
			Default:  booldefault.StaticBool(false),
			Validators: []validator.Bool{
				validatorfw.BoolConflict(true, path.Expressions{
					path.MatchRelative().AtParent().AtName("admin_privileges"),
				}...),
			},
			MarkdownDescription: "When this parameter is set, any new users defined in the system are automatically assigned to this group.",
		},
		"admin_privileges": schema.BoolAttribute{
			Optional:            true,
			Computed:            true,
			Default:             booldefault.StaticBool(false),
			MarkdownDescription: "Any users added to this group will automatically be assigned with admin privileges in the system.",
		},
		"members": schema.SetAttribute{
			ElementType: types.StringType,
			Optional:    true,
			PlanModifiers: []planmodifier.Set{
				setplanmodifier.UseStateForUnknown(),
			},
			MarkdownDescription: "List of users assigned to the group.",
		},
		"realm": schema.StringAttribute{
			Computed: true,
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
			},
			MarkdownDescription: "The realm for the group.",
		},
		"realm_attributes": schema.StringAttribute{
			Computed: true,
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
			},
			MarkdownDescription: "The realm for the group.",
		},
	},
	MarkdownDescription: "Provides a group resource to create and manage groups, and manages membership. A group represents a role and is used with RBAC (Role-Based Access Control) rules. See [JFrog documentation](https://jfrog.com/help/r/jfrog-platform-administration-documentation/create-and-edit-groups) for more details.",
}

var groupSchemaV1 = schema.Schema{
	Version: 1,
	Attributes: lo.Assign(
		groupSchemaV0.Attributes,
		map[string]schema.Attribute{
			"members": schema.SetAttribute{
				ElementType: types.StringType,
				Optional:    true,
				PlanModifiers: []planmodifier.Set{
					setplanmodifier.UseStateForUnknown(),
				},
				MarkdownDescription: "List of users assigned to the group.",
				DeprecationMessage:  "Replaced by `platform_group_members` resource. This should not be used in combination with `platform_group_members` resource. Use `use_group_members_resource` attribute to control which resource manages group membership.",
			},
			"use_group_members_resource": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
				MarkdownDescription: "When set to `true`, this resource will ignore the `members` attributes and allow memberships to be managed by `platform_group_members` resource instead. Default value is `true`.",
			},
			"reports_manager": schema.BoolAttribute{
				Optional: true,
				Computed: true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
				MarkdownDescription: "Whether group has reports manager role. Available from Artifactory 7.128.0.",
			},
			"watch_manager": schema.BoolAttribute{
				Optional: true,
				Computed: true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
				MarkdownDescription: "Whether group has watch manager role. Available from Artifactory 7.128.0.",
			},
			"policy_manager": schema.BoolAttribute{
				Optional: true,
				Computed: true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
				MarkdownDescription: "Whether group has policy manager role. The policy manager role implies the policy viewer role on the server side: setting this to `true` together with `policy_viewer = false` is rejected at plan time. Omit `policy_viewer` or set it to `true`. Available from Artifactory 7.128.0.",
			},
			"policy_viewer": schema.BoolAttribute{
				Optional: true,
				Computed: true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
				MarkdownDescription: "Whether group has policy viewer role. Implied by `policy_manager`: when `policy_manager = true`, the server forces this attribute to `true`. Available from Artifactory 7.128.0.",
			},
			"manage_resources": schema.BoolAttribute{
				Optional: true,
				Computed: true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
				MarkdownDescription: "Whether group manages resources in the default project. Available from Artifactory 7.128.0.",
			},
			"manage_webhook": schema.BoolAttribute{
				Optional: true,
				Computed: true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
				MarkdownDescription: "Whether group has manage webhook permissions. Available from Artifactory 7.128.0.",
			},
		},
	),
	MarkdownDescription: groupSchemaV0.MarkdownDescription,
}

func (r *groupResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = groupSchemaV1
}

// ValidateConfig overrides the embedded JFrogResource.ValidateConfig to
// enforce two cross-field rules at plan time:
//  1. The policy_manager role implies policy_viewer on the server side, so the
//     combination policy_manager=true + policy_viewer=false would be silently
//     coerced by the API and surface as "Provider produced inconsistent result
//     after apply". Reject it up front.
//  2. The role boolean fields were introduced in Artifactory 7.128.0; reject
//     configurations that set any of them against an older server. The
//     resource as a whole still works against the base ValidArtifactoryVersion
//     (7.49.3) when these fields are omitted.
func (r *groupResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	r.JFrogResource.ValidateConfig(ctx, req, resp)
	if resp.Diagnostics.HasError() {
		return
	}

	var data groupResourceModelV1
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Rule 1: policy_manager implies policy_viewer.
	if data.PolicyManager.ValueBool() &&
		!data.PolicyViewer.IsNull() && !data.PolicyViewer.IsUnknown() &&
		!data.PolicyViewer.ValueBool() {
		resp.Diagnostics.AddAttributeError(
			path.Root("policy_viewer"),
			"Invalid Attribute Combination",
			"policy_viewer can not be set to false when policy_manager is true; the policy manager role implies the policy viewer role.",
		)
	}

	// Rule 2: role booleans require Artifactory >= groupRolesArtifactoryVersion.
	if r.ProviderData == nil || r.ProviderData.ArtifactoryVersion == "" {
		return
	}
	ok, err := util.CheckVersion(r.ProviderData.ArtifactoryVersion, groupRolesArtifactoryVersion)
	if err != nil {
		resp.Diagnostics.AddError("Failed to verify Artifactory version", err.Error())
		return
	}
	if ok {
		return
	}

	addIfSet := func(field types.Bool, attr string) {
		if !field.IsNull() && !field.IsUnknown() {
			resp.Diagnostics.AddAttributeError(
				path.Root(attr),
				"Incompatible Artifactory version",
				fmt.Sprintf("Attribute %q requires Artifactory %s or later. Detected version: %s.", attr, groupRolesArtifactoryVersion, r.ProviderData.ArtifactoryVersion),
			)
		}
	}
	addIfSet(data.ReportsManager, "reports_manager")
	addIfSet(data.WatchManager, "watch_manager")
	addIfSet(data.PolicyManager, "policy_manager")
	addIfSet(data.PolicyViewer, "policy_viewer")
	addIfSet(data.ManageResources, "manage_resources")
	addIfSet(data.ManageWebhook, "manage_webhook")
}

type groupResourceModelV0 struct {
	Name            types.String `tfsdk:"name"`
	Description     types.String `tfsdk:"description"`
	ExternalId      types.String `tfsdk:"external_id"`
	AutoJoin        types.Bool   `tfsdk:"auto_join"`
	AdminPrivileges types.Bool   `tfsdk:"admin_privileges"`
	Members         types.Set    `tfsdk:"members"`
	Realm           types.String `tfsdk:"realm"`
	RealmAttributes types.String `tfsdk:"realm_attributes"`
}

type groupResourceModelV1 struct {
	groupResourceModelV0
	UseGroupMembersResource types.Bool `tfsdk:"use_group_members_resource"`
	ReportsManager          types.Bool `tfsdk:"reports_manager"`
	WatchManager            types.Bool `tfsdk:"watch_manager"`
	PolicyManager           types.Bool `tfsdk:"policy_manager"`
	PolicyViewer            types.Bool `tfsdk:"policy_viewer"`
	ManageResources         types.Bool `tfsdk:"manage_resources"`
	ManageWebhook           types.Bool `tfsdk:"manage_webhook"`
}

func (r *groupResourceModelV1) toAPIModel(ctx context.Context, apiModel *groupAPIModel) (ds diag.Diagnostics) {

	var members []string

	if !r.UseGroupMembersResource.ValueBool() {
		ds.Append(r.Members.ElementsAs(ctx, &members, false)...)
		if ds.HasError() {
			return
		}
	}

	*apiModel = groupAPIModel{
		Name:            r.Name.ValueString(),
		Description:     r.Description.ValueStringPointer(),
		ExternalId:      r.ExternalId.ValueStringPointer(),
		AutoJoin:        r.AutoJoin.ValueBoolPointer(),
		AdminPrivileges: r.AdminPrivileges.ValueBoolPointer(),
		Members:         members,
		ReportsManager:  r.ReportsManager.ValueBoolPointer(),
		WatchManager:    r.WatchManager.ValueBoolPointer(),
		PolicyManager:   r.PolicyManager.ValueBoolPointer(),
		PolicyViewer:    r.PolicyViewer.ValueBoolPointer(),
		ManageResources: r.ManageResources.ValueBoolPointer(),
		ManageWebhook:   r.ManageWebhook.ValueBoolPointer(),
	}

	return nil
}

func (r *groupResourceModelV1) fromAPIModel(ctx context.Context, apiModel groupAPIModel, ignoreMembers bool) diag.Diagnostics {
	diags := diag.Diagnostics{}

	r.Name = types.StringValue(apiModel.Name)
	r.Description = types.StringPointerValue(apiModel.Description)
	r.ExternalId = types.StringPointerValue(apiModel.ExternalId)
	r.AutoJoin = types.BoolPointerValue(apiModel.AutoJoin)
	r.AdminPrivileges = types.BoolPointerValue(apiModel.AdminPrivileges)
	r.Realm = types.StringPointerValue(apiModel.Realm)
	r.RealmAttributes = types.StringPointerValue(apiModel.RealmAttributes)
	r.ReportsManager = types.BoolPointerValue(apiModel.ReportsManager)
	r.WatchManager = types.BoolPointerValue(apiModel.WatchManager)
	r.PolicyManager = types.BoolPointerValue(apiModel.PolicyManager)
	r.PolicyViewer = types.BoolPointerValue(apiModel.PolicyViewer)
	r.ManageResources = types.BoolPointerValue(apiModel.ManageResources)
	r.ManageWebhook = types.BoolPointerValue(apiModel.ManageWebhook)

	if !r.UseGroupMembersResource.ValueBool() {
		if r.Members.IsUnknown() {
			r.Members = types.SetNull(types.StringType)
		}

		if !ignoreMembers && len(apiModel.Members) > 0 {
			members, d := types.SetValueFrom(ctx, types.StringType, apiModel.Members)
			if d != nil {
				diags.Append(d...)
				return diags
			}

			r.Members = members
		}
	}

	return diags
}

func (r *groupResource) UpgradeState(ctx context.Context) map[int64]resource.StateUpgrader {
	return map[int64]resource.StateUpgrader{
		// State upgrade implementation from 0 (prior state version) to 1 (Schema.Version)
		0: {
			PriorSchema: &groupSchemaV0,
			StateUpgrader: func(ctx context.Context, req resource.UpgradeStateRequest, resp *resource.UpgradeStateResponse) {
				var priorStateData groupResourceModelV0

				resp.Diagnostics.Append(req.State.Get(ctx, &priorStateData)...)
				if resp.Diagnostics.HasError() {
					return
				}

				upgradedStateData := groupResourceModelV1{
					groupResourceModelV0:    priorStateData,
					UseGroupMembersResource: types.BoolValue(false),
				}

				resp.Diagnostics.Append(resp.State.Set(ctx, upgradedStateData)...)
			},
		},
	}
}

type groupAPIModel struct {
	Name            string   `json:"name"`
	Description     *string  `json:"description,omitempty"`
	ExternalId      *string  `json:"external_id,omitempty"`
	AutoJoin        *bool    `json:"auto_join,omitempty"`
	AdminPrivileges *bool    `json:"admin_privileges,omitempty"`
	Members         []string `json:"members,omitempty"`          // only for create
	Realm           *string  `json:"realm,omitempty"`            // read only
	RealmAttributes *string  `json:"realm_attributes,omitempty"` // read only
	// Available from Artifactory 7.128.0
	ReportsManager  *bool `json:"reports_manager,omitempty"`
	WatchManager    *bool `json:"watch_manager,omitempty"`
	PolicyManager   *bool `json:"policy_manager,omitempty"`
	PolicyViewer    *bool `json:"policy_viewer,omitempty"`
	ManageResources *bool `json:"manage_resources,omitempty"`
	ManageWebhook   *bool `json:"manage_webhook,omitempty"`
}

type groupMembersRequestAPIModel struct {
	Add    []string `json:"add"`
	Remove []string `json:"remove"`
}

type groupMembersResponseAPIModel struct {
	Members []string `json:"members"`
}

func (r *groupResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	go util.SendUsageResourceCreate(ctx, r.ProviderData.Client.R(), r.ProviderData.ProductId, r.TypeName)

	var plan *groupResourceModelV1
	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var group groupAPIModel
	resp.Diagnostics.Append(plan.toAPIModel(ctx, &group)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var newGroup groupAPIModel
	var apiErrs util.JFrogErrors
	response, err := r.ProviderData.Client.R().
		SetBody(group).
		SetResult(&newGroup).
		SetError(&apiErrs).
		Post(r.JFrogResource.CollectionEndpoint)

	if err != nil {
		utilfw.UnableToCreateResourceError(resp, err.Error())
		return
	}

	if response.StatusCode() != http.StatusCreated {
		utilfw.UnableToCreateResourceError(resp, apiErrs.String())
		return
	}

	plan.Realm = types.StringPointerValue(newGroup.Realm)
	plan.RealmAttributes = types.StringPointerValue(newGroup.RealmAttributes)

	// The role booleans are Optional+Computed (no Default), so when the user
	// omits them in HCL the planned value is unknown. We must resolve them to
	// known values from the POST response before saving state, otherwise
	// Terraform errors with "All values must be known after apply". On older
	// Artifactory (< 7.128.0) the response omits these fields and the
	// pointers are nil, which BoolPointerValue maps to null - still known.
	plan.ReportsManager = types.BoolPointerValue(newGroup.ReportsManager)
	plan.WatchManager = types.BoolPointerValue(newGroup.WatchManager)
	plan.PolicyManager = types.BoolPointerValue(newGroup.PolicyManager)
	plan.PolicyViewer = types.BoolPointerValue(newGroup.PolicyViewer)
	plan.ManageResources = types.BoolPointerValue(newGroup.ManageResources)
	plan.ManageWebhook = types.BoolPointerValue(newGroup.ManageWebhook)

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *groupResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	go util.SendUsageResourceRead(ctx, r.ProviderData.Client.R(), r.ProviderData.ProductId, r.TypeName)

	var state *groupResourceModelV1
	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var group groupAPIModel
	var apiErrs util.JFrogErrors
	response, err := r.ProviderData.Client.R().
		SetPathParam("name", state.Name.ValueString()).
		SetResult(&group).
		SetError(&apiErrs).
		Get(r.JFrogResource.DocumentEndpoint)

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
		utilfw.UnableToRefreshResourceError(resp, apiErrs.String())
		return
	}

	// Convert from the API data model to the Terraform data model
	// and refresh any attribute values.
	resp.Diagnostics.Append(state.fromAPIModel(ctx, group, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *groupResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	go util.SendUsageResourceUpdate(ctx, r.ProviderData.Client.R(), r.ProviderData.ProductId, r.TypeName)

	var plan groupResourceModelV1
	var state groupResourceModelV1

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var group groupAPIModel
	resp.Diagnostics.Append(plan.toAPIModel(ctx, &group)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var updatedGroup groupAPIModel
	var apiErrs util.JFrogErrors
	response, err := r.ProviderData.Client.R().
		SetPathParam("name", plan.Name.ValueString()).
		SetBody(group).
		SetResult(&updatedGroup).
		SetError(&apiErrs).
		Patch(r.JFrogResource.DocumentEndpoint)
	if err != nil {
		utilfw.UnableToUpdateResourceError(resp, err.Error())
		return
	}

	if response.IsError() {
		utilfw.UnableToUpdateResourceError(resp, apiErrs.String())
		return
	}

	resp.Diagnostics.Append(plan.fromAPIModel(ctx, updatedGroup, true)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var planMembers []string
	resp.Diagnostics.Append(plan.Members.ElementsAs(ctx, &planMembers, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var stateMembers []string
	resp.Diagnostics.Append(state.Members.ElementsAs(ctx, &stateMembers, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	memebersToAdd, membersToRemove := lo.Difference(planMembers, stateMembers)
	membersReq := groupMembersRequestAPIModel{
		Add:    memebersToAdd,
		Remove: membersToRemove,
	}

	if len(memebersToAdd) > 0 || len(membersToRemove) > 0 {
		var membersRes groupMembersResponseAPIModel
		response, err = r.ProviderData.Client.R().
			SetPathParam("name", plan.Name.ValueString()).
			SetBody(membersReq).
			SetResult(&membersRes).
			SetError(&apiErrs).
			Patch(r.JFrogResource.DocumentEndpoint + "/members")
		if err != nil {
			utilfw.UnableToUpdateResourceError(resp, err.Error())
			return
		}

		if response.IsError() {
			utilfw.UnableToUpdateResourceError(resp, apiErrs.String())
			return
		}

		// only update members attribute if it is set in the configuration
		if !plan.Members.IsNull() {
			ms, d := types.SetValueFrom(ctx, types.StringType, membersRes.Members)
			if d != nil {
				resp.Diagnostics.Append(d...)
				return
			}

			plan.Members = ms
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *groupResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	go util.SendUsageResourceDelete(ctx, r.ProviderData.Client.R(), r.ProviderData.ProductId, r.TypeName)

	var state groupResourceModelV1

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	var apiErrs util.JFrogErrors
	response, err := r.ProviderData.Client.R().
		SetPathParam("name", state.Name.ValueString()).
		SetError(&apiErrs).
		Delete(r.JFrogResource.DocumentEndpoint)

	if err != nil {
		utilfw.UnableToDeleteResourceError(resp, err.Error())
		return
	}

	// Return error if the HTTP status code is not 204 No Content or 404 Not Found
	if response.StatusCode() != http.StatusNotFound && response.StatusCode() != http.StatusNoContent {
		utilfw.UnableToDeleteResourceError(resp, apiErrs.String())
		return
	}

	// If the logic reaches here, it implicitly succeeded and will remove
	// the resource from state if there are no other errors.
}

// ImportState imports the resource into the Terraform state.
func (r *groupResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("name"), req, resp)
}
