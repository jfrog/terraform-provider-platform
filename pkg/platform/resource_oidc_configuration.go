package platform

import (
	"context"
	"fmt"
	"net/http"
	"regexp"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/jfrog/terraform-provider-shared/util"
	utilfw "github.com/jfrog/terraform-provider-shared/util/fw"
)

const (
	gitHubProviderType        = "GitHub"
	gitHubProviderURL         = "https://token.actions.githubusercontent.com/"
	odicConfigurationEndpoint = "/access/api/v1/oidc"
)

var OIDCConfigurationNameValidators = []validator.String{
	stringvalidator.LengthBetween(1, 255),
	stringvalidator.RegexMatches(
		regexp.MustCompile(`^[a-z]{1}[a-z0-9\-]+$`),
		"must start with a lowercase letter and only contain lowercase letters, digits and `-` character.",
	),
}

var _ resource.Resource = (*odicConfigurationResource)(nil)

type odicConfigurationResource struct {
	ProviderData util.ProvderMetadata
}

func NewOIDCConfigurationResource() resource.Resource {
	return &odicConfigurationResource{}
}

func (r *odicConfigurationResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_oidc_configuration"
}

func (r *odicConfigurationResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
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
				Description: fmt.Sprintf("OIDC issuer URL. For GitHub actions, the URL must be %s.", gitHubProviderURL),
			},
			"provider_type": schema.StringAttribute{
				Required: true,
				Validators: []validator.String{
					stringvalidator.OneOf([]string{"generic", gitHubProviderType}...),
				},
				MarkdownDescription: fmt.Sprintf("Type of OIDC provider. Can be `generic` or `%s`.", gitHubProviderType),
			},
			"audience": schema.StringAttribute{
				Optional: true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
				Description: "Informational field that you can use to include details of the audience that uses the OIDC configuration.",
			},
		},
		MarkdownDescription: "Manage OIDC configuration in JFrog platform. See the JFrog [OIDC configuration documentation](https://jfrog.com/help/r/jfrog-platform-administration-documentation/configure-an-oidc-integration) for more information.",
	}
}

func (r odicConfigurationResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var data odicConfigurationResourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if data.ProviderType.ValueString() == gitHubProviderType && data.IssuerURL.ValueString() != gitHubProviderURL {
		resp.Diagnostics.AddAttributeError(
			path.Root("issuer_url"),
			"Invalid Attribute Configuration",
			fmt.Sprintf("issuer_url must be set to %s when provider_type is set to '%s'.", gitHubProviderURL, gitHubProviderType),
		)
	}
}

type odicConfigurationResourceModel struct {
	Name         types.String `tfsdk:"name"`
	Description  types.String `tfsdk:"description"`
	IssuerURL    types.String `tfsdk:"issuer_url"`
	ProviderType types.String `tfsdk:"provider_type"`
	Audience     types.String `tfsdk:"audience"`
}

type odicConfigurationAPIModel struct {
	Name         string `json:"name"`
	Description  string `json:"description,omitempty"`
	IssuerURL    string `json:"issuer_url"`
	ProviderType string `json:"provider_type"`
	Audience     string `json:"audience,omitempty"`
}

func (r *odicConfigurationResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}
	r.ProviderData = req.ProviderData.(util.ProvderMetadata)
}

func (r *odicConfigurationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan odicConfigurationResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	odicConfig := odicConfigurationAPIModel{
		Name:         plan.Name.ValueString(),
		IssuerURL:    plan.IssuerURL.ValueString(),
		ProviderType: plan.ProviderType.ValueString(),
		Audience:     plan.Audience.ValueString(),
		Description:  plan.Description.ValueString(),
	}

	response, err := r.ProviderData.Client.R().
		SetBody(&odicConfig).
		Post(odicConfigurationEndpoint)
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

func (r *odicConfigurationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state odicConfigurationResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var odicConfig odicConfigurationAPIModel

	response, err := r.ProviderData.Client.R().
		SetPathParam("name", state.Name.ValueString()).
		SetResult(&odicConfig).
		Get(odicConfigurationEndpoint + "/{name}")

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
	state.Name = types.StringValue(odicConfig.Name)

	if len(odicConfig.Description) > 0 {
		state.Description = types.StringValue(odicConfig.Description)
	}

	state.IssuerURL = types.StringValue(odicConfig.IssuerURL)

	if len(odicConfig.Audience) > 0 {
		state.Audience = types.StringValue(odicConfig.Audience)
	}

	state.ProviderType = types.StringValue(odicConfig.ProviderType)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *odicConfigurationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan odicConfigurationResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	odicConfig := odicConfigurationAPIModel{
		Name:         plan.Name.ValueString(),
		IssuerURL:    plan.IssuerURL.ValueString(),
		ProviderType: plan.ProviderType.ValueString(),
		Audience:     plan.Audience.ValueString(),
		Description:  plan.Description.ValueString(),
	}

	response, err := r.ProviderData.Client.R().
		SetPathParam("name", plan.Name.ValueString()).
		SetBody(&odicConfig).
		Put(odicConfigurationEndpoint + "/{name}")
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

func (r *odicConfigurationResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state odicConfigurationResourceModel

	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	response, err := r.ProviderData.Client.R().
		SetPathParam("name", state.Name.ValueString()).
		Delete(odicConfigurationEndpoint + "/{name}")
	if err != nil {
		utilfw.UnableToDeleteResourceError(resp, err.Error())
		return
	}

	if response.StatusCode() != http.StatusNoContent {
		utilfw.UnableToDeleteResourceError(resp, response.String())
		return
	}

	// If the logic reaches here, it implicitly succeeded and will remove
	// the resource from state if there are no other errors.
}

func (r *odicConfigurationResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("name"), req, resp)
}
