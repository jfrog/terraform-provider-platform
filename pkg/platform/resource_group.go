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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/jfrog/terraform-provider-shared/util"
	utilfw "github.com/jfrog/terraform-provider-shared/util/fw"
	validatorfw "github.com/jfrog/terraform-provider-shared/validator/fw"
	"github.com/samber/lo"
)

var _ resource.Resource = (*groupResource)(nil)

type groupResource struct {
	util.JFrogResource
}

func NewGroupResource() resource.Resource {
	return &groupResource{
		JFrogResource: util.JFrogResource{
			TypeName:                "platform_group",
			ValidArtifactoryVersion: "7.49.3",
			CollectionEndpoint:      "access/api/v2/groups",
			DocumentEndpoint:        "access/api/v2/groups/{name}",
		},
	}
}

func (r *groupResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Required: true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 64),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				MarkdownDescription: "Name of the group.",
			},
			"description": schema.StringAttribute{
				Optional: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
				MarkdownDescription: "A description for the group.",
			},
			"external_id": schema.StringAttribute{
				Optional: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
				MarkdownDescription: "New external group ID used to configure the corresponding group in Azure AD.",
			},
			"auto_join": schema.BoolAttribute{
				Optional: true,
				Computed: true,
				Default:  booldefault.StaticBool(false),
				Validators: []validator.Bool{
					validatorfw.BoolConflict(true, path.Expressions{
						path.MatchRelative().AtParent().AtName("admin_privileges"),
					}...),
				},
				MarkdownDescription: "When this parameter is set, any new users defined in the system are automatically assigned to this group.",
			},
			"admin_privileges": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
				MarkdownDescription: "Any users added to this group will automatically be assigned with admin privileges in the system.",
			},
			"members": schema.SetAttribute{
				ElementType: types.StringType,
				Optional:    true,
				PlanModifiers: []planmodifier.Set{
					setplanmodifier.UseStateForUnknown(),
				},
				MarkdownDescription: "List of users assigned to the group.",
			},
			"realm": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The realm for the group.",
			},
			"realm_attributes": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The realm for the group.",
			},
		},
		MarkdownDescription: "Provides a group resource to create and manage groups, and manages membership. A group represents a role and is used with RBAC (Role-Based Access Control) rules. See [JFrog documentation](https://jfrog.com/help/r/jfrog-platform-administration-documentation/create-and-edit-groups) for more details.",
	}
}

type groupResourceModel struct {
	Name            types.String `tfsdk:"name"`
	Description     types.String `tfsdk:"description"`
	ExternalId      types.String `tfsdk:"external_id"`
	AutoJoin        types.Bool   `tfsdk:"auto_join"`
	AdminPrivileges types.Bool   `tfsdk:"admin_privileges"`
	Members         types.Set    `tfsdk:"members"`
	Realm           types.String `tfsdk:"realm"`
	RealmAttributes types.String `tfsdk:"realm_attributes"`
}

func (r *groupResourceModel) toAPIModel(ctx context.Context, apiModel *groupAPIModel) (ds diag.Diagnostics) {

	var members []string
	ds.Append(r.Members.ElementsAs(ctx, &members, false)...)
	if ds.HasError() {
		return
	}

	*apiModel = groupAPIModel{
		Name:            r.Name.ValueString(),
		Description:     r.Description.ValueStringPointer(),
		ExternalId:      r.ExternalId.ValueStringPointer(),
		AutoJoin:        r.AutoJoin.ValueBoolPointer(),
		AdminPrivileges: r.AdminPrivileges.ValueBoolPointer(),
		Members:         members,
	}

	return nil
}

func (r *groupResourceModel) fromAPIModel(ctx context.Context, apiModel groupAPIModel, ignoreMembers bool) diag.Diagnostics {
	diags := diag.Diagnostics{}

	r.Name = types.StringValue(apiModel.Name)
	r.Description = types.StringPointerValue(apiModel.Description)
	r.ExternalId = types.StringPointerValue(apiModel.ExternalId)
	r.AutoJoin = types.BoolPointerValue(apiModel.AutoJoin)
	r.AdminPrivileges = types.BoolPointerValue(apiModel.AdminPrivileges)
	r.Realm = types.StringPointerValue(apiModel.Realm)
	r.RealmAttributes = types.StringPointerValue(apiModel.RealmAttributes)

	if r.Members.IsUnknown() {
		r.Members = types.SetNull(types.StringType)
	}

	if !ignoreMembers && len(apiModel.Members) > 0 {
		members, d := types.SetValueFrom(ctx, types.StringType, apiModel.Members)
		if d != nil {
			diags.Append(d...)
			return diags
		}

		r.Members = members
	}

	return diags
}

type groupAPIModel struct {
	Name            string   `json:"name"`
	Description     *string  `json:"description,omitempty"`
	ExternalId      *string  `json:"external_id,omitempty"`
	AutoJoin        *bool    `json:"auto_join,omitempty"`
	AdminPrivileges *bool    `json:"admin_privileges,omitempty"`
	Members         []string `json:"members,omitempty"`          // only for create
	Realm           *string  `json:"realm,omitempty"`            // read only
	RealmAttributes *string  `json:"realm_attributes,omitempty"` // read only
}

