package platform

import (
	"context"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/jfrog/terraform-provider-shared/util"
	utilfw "github.com/jfrog/terraform-provider-shared/util/fw"
)

const (
	FederationsEndpoint = "access/api/v1/system/federation"
	FederationEndpoint  = "access/api/v1/system/federation/{server_name}"
)

func NewSystemFederationResource() resource.Resource {
	return &SystemFederationResource{
		TypeName: "platform_federation",
	}
}

type SystemFederationResource struct {
	ProviderData util.ProviderMetadata
	TypeName     string
}

type SystemFederationResourceModel struct {
	ServerName types.String `tfsdk:"server_name"`
	Name types.String `tfsdk:"name"`
	Url types.String `tfsdk:"url"`
	Active types.Bool `tfsdk:"active"`
	Entities types.String `tfsdk:"entities"`
	PermissionFilters types.Map `tfsdk:"permission_filters"`
	IncludePatterns types.String `tfsdk:"include_patterns"`
	ExcludePatterns types.String `tfsdk:"exclude_patterns"`
	GroupFilters types.Map `tfsdk:"group_filters"`
	ForceOverride types.Bool `tfsdk:"force_override"`
	Targets types.String `tfsdk:"targets"`
	PermissionFilters types.Map `tfsdk:"permissionFilters"`
	IncludePatterns types.String `tfsdk:"includePatterns"`
	ExcludePatterns types.String `tfsdk:"excludePatterns"`
}

type FederationRequestAPIModel struct {
	ServerName string `json:"server_name"`
	Name string `json:"name"`
	Url string `json:"url"`
	Active string `json:"active"`
	Entities string `json:"entities"`
	PermissionFilters string `json:"permission_filters"`
	IncludePatterns string `json:"include_patterns"`
	ExcludePatterns string `json:"exclude_patterns"`
	GroupFilters string `json:"group_filters"`
	ForceOverride string `json:"force_override"`
	Targets string `json:"targets"`
	PermissionFilters string `json:"permissionFilters"`
	IncludePatterns string `json:"includePatterns"`
	ExcludePatterns string `json:"excludePatterns"`
}

type FederationAPIModel struct {
	ServerName string `json:"server_name"`
	Name string `json:"name"`
	Url string `json:"url"`
	Active string `json:"active"`
	Entities string `json:"entities"`
	PermissionFilters string `json:"permission_filters"`
	IncludePatterns string `json:"include_patterns"`
	ExcludePatterns string `json:"exclude_patterns"`
	GroupFilters string `json:"group_filters"`
	ForceOverride string `json:"force_override"`
	Targets string `json:"targets"`
	PermissionFilters string `json:"permissionFilters"`
	IncludePatterns string `json:"includePatterns"`
	ExcludePatterns string `json:"excludePatterns"`
}

func (r *SystemFederationResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = r.TypeName
}

func (r *SystemFederationResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"server_name": schema.StringAttribute{
				Required: true,
				Description: "The server_name of the resource.",
			},
			"name": schema.StringAttribute{
				Optional: true,
				Description: "The name of the resource.",
			},
			"url": schema.StringAttribute{
				Optional: true,
				Description: "The url of the resource.",
			},
			"active": schema.StringAttribute{
				Optional: true,
				Description: "The active of the resource.",
			},
			"entities": schema.StringAttribute{
				Optional: true,
				Description: "The entities of the resource.",
			},
			"permission_filters": schema.StringAttribute{
				Optional: true,
				Description: "The permission_filters of the resource.",
			},
			"include_patterns": schema.StringAttribute{
				Optional: true,
				Description: "The include_patterns of the resource.",
			},
			"exclude_patterns": schema.StringAttribute{
				Optional: true,
				Description: "The exclude_patterns of the resource.",
			},
			"group_filters": schema.StringAttribute{
				Optional: true,
				Description: "The group_filters of the resource.",
			},
			"force_override": schema.StringAttribute{
				Optional: true,
				Description: "The force_override of the resource.",
			},
			"targets": schema.StringAttribute{
				Optional: true,
				Description: "The targets of the resource.",
			},
			"permissionFilters": schema.StringAttribute{
				Optional: true,
				Description: "The permissionFilters of the resource.",
			},
			"includePatterns": schema.StringAttribute{
				Optional: true,
				Description: "The includePatterns of the resource.",
			},
			"excludePatterns": schema.StringAttribute{
				Optional: true,
				Description: "The excludePatterns of the resource.",
			},
		},
		MarkdownDescription: "Manages federation in JFrog Platform.",
	}
}

