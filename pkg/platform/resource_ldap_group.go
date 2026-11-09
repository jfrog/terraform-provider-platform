// Copyright (c) JFrog Ltd. (2025)
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package platform

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
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
)

const (
	ldapGroupEndpoint        = "artifactory/api/security/groups/{name}"
	ldapGroupSettingEndpoint = "access/api/v1/ldap/groups/{setting}"
	ldapGroupImportEndpoint  = "access/api/v1/ldap/groups/import"
)

// The group is visible in Artifactory a moment after the import call returns.
var ldapGroupReadbackDelays = []time.Duration{2 * time.Second, 4 * time.Second, 8 * time.Second}

// NewLDAPGroupResource returns the platform_ldap_group resource.
func NewLDAPGroupResource() resource.Resource {
	return &LDAPGroupResource{
		JFrogResource: util.JFrogResource{
			TypeName:                "platform_ldap_group",
			ValidArtifactoryVersion: "7.57.1",
		},
	}
}

type LDAPGroupResource struct {
	util.JFrogResource
}

type LDAPGroupResourceModel struct {
	ID               types.String `tfsdk:"id"`
	Name             types.String `tfsdk:"name"`
	GroupDN          types.String `tfsdk:"group_dn"`
	LDAPGroupSetting types.String `tfsdk:"ldap_group_setting"`
}

type ldapGroupAPIModel struct {
	Name            string `json:"name"`
	Realm           string `json:"realm"`
	RealmAttributes string `json:"realmAttributes"`
}

type ldapGroupSettingAPIModel struct {
	Name                 string `json:"name"`
	EnabledLDAP          string `json:"enabled_ldap"`
	GroupBaseDN          string `json:"group_base_dn"`
	GroupNameAttribute   string `json:"group_name_attribute"`
	GroupMemberAttribute string `json:"group_member_attribute"`
	SubTree              bool   `json:"sub_tree"`
	ForceAttributeSearch bool   `json:"force_attribute_search"`
	Filter               string `json:"filter"`
	DescriptionAttribute string `json:"description_attribute"`
	Strategy             string `json:"strategy"`
}

type ldapGroupImportAPIModel struct {
	Settings ldapGroupSettingAPIModel `json:"ldap_group_settings"`
	Groups   []ldapGroupImportItem    `json:"import_groups"`
}

type ldapGroupImportItem struct {
	GroupName      string `json:"group_name"`
	GroupDN        string `json:"group_dn"`
	RequiredUpdate string `json:"required_update"`
}

// groupDN returns the DN a group is bound to. Artifactory keeps it in realmAttributes
// as ldapGroupName=...;groupsStrategy=...;groupDn=...
func (g ldapGroupAPIModel) groupDN() (string, bool) {
	for _, part := range strings.Split(g.RealmAttributes, ";") {
		if key, value, found := strings.Cut(part, "="); found && strings.TrimSpace(key) == "groupDn" {
			return strings.TrimSpace(value), true
		}
	}
	return "", false
}

// Schema defines the attributes. name and group_dn force a replacement, ldap_group_setting does not.
func (r *LDAPGroupResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	required := func(description string, modifiers ...planmodifier.String) schema.StringAttribute {
		return schema.StringAttribute{
			Required:            true,
			Validators:          []validator.String{stringvalidator.LengthAtLeast(1)},
			PlanModifiers:       modifiers,
			MarkdownDescription: description,
		}
	}
	replace := stringplanmodifier.RequiresReplace()

	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"name":               required("Name of the group in JFrog.", replace),
			"group_dn":           required("DN of the group in LDAP, for example `cn=my-group,ou=Groups,dc=example,dc=com`.", replace),
			"ldap_group_setting": required("Name of an existing LDAP group setting. Its search settings are used to import the group. Only read when the group is created."),
		},
		MarkdownDescription: "Registers an existing LDAP group in JFrog with the LDAP group import. " +
			"The members stay managed in LDAP. Changing `name` or `group_dn` replaces the group, and destroying the resource deletes the group in JFrog, so check the permissions and roles that reference the group before either. " +
			"See [LDAP Groups](https://jfrog.com/help/r/jfrog-platform-administration-documentation/ldap-groups).",
	}
}

// getGroup returns nil when the group does not exist.
func (r *LDAPGroupResource) getGroup(name string) (*ldapGroupAPIModel, error) {
	var group ldapGroupAPIModel
	var apiErrs util.JFrogErrors
	response, err := r.ProviderData.Client.R().
		SetPathParam("name", name).
		SetResult(&group).
		SetError(&apiErrs).
		Get(ldapGroupEndpoint)
	if err != nil {
		return nil, err
	}
	switch {
	case response.StatusCode() == http.StatusNotFound:
		return nil, nil
	case response.IsError():
		return nil, fmt.Errorf("reading group %s returned %s: %s", name, response.Status(), apiErrs.String())
	}
	return &group, nil
}

