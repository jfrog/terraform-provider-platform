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
	ServerssEndpoint = "platform/api/jmis/v1/servers"
	ServerssEndpoint  = "platform/api/jmis/v1/servers"
)

func NewJmisServersResource() resource.Resource {
	return &JmisServersResource{
		TypeName: "platform_servers",
	}
}

type JmisServersResource struct {
	ProviderData util.ProviderMetadata
	TypeName     string
}

type JmisServersResourceModel struct {
	Servers types.String `tfsdk:"servers"`
	ServerName types.String `tfsdk:"server_name"`
	ServerType types.String `tfsdk:"server_type"`
	CloudProvider types.String `tfsdk:"cloud_provider"`
	Region types.String `tfsdk:"region"`
}

type ServersAPIModel struct {
	Servers string `json:"servers"`
	ServerName string `json:"server_name"`
	ServerType string `json:"server_type"`
	CloudProvider string `json:"cloud_provider"`
	Region string `json:"region"`
}

func (r *JmisServersResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = r.TypeName
}

func (r *JmisServersResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"servers": schema.StringAttribute{
				Optional: true,
				Description: "The servers of the resource.",
			},
			"server_name": schema.StringAttribute{
				Optional: true,
				Description: "The server_name of the resource.",
			},
			"server_type": schema.StringAttribute{
				Optional: true,
				Description: "The server_type of the resource.",
			},
			"cloud_provider": schema.StringAttribute{
				Optional: true,
				Description: "The cloud_provider of the resource.",
			},
			"region": schema.StringAttribute{
				Optional: true,
				Description: "The region of the resource.",
			},
		},
		MarkdownDescription: "Manages servers in JFrog Platform.",
	}
}

func (r *JmisServersResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.ProviderData = req.ProviderData.(util.ProviderMetadata)
}



func (r *JmisServersResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	go util.SendUsageResourceRead(ctx, r.ProviderData.Client.R(), r.ProviderData.ProductId, r.TypeName)

	var state JmisServersResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var result JmisServersAPIModel

	response, err := r.ProviderData.Client.R().
		SetPathParams(map[string]string{
		}).
		SetResult(&result).
		Get(ServerssEndpoint)
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




