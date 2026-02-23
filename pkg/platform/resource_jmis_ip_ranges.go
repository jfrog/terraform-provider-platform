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
	IpRangessEndpoint = "platform/api/jmis/v1/ip-ranges"
	IpRangessEndpoint  = "platform/api/jmis/v1/ip-ranges"
)

func NewJmisIpRangesResource() resource.Resource {
	return &JmisIpRangesResource{
		TypeName: "platform_ip_ranges",
	}
}

type JmisIpRangesResource struct {
	ProviderData util.ProviderMetadata
	TypeName     string
}

type JmisIpRangesResourceModel struct {
	Cidr types.String `tfsdk:"cidr"`
	Region types.String `tfsdk:"region"`
	Service types.String `tfsdk:"service"`
	Cloud types.String `tfsdk:"cloud"`
}

type IpRangesAPIModel struct {
	Cidr string `json:"cidr"`
	Region string `json:"region"`
	Service string `json:"service"`
	Cloud string `json:"cloud"`
}

func (r *JmisIpRangesResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = r.TypeName
}

func (r *JmisIpRangesResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"cidr": schema.StringAttribute{
				Optional: true,
				Description: "The cidr of the resource.",
			},
			"region": schema.StringAttribute{
				Optional: true,
				Description: "The region of the resource.",
			},
			"service": schema.StringAttribute{
				Optional: true,
				Description: "The service of the resource.",
			},
			"cloud": schema.StringAttribute{
				Optional: true,
				Description: "The cloud of the resource.",
			},
		},
		MarkdownDescription: "Manages ip_ranges in JFrog Platform.",
	}
}

func (r *JmisIpRangesResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.ProviderData = req.ProviderData.(util.ProviderMetadata)
}



func (r *JmisIpRangesResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	go util.SendUsageResourceRead(ctx, r.ProviderData.Client.R(), r.ProviderData.ProductId, r.TypeName)

	var state JmisIpRangesResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var result JmisIpRangesAPIModel

	response, err := r.ProviderData.Client.R().
		SetPathParams(map[string]string{
		}).
		SetResult(&result).
		Get(IpRangessEndpoint)
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




