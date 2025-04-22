package platform

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"regexp"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/jfrog/terraform-provider-shared/util"
	utilfw "github.com/jfrog/terraform-provider-shared/util/fw"
	validatorfw_string "github.com/jfrog/terraform-provider-shared/validator/fw/string"
)

const odicIdentityMappingEndpoint = "/access/api/v1/oidc/{provider_name}/identity_mappings"

var _ resource.Resource = (*odicIdentityMappingResource)(nil)

type odicIdentityMappingResource struct {
	ProviderData util.ProviderMetadata
	TypeName     string
}

func NewOIDCIdentityMappingResource() resource.Resource {
	return &odicIdentityMappingResource{
		TypeName: "platform_oidc_identity_mapping",
	}
}

func (r *odicIdentityMappingResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = r.TypeName
}

func (r *odicIdentityMappingResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Required: true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 255),
					stringvalidator.RegexMatches(
						regexp.MustCompile(`^[^ !@#$%^&*()+={}\[\]:;'"<>,\./?~\x60|\\]+$`),
						"name cannot contain spaces or special characters",
					),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Description: "Name of the OIDC identity mapping",
			},
			"description": schema.StringAttribute{
				Optional:    true,
				Description: "Description of the OIDC mapping",
			},
			"provider_name": schema.StringAttribute{
				Required:    true,
				Validators:  OIDCConfigurationNameValidators,
				Description: "Name of the OIDC configuration",
			},
			"priority": schema.Int64Attribute{
				Required: true,
				Validators: []validator.Int64{
					int64validator.Between(1, math.MaxInt64),
				},
				Description: "Priority of the identity mapping. The priority should be a number. The higher priority is set for the lower number. If you do not enter a value, the identity mapping is assigned the lowest priority. We recommend that you assign the highest priority (1) to the strongest permission gate. Set the lowest priority to the weakest permission for a logical and effective access control setup.",
			},
			"claims_json": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Claims JSON from the OIDC provider. Use [Terraform jsonencode function](https://developer.hashicorp.com/terraform/language/functions/jsonencode) to encode the JSON string. Claims constitute the payload part of a JSON web token and represent a set of information exchanged between two parties. The JWT standard distinguishes between reserved claims, public claims, and private claims. In API Gateway context, both public claims and private claims are considered custom claims. For example, an ID token (which is always a JWT) can contain a claim called that asserts that the name of the user authenticating is \"John Doe\". In a JWT, a claim appears as a name/value pair where the name is always a string and the value can be any JSON value.",
			},
			"token_spec": schema.SingleNestedAttribute{
				Required: true,
				Attributes: map[string]schema.Attribute{
					"username": schema.StringAttribute{
						Optional: true,
						Validators: []validator.String{
							stringvalidator.AlsoRequires(
								path.MatchRelative().AtParent().AtName("scope"),
							),
							stringvalidator.ConflictsWith(
								path.MatchRelative().AtParent().AtName("username_pattern"),
								path.MatchRelative().AtParent().AtName("groups_pattern"),
							),
							stringvalidator.LengthAtLeast(1),
						},
						Description: "User name of the OIDC user. Not applicable when `scope` is set to `applied-permissions/groups`. Must be set when `scope` is set to `applied-permissions/roles`.",
					},
					"username_pattern": schema.StringAttribute{
						Optional: true,
						Validators: []validator.String{
							stringvalidator.ConflictsWith(
								path.MatchRelative().AtParent().AtName("username"),
								path.MatchRelative().AtParent().AtName("groups_pattern"),
							),
							stringvalidator.LengthAtLeast(1),
						},
						Description: "Provide a pattern which is used to map OIDC user to Artifactory user.",
					},
					"groups_pattern": schema.StringAttribute{
						Optional: true,
						Validators: []validator.String{
							stringvalidator.ConflictsWith(
								path.MatchRelative().AtParent().AtName("username"),
								path.MatchRelative().AtParent().AtName("username_pattern"),
							),
							stringvalidator.LengthAtLeast(1),
						},
						Description: "Provide a pattern which is used to map OIDC groups to Artifactory groups.",
					},
					"scope": schema.StringAttribute{
						Optional: true,
						Validators: []validator.String{
							stringvalidator.RegexMatches(
								regexp.MustCompile(`^(applied-permissions\/admin|applied-permissions\/user|applied-permissions\/groups:.+|applied-permissions\/roles:.+)$`),
								"must start with either 'applied-permissions/admin', 'applied-permissions/user', 'applied-permissions/groups:', or 'applied-permissions/roles:'",
							),
						},
						MarkdownDescription: "Scope of the token. Must start with `applied-permissions/user`, `applied-permissions/admin`, `applied-permissions/roles:`, or `applied-permissions/groups:`. Group names must be comma-separated, double quotes wrapped, e.g. `applied-permissions/groups:\\\"readers\\\",\\\"my-group\\\",` Role permissions are only applicable when in project scope and must be comma-separated, double quotes wrapped, e.g. `applied-permissions:roles:<project-key>:\"Developer\",\"Viewer\". `username` is also required when setting role permission.",
					},
					"audience": schema.StringAttribute{
						Optional: true,
						Computed: true,
						Default:  stringdefault.StaticString("*@*"),
						Validators: []validator.String{
							stringvalidator.LengthAtLeast(1),
						},
						MarkdownDescription: "Sets of (space separated) the JFrog services to which the mapping applies. Default value is `*@*`, which applies to all services.",
					},
					"expires_in": schema.Int64Attribute{
						Optional: true,
						Computed: true,
						Default:  int64default.StaticInt64(60),
						Validators: []validator.Int64{
							int64validator.AtLeast(1),
						},
						MarkdownDescription: "Token expiry time in seconds. Default value is 60.",
					},
				},
				Description: "Specifications of the token. In case of success, a token with the following details will be generated and passed to OIDC Provider.",
			},
			"project_key": schema.StringAttribute{
				Optional: true,
				Validators: []validator.String{
					validatorfw_string.ProjectKey(),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Description: "If set, this Identity Mapping will be available in the scope of the given project (editable by platform admin and project admin). If not set, this Identity Mapping will be global and only editable by platform admin. Once set, the projectKey cannot be changed.",
			},
		},
		MarkdownDescription: "Manage OIDC identity mapping for an OIDC configuration in JFrog platform. See the JFrog [OIDC identity mappings documentation](https://jfrog.com/help/r/jfrog-platform-administration-documentation/configure-identity-mappings) for more information.",
	}
}

