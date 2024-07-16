package platform

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	utilfw "github.com/jfrog/terraform-provider-shared/util/fw"
)

const reversProxyEndpoint = "/artifactory/api/system/configuration/webServer"
const maxPortNumber = 65535

var supportedDockerProxyMethods = []string{"SUBDOMAIN", "REPOPATHPREFIX", "PORTPERREPO"}
var supportedServerProviderTypes = []string{"DIRECT", "NGINX", "APACHE"}

var _ resource.Resource = (*reverseProxyResource)(nil)

type reverseProxyResource struct {
	ProviderData PlatformProviderMetadata
}

func NewReverseProxyResource() resource.Resource {
	return &reverseProxyResource{}
}

func (r *reverseProxyResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_reverse_proxy"
}

func (r *reverseProxyResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"docker_reverse_proxy_method": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: fmt.Sprintf("Docker access method. The default value is SUBDOMAIN. Supported values: %s.", strings.Join(supportedDockerProxyMethods, ", ")),
				Default:     stringdefault.StaticString("SUBDOMAIN"),
				Validators:  []validator.String{stringvalidator.OneOf(supportedDockerProxyMethods...)},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"server_provider": schema.StringAttribute{
				Required:    true,
				Description: fmt.Sprintf("Set the server provider type. Supported values: %s.", strings.Join(supportedServerProviderTypes, ", ")),
				Validators:  []validator.String{stringvalidator.OneOf(supportedServerProviderTypes...)},
			},
			"public_server_name": schema.StringAttribute{
				Optional:    true,
				Description: "The server name that will be used to access Artifactory. Should be correlated with the base URL value. Must be set when `server_provider` is set to `NIGNIX` or `APACHE`",
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"internal_hostname": schema.StringAttribute{
				Optional:    true,
				Description: "The internal server name for Artifactory which will be used by the web server to access the Artifactory machine. If the web server is installed on the same machine as Artifactory you can use localhost, otherwise use the IP or hostname. Must be set when `server_provider` is set to `NIGNIX` or `APACHE`",
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"use_https": schema.BoolAttribute{
				Optional: true,
				Computed: true,
				Default:  booldefault.StaticBool(false),
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
				Description: "When set, Artifactory will be accessible via HTTPS at the corresponding port that is set. Only settable when `server_provider` is set to `NIGNIX` or `APACHE`",
			},
			"http_port": schema.Int64Attribute{
				Optional: true,
				Computed: true,
				Default:  int64default.StaticInt64(80),
				Validators: []validator.Int64{
					int64validator.AtMost(maxPortNumber),
				},
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
				Description: "The port for access via HTTP. The default value is 80. Only settable when `server_provider` is set to `NIGNIX` or `APACHE`",
			},
			"https_port": schema.Int64Attribute{
				Optional: true,
				Computed: true,
				Default:  int64default.StaticInt64(443),
				Validators: []validator.Int64{
					int64validator.AtMost(maxPortNumber),
				},
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
				Description: "The port for access via HTTPS. The default value is 443. Only settable when `use_https` is set to `true`",
			},
			"ssl_key_path": schema.StringAttribute{
				Optional:    true,
				Description: "The full path of the key file on the web server, e.g. `/etc/ssl/private/myserver.key`. Must be set when `use_https` is set to `true`",
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"ssl_certificate_path": schema.StringAttribute{
				Optional:    true,
				Description: "The full path of the certificate file on the web server, e.g. `/etc/ssl/certs/myserver.crt`. Must be set when `use_https` is set to `true`",
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
		MarkdownDescription: "Provides a JFrog [Reverse Proxy](https://jfrog.com/help/r/jfrog-artifactory-documentation/reverse-proxy-settings) resource.\n\n~>Only available for self-hosted instances.",
	}
}

type reverseProxyResourceModel struct {
	DockerReverseProxyMethod types.String `tfsdk:"docker_reverse_proxy_method"`
	ServerProvider           types.String `tfsdk:"server_provider"`
	PublicServerName         types.String `tfsdk:"public_server_name"`
	InternalHostname         types.String `tfsdk:"internal_hostname"`
	UseHttps                 types.Bool   `tfsdk:"use_https"`
	HttpPort                 types.Int64  `tfsdk:"http_port"`
	HttpsPort                types.Int64  `tfsdk:"https_port"`
	SslKeyPath               types.String `tfsdk:"ssl_key_path"`
	SslCertificatePath       types.String `tfsdk:"ssl_certificate_path"`
}

func (r *reverseProxyResourceModel) toAPIModel(_ context.Context, apiModel *reverseProxyAPIModel) (ds diag.Diagnostics) {
	*apiModel = reverseProxyAPIModel{
		Key:                      strings.ToLower(r.ServerProvider.ValueString()),
		WebServerType:            r.ServerProvider.ValueString(),
		ArtifactoryAppContext:    "artifactory",
		PublicAppContext:         "artifactory",
		ServerName:               r.PublicServerName.ValueString(),
		ArtifactoryServerName:    r.InternalHostname.ValueString(),
		DockerReverseProxyMethod: r.DockerReverseProxyMethod.ValueString(),
		UseHttp:                  true,
		UseHttps:                 r.UseHttps.ValueBool(),
		HttpPort:                 r.HttpPort.ValueInt64(),
		HttpsPort:                r.HttpsPort.ValueInt64(),
		SslKey:                   r.SslKeyPath.ValueString(),
		SslCertificate:           r.SslCertificatePath.ValueString(),
	}

	return nil
}

func (r *reverseProxyResourceModel) fromAPIModel(_ context.Context, apiModel *reverseProxyAPIModel) (ds diag.Diagnostics) {
	r.ServerProvider = types.StringValue(apiModel.WebServerType)
	r.DockerReverseProxyMethod = types.StringValue(apiModel.DockerReverseProxyMethod)
	r.PublicServerName = types.StringValue(apiModel.ServerName)
	r.InternalHostname = types.StringValue(apiModel.ArtifactoryServerName)
	r.UseHttps = types.BoolValue(apiModel.UseHttps)
	r.HttpPort = types.Int64Value(apiModel.HttpPort)
	r.HttpsPort = types.Int64Value(apiModel.HttpsPort)

	// API will return empty string if not configured. Therefore we only update if it isn't empty
	if apiModel.SslKey != "" {
		r.SslKeyPath = types.StringValue(apiModel.SslKey)
	}

	// API will return empty string if not configured. Therefore we only update if it isn't empty
	if apiModel.SslCertificate != "" {
		r.SslCertificatePath = types.StringValue(apiModel.SslCertificate)
	}

	return
}

type reverseProxyAPIModel struct {
	Key                      string `json:"key"`
	WebServerType            string `json:"webServerType"`
	ArtifactoryAppContext    string `json:"artifactoryAppContext"`
	PublicAppContext         string `json:"publicAppContext"`
	ServerName               string `json:"serverName"`
	ArtifactoryServerName    string `json:"artifactoryServerName"`
	DockerReverseProxyMethod string `json:"dockerReverseProxyMethod"`
	UseHttp                  bool   `json:"useHttp"`
	UseHttps                 bool   `json:"useHttps"`
	HttpPort                 int64  `json:"httpPort"`
	HttpsPort                int64  `json:"httpsPort"`
	SslKey                   string `json:"sslKey,omitempty"`
	SslCertificate           string `json:"sslCertificate,omitempty"`
}

func (r *reverseProxyResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}
	r.ProviderData = req.ProviderData.(PlatformProviderMetadata)
}

func (r *reverseProxyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan reverseProxyResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var reverseProxy reverseProxyAPIModel
	resp.Diagnostics.Append(plan.toAPIModel(ctx, &reverseProxy)...)
	if resp.Diagnostics.HasError() {
		return
	}

	response, err := r.ProviderData.Client.R().
		SetBody(&reverseProxy).
		Post(reversProxyEndpoint)
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

func (r *reverseProxyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state reverseProxyResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var reverseProxy reverseProxyAPIModel

	response, err := r.ProviderData.Client.R().
		SetResult(&reverseProxy).
		Get(reversProxyEndpoint)

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
	resp.Diagnostics.Append(state.fromAPIModel(ctx, &reverseProxy)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *reverseProxyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan reverseProxyResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var reverseProxy reverseProxyAPIModel
	resp.Diagnostics.Append(plan.toAPIModel(ctx, &reverseProxy)...)
	if resp.Diagnostics.HasError() {
		return
	}

	response, err := r.ProviderData.Client.R().
		SetBody(&reverseProxy).
		Post(reversProxyEndpoint)
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

func (r *reverseProxyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	resp.Diagnostics.AddWarning(
		"Unable to Delete Resource",
		"Reverse proxy cannot be deleted.",
	)
}

func (r *reverseProxyResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("server_provider"), req, resp)
}

func (r reverseProxyResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var config reverseProxyResourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	switch serverProvider := config.ServerProvider.ValueString(); serverProvider {
	case "NGINX", "APACHE":
		if config.InternalHostname.IsNull() || config.InternalHostname.IsUnknown() {
			resp.Diagnostics.AddAttributeError(
				path.Root("internal_hostname"),
				"Missing Attribute Configuration",
				fmt.Sprintf("internal_hostname must be configured when server_provider is set to '%s'.", serverProvider),
			)
		}

		if config.PublicServerName.IsNull() || config.PublicServerName.IsUnknown() {
			resp.Diagnostics.AddAttributeError(
				path.Root("public_server_name"),
				"Missing Attribute Configuration",
				fmt.Sprintf("public_server_name must be configured when server_provider is set to '%s'.", serverProvider),
			)
		}
	}

	if config.UseHttps.ValueBool() {
		if config.SslKeyPath.IsNull() || config.SslKeyPath.IsUnknown() {
			resp.Diagnostics.AddAttributeError(
				path.Root("ssl_key_path"),
				"Missing Attribute Configuration",
				"ssl_key_path must be configured when use_https is set to 'true'.",
			)
		}

		if config.SslCertificatePath.IsNull() || config.SslCertificatePath.IsUnknown() {
			resp.Diagnostics.AddAttributeError(
				path.Root("ssl_certificate_path"),
				"Missing Attribute Configuration",
				"ssl_certificate_path must be configured when use_https is set to 'true'.",
			)
		}
	}
}
