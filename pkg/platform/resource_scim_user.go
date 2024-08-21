package platform

import (
	"context"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/jfrog/terraform-provider-shared/util"
	utilfw "github.com/jfrog/terraform-provider-shared/util/fw"
	"github.com/samber/lo"
)

const (
	SCIMUsersEndpoint = "access/api/v1/scim/v2/Users"
	SCIMUserEndpoint  = "access/api/v1/scim/v2/Users/{id}"
)

func NewSCIMUserResource() resource.Resource {
	return &SCIMUserResource{
		TypeName: "platform_scim_user",
	}
}

type SCIMUserResource struct {
	ProviderData PlatformProviderMetadata
	TypeName     string
}

type SCIMUserResourceModel struct {
	Username types.String `tfsdk:"username"`
	Active   types.Bool   `tfsdk:"active"`
	Emails   types.Set    `tfsdk:"emails"`
	Groups   types.Set    `tfsdk:"groups"`
	Meta     types.Map    `tfsdk:"meta"`
}

func (r *SCIMUserResourceModel) toAPIModel(_ context.Context, apiModel *SCIMUserAPIModel) (ds diag.Diagnostics) {
	apiModel.Schemas = []string{"urn:ietf:params:scim:schemas:core:2.0:User"}

	apiModel.Username = r.Username.ValueString()
	apiModel.Active = r.Active.ValueBool()

	emails := lo.Map[attr.Value](
		r.Emails.Elements(),
		func(elem attr.Value, index int) SCIMUserEmailAPIModel {
			attr := elem.(types.Object).Attributes()

			return SCIMUserEmailAPIModel{
				Value:   attr["value"].(types.String).ValueString(),
				Primary: attr["primary"].(types.Bool).ValueBool(),
			}
		},
	)
	apiModel.Emails = emails

	return
}

var SCIMUserEmailResourceModelAttributeType map[string]attr.Type = map[string]attr.Type{
	"value":   types.StringType,
	"primary": types.BoolType,
}

var SCIMUserGroupResourceModelAttributeType map[string]attr.Type = map[string]attr.Type{
	"value": types.StringType,
}

func (r *SCIMUserResourceModel) fromAPIModel(_ context.Context, apiModel *SCIMUserAPIModel) (ds diag.Diagnostics) {
	r.Username = types.StringValue(apiModel.Username)
	r.Active = types.BoolValue(apiModel.Active)

	emails := lo.Map(
		apiModel.Emails,
		func(email SCIMUserEmailAPIModel, _ int) attr.Value {
			e, d := types.ObjectValue(
				SCIMUserEmailResourceModelAttributeType,
				map[string]attr.Value{
					"value":   types.StringValue(email.Value),
					"primary": types.BoolValue(email.Primary),
				},
			)
			if d.HasError() {
				ds.Append(d...)
			}

			return e
		},
	)
	emailsSet, d := types.SetValue(
		types.ObjectType{AttrTypes: SCIMUserEmailResourceModelAttributeType},
		emails,
	)
	if d.HasError() {
		ds.Append(d...)
	}
	r.Emails = emailsSet

	groups := lo.Map(
		apiModel.Groups,
		func(group SCIMUserGroupAPIModel, _ int) attr.Value {
			e, d := types.ObjectValue(
				SCIMUserGroupResourceModelAttributeType,
				map[string]attr.Value{
					"value": types.StringValue(group.Value),
				},
			)
			if d.HasError() {
				ds.Append(d...)
			}

			return e
		},
	)
	groupsSet, d := types.SetValue(
		types.ObjectType{AttrTypes: SCIMUserGroupResourceModelAttributeType},
		groups,
	)
	if d.HasError() {
		ds.Append(d...)
	}
	r.Groups = groupsSet

	metas := lo.MapEntries(
		apiModel.Meta,
		func(k, v string) (string, attr.Value) {
			return k, types.StringValue(v)
		},
	)

	metasMap, d := types.MapValue(
		types.StringType,
		metas,
	)
	if d.HasError() {
		ds.Append(d...)
	}
	r.Meta = metasMap

	return
}

type SCIMUserAPIModel struct {
	Schemas  []string                `json:"schemas"`
	Username string                  `json:"userName"`
	Active   bool                    `json:"active"`
	Emails   []SCIMUserEmailAPIModel `json:"emails"`
	Groups   []SCIMUserGroupAPIModel `json:"groups,omitempty"`
	Meta     map[string]string       `json:"meta,omitempty"`
}

type SCIMUserEmailAPIModel struct {
	Value   string `json:"value"`
	Primary bool   `json:"primary"`
}

type SCIMUserGroupAPIModel struct {
	Value string `json:"value"`
}

type SCIMErrorAPIModel struct {
	Status  int      `json:"status"`
	Detail  string   `json:"detail"`
	Schemas []string `json:"schemas"`
}

func (r *SCIMUserResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = r.TypeName
}

