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
	RegionssEndpoint = "platform/api/jmis/v1/regions"
	RegionssEndpoint  = "platform/api/jmis/v1/regions"
)

func NewJmisRegionsResource() resource.Resource {
	return &JmisRegionsResource{
		TypeName: "platform_regions",
	}
}

type JmisRegionsResource struct {
	ProviderData util.ProviderMetadata
	TypeName     string
}

type JmisRegionsResourceModel struct {
	Regions types.String `tfsdk:"regions"`
	Cloud types.String `tfsdk:"cloud"`
	RegionName types.String `tfsdk:"region_name"`
	RegionCode types.String `tfsdk:"region_code"`
}

type RegionsAPIModel struct {
	Regions string `json:"regions"`
	Cloud string `json:"cloud"`
	RegionName string `json:"region_name"`
	RegionCode string `json:"region_code"`
}

func (r *JmisRegionsResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = r.TypeName
}

func (r *JmisRegionsResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"regions": schema.StringAttribute{
				Optional: true,
				Description: "The regions of the resource.",
			},
			"cloud": schema.StringAttribute{
				Optional: true,
				Description: "The cloud of the resource.",
			},
			"region_name": schema.StringAttribute{
				Optional: true,
				Description: "The region_name of the resource.",
			},
			"region_code": schema.StringAttribute{
				Optional: true,
				Description: "The region_code of the resource.",
			},
		},
		MarkdownDescription: "Manages regions in JFrog Platform.",
	}
}

func (r *JmisRegionsResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.ProviderData = req.ProviderData.(util.ProviderMetadata)
}



func (r *JmisRegionsResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	go util.SendUsageResourceRead(ctx, r.ProviderData.Client.R(), r.ProviderData.ProductId, r.TypeName)

	var state JmisRegionsResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var result JmisRegionsAPIModel

	response, err := r.ProviderData.Client.R().
		SetPathParams(map[string]string{
		}).
		SetResult(&result).
		Get(RegionssEndpoint)
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




