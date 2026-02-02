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

	"github.com/hashicorp/terraform-plugin-framework-validators/boolvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/jfrog/terraform-provider-shared/util"
	utilfw "github.com/jfrog/terraform-provider-shared/util/fw"
	validatorfw_string "github.com/jfrog/terraform-provider-shared/validator/fw/string"
	"github.com/samber/lo"
)

func NewSAMLSettingsResource() resource.Resource {
	return &SAMLSettingsResource{
		JFrogResource: util.JFrogResource{
			TypeName:                "platform_saml_settings",
			ValidArtifactoryVersion: "7.83.1",
			DocumentEndpoint:        "access/api/v1/saml/{name}",
			CollectionEndpoint:      "access/api/v1/saml",
		},
	}
}

type SAMLSettingsResource struct {
	util.JFrogResource
}

type SAMLSettingsResourceModelV0 struct {
	Name                      types.String `tfsdk:"name"`
	Enable                    types.Bool   `tfsdk:"enable"`
	Certificate               types.String `tfsdk:"certificate"`
	EmailAttribute            types.String `tfsdk:"email_attribute"`
	GroupAttribute            types.String `tfsdk:"group_attribute"`
	NameIDAttribute           types.String `tfsdk:"name_id_attribute"`
	LoginURL                  types.String `tfsdk:"login_url"`
	LogoutURL                 types.String `tfsdk:"logout_url"`
	NoAutoUserCreation        types.Bool   `tfsdk:"no_auto_user_creation"`
	ServiceProviderName       types.String `tfsdk:"service_provider_name"`
	AllowUserToAccessProfile  types.Bool   `tfsdk:"allow_user_to_access_profile"`
	AutoRedirect              types.Bool   `tfsdk:"auto_redirect"`
	SyncGroups                types.Bool   `tfsdk:"sync_groups"`
	VerifyAudienceRestriction types.Bool   `tfsdk:"verify_audience_restriction"`
	UseEncryptedAssertion     types.Bool   `tfsdk:"use_encrypted_assertion"`
}

type SAMLSettingsResourceModelV1 struct {
	SAMLSettingsResourceModelV0
	AutoUserCreation  types.Bool `tfsdk:"auto_user_creation"`
	LDAPGroupSettings types.Set  `tfsdk:"ldap_group_settings"`
}

type SAMLSettingsResourceModelV2 struct {
	Name                      types.String `tfsdk:"name"`
	Enable                    types.Bool   `tfsdk:"enable"`
	Certificate               types.String `tfsdk:"certificate"`
	EmailAttribute            types.String `tfsdk:"email_attribute"`
	GroupAttribute            types.String `tfsdk:"group_attribute"`
	NameIDAttribute           types.String `tfsdk:"name_id_attribute"`
	LoginURL                  types.String `tfsdk:"login_url"`
	LogoutURL                 types.String `tfsdk:"logout_url"`
	ServiceProviderName       types.String `tfsdk:"service_provider_name"`
	AllowUserToAccessProfile  types.Bool   `tfsdk:"allow_user_to_access_profile"`
	AutoRedirect              types.Bool   `tfsdk:"auto_redirect"`
	SyncGroups                types.Bool   `tfsdk:"sync_groups"`
	VerifyAudienceRestriction types.Bool   `tfsdk:"verify_audience_restriction"`
	UseEncryptedAssertion     types.Bool   `tfsdk:"use_encrypted_assertion"`
	AutoUserCreation          types.Bool   `tfsdk:"auto_user_creation"`
	LDAPGroupSettings         types.Set    `tfsdk:"ldap_group_settings"`
}

