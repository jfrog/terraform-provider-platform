package platform

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/jfrog/terraform-provider-shared/client"
	"github.com/jfrog/terraform-provider-shared/util"
	utilfw "github.com/jfrog/terraform-provider-shared/util/fw"
	validator_string "github.com/jfrog/terraform-provider-shared/validator/fw/string"
)

var Version = "0.0.1"

// needs to be exported so make file can update this
var productId = "terraform-provider-platform/" + Version

var _ provider.Provider = (*platformProvider)(nil)

type platformProvider struct {
	Url          types.String `tfsdk:"url"`
	AccessToken  types.String `tfsdk:"access_token"`
	CheckLicense types.Bool   `tfsdk:"check_license"`
}

func New() func() provider.Provider {
	return func() provider.Provider {
		return &platformProvider{}
	}
}

func (p *platformProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	// check if Terraform version is >=1.0.0, i.e. support protocol v6
	supportProtocolV6, err := util.CheckVersion(req.TerraformVersion, "1.0.0")
	if err != nil {
		resp.Diagnostics.Append(diag.NewWarningDiagnostic("failed to check Terraform version", err.Error()))
	}

	if !supportProtocolV6 {
		resp.Diagnostics.Append(diag.NewWarningDiagnostic(
			"Terraform CLI version deprecation",
			"Terraform version older than 1.0 will no longer be supported in Q1 2024. Please upgrade to latest Terraform CLI.",
		))
	}

	// Check environment variables, first available OS variable will be assigned to the var
	url := util.CheckEnvVars([]string{"JFROG_URL", "ARTIFACTORY_URL"}, "")
	accessToken := util.CheckEnvVars([]string{"JFROG_ACCESS_TOKEN", "ARTIFACTORY_ACCESS_TOKEN"}, "")

	var config platformProvider

	// Read configuration data into model
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Check configuration data, which should take precedence over
	// environment variable data, if found.
	if config.AccessToken.ValueString() != "" {
		accessToken = config.AccessToken.ValueString()
	}

	if accessToken == "" {
		resp.Diagnostics.AddError(
			"Missing JFrog Access Token",
			"While configuring the provider, the Access Token was not found in "+
				"the JFROG_ACCESS_TOKEN/ARTIFACTORY_ACCESS_TOKEN environment variable or provider "+
				"configuration block access_token attribute.",
		)
		return
	}

	if config.Url.ValueString() != "" {
		url = config.Url.ValueString()
	}

	if url == "" {
		resp.Diagnostics.AddError(
			"Missing URL Configuration",
			"While configuring the provider, the url was not found in "+
				"the JFROG_URL/ARTIFACTORY_URL environment variable or provider "+
				"configuration block url attribute.",
		)
		return
	}

	restyBase, err := client.Build(url, productId)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating Resty client",
			err.Error(),
		)
	}

	restyBase, err = client.AddAuth(restyBase, "", accessToken)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error adding Auth to Resty client",
			err.Error(),
		)
	}

	if config.CheckLicense.IsNull() || config.CheckLicense.ValueBool() {
		if licenseErr := utilfw.CheckArtifactoryLicense(restyBase, "Enterprise", "Commercial", "Edge"); licenseErr != nil {
			resp.Diagnostics.AddError(
				"Error getting Artifactory license",
				fmt.Sprintf("%v", licenseErr),
			)
			return
		}
	}

	version, err := util.GetArtifactoryVersion(restyBase)
	if err != nil {
		resp.Diagnostics.AddWarning(
			"Error getting Artifactory version",
			fmt.Sprintf("The provider functionality might be affected by the absence of Artifactory version in the context. %v", err),
		)
		return
	}

	featureUsage := fmt.Sprintf("Terraform/%s", req.TerraformVersion)
	util.SendUsage(ctx, restyBase, productId, featureUsage)

	resp.DataSourceData = util.ProvderMetadata{
		Client:             restyBase,
		ArtifactoryVersion: version,
	}

	resp.ResourceData = util.ProvderMetadata{
		Client:             restyBase,
		ArtifactoryVersion: version,
	}
}

func (p *platformProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "platform"
	resp.Version = Version
}

func (p *platformProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		// NewDataSource,
	}
}

func (p *platformProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		// NewResourceWorkerService,
	}
}

func (p *platformProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"url": schema.StringAttribute{
				Description: "Artifactory URL.",
				Optional:    true,
				Validators: []validator.String{
					validator_string.IsURLHttpOrHttps(),
				},
			},
			"access_token": schema.StringAttribute{
				Description: "This is a access token that can be given to you by your admin under `User Management -> Access Tokens`. If not set, the 'api_key' attribute value will be used.",
				Optional:    true,
				Sensitive:   true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"check_license": schema.BoolAttribute{
				Description: "Toggle for pre-flight checking of Artifactory Pro and Enterprise license. Default to `true`.",
				Optional:    true,
			},
		},
	}
}
