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
	"regexp"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
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
)

const (
	AccessVersion                 = "7.138.0"
	GithubEnterpriseAccessVersion = "7.144.0"
	gitHubProviderType            = "GitHub"
	githubEnterpriseType          = "GitHubEnterprise"
	azureProviderType             = "Azure"
	gitHubProviderURL             = "https://token.actions.githubusercontent.com"
)

var OIDCConfigurationNameValidators = []validator.String{
	stringvalidator.LengthBetween(1, 255),
	stringvalidator.RegexMatches(
		regexp.MustCompile(`^[a-z]{1}[a-z0-9\-]+$`),
		"must start with a lowercase letter and only contain lowercase letters, digits and `-` character.",
	),
}

// apiProviderType maps a schema `provider_type` value onto the value the Access API expects.
func apiProviderType(providerType string) string {
	switch providerType {
	case "generic":
		return "Generic OpenID Connect"
	case githubEnterpriseType:
		return "GitHub Enterprise"
	default:
		return providerType
	}
}

func isGitHubProviderType(providerType string) bool {
	return providerType == gitHubProviderType || providerType == githubEnterpriseType
}

var _ resource.Resource = (*oidcConfigurationResource)(nil)

type oidcConfigurationResource struct {
	util.JFrogResource
	ProviderData util.ProviderMetadata
}

func NewOIDCConfigurationResource() resource.Resource {
	return &oidcConfigurationResource{
		JFrogResource: util.JFrogResource{
			TypeName:                "platform_oidc_configuration",
			ValidArtifactoryVersion: "7.73.1",
			CollectionEndpoint:      "/access/api/v1/oidc",
			DocumentEndpoint:        "/access/api/v1/oidc/{name}",
		},
	}
}

func (r *oidcConfigurationResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Required:   true,
				Validators: OIDCConfigurationNameValidators,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Description: "Name of the OIDC provider",
			},
			"description": schema.StringAttribute{
				Optional:    true,
				Description: "Description of the OIDC provider",
			},
			"issuer_url": schema.StringAttribute{
				Required: true,
				Validators: []validator.String{
					stringvalidator.RegexMatches(
						regexp.MustCompile(`^https:\/\/`),
						"must use https protocol.",
					),
				},
				Description: fmt.Sprintf("OIDC issuer URL. For GitHub actions, the URL must start with %s.", gitHubProviderURL),
			},
			"provider_type": schema.StringAttribute{
				Required: true,
				Validators: []validator.String{
					stringvalidator.OneOf([]string{"generic", gitHubProviderType, githubEnterpriseType, azureProviderType}...),
				},
				MarkdownDescription: fmt.Sprintf("Type of OIDC provider. Can be `generic`, `%s`, `%s` or `%s`.", gitHubProviderType, githubEnterpriseType, azureProviderType),
			},
			"audience": schema.StringAttribute{
				Optional: true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
				Description: "Informational field that you can use to include details of the audience that uses the OIDC configuration.",
			},
			"organization": schema.StringAttribute{
				// Only required when provider_type is set to GitHub
				Optional: true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
				MarkdownDescription: fmt.Sprintf("This field is mandatory when `provider_type` is `%[1]s` or `%[2]s`. It is only applicable to those provider types: the JFrog platform discards it for `generic` and `%[3]s`, so configuring it with any other `provider_type` is rejected at plan time. For `%[2]s`, the platform stores the value in `token_issuer`. Informational field that you can use to include details of the organization that uses the OIDC configuration. **Upgrade caveat (provider 2.2.6 through 2.2.10):** if you created a `%[2]s` configuration with `organization` set on those provider versions, upgrading to 2.2.11 with an unchanged configuration can leave the value in Terraform state but not on the platform — `terraform plan` and `terraform plan -refresh-only` report no changes. Perform a one-time remediation: change any attribute (for example `description`) and run `terraform apply`, run `terraform apply -replace='platform_oidc_configuration.<name>'`, change `organization` and apply, or `terraform import` the resource and apply. Leaving the configuration unchanged leaves a silent security gap: the platform does not enforce the organization restriction Terraform state asserts.", gitHubProviderType, githubEnterpriseType, azureProviderType),
			},
			"project_key": schema.StringAttribute{
				Optional: true,
				Validators: []validator.String{
					validatorfw_string.ProjectKey(),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Description: "If set, this Identity Configuration will be available in the scope of the given project (editable by platform admin and project admin). If not set, this Identity Configuration will be global and only editable by platform admin. Once set, the projectKey cannot be changed.",
			},
			"token_issuer": schema.StringAttribute{
				Optional:    true,
				Description: fmt.Sprintf("Token issuer URL of the identity provider. Not allowed when `provider_type` is `%s` or `%s`.", gitHubProviderType, githubEnterpriseType),
			},
			"azure_app_id": schema.StringAttribute{
				Optional:    true,
				Description: fmt.Sprintf("Azure Application ID. Only applicable when `provider_type` is `%s`.", azureProviderType),
			},
			"use_default_proxy": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
				MarkdownDescription: "This enables and disables the default proxy for OIDC integration. If enabled, the OIDC mechanism will utilize the default proxy for all OIDC requests. If disabled, the OIDC mechanism does not use any proxy for all OIDC requests. Before enabling this functionality you must configure the default proxy.",
			},
			"enable_permissive_configuration": schema.BoolAttribute{
				Optional: true,
				MarkdownDescription: fmt.Sprintf(
					"Only `%[1]s` accepts `false`. Setting `enable_permissive_configuration = false` for `%[2]s`, `generic`, or `%[3]s` is rejected at plan time: "+
						"the JFrog platform does not enforce restrictive (non-permissive) authentication for those provider types and keeps permissive authentication enabled regardless. "+
						"When set to `true` or left unset, the attribute is accepted for all provider types but only enforced for `%[1]s`. "+
						"When set, allows authentication without any restrictions. For security best practices, it is recommended to add restrictions to limit access and enforce stricter controls. Use with caution, as this may grant broader access.",
					gitHubProviderType, githubEnterpriseType, azureProviderType,
				),
			},
		},
		MarkdownDescription: "Manage OIDC configuration in JFrog platform. See the JFrog [OIDC configuration documentation](https://jfrog.com/help/r/jfrog-platform-administration-documentation/configure-an-oidc-integration) for more information.",
	}
}

