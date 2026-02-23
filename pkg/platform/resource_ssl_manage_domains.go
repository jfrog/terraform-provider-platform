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
	ManageDomainssEndpoint = "platform/api/jmis/v1/ssl/manage_domains"
	ManageDomainssEndpoint  = "platform/api/jmis/v1/ssl/manage_domains"
)

func NewSslManageDomainsResource() resource.Resource {
	return &SslManageDomainsResource{
		TypeName: "platform_manage_domains",
	}
}

type SslManageDomainsResource struct {
	ProviderData util.ProviderMetadata
	TypeName     string
}

type SslManageDomainsResourceModel struct {
	Status types.String `tfsdk:"status"`
	Message types.String `tfsdk:"message"`
}

type ManageDomainsRequestAPIModel struct {
	Status string `json:"status"`
	Message string `json:"message"`
}

type ManageDomainsAPIModel struct {
	Status string `json:"status"`
	Message string `json:"message"`
}

func (r *SslManageDomainsResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = r.TypeName
}

func (r *SslManageDomainsResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"status": schema.StringAttribute{
				Optional: true,
				Description: "The status of the resource.",
			},
			"message": schema.StringAttribute{
				Optional: true,
				Description: "The message of the resource.",
			},
		},
		MarkdownDescription: "Manages manage_domains in JFrog Platform.",
	}
}

func (r *SslManageDomainsResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.ProviderData = req.ProviderData.(util.ProviderMetadata)
}


func (r *SslManageDomainsResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	go util.SendUsageResourceCreate(ctx, r.ProviderData.Client.R(), r.ProviderData.ProductId, r.TypeName)

	var plan SslManageDomainsResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	requestBody := SslManageDomainsRequestAPIModel{
		Status: plan.Status.ValueString(),
		Message: plan.Message.ValueString(),
	}

	var result SslManageDomainsAPIModel

	response, err := r.ProviderData.Client.R().
		SetBody(requestBody).
		SetResult(&result).
		Post(ManageDomainssEndpoint)
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


func (r *SslManageDomainsResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	go util.SendUsageResourceRead(ctx, r.ProviderData.Client.R(), r.ProviderData.ProductId, r.TypeName)

	var state SslManageDomainsResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var result SslManageDomainsAPIModel

	response, err := r.ProviderData.Client.R().
		SetPathParams(map[string]string{
		}).
		SetResult(&result).
		Get(ManageDomainssEndpoint)
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




