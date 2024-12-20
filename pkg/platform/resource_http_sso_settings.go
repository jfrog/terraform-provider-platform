package platform

import (
	"context"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/jfrog/terraform-provider-shared/util"
	utilfw "github.com/jfrog/terraform-provider-shared/util/fw"
)

func NewHTTPSSOSettingsResource() resource.Resource {
	return &HTTPSSOSettingsResource{
		JFrogResource: util.JFrogResource{
			TypeName:                "platform_http_sso_settings",
			ValidArtifactoryVersion: "7.57.1",
			DocumentEndpoint:        "access/api/v1/httpsso",
		},
	}
}

type HTTPSSOSettingsResource struct {
	util.JFrogResource
}

type HTTPSSOSettingsResourceModel struct {
	Proxied                   types.Bool   `tfsdk:"proxied"`
	AutoCreateUser            types.Bool   `tfsdk:"auto_create_user"`
	AllowUserToAccessProfile  types.Bool   `tfsdk:"allow_user_to_access_profile"`
	RemoteUserRequestVariable types.String `tfsdk:"remote_user_request_variable"`
	SyncLDAPGroups            types.Bool   `tfsdk:"sync_ldap_groups"`
}

func (r *HTTPSSOSettingsResourceModel) toAPIModel(_ context.Context, apiModel *HTTPSSOSettingsAPIModel) diag.Diagnostics {
	diags := diag.Diagnostics{}

	apiModel.Proxied = r.Proxied.ValueBool()
	apiModel.AutoCreateUser = r.AutoCreateUser.ValueBool()
	apiModel.AllowUserToAccessProfile = r.AllowUserToAccessProfile.ValueBool()
	apiModel.RemoteUserRequestVariable = r.RemoteUserRequestVariable.ValueString()
	apiModel.SyncLDAPGroups = r.SyncLDAPGroups.ValueBool()

	return diags
}

func (r *HTTPSSOSettingsResourceModel) fromAPIModel(_ context.Context, apiModel *HTTPSSOSettingsAPIModel) (ds diag.Diagnostics) {
	r.Proxied = types.BoolValue(apiModel.Proxied)
	r.AutoCreateUser = types.BoolValue(apiModel.AutoCreateUser)
	r.AllowUserToAccessProfile = types.BoolValue(apiModel.AllowUserToAccessProfile)
	r.RemoteUserRequestVariable = types.StringValue(apiModel.RemoteUserRequestVariable)
	r.SyncLDAPGroups = types.BoolValue(apiModel.SyncLDAPGroups)

	return
}

type HTTPSSOSettingsAPIModel struct {
	Proxied                   bool   `json:"http_sso_proxied"`
	AutoCreateUser            bool   `json:"auto_create_user"`
	AllowUserToAccessProfile  bool   `json:"allow_user_to_access_profile"`
	RemoteUserRequestVariable string `json:"remote_user_request_variable"`
	SyncLDAPGroups            bool   `json:"sync_ldap_groups"`
}

func (r *HTTPSSOSettingsResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"proxied": schema.BoolAttribute{
				Required:            true,
				MarkdownDescription: "When set, Artifactory trusts incoming requests and reuses the remote user originally set on the request by the SSO of the HTTP server. This is useful if you want to use existing enterprise SSO integrations, such as the powerful authentication schemes provided by Apache (mod_auth_ldap, mod_auth_ntlm, mod_auth_kerb, etc.). When Artifactory is deployed as a webapp on Tomcat behind Apache: If using mod_jk, be sure to use the `JkEnvVar REMOTE_USER` directive in Apache's configuration.",
			},
			"auto_create_user": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
				MarkdownDescription: "When set, authenticated users are automatically created in Artifactory. When not set, for every request from an SSO user, the user is temporarily associated with default groups (if such groups are defined), and the permissions for these groups apply. Without automatic user creation, you must manually create the user inside Artifactory to manage user permissions not attached to their default groups. Default to `false`.",
			},
			"allow_user_to_access_profile": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
				MarkdownDescription: "Auto created users will have access to their profile page and will be able to perform actions such as generating an API key. Default to `false`.",
			},
			"remote_user_request_variable": schema.StringAttribute{
				Optional: true,
				Computed: true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
				Default:             stringdefault.StaticString("REMOTE_USER"),
				MarkdownDescription: "The name of the HTTP request variable to use for extracting the user identity. Default to `REMOTE_USER`.",
			},
			"sync_ldap_groups": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
				MarkdownDescription: "When set, the user will be associated with the groups returned in the LDAP login response. Note that the user's association with the returned groups is persistent if the `auto_create_user` is set. Default to `false`.",
			},
		},
		MarkdownDescription: "Provides a JFrog [HTTP SSO Settings](https://jfrog.com/help/r/jfrog-platform-administration-documentation/http-sso) resource. This allows you to reuse existing HTTP-based SSO infrastructures with the JFrog Platform Unit (JPD), such as the SSO modules offered by Apache HTTPd.",
	}
}

func (r *HTTPSSOSettingsResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	go util.SendUsageResourceCreate(ctx, r.ProviderData.Client.R(), r.ProviderData.ProductId, r.TypeName)

	var plan HTTPSSOSettingsResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var httpSSOSettings HTTPSSOSettingsAPIModel
	resp.Diagnostics.Append(plan.toAPIModel(ctx, &httpSSOSettings)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var jfrogErrors util.JFrogErrors
	response, err := r.ProviderData.Client.R().
		SetBody(httpSSOSettings).
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

func (r *HTTPSSOSettingsResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	go util.SendUsageResourceRead(ctx, r.ProviderData.Client.R(), r.ProviderData.ProductId, r.TypeName)

	var state HTTPSSOSettingsResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var httpSSOSettings HTTPSSOSettingsAPIModel

	response, err := r.ProviderData.Client.R().
		SetResult(&httpSSOSettings).
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
	resp.Diagnostics.Append(state.fromAPIModel(ctx, &httpSSOSettings)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *HTTPSSOSettingsResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	go util.SendUsageResourceUpdate(ctx, r.ProviderData.Client.R(), r.ProviderData.ProductId, r.TypeName)

	var plan HTTPSSOSettingsResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var httpSSOSettings HTTPSSOSettingsAPIModel
	resp.Diagnostics.Append(plan.toAPIModel(ctx, &httpSSOSettings)...)
	if resp.Diagnostics.HasError() {
		return
	}

	response, err := r.ProviderData.Client.R().
		SetBody(httpSSOSettings).
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

func (r *HTTPSSOSettingsResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	go util.SendUsageResourceDelete(ctx, r.ProviderData.Client.R(), r.ProviderData.ProductId, r.TypeName)

	resp.Diagnostics.AddWarning(
		"Unable to Delete Resource",
		"HTTP SSO settings cannot be deleted.",
	)
}

func (r *HTTPSSOSettingsResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("remote_user_request_variable"), req, resp)
}