func (r *SCIMUserResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"username": schema.StringAttribute{
				Required: true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"active": schema.BoolAttribute{
				Optional: true,
				Computed: true,
				Default:  booldefault.StaticBool(true),
			},
			"emails": schema.SetNestedAttribute{
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"value": schema.StringAttribute{
							Required: true,
							Validators: []validator.String{
								stringvalidator.LengthAtLeast(1),
							},
						},
						"primary": schema.BoolAttribute{
							Required: true,
						},
					},
				},
				Required: true,
				Validators: []validator.Set{
					setvalidator.SizeAtLeast(1),
				},
				PlanModifiers: []planmodifier.Set{
					setplanmodifier.RequiresReplace(),
				},
			},
			"groups": schema.SetNestedAttribute{
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"value": schema.StringAttribute{
							Required: true,
							Validators: []validator.String{
								stringvalidator.LengthAtLeast(1),
							},
						},
					},
				},
				Computed: true,
			},
			"meta": schema.MapAttribute{
				ElementType: types.StringType,
				Computed:    true,
			},
		},
		MarkdownDescription: "Provides a JFrog [SCIM User](https://jfrog.com/help/r/jfrog-platform-administration-documentation/scim) resource to manage users with the SCIM protocol.",
	}
}

func (r *SCIMUserResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}
	r.ProviderData = req.ProviderData.(PlatformProviderMetadata)
}

func (r *SCIMUserResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	go util.SendUsageResourceCreate(ctx, r.ProviderData.Client.R(), r.ProviderData.ProductId, r.TypeName)

	var plan SCIMUserResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var user SCIMUserAPIModel
	resp.Diagnostics.Append(plan.toAPIModel(ctx, &user)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var result SCIMUserAPIModel
	var scimErr SCIMErrorAPIModel
	response, err := r.ProviderData.Client.R().
		SetBody(user).
		SetResult(&result).
		SetError(&scimErr).
		Post(SCIMUsersEndpoint)

	if err != nil {
		utilfw.UnableToCreateResourceError(resp, err.Error())
		return
	}

	if response.IsError() {
		utilfw.UnableToCreateResourceError(resp, scimErr.Detail)
		return
	}

	// Convert from the API data model to the Terraform data model
	// and refresh any attribute values.
	resp.Diagnostics.Append(plan.fromAPIModel(ctx, &result)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *SCIMUserResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	go util.SendUsageResourceRead(ctx, r.ProviderData.Client.R(), r.ProviderData.ProductId, r.TypeName)

	var state SCIMUserResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var user SCIMUserAPIModel
	var scimErr SCIMErrorAPIModel

	response, err := r.ProviderData.Client.R().
		SetPathParam("id", state.Username.ValueString()).
		SetResult(&user).
		SetError(&scimErr).
		Get(SCIMUserEndpoint)

	if err != nil {
		utilfw.UnableToRefreshResourceError(resp, err.Error())
		return
	}

	if response.IsError() {
		utilfw.UnableToRefreshResourceError(resp, scimErr.Detail)
		return
	}

	// Treat HTTP 404 Not Found status as a signal to recreate resource
	// and return early
	if response.StatusCode() == http.StatusNotFound {
		resp.State.RemoveResource(ctx)
		return
	}

	// Convert from the API data model to the Terraform data model
	// and refresh any attribute values.
	resp.Diagnostics.Append(state.fromAPIModel(ctx, &user)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *SCIMUserResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	go util.SendUsageResourceUpdate(ctx, r.ProviderData.Client.R(), r.ProviderData.ProductId, r.TypeName)

	var plan SCIMUserResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var user SCIMUserAPIModel
	resp.Diagnostics.Append(plan.toAPIModel(ctx, &user)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var result SCIMUserAPIModel
	var scimErr SCIMErrorAPIModel
	response, err := r.ProviderData.Client.R().
		SetPathParam("id", plan.Username.ValueString()).
		SetBody(user).
		SetResult(&result).
		SetError(&scimErr).
		Put(SCIMUserEndpoint)

	if err != nil {
		utilfw.UnableToUpdateResourceError(resp, err.Error())
		return
	}

	if response.IsError() {
		utilfw.UnableToUpdateResourceError(resp, scimErr.Detail)
		return
	}

	// Convert from the API data model to the Terraform data model
	// and refresh any attribute values.
	resp.Diagnostics.Append(plan.fromAPIModel(ctx, &result)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *SCIMUserResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	go util.SendUsageResourceDelete(ctx, r.ProviderData.Client.R(), r.ProviderData.ProductId, r.TypeName)

	var state SCIMUserResourceModel

	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var scimErr SCIMErrorAPIModel
	response, err := r.ProviderData.Client.R().
		SetPathParam("id", state.Username.ValueString()).
		SetError(&scimErr).
		Delete(SCIMUserEndpoint)

	if err != nil {
		utilfw.UnableToDeleteResourceError(resp, err.Error())
		return
	}

	if response.IsError() {
		utilfw.UnableToDeleteResourceError(resp, scimErr.Detail)
		return
	}

	// If the logic reaches here, it implicitly succeeded and will remove
	// the resource from state if there are no other errors.
}

func (r *SCIMUserResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("username"), req, resp)
}
