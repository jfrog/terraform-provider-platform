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
	DeletesEndpoint = "platform/api/jmis/v1/private-link/delete"
	DeletesEndpoint  = "platform/api/jmis/v1/private-link/delete"
)

func NewPrivateLinkDeleteResource() resource.Resource {
	return &PrivateLinkDeleteResource{
		TypeName: "platform_delete",
	}
}

type PrivateLinkDeleteResource struct {
	ProviderData util.ProviderMetadata
	TypeName     string
}

type PrivateLinkDeleteResourceModel struct {
	PrivateLinkId types.String `tfsdk:"privateLinkId"`
	Results types.String `tfsdk:"results"`
	ServerName types.String `tfsdk:"serverName"`
	PrivateLinkStatus types.String `tfsdk:"privateLinkStatus"`
}

type DeleteAPIModel struct {
	PrivateLinkId string `json:"privateLinkId"`
	Results string `json:"results"`
	ServerName string `json:"serverName"`
	PrivateLinkStatus string `json:"privateLinkStatus"`
}

func (r *PrivateLinkDeleteResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = r.TypeName
}

func (r *PrivateLinkDeleteResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"privateLinkId": schema.StringAttribute{
				Optional: true,
				Description: "The privateLinkId of the resource.",
			},
			"results": schema.StringAttribute{
				Optional: true,
				Description: "The results of the resource.",
			},
			"serverName": schema.StringAttribute{
				Optional: true,
				Description: "The serverName of the resource.",
			},
			"privateLinkStatus": schema.StringAttribute{
				Optional: true,
				Description: "The privateLinkStatus of the resource.",
			},
		},
		MarkdownDescription: "Manages delete in JFrog Platform.",
	}
}

func (r *PrivateLinkDeleteResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.ProviderData = req.ProviderData.(util.ProviderMetadata)
}



func (r *PrivateLinkDeleteResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	go util.SendUsageResourceRead(ctx, r.ProviderData.Client.R(), r.ProviderData.ProductId, r.TypeName)

	var state PrivateLinkDeleteResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var result PrivateLinkDeleteAPIModel

	response, err := r.ProviderData.Client.R().
		SetPathParams(map[string]string{
		}).
		SetResult(&result).
		Get(DeletesEndpoint)
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




func (r *PrivateLinkDeleteResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	go util.SendUsageResourceDelete(ctx, r.ProviderData.Client.R(), r.ProviderData.ProductId, r.TypeName)

	var state PrivateLinkDeleteResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	response, err := r.ProviderData.Client.R().
		SetPathParams(map[string]string{
		}).
		Delete(DeletesEndpoint)
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

