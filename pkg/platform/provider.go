package platform

import (
	"context"
	"fmt"

	"github.com/go-resty/resty/v2"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/jfrog/terraform-provider-shared/client"
	"github.com/jfrog/terraform-provider-shared/util"
	validator_string "github.com/jfrog/terraform-provider-shared/validator/fw/string"
)

var Version = "1.0.0"

// needs to be exported so make file can update this
var productId = "terraform-provider-platform/" + Version

var _ provider.Provider = (*PlatformProvider)(nil)

type PlatformProviderMetadata struct {
	util.ProviderMetadata
	MyJFrogClient *resty.Client
}

type PlatformProvider struct {
	Meta PlatformProviderMetadata
}

type platformProviderModel struct {
	Url                  types.String `tfsdk:"url"`
	AccessToken          types.String `tfsdk:"access_token"`
	MyJFrogAPIToken      types.String `tfsdk:"myjfrog_api_token"`
	OIDCProviderName     types.String `tfsdk:"oidc_provider_name"`
	TFCCredentialTagName types.String `tfsdk:"tfc_credential_tag_name"`
	CheckLicense         types.Bool   `tfsdk:"check_license"`
}

func NewProvider() func() provider.Provider {
	return func() provider.Provider {
		return &PlatformProvider{}
	}
}

func (p *PlatformProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	// Check environment variables, first available OS variable will be assigned to the var
	url := util.CheckEnvVars([]string{"JFROG_URL"}, "")
	accessToken := util.CheckEnvVars([]string{"JFROG_ACCESS_TOKEN"}, "")

	var config platformProviderModel

	// Read configuration data into model
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if config.Url.ValueString() != "" {
		url = config.Url.ValueString()
	}

	if url == "" {
		resp.Diagnostics.AddError(
			"Missing URL Configuration",
			"While configuring the provider, the url was not found in the JFROG_URL environment variable or provider configuration block url attribute.",
		)
		return
	}

	platformClient, err := client.Build(url, productId)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating Resty client",
			err.Error(),
		)
		return
	}

	oidcProviderName := config.OIDCProviderName.ValueString()
	if oidcProviderName != "" {
		oidcAccessToken, err := util.OIDCTokenExchange(ctx, platformClient, oidcProviderName, config.TFCCredentialTagName.ValueString())
		if err != nil {
			resp.Diagnostics.AddError(
				"Failed OIDC ID token exchange",
				err.Error(),
			)
			return
		}

		// use token from OIDC provider, which should take precedence over
		// environment variable data, if found.
		if oidcAccessToken != "" {
			accessToken = oidcAccessToken
		}
	}

	// use token from configuration, which should take precedence over
	// environment variable data or OIDC provider, if found.
	if config.AccessToken.ValueString() != "" {
		accessToken = config.AccessToken.ValueString()
	}

	myJFrogAPIToken := util.CheckEnvVars([]string{"JFROG_MYJFROG_API_TOKEN"}, "")
	if config.MyJFrogAPIToken.ValueString() != "" {
		myJFrogAPIToken = config.MyJFrogAPIToken.ValueString()
	}

	if accessToken == "" && myJFrogAPIToken == "" {
		resp.Diagnostics.AddError(
			"Missing JFrog Access Token and MyJFrog API token",
			"Neither Access Token nor MyJFrog API Token were found in environment variables or provider configuration. Provider will not function.",
		)
		return
	}

	if accessToken == "" {
		resp.Diagnostics.AddWarning(
			"Missing JFrog Access Token",
			"Access Token was not found in the JFROG_ACCESS_TOKEN environment variable, provider configuration block access_token attribute, or Terraform Cloud TFC_WORKLOAD_IDENTITY_TOKEN environment variable. Platform functionality will be affected.",
		)
	}

	if myJFrogAPIToken == "" {
		resp.Diagnostics.AddWarning(
			"Missing MyJFrog API Token",
			"MyJFrog API Token was not found in the JFROG_MYJFROG_API_TOKEN environment variable or provider configuration block myjfrog_api_token attribute. MyJFrog functionality will be affected.",
		)
	}

	artifactoryVersion := ""
	if len(accessToken) > 0 {
		_, err = client.AddAuth(platformClient, "", accessToken)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error adding Auth to Resty client",
				err.Error(),
			)
			return
		}

		version, err := util.GetArtifactoryVersion(platformClient)
		if err != nil {
			resp.Diagnostics.AddWarning(
				"Error getting Artifactory version",
				fmt.Sprintf("Provider functionality might be affected by the absence of Artifactory version. %v", err),
			)
		}

		artifactoryVersion = version

		featureUsage := fmt.Sprintf("Terraform/%s", req.TerraformVersion)
		go util.SendUsage(ctx, platformClient.R(), productId, featureUsage)
	}

	myJFrogClient, err := client.Build("https://my.jfrog.com", productId)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating Resty client for MyJFrog",
			err.Error(),
		)
		return
	}

	if len(myJFrogAPIToken) > 0 {
		_, err := client.AddAuth(myJFrogClient, "", myJFrogAPIToken)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error adding Auth to Resty client for MyJFrog",
				err.Error(),
			)
			return
		}
	}

	featureUsage := fmt.Sprintf("Terraform/%s", req.TerraformVersion)
	go util.SendUsage(ctx, platformClient.R(), productId, featureUsage)

	meta := PlatformProviderMetadata{
		ProviderMetadata: util.ProviderMetadata{
			Client:             platformClient,
			ArtifactoryVersion: artifactoryVersion,
			ProductId:          productId,
		},
		MyJFrogClient: myJFrogClient,
	}

	p.Meta = meta

	resp.DataSourceData = meta
	resp.ResourceData = meta
}