func (r *oidcConfigurationResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}
	r.ProviderData = req.ProviderData.(util.ProviderMetadata)
}

func (r oidcConfigurationResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var data oidcConfigurationResourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if data.ProviderType.ValueString() == gitHubProviderType && !strings.HasPrefix(data.IssuerURL.ValueString(), gitHubProviderURL) {
		resp.Diagnostics.AddAttributeError(
			path.Root("issuer_url"),
			"Invalid Attribute Configuration",
			fmt.Sprintf("issuer_url must start with %s when provider_type is set to '%s'.", gitHubProviderURL, gitHubProviderType),
		)
	}

	// Access version 7.144.0 or later is required for the `issuer_url` attribute when `provider_type` is set to `GitHubEnterprise`
	if data.ProviderType.ValueString() == githubEnterpriseType {
		if ok, err := util.CheckVersion(r.ProviderData.AccessVersion, GithubEnterpriseAccessVersion); err == nil && ok {
			if data.IssuerURL.IsNull() || data.IssuerURL.ValueString() == "" {
				resp.Diagnostics.AddAttributeError(
					path.Root("issuer_url"),
					"Missing Attribute Configuration",
					fmt.Sprintf("issuer_url must be configured when provider_type is set to '%s'.", githubEnterpriseType),
				)
			}
		}
	}

	enablePermissiveConfiguration := data.EnablePermissiveConfiguration.ValueBool()

	// Access version 7.138.0 or later is required for the `organization` attribute when `provider_type` is set to `GitHub`
	if data.ProviderType.ValueString() == gitHubProviderType {
		if ok, err := util.CheckVersion(r.ProviderData.AccessVersion, AccessVersion); err == nil && ok {
			if !enablePermissiveConfiguration && data.Organization.IsNull() {
				resp.Diagnostics.AddAttributeError(
					path.Root("organization"),
					"Missing Attribute Configuration",
					fmt.Sprintf("organization must be configured when provider_type is set to '%s'.", gitHubProviderType),
				)
			}
		}
	}

	if data.ProviderType.ValueString() == githubEnterpriseType {
		if ok, err := util.CheckVersion(r.ProviderData.AccessVersion, GithubEnterpriseAccessVersion); err == nil && ok {
			if !enablePermissiveConfiguration && data.Organization.IsNull() {
				resp.Diagnostics.AddAttributeError(
					path.Root("organization"),
					"Missing Attribute Configuration",
					fmt.Sprintf("organization must be configured when provider_type is set to '%s'.", githubEnterpriseType),
				)
			}
		}
	}

	// The platform only stores `organization` for the GitHub provider types, where the API
	// carries it in the `token_issuer` field. For every other provider type the value is
	// discarded, so it would sit in state forever without the platform ever knowing about
	// it. Validation is deferred while `provider_type` is unknown.
	if !data.Organization.IsNull() &&
		!data.ProviderType.IsNull() && !data.ProviderType.IsUnknown() &&
		!isGitHubProviderType(data.ProviderType.ValueString()) {
		resp.Diagnostics.AddAttributeError(
			path.Root("organization"),
			"Invalid Attribute Configuration",
			fmt.Sprintf(
				"organization is only applicable when provider_type is set to '%s' or '%s'. "+
					"The JFrog platform discards it for provider_type '%s', so the value never takes effect. "+
					"Remove the attribute from your configuration.",
				gitHubProviderType, githubEnterpriseType, data.ProviderType.ValueString(),
			),
		)
	}

	// Validation is deferred while `provider_type` is unknown: an unknown value may still
	// resolve to `Azure`, and rejecting it here fails the plan of a configuration that
	// would apply cleanly.
	if !data.AzureAppId.IsNull() &&
		!data.ProviderType.IsNull() && !data.ProviderType.IsUnknown() &&
		data.ProviderType.ValueString() != azureProviderType {
		resp.Diagnostics.AddAttributeError(
			path.Root("azure_app_id"),
			"Invalid Attribute Configuration",
			fmt.Sprintf("azure_app_id is only applicable when provider_type is set to '%s'.", azureProviderType),
		)
	}

	// Validation is deferred while `provider_type` is unknown. The guard is redundant
	// today, because an unknown value matches neither GitHub provider type, but it keeps
	// the deferral independent of the polarity of the comparison below.
	if !data.TokenIssuer.IsNull() &&
		!data.ProviderType.IsNull() && !data.ProviderType.IsUnknown() &&
		(data.ProviderType.ValueString() == gitHubProviderType ||
			data.ProviderType.ValueString() == githubEnterpriseType) {
		resp.Diagnostics.AddAttributeError(
			path.Root("token_issuer"),
			"Invalid Attribute Configuration",
			fmt.Sprintf("token_issuer is not allowed when provider_type is set to '%s' or '%s'.", gitHubProviderType, githubEnterpriseType),
		)
	}

	// The platform only enforces `enable_permissive_configuration` for `GitHub`. Setting
	// `false` for any other provider type creates a false sense of security: the platform
	// keeps permissive authentication enabled regardless. Validation is deferred while
	// `provider_type` is unknown.
	if !data.EnablePermissiveConfiguration.IsNull() &&
		!data.EnablePermissiveConfiguration.IsUnknown() &&
		!data.EnablePermissiveConfiguration.ValueBool() &&
		!data.ProviderType.IsNull() && !data.ProviderType.IsUnknown() &&
		data.ProviderType.ValueString() != gitHubProviderType {
		providerType := data.ProviderType.ValueString()
		resp.Diagnostics.AddAttributeError(
			path.Root("enable_permissive_configuration"),
			"Invalid Attribute Configuration",
			fmt.Sprintf(
				"enable_permissive_configuration = false is only applicable when provider_type is set to '%s'. "+
					"The JFrog platform does not enforce restrictive (non-permissive) authentication for provider_type '%s': "+
					"permissive authentication remains enabled regardless of this setting. "+
					"Remove enable_permissive_configuration from your configuration, or set provider_type to '%s' if you need to restrict authentication.",
				gitHubProviderType, providerType, gitHubProviderType,
			),
		)
	}
}