func (r *SystemFederationResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.ProviderData = req.ProviderData.(util.ProviderMetadata)
}


func (r *SystemFederationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	go util.SendUsageResourceCreate(ctx, r.ProviderData.Client.R(), r.ProviderData.ProductId, r.TypeName)

	var plan SystemFederationResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	requestBody := SystemFederationRequestAPIModel{
		ServerName: plan.ServerName.ValueString(),
		Name: plan.Name.ValueString(),
		Url: plan.Url.ValueString(),
		Active: plan.Active.ValueString(),
		Entities: plan.Entities.ValueString(),
		PermissionFilters: plan.PermissionFilters.ValueString(),
		IncludePatterns: plan.IncludePatterns.ValueString(),
		ExcludePatterns: plan.ExcludePatterns.ValueString(),
		GroupFilters: plan.GroupFilters.ValueString(),
		ForceOverride: plan.ForceOverride.ValueString(),
		Targets: plan.Targets.ValueString(),
		PermissionFilters: plan.PermissionFilters.ValueString(),
		IncludePatterns: plan.IncludePatterns.ValueString(),
		ExcludePatterns: plan.ExcludePatterns.ValueString(),
	}

	var result SystemFederationAPIModel

	response, err := r.ProviderData.Client.R().
		SetBody(requestBody).
		SetResult(&result).
		Post(FederationsEndpoint)
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


func (r *SystemFederationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	go util.SendUsageResourceRead(ctx, r.ProviderData.Client.R(), r.ProviderData.ProductId, r.TypeName)

	var state SystemFederationResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var result SystemFederationAPIModel

	response, err := r.ProviderData.Client.R().
		SetPathParams(map[string]string{
			"server_name": state.ServerName.ValueString(),
		}).
		SetResult(&result).
		Get(FederationEndpoint)
	if err != nil {
		utilfw.UnableToRefreshResourceError(resp, err.Error())
		return
	}

	if response.StatusCode() == http.StatusNotFound {
		resp.State.RemoveResource(ctx)
		return
	}

	if response.IsError() {
		utilfw.UnableToRefreshResourceError(resp, response.String())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}


func (r *SystemFederationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	go util.SendUsageResourceUpdate(ctx, r.ProviderData.Client.R(), r.ProviderData.ProductId, r.TypeName)

	var plan SystemFederationResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	requestBody := SystemFederationRequestAPIModel{
		ServerName: plan.ServerName.ValueString(),
		Name: plan.Name.ValueString(),
		Url: plan.Url.ValueString(),
		Active: plan.Active.ValueString(),
		Entities: plan.Entities.ValueString(),
		PermissionFilters: plan.PermissionFilters.ValueString(),
		IncludePatterns: plan.IncludePatterns.ValueString(),
		ExcludePatterns: plan.ExcludePatterns.ValueString(),
		GroupFilters: plan.GroupFilters.ValueString(),
		ForceOverride: plan.ForceOverride.ValueString(),
		Targets: plan.Targets.ValueString(),
		PermissionFilters: plan.PermissionFilters.ValueString(),
		IncludePatterns: plan.IncludePatterns.ValueString(),
		ExcludePatterns: plan.ExcludePatterns.ValueString(),
	}

	response, err := r.ProviderData.Client.R().
		SetPathParams(map[string]string{
			"server_name": plan.ServerName.ValueString(),
		}).
		SetBody(requestBody).
		Patch(FederationEndpoint)
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



func (r *SystemFederationResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	go util.SendUsageResourceDelete(ctx, r.ProviderData.Client.R(), r.ProviderData.ProductId, r.TypeName)

	var state SystemFederationResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	response, err := r.ProviderData.Client.R().
		SetPathParams(map[string]string{
			"server_name": state.ServerName.ValueString(),
		}).
		Delete(FederationEndpoint)
	if err != nil {
		utilfw.UnableToDeleteResourceError(resp, err.Error())
		return
	}

	if response.StatusCode() == http.StatusNotFound {
		return
	}

	if response.IsError() {
		utilfw.UnableToDeleteResourceError(resp, response.String())
		return
	}
}

