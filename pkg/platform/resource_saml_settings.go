package platform

import (
	"context"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework-validators/boolvalidator"
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

const (
	SAMLSettingsEndpoint = "access/api/v1/saml"
	SAMLSettingEndpoint  = "access/api/v1/saml/{name}"
)

func NewSAMLSettingsResource() resource.Resource {
	return &SAMLSettingsResource{
		TypeName: "platform_saml_settings",
	}
}

type SAMLSettingsResource struct {
	ProviderData PlatformProviderMetadata
	TypeName     string
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
	AutoUserCreation types.Bool `tfsdk:"auto_user_creation"`
}

func (r *SAMLSettingsResourceModelV1) toAPIModel(_ context.Context, apiModel *SAMLSettingsAPIModel) diag.Diagnostics {

	apiModel.Name = r.Name.ValueString()
	apiModel.Enable = r.Enable.ValueBool()
	apiModel.Certificate = r.Certificate.ValueString()
	apiModel.EmailAttribute = r.EmailAttribute.ValueString()
	apiModel.GroupAttribute = r.GroupAttribute.ValueString()
	apiModel.NameIDAttribute = r.NameIDAttribute.ValueString()
	apiModel.LoginURL = r.LoginURL.ValueString()
	apiModel.LogoutURL = r.LogoutURL.ValueString()

	if !r.AutoUserCreation.IsNull() {
		apiModel.AutoUserCreation = r.AutoUserCreation.ValueBool()
	} else {
		apiModel.AutoUserCreation = !r.NoAutoUserCreation.ValueBool()
	}

	apiModel.ServiceProviderName = r.ServiceProviderName.ValueString()
	apiModel.AllowUserToAccessProfile = r.AllowUserToAccessProfile.ValueBool()
	apiModel.AutoRedirect = r.AutoRedirect.ValueBool()
	apiModel.SyncGroups = r.SyncGroups.ValueBool()
	apiModel.VerifyAudienceRestriction = r.VerifyAudienceRestriction.ValueBool()
	apiModel.UseEncryptedAssertion = r.UseEncryptedAssertion.ValueBool()

	return nil
}

func (r *SAMLSettingsResourceModelV1) fromAPIModel(_ context.Context, apiModel *SAMLSettingsAPIModel) (ds diag.Diagnostics) {
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

	return
}

type SAMLSettingsAPIModel struct {
	Name                      string `json:"name"`
	Enable                    bool   `json:"enable_integration"`
	VerifyAudienceRestriction bool   `json:"verify_audience_restriction"`
	LoginURL                  string `json:"login_url"`
	LogoutURL                 string `json:"logout_url"`
	ServiceProviderName       string `json:"service_provider_name"`
	AutoUserCreation          bool   `json:"auto_user_creation"`
	AllowUserToAccessProfile  bool   `json:"allow_user_to_access_profile"`
	UseEncryptedAssertion     bool   `json:"use_encrypted_assertion"`
	AutoRedirect              bool   `json:"auto_redirect"`
	SyncGroups                bool   `json:"sync_groups"`
	Certificate               string `json:"certificate"`
	GroupAttribute            string `json:"group_attribute"`
	EmailAttribute            string `json:"email_attribute"`
	NameIDAttribute           string `json:"name_id_attribute"`
}

func (r *SAMLSettingsResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = r.TypeName
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
		MarkdownDescription: "If `no_auto_user_creation` is diabled or an internal user exists, the system will set the user's email to the value in this attribute that is returned by the SAML login XML response.",
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
	},
)

func (r *SAMLSettingsResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Version:             1,
		Attributes:          samlSettingsSchemaV1,
		MarkdownDescription: "Provides a JFrog [SAML SSO Settings](https://jfrog.com/help/r/jfrog-platform-administration-documentation/saml-sso) resource.\n\n~>Only available for self-hosted instances.",
	}
}

func (r *SAMLSettingsResource) UpgradeState(ctx context.Context) map[int64]resource.StateUpgrader {
	return map[int64]resource.StateUpgrader{
		// State upgrade implementation from 0 (prior state version) to 1 (Schema.Version)
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

				upgradedStateData := SAMLSettingsResourceModelV1{
					SAMLSettingsResourceModelV0: priorStateData,
					AutoUserCreation:            types.BoolValue(true),
				}

				upgradedStateData.AutoUserCreation = types.BoolValue(!priorStateData.NoAutoUserCreation.ValueBool())

				resp.Diagnostics.Append(resp.State.Set(ctx, upgradedStateData)...)
			},
		},
	}
}

func (r *SAMLSettingsResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}
	r.ProviderData = req.ProviderData.(PlatformProviderMetadata)

	supported, err := util.CheckVersion(r.ProviderData.ArtifactoryVersion, "7.83.1")
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to check Artifactory version",
			err.Error(),
		)
		return
	}

	if !supported {
		resp.Diagnostics.AddError(
			"Unsupported Artifactory version",
			fmt.Sprintf("This resource is supported by Artifactory version 7.83.1 or later. Current version: %s", r.ProviderData.ArtifactoryVersion),
		)
		return
	}
}

func (r *SAMLSettingsResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	go util.SendUsageResourceCreate(ctx, r.ProviderData.Client.R(), r.ProviderData.ProductId, r.TypeName)

	var plan SAMLSettingsResourceModelV1

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
		Post(SAMLSettingsEndpoint)

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

	var state SAMLSettingsResourceModelV1

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var samlSettings SAMLSettingsAPIModel

	response, err := r.ProviderData.Client.R().
		SetPathParam("name", state.Name.ValueString()).
		SetResult(&samlSettings).
		Get(SAMLSettingEndpoint)

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

	var plan SAMLSettingsResourceModelV1

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
		Put(SAMLSettingEndpoint)

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

	var state SAMLSettingsResourceModelV1

	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	response, err := r.ProviderData.Client.R().
		SetPathParam("name", state.Name.ValueString()).
		Delete(SAMLSettingEndpoint)
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