type oidcConfigurationResourceModel struct {
	Name                          types.String `tfsdk:"name"`
	Description                   types.String `tfsdk:"description"`
	IssuerURL                     types.String `tfsdk:"issuer_url"`
	ProviderType                  types.String `tfsdk:"provider_type"`
	Audience                      types.String `tfsdk:"audience"`
	Organization                  types.String `tfsdk:"organization"`
	ProjectKey                    types.String `tfsdk:"project_key"`
	TokenIssuer                   types.String `tfsdk:"token_issuer"`
	AzureAppId                    types.String `tfsdk:"azure_app_id"`
	UseDefaultProxy               types.Bool   `tfsdk:"use_default_proxy"`
	EnablePermissiveConfiguration types.Bool   `tfsdk:"enable_permissive_configuration"`
}

type oidcConfigurationAPIModel struct {
	Name            string `json:"name"`
	Description     string `json:"description,omitempty"`
	IssuerURL       string `json:"issuer_url"`
	ProviderType    string `json:"provider_type"`
	Audience        string `json:"audience,omitempty"`
	Organization    string `json:"organization"`
	ProjectKey      string `json:"project_key,omitempty"`
	TokenIssuer     string `json:"token_issuer,omitempty"`
	AzureAppId      string `json:"azure_app_id,omitempty"`
	UseDefaultProxy bool   `json:"use_default_proxy"`
	// Pointer so that an explicit `false` is sent to the API and only an unset
	// attribute is omitted.
	EnablePermissiveConfiguration *bool `json:"enable_permissive_configuration,omitempty"`
}