func (r *SAMLSettingsResourceModelV2) toAPIModel(ctx context.Context, apiModel *SAMLSettingsAPIModel) diag.Diagnostics {
	diags := diag.Diagnostics{}

	apiModel.Name = r.Name.ValueString()
	apiModel.Enable = r.Enable.ValueBool()
	apiModel.Certificate = r.Certificate.ValueString()
	apiModel.EmailAttribute = r.EmailAttribute.ValueString()
	apiModel.GroupAttribute = r.GroupAttribute.ValueString()
	apiModel.NameIDAttribute = r.NameIDAttribute.ValueString()
	apiModel.LoginURL = r.LoginURL.ValueString()
	apiModel.LogoutURL = r.LogoutURL.ValueString()
	apiModel.AutoUserCreation = r.AutoUserCreation.ValueBool()
	apiModel.ServiceProviderName = r.ServiceProviderName.ValueString()
	apiModel.AllowUserToAccessProfile = r.AllowUserToAccessProfile.ValueBool()
	apiModel.AutoRedirect = r.AutoRedirect.ValueBool()
	apiModel.SyncGroups = r.SyncGroups.ValueBool()
	apiModel.VerifyAudienceRestriction = r.VerifyAudienceRestriction.ValueBool()
	apiModel.UseEncryptedAssertion = r.UseEncryptedAssertion.ValueBool()

	ldapGroupSettings := []string{} // API treats absent or null value as noop so needs empty array to reset
	if !r.LDAPGroupSettings.IsNull() {
		d := r.LDAPGroupSettings.ElementsAs(ctx, &ldapGroupSettings, false)
		if d.HasError() {
			diags.Append(d...)
			return diags
		}
	}

	apiModel.LDAPGroupSettings = ldapGroupSettings

	return diags
}

func (r *SAMLSettingsResourceModelV2) fromAPIModel(ctx context.Context, apiModel *SAMLSettingsAPIModel) (ds diag.Diagnostics) {
	r.Name = types.StringValue(apiModel.Name)
	r.Enable = types.BoolValue(apiModel.Enable)
	r.Certificate = types.StringValue(apiModel.Certificate)

	if len(apiModel.EmailAttribute) > 0 {
		r.EmailAttribute = types.StringValue(apiModel.EmailAttribute)
	}

	if len(apiModel.GroupAttribute) > 0 {
		r.GroupAttribute = types.StringValue(apiModel.GroupAttribute)
	}

	if len(apiModel.NameIDAttribute) > 0 {
		r.NameIDAttribute = types.StringValue(apiModel.NameIDAttribute)
	}

	r.LoginURL = types.StringValue(apiModel.LoginURL)
	r.LogoutURL = types.StringValue(apiModel.LogoutURL)
	r.AutoUserCreation = types.BoolValue(apiModel.AutoUserCreation)
	r.ServiceProviderName = types.StringValue(apiModel.ServiceProviderName)
	r.AllowUserToAccessProfile = types.BoolValue(apiModel.AllowUserToAccessProfile)
	r.AutoRedirect = types.BoolValue(apiModel.AutoRedirect)
	r.SyncGroups = types.BoolValue(apiModel.SyncGroups)
	r.VerifyAudienceRestriction = types.BoolValue(apiModel.VerifyAudienceRestriction)
	r.UseEncryptedAssertion = types.BoolValue(apiModel.UseEncryptedAssertion)

	ldapGroupSettings := types.SetNull(types.StringType)
	if len(apiModel.LDAPGroupSettings) > 0 {
		ls, d := types.SetValueFrom(ctx, types.StringType, apiModel.LDAPGroupSettings)
		if d.HasError() {
			ds.Append(d...)
			return
		}

		ldapGroupSettings = ls
	}

	r.LDAPGroupSettings = ldapGroupSettings

	return
}

type SAMLSettingsAPIModel struct {
	Name                      string   `json:"name"`
	Enable                    bool     `json:"enable_integration"`
	VerifyAudienceRestriction bool     `json:"verify_audience_restriction"`
	LoginURL                  string   `json:"login_url"`
	LogoutURL                 string   `json:"logout_url"`
	ServiceProviderName       string   `json:"service_provider_name"`
	AutoUserCreation          bool     `json:"auto_user_creation"`
	AllowUserToAccessProfile  bool     `json:"allow_user_to_access_profile"`
	UseEncryptedAssertion     bool     `json:"use_encrypted_assertion"`
	AutoRedirect              bool     `json:"auto_redirect"`
	SyncGroups                bool     `json:"sync_groups"`
	Certificate               string   `json:"certificate"`
	GroupAttribute            string   `json:"group_attribute"`
	EmailAttribute            string   `json:"email_attribute"`
	NameIDAttribute           string   `json:"name_id_attribute"`
	LDAPGroupSettings         []string `json:"ldap_group_settings"`
}

