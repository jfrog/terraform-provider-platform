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
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/jfrog/terraform-provider-shared/util"
	utilfw "github.com/jfrog/terraform-provider-shared/util/fw"
	validatorfw_string "github.com/jfrog/terraform-provider-shared/validator/fw/string"
)

func NewCrowdSettingsResource() resource.Resource {
	return &CrowdSettingsResource{
		JFrogResource: util.JFrogResource{
			TypeName:                "platform_crowd_settings",
			ValidArtifactoryVersion: "7.64.0",
			DocumentEndpoint:        "access/api/v1/crowd",
		},
	}
}

type CrowdSettingsResource struct {
	util.JFrogResource
}

type CrowdSettingsResourceModel struct {
	Enable                     types.Bool   `tfsdk:"enable"`
	ServerURL                  types.String `tfsdk:"server_url"`
	ApplicationName            types.String `tfsdk:"application_name"`
	Password                   types.String `tfsdk:"password"`
	SessionValidationInterval  types.Int64  `tfsdk:"session_validation_interval"`
	UseDefaultProxy            types.Bool   `tfsdk:"use_default_proxy"`
	AutoUserCreation           types.Bool   `tfsdk:"auto_user_creation"`
	AllowUserToAccessProfile   types.Bool   `tfsdk:"allow_user_to_access_profile"`
	DirectAuthentication       types.Bool   `tfsdk:"direct_authentication"`
	OverrideAllGroupsUponLogin types.Bool   `tfsdk:"override_all_groups_upon_login"`
}

func (r *CrowdSettingsResourceModel) toAPIModel(_ context.Context, apiModel *CrowdSettingsAPIModel) diag.Diagnostics {
	diags := diag.Diagnostics{}

	apiModel.Enable = r.Enable.ValueBool()
	apiModel.ServerURL = r.ServerURL.ValueString()
	apiModel.ApplicationName = r.ApplicationName.ValueString()
	apiModel.Password = r.Password.ValueString()
	apiModel.SessionValidationInterval = r.SessionValidationInterval.ValueInt64()
	apiModel.UseDefaultProxy = r.UseDefaultProxy.ValueBoolPointer()
	apiModel.AutoUserCreation = r.AutoUserCreation.ValueBoolPointer()
	apiModel.AllowUserToAccessProfile = r.AllowUserToAccessProfile.ValueBoolPointer()
	apiModel.DirectAuthentication = r.DirectAuthentication.ValueBoolPointer()
	apiModel.OverrideAllGroupsUponLogin = r.OverrideAllGroupsUponLogin.ValueBoolPointer()

	return diags
}

func (r *CrowdSettingsResourceModel) fromAPIModel(_ context.Context, apiModel *CrowdSettingsAPIModel) (ds diag.Diagnostics) {
	r.Enable = types.BoolValue(apiModel.Enable)
	r.ServerURL = types.StringValue(apiModel.ServerURL)
	r.ApplicationName = types.StringValue(apiModel.ApplicationName)
	r.SessionValidationInterval = types.Int64Value(apiModel.SessionValidationInterval)
	r.UseDefaultProxy = types.BoolPointerValue(apiModel.UseDefaultProxy)
	r.AutoUserCreation = types.BoolPointerValue(apiModel.AutoUserCreation)
	r.AllowUserToAccessProfile = types.BoolPointerValue(apiModel.AllowUserToAccessProfile)
	r.DirectAuthentication = types.BoolPointerValue(apiModel.DirectAuthentication)
	r.OverrideAllGroupsUponLogin = types.BoolPointerValue(apiModel.OverrideAllGroupsUponLogin)

	return
}

type CrowdSettingsAPIModel struct {
	Enable                     bool   `json:"enable_integration"`
	ServerURL                  string `json:"server_url"`
	ApplicationName            string `json:"application_name"`
	Password                   string `json:"password"`
	SessionValidationInterval  int64  `json:"session_validation_interval"`
	UseDefaultProxy            *bool  `json:"use_default_proxy"`
	AutoUserCreation           *bool  `json:"auto_user_creation"`
	AllowUserToAccessProfile   *bool  `json:"allow_user_to_access_profile"`
	DirectAuthentication       *bool  `json:"direct_authentication"`
	OverrideAllGroupsUponLogin *bool  `json:"override_all_groups_upon_login"`
}

