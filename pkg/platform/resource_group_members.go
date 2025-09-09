package platform

import (
	"context"
	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
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
	"github.com/samber/lo"
	"net/http"
)

var _ resource.Resource = (*groupMembersResource)(nil)

type groupMembersResource struct {
	util.JFrogResource
}

func NewGroupMembersResource() resource.Resource {
	return &groupMembersResource{
		JFrogResource: util.JFrogResource{
			TypeName:         "platform_group_members",
			DocumentEndpoint: "access/api/v2/groups/{name}/members",
		},
	}
}

func (r *groupMembersResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
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
			"members": schema.SetAttribute{
				ElementType: types.StringType,
				Required:    true,
				Validators: []validator.Set{
					setvalidator.SizeAtLeast(1),
				},
				MarkdownDescription: "List of users assigned to the group.",
			},
		},
		MarkdownDescription: "Provides a resource to manage group membership. See [JFrog documentation](https://jfrog.com/help/r/jfrog-platform-administration-documentation/assign-users-to-groups) for more details.",
	}
}

type groupMembersResourceModel struct {
	Name    types.String `tfsdk:"name"`
	Members types.Set    `tfsdk:"members"`
}

func (r *groupMembersResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	go util.SendUsageResourceCreate(ctx, r.ProviderData.Client.R(), r.ProviderData.ProductId, r.TypeName)

	var plan groupMembersResourceModel
	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var members []string
	resp.Diagnostics.Append(plan.Members.ElementsAs(ctx, &members, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	groupMembers := groupMembersRequestAPIModel{
		Add: members,
	}

	var updatedGroupMembers groupMembersResponseAPIModel
	var apiErrs util.JFrogErrors
	response, err := r.ProviderData.Client.R().
		SetPathParam("name", plan.Name.ValueString()).
		SetBody(groupMembers).
		SetResult(&updatedGroupMembers).
		SetError(&apiErrs).
		Patch(r.JFrogResource.DocumentEndpoint)

	if err != nil {
		utilfw.UnableToCreateResourceError(resp, err.Error())
		return
	}

	if response.IsError() {
		utilfw.UnableToCreateResourceError(resp, apiErrs.String())
		return
	}

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *groupMembersResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	go util.SendUsageResourceRead(ctx, r.ProviderData.Client.R(), r.ProviderData.ProductId, r.TypeName)

	var state groupMembersResourceModel
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
		Get("access/api/v2/groups/{name}")

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
	members, d := types.SetValueFrom(ctx, types.StringType, group.Members)
	if d != nil {
		resp.Diagnostics.Append(d...)
	}
	if resp.Diagnostics.HasError() {
		return
	}

	state.Members = members

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *groupMembersResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	go util.SendUsageResourceUpdate(ctx, r.ProviderData.Client.R(), r.ProviderData.ProductId, r.TypeName)

	var plan groupMembersResourceModel
	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state groupMembersResourceModel
	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
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
	groupMembers := groupMembersRequestAPIModel{
		Add:    memebersToAdd,
		Remove: membersToRemove,
	}

	var updatedGroupMembers groupMembersResponseAPIModel
	var apiErrs util.JFrogErrors
	response, err := r.ProviderData.Client.R().
		SetPathParam("name", plan.Name.ValueString()).
		SetBody(groupMembers).
		SetResult(&updatedGroupMembers).
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

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *groupMembersResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	go util.SendUsageResourceDelete(ctx, r.ProviderData.Client.R(), r.ProviderData.ProductId, r.TypeName)

	var state groupMembersResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	var members []string
	resp.Diagnostics.Append(state.Members.ElementsAs(ctx, &members, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	groupMembers := groupMembersRequestAPIModel{
		Remove: members,
	}

	var updatedGroupMembers groupMembersResponseAPIModel
	var apiErrs util.JFrogErrors
	response, err := r.ProviderData.Client.R().
		SetPathParam("name", state.Name.ValueString()).
		SetBody(groupMembers).
		SetResult(&updatedGroupMembers).
		SetError(&apiErrs).
		Patch(r.JFrogResource.DocumentEndpoint)

	if err != nil {
		utilfw.UnableToDeleteResourceError(resp, err.Error())
		return
	}

	if response.IsError() {
		utilfw.UnableToDeleteResourceError(resp, apiErrs.String())
		return
	}

	// If the logic reaches here, it implicitly succeeded and will remove
	// the resource from state if there are no other errors.
}

// ImportState imports the resource into the Terraform state.
func (r *groupMembersResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("name"), req, resp)
}
