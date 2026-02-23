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
	PrivateLinkEndpoint = "platform/api/jmis/v1/private-link/{privateLinkId}"
	PrivateLinkEndpoint  = "platform/api/jmis/v1/private-link/{privateLinkId}"
)

func NewJmisPrivateLinkResource() resource.Resource {
	return &JmisPrivateLinkResource{
		TypeName: "platform_private_link",
	}
}

type JmisPrivateLinkResource struct {
	ProviderData util.ProviderMetadata
	TypeName     string
}

type JmisPrivateLinkResourceModel struct {
	PrivateLinkId types.String `tfsdk:"privateLinkId"`
	Servers types.String `tfsdk:"servers"`
	ServerName types.String `tfsdk:"serverName"`
	PrivateLinkStatus types.String `tfsdk:"privateLinkStatus"`
}

type PrivateLinkAPIModel struct {
	PrivateLinkId string `json:"privateLinkId"`
	Servers string `json:"servers"`
	ServerName string `json:"serverName"`
	PrivateLinkStatus string `json:"privateLinkStatus"`
}

func (r *JmisPrivateLinkResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = r.TypeName
}

func (r *JmisPrivateLinkResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"privateLinkId": schema.StringAttribute{
				Required: true,
				Description: "The privateLinkId of the resource.",
			},
			"servers": schema.StringAttribute{
				Optional: true,
				Description: "The servers of the resource.",
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
		MarkdownDescription: "Manages private_link in JFrog Platform.",
	}
}

func (r *JmisPrivateLinkResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.ProviderData = req.ProviderData.(util.ProviderMetadata)
}



func (r *JmisPrivateLinkResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	go util.SendUsageResourceRead(ctx, r.ProviderData.Client.R(), r.ProviderData.ProductId, r.TypeName)

	var state JmisPrivateLinkResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var result JmisPrivateLinkAPIModel

	response, err := r.ProviderData.Client.R().
		SetPathParams(map[string]string{
			"privateLinkId": state.PrivateLinkId.ValueString(),
		}).
		SetResult(&result).
		Get(PrivateLinkEndpoint)
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