type groupMembersRequestAPIModel struct {
	Add    []string `json:"add"`
	Remove []string `json:"remove"`
}

type groupMembersResponseAPIModel struct {
	Members []string `json:"members"`
}

func (r *groupResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	go util.SendUsageResourceCreate(ctx, r.ProviderData.Client.R(), r.ProviderData.ProductId, r.TypeName)

	var plan *groupResourceModel
	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var group groupAPIModel
	resp.Diagnostics.Append(plan.toAPIModel(ctx, &group)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var newGroup groupAPIModel
	var apiErrs util.JFrogErrors
	response, err := r.ProviderData.Client.R().
		SetBody(group).
		SetResult(&newGroup).
		SetError(&apiErrs).
		Post(r.JFrogResource.CollectionEndpoint)

	if err != nil {
		utilfw.UnableToCreateResourceError(resp, err.Error())
		return
	}

	if response.StatusCode() != http.StatusCreated {
		utilfw.UnableToCreateResourceError(resp, apiErrs.String())
		return
	}

	plan.Realm = types.StringPointerValue(newGroup.Realm)
	plan.RealmAttributes = types.StringPointerValue(newGroup.RealmAttributes)

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *groupResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	go util.SendUsageResourceRead(ctx, r.ProviderData.Client.R(), r.ProviderData.ProductId, r.TypeName)

	var state *groupResourceModel
	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var group groupAPIModel
	var apiErrs util.JFrogErrors
	response, err := r.ProviderData.Client.R().
		SetPathParam("name", state.Name.ValueString()).
		SetResult(&group).
		SetError(&apiErrs).
		Get(r.JFrogResource.DocumentEndpoint)

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
		utilfw.UnableToRefreshResourceError(resp, apiErrs.String())
		return
	}

	// Convert from the API data model to the Terraform data model
	// and refresh any attribute values.
	resp.Diagnostics.Append(state.fromAPIModel(ctx, group, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *groupResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	go util.SendUsageResourceUpdate(ctx, r.ProviderData.Client.R(), r.ProviderData.ProductId, r.TypeName)

	var plan groupResourceModel
	var state groupResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var group groupAPIModel
	resp.Diagnostics.Append(plan.toAPIModel(ctx, &group)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var updatedGroup groupAPIModel
	var apiErrs util.JFrogErrors
	response, err := r.ProviderData.Client.R().
		SetPathParam("name", plan.Name.ValueString()).
		SetBody(group).
		SetResult(&updatedGroup).
		SetError(&apiErrs).
		Patch(r.JFrogResource.DocumentEndpoint)
	if err != nil {
		utilfw.UnableToUpdateResourceError(resp, err.Error())
		return
	}

	if response.IsError() {
		utilfw.UnableToUpdateResourceError(resp, apiErrs.String())
		return
	}

	resp.Diagnostics.Append(plan.fromAPIModel(ctx, updatedGroup, true)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var planMembers []string
	resp.Diagnostics.Append(plan.Members.ElementsAs(ctx, &planMembers, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var stateMembers []string
	resp.Diagnostics.Append(state.Members.ElementsAs(ctx, &stateMembers, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	memebersToAdd, membersToRemove := lo.Difference(planMembers, stateMembers)
	membersReq := groupMembersRequestAPIModel{
		Add:    memebersToAdd,
		Remove: membersToRemove,
	}

	if len(memebersToAdd) > 0 || len(membersToRemove) > 0 {
		var membersRes groupMembersResponseAPIModel
		response, err = r.ProviderData.Client.R().
			SetPathParam("name", plan.Name.ValueString()).
			SetBody(membersReq).
			SetResult(&membersRes).
			SetError(&apiErrs).
			Patch(r.JFrogResource.DocumentEndpoint + "/members")
		if err != nil {
			utilfw.UnableToUpdateResourceError(resp, err.Error())
			return
		}

		if response.IsError() {
			utilfw.UnableToUpdateResourceError(resp, apiErrs.String())
			return
		}

		// only update members attribute if it is set in the configuration
		if !plan.Members.IsNull() {
			ms, d := types.SetValueFrom(ctx, types.StringType, membersRes.Members)
			if d != nil {
				resp.Diagnostics.Append(d...)
				return
			}

			plan.Members = ms
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *groupResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	go util.SendUsageResourceDelete(ctx, r.ProviderData.Client.R(), r.ProviderData.ProductId, r.TypeName)

	var state groupResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	var apiErrs util.JFrogErrors
	response, err := r.ProviderData.Client.R().
		SetPathParam("name", state.Name.ValueString()).
		SetError(&apiErrs).
		Delete(r.JFrogResource.DocumentEndpoint)

	if err != nil {
		utilfw.UnableToDeleteResourceError(resp, err.Error())
		return
	}

	// Return error if the HTTP status code is not 204 No Content or 404 Not Found
	if response.StatusCode() != http.StatusNotFound && response.StatusCode() != http.StatusNoContent {
		utilfw.UnableToDeleteResourceError(resp, apiErrs.String())
		return
	}

	// If the logic reaches here, it implicitly succeeded and will remove
	// the resource from state if there are no other errors.
}

// ImportState imports the resource into the Terraform state.
func (r *groupResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("name"), req, resp)
}