type odicIdentityMappingResourceModel struct {
	Name         types.String `tfsdk:"name"`
	Description  types.String `tfsdk:"description"`
	ProviderName types.String `tfsdk:"provider_name"`
	Priority     types.Int64  `tfsdk:"priority"`
	ClaimsJSON   types.String `tfsdk:"claims_json"`
	TokenSpec    types.Object `tfsdk:"token_spec"`
	ProjectKey   types.String `tfsdk:"project_key"`
}

type odicIdentityMappingTokenSpecResourceModel struct {
	Username        types.String `tfsdk:"username"`
	UsernamePattern types.String `tfsdk:"username_pattern"`
	GroupsPattern   types.String `tfsdk:"groups_pattern"`
	Scope           types.String `tfsdk:"scope"`
	Audience        types.String `tfsdk:"audience"`
	ExpiresIn       types.Int64  `tfsdk:"expires_in"`
}

var odicIdentityMappingTokenSpecResourceModelAttributeType map[string]attr.Type = map[string]attr.Type{
	"username":         types.StringType,
	"username_pattern": types.StringType,
	"groups_pattern":   types.StringType,
	"scope":            types.StringType,
	"audience":         types.StringType,
	"expires_in":       types.Int64Type,
}

func (r *odicIdentityMappingResourceModel) toAPIModel(ctx context.Context, apiModel *odicIdentityMappingAPIModel) (ds diag.Diagnostics) {
	var claims map[string]any
	err := json.Unmarshal([]byte(r.ClaimsJSON.ValueString()), &claims)
	if err != nil {
		ds.AddError(
			"fails to unmarshal claims",
			err.Error(),
		)
		return
	}
	var tokenSpec odicIdentityMappingTokenSpecResourceModel
	ds.Append(r.TokenSpec.As(ctx, &tokenSpec, basetypes.ObjectAsOptions{})...)
	if ds.HasError() {
		return
	}

	*apiModel = odicIdentityMappingAPIModel{
		Name:         r.Name.ValueString(),
		Description:  r.Description.ValueString(),
		ProviderName: r.ProviderName.ValueString(),
		Priority:     r.Priority.ValueInt64(),
		Claims:       claims,
		TokenSpec: odicIdentityMappingTokenSpecAPIModel{
			Username:        tokenSpec.Username.ValueString(),
			UsernamePattern: tokenSpec.UsernamePattern.ValueString(),
			GroupsPattern:   tokenSpec.GroupsPattern.ValueString(),
			Scope:           tokenSpec.Scope.ValueString(),
			Audience:        tokenSpec.Audience.ValueString(),
			ExpiresIn:       tokenSpec.ExpiresIn.ValueInt64(),
		},
		ProjectKey: r.ProjectKey.ValueString(),
	}

	return
}

