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
	gitHubProviderType = "GitHub"
	gitHubProviderURL  = "https://token.actions.githubusercontent.com"
)

var OIDCConfigurationNameValidators = []validator.String{
	stringvalidator.LengthBetween(1, 255),
	stringvalidator.RegexMatches(
		regexp.MustCompile(`^[a-z]{1}[a-z0-9\-]+$`),
		"must start with a lowercase letter and only contain lowercase letters, digits and `-` character.",
	),
}

var _ resource.Resource = (*oidcConfigurationResource)(nil)

type oidcConfigurationResource struct {
	util.JFrogResource
}

func NewOIDCConfigurationResource() resource.Resource {
	return &oidcConfigurationResource{
		JFrogResource: util.JFrogResource{
			TypeName:           "platform_oidc_configuration",
			CollectionEndpoint: "/access/api/v1/oidc",
			DocumentEndpoint:   "/access/api/v1/oidc/{name}",
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
					stringvalidator.OneOf([]string{"generic", gitHubProviderType, "Azure"}...),
				},
				MarkdownDescription: fmt.Sprintf("Type of OIDC provider. Can be `generic`, `%s`, or `Azure`.", gitHubProviderType),
			},
			"audience": schema.StringAttribute{
				Optional: true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
				Description: "Informational field that you can use to include details of the audience that uses the OIDC configuration.",
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
			"use_default_proxy": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
				MarkdownDescription: "This enables and disables the default proxy for OIDC integration. If enabled, the OIDC mechanism will utilize the default proxy for all OIDC requests. If disabled, the OIDC mechanism does not use any proxy for all OIDC requests. Before enabling this functionality you must configure the default proxy.",
			},
		},
		MarkdownDescription: "Manage OIDC configuration in JFrog platform. See the JFrog [OIDC configuration documentation](https://jfrog.com/help/r/jfrog-platform-administration-documentation/configure-an-oidc-integration) for more information.",
	}
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
}

type oidcConfigurationResourceModel struct {
	Name            types.String `tfsdk:"name"`
	Description     types.String `tfsdk:"description"`
	IssuerURL       types.String `tfsdk:"issuer_url"`
	ProviderType    types.String `tfsdk:"provider_type"`
	Audience        types.String `tfsdk:"audience"`
	ProjectKey      types.String `tfsdk:"project_key"`
	UseDefaultProxy types.Bool   `tfsdk:"use_default_proxy"`
}

type oidcConfigurationAPIModel struct {
	Name            string `json:"name"`
	Description     string `json:"description,omitempty"`
	IssuerURL       string `json:"issuer_url"`
	ProviderType    string `json:"provider_type"`
	Audience        string `json:"audience,omitempty"`
	ProjectKey      string `json:"project_key,omitempty"`
	UseDefaultProxy bool   `json:"use_default_proxy"`
}

func (r *oidcConfigurationResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}
	m := req.ProviderData.(PlatformProviderMetadata).ProviderMetadata
	r.ProviderData = &m
}

func (r *oidcConfigurationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	go util.SendUsageResourceCreate(ctx, r.ProviderData.Client.R(), r.ProviderData.ProductId, r.TypeName)

	var plan oidcConfigurationResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	providerType := plan.ProviderType.ValueString()
	if providerType == "generic" {
		providerType = "Generic OpenID Connect"
	}

	oidcConfig := oidcConfigurationAPIModel{
		Name:            plan.Name.ValueString(),
		IssuerURL:       plan.IssuerURL.ValueString(),
		ProviderType:    providerType,
		Audience:        plan.Audience.ValueString(),
		Description:     plan.Description.ValueString(),
		ProjectKey:      plan.ProjectKey.ValueString(),
		UseDefaultProxy: plan.UseDefaultProxy.ValueBool(),
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

	if oidcConfig.ProviderType == "Generic OpenID Connect" {
		state.ProviderType = types.StringValue("generic")
	} else {
		state.ProviderType = types.StringValue(oidcConfig.ProviderType)
	}

	state.UseDefaultProxy = types.BoolValue(oidcConfig.UseDefaultProxy)

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
	if providerType == "generic" {
		providerType = "Generic OpenID Connect"
	}

	oidcConfig := oidcConfigurationAPIModel{
		Name:            plan.Name.ValueString(),
		IssuerURL:       plan.IssuerURL.ValueString(),
		ProviderType:    providerType,
		Audience:        plan.Audience.ValueString(),
		Description:     plan.Description.ValueString(),
		ProjectKey:      plan.ProjectKey.ValueString(),
		UseDefaultProxy: plan.UseDefaultProxy.ValueBool(),
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
