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
	ServersEndpoint = "platform/api/jmis/v1/private-link/servers/{serverName}"
	ServersEndpoint  = "platform/api/jmis/v1/private-link/servers/{serverName}"
)

func NewPrivateLinkServersResource() resource.Resource {
	return &PrivateLinkServersResource{
		TypeName: "platform_servers",
	}
}

type PrivateLinkServersResource struct {
	ProviderData util.ProviderMetadata
	TypeName     string
}

type PrivateLinkServersResourceModel struct {
	ServerName types.String `tfsdk:"serverName"`
	PrivateLinks types.String `tfsdk:"private_links"`
	PrivateLinkId types.String `tfsdk:"privateLinkId"`
	Status types.String `tfsdk:"status"`
}

type ServersAPIModel struct {
	ServerName string `json:"serverName"`
	PrivateLinks string `json:"private_links"`
	PrivateLinkId string `json:"privateLinkId"`
	Status string `json:"status"`
}

func (r *PrivateLinkServersResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = r.TypeName
}

func (r *PrivateLinkServersResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"serverName": schema.StringAttribute{
				Required: true,
				Description: "The serverName of the resource.",
			},
			"private_links": schema.StringAttribute{
				Optional: true,
				Description: "The private_links of the resource.",
			},
			"privateLinkId": schema.StringAttribute{
				Optional: true,
				Description: "The privateLinkId of the resource.",
			},
			"status": schema.StringAttribute{
				Optional: true,
				Description: "The status of the resource.",
			},
		},
		MarkdownDescription: "Manages servers in JFrog Platform.",
	}
}

func (r *PrivateLinkServersResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.ProviderData = req.ProviderData.(util.ProviderMetadata)
}



func (r *PrivateLinkServersResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	go util.SendUsageResourceRead(ctx, r.ProviderData.Client.R(), r.ProviderData.ProductId, r.TypeName)

	var state PrivateLinkServersResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var result PrivateLinkServersAPIModel

	response, err := r.ProviderData.Client.R().
		SetPathParams(map[string]string{
			"serverName": state.ServerName.ValueString(),
		}).
		SetResult(&result).
		Get(ServersEndpoint)
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




