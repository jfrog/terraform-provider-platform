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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/jfrog/terraform-provider-shared/util"
	utilfw "github.com/jfrog/terraform-provider-shared/util/fw"
	"github.com/samber/lo"
)

const (
	SCIMGroupsEndpoint = "access/api/v1/scim/v2/Groups"
	SCIMGroupEndpoint  = "access/api/v1/scim/v2/Groups/{name}"
)

func NewSCIMGroupResource() resource.Resource {
	return &SCIMGroupResource{
		TypeName: "platform_scim_group",
	}
}

type SCIMGroupResource struct {
	ProviderData PlatformProviderMetadata
	TypeName     string
}

type SCIMGroupResourceModel struct {
	ID          types.String `tfsdk:"id"`
	DisplayName types.String `tfsdk:"display_name"`
	Members     types.Set    `tfsdk:"members"`
	Meta        types.Map    `tfsdk:"meta"`
}

type SCIMGroupAPIModel struct {
	Schemas     []string                  `json:"schemas"`
	ID          string                    `json:"id"`
	DisplayName string                    `json:"displayName"`
	Members     []SCIMGroupMemberAPIModel `json:"members"`
	Meta        map[string]string         `json:"meta,omitempty"`
}

type SCIMGroupMemberAPIModel struct {
	Value   string `json:"value"`
	Display string `json:"display"`
}

func (r *SCIMGroupResourceModel) toAPIModel(_ context.Context, apiModel *SCIMGroupAPIModel) (ds diag.Diagnostics) {
	apiModel.Schemas = []string{"urn:ietf:params:scim:schemas:core:2.0:Group"}

	apiModel.ID = r.ID.ValueString()
	apiModel.DisplayName = r.DisplayName.ValueString()

	members := lo.Map[attr.Value](
		r.Members.Elements(),
		func(elem attr.Value, index int) SCIMGroupMemberAPIModel {
			attr := elem.(types.Object).Attributes()

			return SCIMGroupMemberAPIModel{
				Value:   attr["value"].(types.String).ValueString(),
				Display: attr["display"].(types.String).ValueString(),
			}
		},
	)
	apiModel.Members = members

	return
}

var SCIMGroupMemberResourceModelAttributeType map[string]attr.Type = map[string]attr.Type{
	"value":   types.StringType,
	"display": types.StringType,
}

func (r *SCIMGroupResourceModel) fromAPIModel(_ context.Context, apiModel *SCIMGroupAPIModel) (ds diag.Diagnostics) {
	r.ID = types.StringValue(apiModel.ID)
	r.DisplayName = types.StringValue(apiModel.DisplayName)

	members := lo.Map(
		apiModel.Members,
		func(member SCIMGroupMemberAPIModel, _ int) attr.Value {
			e, d := types.ObjectValue(
				SCIMGroupMemberResourceModelAttributeType,
				map[string]attr.Value{
					"value":   types.StringValue(member.Value),
					"display": types.StringValue(member.Display),
				},
			)
			if d.HasError() {
				ds.Append(d...)
			}

			return e
		},
	)
	membersSet, d := types.SetValue(
		types.ObjectType{AttrTypes: SCIMGroupMemberResourceModelAttributeType},
		members,
	)
	if d.HasError() {
		ds.Append(d...)
	}
	r.Members = membersSet

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

func (r *SCIMGroupResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = r.TypeName
}

func (r *SCIMGroupResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Required: true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Description: "Group ID",
			},
			"display_name": schema.StringAttribute{
				Required: true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"members": schema.SetNestedAttribute{
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"value": schema.StringAttribute{
							Required: true,
							Validators: []validator.String{
								stringvalidator.LengthAtLeast(1),
							},
						},
						"display": schema.StringAttribute{
							Required: true,
							Validators: []validator.String{
								stringvalidator.LengthAtLeast(1),
							},
						},
					},
				},
				Required: true,
				Validators: []validator.Set{
					setvalidator.SizeAtLeast(1),
				},
			},
			"meta": schema.MapAttribute{
				ElementType: types.StringType,
				Computed:    true,
			},
		},
		MarkdownDescription: "Provides a JFrog [SCIM Group](https://jfrog.com/help/r/jfrog-platform-administration-documentation/scim) resource to manage groups with the SCIM protocol.",
	}
}

func (r *SCIMGroupResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}
	r.ProviderData = req.ProviderData.(PlatformProviderMetadata)
}

func (r *SCIMGroupResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	go util.SendUsageResourceCreate(ctx, r.ProviderData.Client.R(), r.ProviderData.ProductId, r.TypeName)

	var plan SCIMGroupResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var group SCIMGroupAPIModel
	resp.Diagnostics.Append(plan.toAPIModel(ctx, &group)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var result SCIMGroupAPIModel
	var scimErr SCIMErrorAPIModel
	response, err := r.ProviderData.Client.R().
		SetBody(group).
		SetResult(&result).
		SetError(&scimErr).
		Post(SCIMGroupsEndpoint)

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

func (r *SCIMGroupResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	go util.SendUsageResourceRead(ctx, r.ProviderData.Client.R(), r.ProviderData.ProductId, r.TypeName)

	var state SCIMGroupResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var group SCIMGroupAPIModel
	var scimErr SCIMErrorAPIModel

	response, err := r.ProviderData.Client.R().
		SetPathParam("name", state.ID.ValueString()).
		SetResult(&group).
		SetError(&scimErr).
		Get(SCIMGroupEndpoint)

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
	resp.Diagnostics.Append(state.fromAPIModel(ctx, &group)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *SCIMGroupResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	go util.SendUsageResourceUpdate(ctx, r.ProviderData.Client.R(), r.ProviderData.ProductId, r.TypeName)

	var plan SCIMGroupResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var group SCIMGroupAPIModel
	resp.Diagnostics.Append(plan.toAPIModel(ctx, &group)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var result SCIMGroupAPIModel
	var scimErr SCIMErrorAPIModel
	response, err := r.ProviderData.Client.R().
		SetPathParam("name", plan.ID.ValueString()).
		SetBody(group).
		SetResult(&result).
		SetError(&scimErr).
		Put(SCIMGroupEndpoint)

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

func (r *SCIMGroupResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	go util.SendUsageResourceDelete(ctx, r.ProviderData.Client.R(), r.ProviderData.ProductId, r.TypeName)

	var state SCIMGroupResourceModel

	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var scimErr SCIMErrorAPIModel
	response, err := r.ProviderData.Client.R().
		SetPathParam("name", state.ID.ValueString()).
		SetError(&scimErr).
		Delete(SCIMGroupEndpoint)

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

func (r *SCIMGroupResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