var samlSettingsSchemaV0 = map[string]schema.Attribute{
	"name": schema.StringAttribute{
		Required: true,
		Validators: []validator.String{
			stringvalidator.LengthAtLeast(1),
		},
		PlanModifiers: []planmodifier.String{
			stringplanmodifier.RequiresReplace(),
		},
		Description: "SAML Settings name.",
	},
	"enable": schema.BoolAttribute{
		Optional:            true,
		Computed:            true,
		Default:             booldefault.StaticBool(true),
		MarkdownDescription: "When set, SAML integration is enabled and users may be authenticated via a SAML server. Default value is `true`.",
	},
	"certificate": schema.StringAttribute{
		Required:  true,
		Sensitive: true,
		Validators: []validator.String{
			stringvalidator.LengthAtLeast(1),
		},
		MarkdownDescription: "The certificate for SAML Authentication in Base64 format. NOTE! The certificate must contain the public key to allow Artifactory to verify sign-in requests.",
	},
	"email_attribute": schema.StringAttribute{
		Optional: true,
		Validators: []validator.String{
			stringvalidator.LengthAtLeast(1),
		},
		MarkdownDescription: "If `auto_user_creation` is enabled or an internal user exists, the system will set the user's email to the value in this attribute that is returned by the SAML login XML response.",
	},
	"group_attribute": schema.StringAttribute{
		Optional: true,
		Validators: []validator.String{
			stringvalidator.LengthAtLeast(1),
		},
		MarkdownDescription: "The group attribute in the SAML login XML response. Note that the system will search for a case-sensitive match to an existing group..",
	},
	"name_id_attribute": schema.StringAttribute{
		Optional: true,
		Validators: []validator.String{
			stringvalidator.LengthAtLeast(1),
		},
		MarkdownDescription: "The username attribute used to configure the SSO URL for the identity provider.",
	},
	"login_url": schema.StringAttribute{
		Required: true,
		Validators: []validator.String{
			stringvalidator.LengthAtLeast(1),
			validatorfw_string.IsURLHttpOrHttps(),
		},
		Description: "The identity provider login URL (when you try to login, the service provider redirects to this URL).",
	},
	"logout_url": schema.StringAttribute{
		Required: true,
		Validators: []validator.String{
			stringvalidator.LengthAtLeast(1),
			validatorfw_string.IsURLHttpOrHttps(),
		},
		Description: "The identity provider logout URL (when you try to logout, the service provider redirects to this URL).",
	},
	"no_auto_user_creation": schema.BoolAttribute{
		Optional:            true,
		Computed:            true,
		Default:             booldefault.StaticBool(false),
		MarkdownDescription: "When disabled, the system will automatically create new users for those who have logged in using SAML, and assign them to the default groups. Default value is `false`.",
	},
	"service_provider_name": schema.StringAttribute{
		Required: true,
		Validators: []validator.String{
			stringvalidator.LengthAtLeast(1),
		},
		MarkdownDescription: "The SAML service provider name. This should be a URI that is also known as the entityID, providerID, or entity identity.",
	},
	"allow_user_to_access_profile": schema.BoolAttribute{
		Optional:            true,
		Computed:            true,
		Default:             booldefault.StaticBool(true),
		MarkdownDescription: "When set, auto created users will have access to their profile page and will be able to perform actions such as generating an API key. Default value is `false`.",
	},
	"auto_redirect": schema.BoolAttribute{
		Optional:            true,
		Computed:            true,
		Default:             booldefault.StaticBool(false),
		MarkdownDescription: "When set, clicking on the login link will direct users to the configured SAML login URL. Default value is `false`.",
	},
	"sync_groups": schema.BoolAttribute{
		Optional:            true,
		Computed:            true,
		Default:             booldefault.StaticBool(false),
		MarkdownDescription: "When set, in addition to the groups the user is already associated with, he will also be associated with the groups returned in the SAML login response. Note that the user's association with the returned groups is not persistent. It is only valid for the current login session. Default value is `false`.",
	},
	"verify_audience_restriction": schema.BoolAttribute{
		Optional:            true,
		Computed:            true,
		Default:             booldefault.StaticBool(true),
		MarkdownDescription: "Set this flag to specify who the assertion is intended for. The \"audience\" will be the service provider and is typically a URL but can technically be formatted as any string of data. Default value is `true`.",
	},
	"use_encrypted_assertion": schema.BoolAttribute{
		Optional:            true,
		Computed:            true,
		Default:             booldefault.StaticBool(false),
		MarkdownDescription: "When set, an X.509 public certificate will be created by Artifactory. Download this certificate and upload it to your IDP and choose your own encryption algorithm. This process will let you encrypt the assertion section in your SAML response. Default value is `false`.",
	},
}