func (r *odicIdentityMappingResourceModel) fromAPIModel(ctx context.Context, apiModel *odicIdentityMappingAPIModel) (ds diag.Diagnostics) {
	r.Name = types.StringValue(apiModel.Name)

	if len(apiModel.Description) > 0 {
		r.Description = types.StringValue(apiModel.Description)
	}

	r.Priority = types.Int64Value(apiModel.Priority)

	claimsBytes, err := json.Marshal(apiModel.Claims)
	if err != nil {
		ds.AddError(
			"fails to marshal claims JSON",
			err.Error(),
		)
		return
	}
	r.ClaimsJSON = types.StringValue(string(claimsBytes))

	tokenSpecResource := odicIdentityMappingTokenSpecResourceModel{
		ExpiresIn: types.Int64Value(apiModel.TokenSpec.ExpiresIn),
	}

	if len(apiModel.TokenSpec.Username) > 0 {
		tokenSpecResource.Username = types.StringValue(apiModel.TokenSpec.Username)
	}

	if len(apiModel.TokenSpec.UsernamePattern) > 0 {
		tokenSpecResource.UsernamePattern = types.StringValue(apiModel.TokenSpec.UsernamePattern)
	}

	if len(apiModel.TokenSpec.GroupsPattern) > 0 {
		tokenSpecResource.GroupsPattern = types.StringValue(apiModel.TokenSpec.GroupsPattern)
	}

	if len(apiModel.TokenSpec.Scope) > 0 {
		tokenSpecResource.Scope = types.StringValue(apiModel.TokenSpec.Scope)
	}

	if len(apiModel.TokenSpec.Audience) > 0 {
		tokenSpecResource.Audience = types.StringValue(apiModel.TokenSpec.Audience)
	}

	tokenSpec, d := types.ObjectValueFrom(
		ctx,
		odicIdentityMappingTokenSpecResourceModelAttributeType,
		tokenSpecResource,
	)
	if d != nil {
		ds = append(ds, d...)
	}
	if ds.HasError() {
		return
	}
	r.TokenSpec = tokenSpec

	return
}

type odicIdentityMappingAPIModel struct {
	Name         string                               `json:"name"`
	Description  string                               `json:"description,omitempty"`
	ProviderName string                               `json:"provider_name"`
	Priority     int64                                `json:"priority"`
	Claims       map[string]any                       `json:"claims"`
	TokenSpec    odicIdentityMappingTokenSpecAPIModel `json:"token_spec"`
	ProjectKey   string                               `json:"project_key,omitempty"`
}