func (r *oidcConfigurationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	go util.SendUsageResourceCreate(ctx, r.ProviderData.Client.R(), r.ProviderData.ProductId, r.TypeName)

	var plan oidcConfigurationResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	providerType := plan.ProviderType.ValueString()

	oidcConfig := oidcConfigurationAPIModel{
		Name:            plan.Name.ValueString(),
		IssuerURL:       plan.IssuerURL.ValueString(),
		ProviderType:    apiProviderType(providerType),
		Audience:        plan.Audience.ValueString(),
		Description:     plan.Description.ValueString(),
		ProjectKey:      plan.ProjectKey.ValueString(),
		UseDefaultProxy: plan.UseDefaultProxy.ValueBool(),
	}

	// For GitHub provider types the API reuses the `token_issuer` field to carry the
	// organization, so sending it here would override `organization`.
	if !isGitHubProviderType(providerType) {
		oidcConfig.TokenIssuer = plan.TokenIssuer.ValueString()
	}

	if providerType == azureProviderType {
		oidcConfig.AzureAppId = plan.AzureAppId.ValueString()
	}

	// Access version 7.138.0 or later is required for the `organization` attribute when `provider_type` is set to `GitHub`
	if providerType == gitHubProviderType {
		if ok, err := util.CheckVersion(r.ProviderData.AccessVersion, AccessVersion); err == nil && ok {
			oidcConfig.Organization = plan.Organization.ValueString()
			oidcConfig.EnablePermissiveConfiguration = plan.EnablePermissiveConfiguration.ValueBoolPointer()
		}
	}

	// Access version 7.144.0 or later is required for the `organization` attribute when `provider_type` is set to `GitHubEnterprise`
	if providerType == githubEnterpriseType {
		if ok, err := util.CheckVersion(r.ProviderData.AccessVersion, GithubEnterpriseAccessVersion); err == nil && ok {
			oidcConfig.Organization = plan.Organization.ValueString()
			oidcConfig.EnablePermissiveConfiguration = plan.EnablePermissiveConfiguration.ValueBoolPointer()
		}
	}

	response, err := r.ProviderData.Client.R().
		SetBody(&oidcConfig).
		Post(r.CollectionEndpoint)
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

func (r *oidcConfigurationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	go util.SendUsageResourceRead(ctx, r.ProviderData.Client.R(), r.ProviderData.ProductId, r.TypeName)

	var state oidcConfigurationResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var oidcConfig oidcConfigurationAPIModel

	response, err := r.ProviderData.Client.R().
		SetPathParam("name", state.Name.ValueString()).
		SetResult(&oidcConfig).
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
	state.Name = types.StringValue(oidcConfig.Name)

	if len(oidcConfig.Description) > 0 {
		state.Description = types.StringValue(oidcConfig.Description)
	}

	state.IssuerURL = types.StringValue(oidcConfig.IssuerURL)

	if len(oidcConfig.Audience) > 0 {
		state.Audience = types.StringValue(oidcConfig.Audience)
	}

	// The API reports the provider type in its own vocabulary, so normalize it back to
	// the schema vocabulary before using it for any provider-specific handling below.
	switch oidcConfig.ProviderType {
	case "Generic OpenID Connect":
		state.ProviderType = types.StringValue("generic")
	case "GitHub Enterprise":
		state.ProviderType = types.StringValue(githubEnterpriseType)
	default:
		state.ProviderType = types.StringValue(oidcConfig.ProviderType)
	}
	providerType := state.ProviderType.ValueString()

	if isGitHubProviderType(providerType) {
		// Access version 7.138.0 (GitHub) or 7.144.0 (GitHubEnterprise) or later is
		// required for the `organization` attribute.
		requiredVersion := AccessVersion
		if providerType == githubEnterpriseType {
			requiredVersion = GithubEnterpriseAccessVersion
		}

		if ok, err := util.CheckVersion(r.ProviderData.AccessVersion, requiredVersion); err == nil && ok {
			// The API returns the organization in the `token_issuer` field and omits
			// `organization` entirely, so fall back to it to keep the attribute
			// refreshed on import and to detect out-of-band changes.
			organization := oidcConfig.Organization
			if len(organization) == 0 {
				organization = oidcConfig.TokenIssuer
			}
			if len(organization) > 0 {
				state.Organization = types.StringValue(organization)
			}
			// The platform only honours enable_permissive_configuration for GitHub. For
			// GitHubEnterprise it always reports true, so refreshing it there would
			// produce a diff that can never converge.
			//
			// Only refresh when the attribute is set in state, so that an unset
			// Optional attribute is never populated from the API response.
			if providerType == gitHubProviderType &&
				!state.EnablePermissiveConfiguration.IsNull() &&
				oidcConfig.EnablePermissiveConfiguration != nil {
				state.EnablePermissiveConfiguration = types.BoolPointerValue(oidcConfig.EnablePermissiveConfiguration)
			}
		}
	}

	// Only refresh when the attribute is already tracked in state, so that a value
	// configured outside Terraform is not planned for removal.
	if !state.AzureAppId.IsNull() {
		if len(oidcConfig.AzureAppId) > 0 {
			state.AzureAppId = types.StringValue(oidcConfig.AzureAppId)
		} else {
			state.AzureAppId = types.StringNull()
		}
	}

	state.UseDefaultProxy = types.BoolValue(oidcConfig.UseDefaultProxy)

	// For GitHub provider types the `token_issuer` field carries the organization,
	// which is not a valid value for this attribute.
	if len(oidcConfig.TokenIssuer) > 0 && !isGitHubProviderType(providerType) {
		state.TokenIssuer = types.StringValue(oidcConfig.TokenIssuer)
	} else {
		state.TokenIssuer = types.StringNull()
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *oidcConfigurationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	go util.SendUsageResourceUpdate(ctx, r.ProviderData.Client.R(), r.ProviderData.ProductId, r.TypeName)

	var plan oidcConfigurationResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	providerType := plan.ProviderType.ValueString()

	oidcConfig := oidcConfigurationAPIModel{
		Name:            plan.Name.ValueString(),
		IssuerURL:       plan.IssuerURL.ValueString(),
		ProviderType:    apiProviderType(providerType),
		Audience:        plan.Audience.ValueString(),
		Description:     plan.Description.ValueString(),
		ProjectKey:      plan.ProjectKey.ValueString(),
		UseDefaultProxy: plan.UseDefaultProxy.ValueBool(),
	}

	// For GitHub provider types the API reuses the `token_issuer` field to carry the
	// organization, so sending it here would override `organization`.
	if !isGitHubProviderType(providerType) {
		oidcConfig.TokenIssuer = plan.TokenIssuer.ValueString()
	}

	if providerType == azureProviderType {
		oidcConfig.AzureAppId = plan.AzureAppId.ValueString()
	}

	// Access version 7.138.0 or later is required for the `organization` attribute when `provider_type` is set to `GitHub`
	if providerType == gitHubProviderType {
		if ok, err := util.CheckVersion(r.ProviderData.AccessVersion, AccessVersion); err == nil && ok {
			oidcConfig.Organization = plan.Organization.ValueString()
			oidcConfig.EnablePermissiveConfiguration = plan.EnablePermissiveConfiguration.ValueBoolPointer()
		}
	}

	// Access version 7.144.0 or later is required for the `organization` attribute when `provider_type` is set to `GitHubEnterprise`
	if providerType == githubEnterpriseType {
		if ok, err := util.CheckVersion(r.ProviderData.AccessVersion, GithubEnterpriseAccessVersion); err == nil && ok {
			oidcConfig.Organization = plan.Organization.ValueString()
			oidcConfig.EnablePermissiveConfiguration = plan.EnablePermissiveConfiguration.ValueBoolPointer()
		}
	}

	response, err := r.ProviderData.Client.R().
		SetPathParam("name", plan.Name.ValueString()).
		SetBody(&oidcConfig).
		Put(r.DocumentEndpoint)
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

func (r *oidcConfigurationResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	go util.SendUsageResourceDelete(ctx, r.ProviderData.Client.R(), r.ProviderData.ProductId, r.TypeName)

	var state oidcConfigurationResourceModel

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

func (r *oidcConfigurationResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, ":", 2)

	if len(parts) > 0 && parts[0] != "" {
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("name"), parts[0])...)
	}

	if len(parts) == 2 && parts[1] != "" {
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("project_key"), parts[1])...)
	}
}