var samlSettingsSchemaV1 = lo.Assign(
	samlSettingsSchemaV0,
	map[string]schema.Attribute{
		"no_auto_user_creation": schema.BoolAttribute{
			Optional:            true,
			Computed:            true,
			Default:             booldefault.StaticBool(false),
			MarkdownDescription: "**Deprecated** Use `auto_user_creation` instead. When disabled, the system will automatically create new users for those who have logged in using SAML, and assign them to the default groups. Default value is `false`.",
			DeprecationMessage:  "Use `auto_user_creation` instead.",
		},
		"auto_user_creation": schema.BoolAttribute{
			Optional: true,
			Computed: true,
			Default:  booldefault.StaticBool(true),
			Validators: []validator.Bool{
				boolvalidator.ExactlyOneOf(
					path.MatchRoot("no_auto_user_creation"),
				),
			},
			MarkdownDescription: "When set, authenticated users are automatically created in Artifactory. When not set, for every request from an SSO user, the user is temporarily associated with default groups (if such groups are defined), and the permissions for these groups apply. Without automatic user creation, you must manually create the user inside Artifactory to manage user permissions not attached to their default groups. Default value is `true`.",
		},
		"ldap_group_settings": schema.SetAttribute{
			ElementType: types.StringType,
			Validators: []validator.Set{
				setvalidator.SizeAtLeast(1),
			},
			Optional:            true,
			MarkdownDescription: "List of LDAP group setting names. Only support in Artifactory 7.98 or later. See [Enabling Synchronization of LDAP Groups for SAML SSO](https://jfrog.com/help/r/jfrog-platform-administration-documentation/enabling-synchronization-of-ldap-groups-for-saml-sso) for more details.",
		},
	},
)

var samlSettingsSchemaV2 = lo.Assign(
	lo.OmitByKeys(
		samlSettingsSchemaV1,
		[]string{"no_auto_user_creation"},
	),
	map[string]schema.Attribute{
		"auto_user_creation": schema.BoolAttribute{
			Optional:            true,
			Computed:            true,
			Default:             booldefault.StaticBool(true),
			MarkdownDescription: "When set, authenticated users are automatically created in Artifactory. When not set, for every request from an SSO user, the user is temporarily associated with default groups (if such groups are defined), and the permissions for these groups apply. Without automatic user creation, you must manually create the user inside Artifactory to manage user permissions not attached to their default groups. Default value is `true`.",
		},
	},
)

func (r *SAMLSettingsResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Version:             2,
		Attributes:          samlSettingsSchemaV2,
		MarkdownDescription: "Provides a JFrog [SAML SSO Settings](https://jfrog.com/help/r/jfrog-platform-administration-documentation/saml-sso) resource.\n\n~>This resource supports both JFrog SaaS and Self-Hosted instances. For SaaS instances, the `enable` parameter must currently be activated via a manual API call after the Terraform apply is complete.",
	}
}