func (r *CrowdSettingsResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"enable": schema.BoolAttribute{
				Required:            true,
				MarkdownDescription: "Use this to enable security integration with Atlassian Crowd or JIRA.",
			},
			"server_url": schema.StringAttribute{
				Required: true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
					validatorfw_string.IsURLHttpOrHttps(),
				},
				MarkdownDescription: "The full URL of the server to use.",
			},
			"application_name": schema.StringAttribute{
				Required: true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
				MarkdownDescription: "The application name configured for JPD in Crowd/JIRA.",
			},
			"password": schema.StringAttribute{
				Required:  true,
				Sensitive: true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
				MarkdownDescription: "The application password configured for JPD in Crowd/JIRA.",
			},
			"session_validation_interval": schema.Int64Attribute{
				Required: true,
				Validators: []validator.Int64{
					int64validator.AtLeast(0),
				},
				MarkdownDescription: "The time window (min) during which the session does not need to be validated. If set to `0`, the token expires only when the session expires.",
			},
			"use_default_proxy": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
				MarkdownDescription: "If a default proxy definition exists, it is used to pass through to the Crowd Server. Default value is `false`.",
			},
			"auto_user_creation": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
				MarkdownDescription: "When set, authenticated users are automatically created in Artifactory. When not set, for every request from a Crowd user, the user is temporarily associated with default groups (if such groups are defined), and the permissions for these groups apply. Without automatic user creation, you must manually create the user in Artifactory to manage user permissions not attached to their default groups. Default value is `true`.",
			},
			"allow_user_to_access_profile": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
				MarkdownDescription: "Auto created users will have access to their profile page and will be able to perform actions such as generating an API key. Default value is `false`.",
			},
			"direct_authentication": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
				MarkdownDescription: "This corresponds to 'Users Management Server' option in Artifactory UI (`true` = JIRA, `false` = Crowd). Default value is `false`.",
			},
			"override_all_groups_upon_login": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
				MarkdownDescription: "When a user logs in with CROWD, only groups retrieved from CROWD will be associated with the user. Default value is `false`.",
			},
		},
		MarkdownDescription: "Provides a JFrog [Crowd Settings](https://jfrog.com/help/r/jfrog-platform-administration-documentation/atlassian-crowd-and-jira-integration) resource. This allows you to delegate authentication requests to Atlassian Crowd/JIRA, use authenticated Crowd/JIRA users and have the JPD participate in a transparent SSO environment managed by Crowd/JIRA.",
	}
}

func (r *CrowdSettingsResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	go util.SendUsageResourceCreate(ctx, r.ProviderData.Client.R(), r.ProviderData.ProductId, r.TypeName)

	var plan CrowdSettingsResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var crowdSettings CrowdSettingsAPIModel
	resp.Diagnostics.Append(plan.toAPIModel(ctx, &crowdSettings)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var jfrogErrors util.JFrogErrors
	response, err := r.ProviderData.Client.R().
		SetBody(crowdSettings).
		SetError(&jfrogErrors).
		Put(r.DocumentEndpoint)

	if err != nil {
		utilfw.UnableToCreateResourceError(resp, err.Error())
		return
	}

	if response.IsError() {
		utilfw.UnableToCreateResourceError(resp, jfrogErrors.String())
		return
	}

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *CrowdSettingsResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	go util.SendUsageResourceRead(ctx, r.ProviderData.Client.R(), r.ProviderData.ProductId, r.TypeName)

	var state CrowdSettingsResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var crowdSettings CrowdSettingsAPIModel

	response, err := r.ProviderData.Client.R().
		SetResult(&crowdSettings).
		Get(r.DocumentEndpoint)

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
	resp.Diagnostics.Append(state.fromAPIModel(ctx, &crowdSettings)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *CrowdSettingsResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	go util.SendUsageResourceUpdate(ctx, r.ProviderData.Client.R(), r.ProviderData.ProductId, r.TypeName)

	var plan CrowdSettingsResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var crowdSettings CrowdSettingsAPIModel
	resp.Diagnostics.Append(plan.toAPIModel(ctx, &crowdSettings)...)
	if resp.Diagnostics.HasError() {
		return
	}

	response, err := r.ProviderData.Client.R().
		SetBody(crowdSettings).
		Put(r.DocumentEndpoint)

	if err != nil {
		utilfw.UnableToUpdateResourceError(resp, err.Error())
		return
	}

	if response.IsError() {
		utilfw.UnableToUpdateResourceError(resp, response.String())
		return
	}

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *CrowdSettingsResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	go util.SendUsageResourceDelete(ctx, r.ProviderData.Client.R(), r.ProviderData.ProductId, r.TypeName)

	resp.Diagnostics.AddWarning(
		"Unable to Delete Resource",
		"Crowd settings cannot be deleted.",
	)
}

func (r *CrowdSettingsResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("server_url"), req, resp)
}