func (p *PlatformProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "platform"
	resp.Version = Version
}

func (p *PlatformProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		// NewDataSource,
	}
}

func (p *PlatformProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewAWSIAMRoleResource,
		NewLicenseResource,
		NewGlobalRoleResource,
		NewOIDCConfigurationResource,
		NewOIDCIdentityMappingResource,
		NewMyJFrogIPAllowListResource,
		NewPermissionResource,
		NewReverseProxyResource,
		NewSAMLSettingsResource,
		NewSCIMUserResource,
		NewSCIMGroupResource,
		NewWorkerServiceResource,
	}
}

func (p *PlatformProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"url": schema.StringAttribute{
				Optional: true,
				Validators: []validator.String{
					validator_string.IsURLHttpOrHttps(),
				},
				MarkdownDescription: "JFrog Platform URL. This can also be sourced from the `JFROG_URL` environment variable.",
			},
			"access_token": schema.StringAttribute{
				Optional:  true,
				Sensitive: true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
				MarkdownDescription: "This is a access token that can be given to you by your admin under `Platform Configuration -> User Management -> Access Tokens`. This can also be sourced from the `JFROG_ACCESS_TOKEN` environment variable.",
			},
			"myjfrog_api_token": schema.StringAttribute{
				Optional:  true,
				Sensitive: true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
				MarkdownDescription: "MyJFrog API token that allows you to make changes to your JFrog account. See [Generate a Token in MyJFrog](https://jfrog.com/help/r/jfrog-hosting-models-documentation/generate-a-token-in-myjfrog) for more details. This can also be sourced from the `JFROG_MYJFROG_API_TOKEN` environment variable.",
				DeprecationMessage:  "MyJFrog API token is deprecated. Use provider 'jfrog/myjfrog' https://registry.terraform.io/providers/jfrog/myjfrog instead.",
			},
			"oidc_provider_name": schema.StringAttribute{
				Optional: true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
				MarkdownDescription: "OIDC provider name. See [Configure an OIDC Integration](https://jfrog.com/help/r/jfrog-platform-administration-documentation/configure-an-oidc-integration) for more details.",
			},
			"tfc_credential_tag_name": schema.StringAttribute{
				Optional: true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
				Description: "Terraform Cloud Workload Identity Token tag name. Use for generating multiple TFC workload identity tokens. When set, the provider will attempt to use env var with this tag name as suffix. **Note:** this is case sensitive, so if set to `JFROG`, then env var `TFC_WORKLOAD_IDENTITY_TOKEN_JFROG` is used instead of `TFC_WORKLOAD_IDENTITY_TOKEN`. See [Generating Multiple Tokens](https://developer.hashicorp.com/terraform/cloud-docs/workspaces/dynamic-provider-credentials/manual-generation#generating-multiple-tokens) on HCP Terraform for more details.",
			},
			"check_license": schema.BoolAttribute{
				Optional:            true,
				MarkdownDescription: "Toggle for pre-flight checking of Artifactory Pro and Enterprise license. Default to `true`.",
				DeprecationMessage:  "Remove this attribute from your provider configuration as it is no longer used and the attribute will be removed in the next major version of the provider.",
			},
		},
	}
}