func (r *SAMLSettingsResource) UpgradeState(ctx context.Context) map[int64]resource.StateUpgrader {
	return map[int64]resource.StateUpgrader{
		// State upgrade implementation from 0 (prior state version) to 2 (Schema.Version)
		0: {
			PriorSchema: &schema.Schema{
				Attributes: samlSettingsSchemaV0,
			},
			StateUpgrader: func(ctx context.Context, req resource.UpgradeStateRequest, resp *resource.UpgradeStateResponse) {
				var priorStateData SAMLSettingsResourceModelV0

				resp.Diagnostics.Append(req.State.Get(ctx, &priorStateData)...)
				if resp.Diagnostics.HasError() {
					return
				}

				upgradedStateData := SAMLSettingsResourceModelV2{
					Name:                      priorStateData.Name,
					Enable:                    priorStateData.Enable,
					Certificate:               priorStateData.Certificate,
					EmailAttribute:            priorStateData.EmailAttribute,
					GroupAttribute:            priorStateData.GroupAttribute,
					NameIDAttribute:           priorStateData.NameIDAttribute,
					LoginURL:                  priorStateData.LoginURL,
					LogoutURL:                 priorStateData.LogoutURL,
					ServiceProviderName:       priorStateData.ServiceProviderName,
					AllowUserToAccessProfile:  priorStateData.AllowUserToAccessProfile,
					AutoRedirect:              priorStateData.AutoRedirect,
					SyncGroups:                priorStateData.SyncGroups,
					VerifyAudienceRestriction: priorStateData.VerifyAudienceRestriction,
					UseEncryptedAssertion:     priorStateData.UseEncryptedAssertion,
					AutoUserCreation:          types.BoolValue(!priorStateData.NoAutoUserCreation.ValueBool()),
					LDAPGroupSettings:         types.SetNull(types.StringType),
				}

				resp.Diagnostics.Append(resp.State.Set(ctx, upgradedStateData)...)
			},
		},
		// State upgrade implementation from 1 (prior state version) to 2 (Schema.Version)
		1: {
			PriorSchema: &schema.Schema{
				Attributes: samlSettingsSchemaV1,
			},
			StateUpgrader: func(ctx context.Context, req resource.UpgradeStateRequest, resp *resource.UpgradeStateResponse) {
				var priorStateData SAMLSettingsResourceModelV1

				resp.Diagnostics.Append(req.State.Get(ctx, &priorStateData)...)
				if resp.Diagnostics.HasError() {
					return
				}

				autoUserCreation := priorStateData.AutoUserCreation.ValueBool()
				if !priorStateData.NoAutoUserCreation.IsNull() {
					autoUserCreation = priorStateData.NoAutoUserCreation.ValueBool()
				}

				upgradedStateData := SAMLSettingsResourceModelV2{
					Name:                      priorStateData.Name,
					Enable:                    priorStateData.Enable,
					Certificate:               priorStateData.Certificate,
					EmailAttribute:            priorStateData.EmailAttribute,
					GroupAttribute:            priorStateData.GroupAttribute,
					NameIDAttribute:           priorStateData.NameIDAttribute,
					LoginURL:                  priorStateData.LoginURL,
					LogoutURL:                 priorStateData.LogoutURL,
					ServiceProviderName:       priorStateData.ServiceProviderName,
					AllowUserToAccessProfile:  priorStateData.AllowUserToAccessProfile,
					AutoRedirect:              priorStateData.AutoRedirect,
					SyncGroups:                priorStateData.SyncGroups,
					VerifyAudienceRestriction: priorStateData.VerifyAudienceRestriction,
					UseEncryptedAssertion:     priorStateData.UseEncryptedAssertion,
					AutoUserCreation:          types.BoolValue(autoUserCreation),
					LDAPGroupSettings:         priorStateData.LDAPGroupSettings,
				}

				resp.Diagnostics.Append(resp.State.Set(ctx, upgradedStateData)...)
			},
		},
	}
}

func (r *SAMLSettingsResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	go util.SendUsageResourceCreate(ctx, r.ProviderData.Client.R(), r.ProviderData.ProductId, r.TypeName)

	var plan SAMLSettingsResourceModelV2

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var samlSettings SAMLSettingsAPIModel
	resp.Diagnostics.Append(plan.toAPIModel(ctx, &samlSettings)...)
	if resp.Diagnostics.HasError() {
		return
	}

	response, err := r.ProviderData.Client.R().
		SetBody(samlSettings).
		Post(r.CollectionEndpoint)

	if err != nil {
		utilfw.UnableToCreateResourceError(resp, err.Error())
		return
	}

	if response.IsError() {
		utilfw.UnableToCreateResourceError(resp, response.String())
		return
	}

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *SAMLSettingsResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	go util.SendUsageResourceRead(ctx, r.ProviderData.Client.R(), r.ProviderData.ProductId, r.TypeName)

	var state SAMLSettingsResourceModelV2

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var samlSettings SAMLSettingsAPIModel

	response, err := r.ProviderData.Client.R().
		SetPathParam("name", state.Name.ValueString()).
		SetResult(&samlSettings).
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
	resp.Diagnostics.Append(state.fromAPIModel(ctx, &samlSettings)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *SAMLSettingsResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	go util.SendUsageResourceUpdate(ctx, r.ProviderData.Client.R(), r.ProviderData.ProductId, r.TypeName)

	var plan SAMLSettingsResourceModelV2

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var samlSettings SAMLSettingsAPIModel
	resp.Diagnostics.Append(plan.toAPIModel(ctx, &samlSettings)...)
	if resp.Diagnostics.HasError() {
		return
	}

	response, err := r.ProviderData.Client.R().
		SetPathParam("name", plan.Name.ValueString()).
		SetBody(samlSettings).
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

func (r *SAMLSettingsResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	go util.SendUsageResourceDelete(ctx, r.ProviderData.Client.R(), r.ProviderData.ProductId, r.TypeName)

	var state SAMLSettingsResourceModelV2

	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	response, err := r.ProviderData.Client.R().
		SetPathParam("name", state.Name.ValueString()).
		Delete(r.DocumentEndpoint)
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

func (r *SAMLSettingsResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("name"), req, resp)
}