// Create imports the group from LDAP. It fails when the group already exists in JFrog.
func (r *LDAPGroupResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	go util.SendUsageResourceCreate(ctx, r.ProviderData.Client.R(), r.ProviderData.ProductId, r.TypeName)

	var plan LDAPGroupResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	name := plan.Name.ValueString()

	existing, err := r.getGroup(name)
	if err != nil {
		utilfw.UnableToCreateResourceError(resp, err.Error())
		return
	}
	if existing != nil {
		if existing.Realm != "ldap" {
			utilfw.UnableToCreateResourceError(resp, fmt.Sprintf("group %s already exists with realm %q, so it cannot be registered as an LDAP group", name, existing.Realm))
			return
		}
		utilfw.UnableToCreateResourceError(resp, fmt.Sprintf("group %s already exists, import it with `terraform import` instead", name))
		return
	}

	var setting ldapGroupSettingAPIModel
	var apiErrs util.JFrogErrors
	response, err := r.ProviderData.Client.R().
		SetPathParam("setting", plan.LDAPGroupSetting.ValueString()).
		SetResult(&setting).
		SetError(&apiErrs).
		Get(ldapGroupSettingEndpoint)
	if err != nil {
		utilfw.UnableToCreateResourceError(resp, err.Error())
		return
	}
	if response.IsError() {
		utilfw.UnableToCreateResourceError(resp, fmt.Sprintf("reading LDAP group setting %s returned %s: %s", plan.LDAPGroupSetting.ValueString(), response.Status(), apiErrs.String()))
		return
	}

	apiErrs = util.JFrogErrors{}
	response, err = r.ProviderData.Client.R().
		SetBody(ldapGroupImportAPIModel{
			Settings: setting,
			Groups:   []ldapGroupImportItem{{GroupName: name, GroupDN: plan.GroupDN.ValueString(), RequiredUpdate: "DOES_NOT_EXIST"}},
		}).
		SetError(&apiErrs).
		Post(ldapGroupImportEndpoint)
	if err != nil {
		utilfw.UnableToCreateResourceError(resp, err.Error())
		return
	}
	if response.IsError() {
		utilfw.UnableToCreateResourceError(resp, fmt.Sprintf("LDAP group import returned %s: %s", response.Status(), apiErrs.String()))
		return
	}

	// The group exists in JFrog from here on. Saving state first means a failure below
	// leaves the resource tainted, so the next apply replaces it instead of hitting "already exists".
	plan.ID = plan.Name
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)

	group, err := r.waitForGroup(ctx, name)
	if err != nil {
		utilfw.UnableToCreateResourceError(resp, err.Error())
		return
	}

	resp.Diagnostics.Append(r.fillModel(&plan, group)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// waitForGroup reads the group until it is visible, or fails after the last retry.
func (r *LDAPGroupResource) waitForGroup(ctx context.Context, name string) (*ldapGroupAPIModel, error) {
	for attempt := 0; ; attempt++ {
		group, err := r.getGroup(name)
		if err != nil {
			return nil, err
		}
		if group != nil {
			return group, nil
		}
		if attempt == len(ldapGroupReadbackDelays) {
			return nil, fmt.Errorf("group %s was imported but is not visible yet", name)
		}
		select {
		case <-time.After(ldapGroupReadbackDelays[attempt]):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
}

// fillModel sets the ID and group_dn from the group in JFrog. It fails when the group is not an LDAP group.
func (r *LDAPGroupResource) fillModel(model *LDAPGroupResourceModel, group *ldapGroupAPIModel) (ds diag.Diagnostics) {
	if group.Realm != "ldap" {
		ds.AddError("Group is not an LDAP group", fmt.Sprintf("group %s has realm %q, expected \"ldap\". Delete the group in JFrog or remove it from the state with `terraform state rm`.", model.Name.ValueString(), group.Realm))
		return
	}
	dn, ok := group.groupDN()
	if !ok {
		ds.AddError("Group has no LDAP binding", fmt.Sprintf("group %s has no groupDn in its realm attributes", model.Name.ValueString()))
		return
	}
	model.ID = model.Name
	// LDAP DNs are case-insensitive and JFrog may return another spelling than the config.
	if !strings.EqualFold(model.GroupDN.ValueString(), dn) {
		model.GroupDN = types.StringValue(dn)
	}
	return
}

// Read refreshes the state and removes the resource when the group no longer exists.
func (r *LDAPGroupResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	go util.SendUsageResourceRead(ctx, r.ProviderData.Client.R(), r.ProviderData.ProductId, r.TypeName)

	var state LDAPGroupResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	group, err := r.getGroup(state.Name.ValueString())
	if err != nil {
		utilfw.UnableToRefreshResourceError(resp, err.Error())
		return
	}
	if group == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	resp.Diagnostics.Append(r.fillModel(&state, group)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Update only stores the plan. Only ldap_group_setting can change in place, and JFrog does not store it.
func (r *LDAPGroupResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	go util.SendUsageResourceUpdate(ctx, r.ProviderData.Client.R(), r.ProviderData.ProductId, r.TypeName)

	var plan LDAPGroupResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// Delete deletes the group in JFrog. A group that is already gone is not an error.
func (r *LDAPGroupResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	go util.SendUsageResourceDelete(ctx, r.ProviderData.Client.R(), r.ProviderData.ProductId, r.TypeName)

	var state LDAPGroupResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var apiErrs util.JFrogErrors
	response, err := r.ProviderData.Client.R().
		SetPathParam("name", state.Name.ValueString()).
		SetError(&apiErrs).
		Delete(ldapGroupEndpoint)
	if err != nil {
		utilfw.UnableToDeleteResourceError(resp, err.Error())
		return
	}
	if response.IsError() && response.StatusCode() != http.StatusNotFound {
		utilfw.UnableToDeleteResourceError(resp, fmt.Sprintf("deleting group %s returned %s: %s", state.Name.ValueString(), response.Status(), apiErrs.String()))
	}
}

// ImportState takes an ID of the form <ldap_group_setting>/<name>.
func (r *LDAPGroupResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	setting, name, ok := strings.Cut(req.ID, "/")
	if !ok || setting == "" || name == "" {
		resp.Diagnostics.AddError("Invalid import ID", "expected <ldap_group_setting>/<name>")
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), name)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("name"), name)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("ldap_group_setting"), setting)...)
}