type odicIdentityMappingTokenSpecAPIModel struct {
	Username        string `json:"username,omitempty"`
	UsernamePattern string `json:"username_pattern,omitempty"`
	GroupsPattern   string `json:"groups_pattern,omitempty"`
	Scope           string `json:"scope"`
	Audience        string `json:"audience"`
	ExpiresIn       int64  `json:"expires_in"`
}

func (r *odicIdentityMappingResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}
	r.ProviderData = req.ProviderData.(util.ProviderMetadata)
}

func (r *odicIdentityMappingResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	go util.SendUsageResourceCreate(ctx, r.ProviderData.Client.R(), r.ProviderData.ProductId, r.TypeName)

	var plan odicIdentityMappingResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var odicIdentityMapping odicIdentityMappingAPIModel
	resp.Diagnostics.Append(plan.toAPIModel(ctx, &odicIdentityMapping)...)
	if resp.Diagnostics.HasError() {
		return
	}

	response, err := r.ProviderData.Client.R().
		SetPathParam("provider_name", plan.ProviderName.ValueString()).
		SetBody(&odicIdentityMapping).
		Post(odicIdentityMappingEndpoint)
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

func (r *odicIdentityMappingResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	go util.SendUsageResourceRead(ctx, r.ProviderData.Client.R(), r.ProviderData.ProductId, r.TypeName)

	var state odicIdentityMappingResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var odicIdentityMapping odicIdentityMappingAPIModel

	response, err := r.ProviderData.Client.R().
		SetPathParams(map[string]string{
			"provider_name": state.ProviderName.ValueString(),
			"name":          state.Name.ValueString(),
		}).
		SetResult(&odicIdentityMapping).
		Get(odicIdentityMappingEndpoint + "/{name}")

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
	resp.Diagnostics.Append(state.fromAPIModel(ctx, &odicIdentityMapping)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *odicIdentityMappingResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	go util.SendUsageResourceUpdate(ctx, r.ProviderData.Client.R(), r.ProviderData.ProductId, r.TypeName)

	var plan odicIdentityMappingResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var odicIdentityMapping odicIdentityMappingAPIModel
	resp.Diagnostics.Append(plan.toAPIModel(ctx, &odicIdentityMapping)...)
	if resp.Diagnostics.HasError() {
		return
	}

	response, err := r.ProviderData.Client.R().
		SetPathParams(map[string]string{
			"provider_name": plan.ProviderName.ValueString(),
			"name":          plan.Name.ValueString(),
		}).
		SetBody(&odicIdentityMapping).
		Put(odicIdentityMappingEndpoint + "/{name}")
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

func (r *odicIdentityMappingResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	go util.SendUsageResourceDelete(ctx, r.ProviderData.Client.R(), r.ProviderData.ProductId, r.TypeName)

	var state odicIdentityMappingResourceModel

	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	response, err := r.ProviderData.Client.R().
		SetPathParams(map[string]string{
			"provider_name": state.ProviderName.ValueString(),
			"name":          state.Name.ValueString(),
		}).
		Delete(odicIdentityMappingEndpoint + "/{name}")
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

func (r *odicIdentityMappingResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	idParts := strings.Split(req.ID, ":")

	if len(idParts) < 2 || len(idParts[0]) == 0 || len(idParts[1]) == 0 {
		resp.Diagnostics.AddError(
			"Unexpected Import Identifier",
			fmt.Sprintf("Expected import identifier with format: identity_mapping_name:provider_name. Got: %q", req.ID),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("name"), idParts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("provider_name"), idParts[1])...)

	if len(idParts) == 3 && idParts[2] != "" {
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("project_key"), idParts[2])...)
	}
}
