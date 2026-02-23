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
	DeleteEndpoint = "platform/api/jmis/v1/ssl/delete/{{certificate_id}}"
	DeleteEndpoint  = "platform/api/jmis/v1/ssl/delete/{{certificate_id}}"
)

func NewSslDeleteResource() resource.Resource {
	return &SslDeleteResource{
		TypeName: "platform_delete",
	}
}

type SslDeleteResource struct {
	ProviderData util.ProviderMetadata
	TypeName     string
}

type SslDeleteResourceModel struct {
	CertificateId types.String `tfsdk:"certificate_id"`
	Status types.String `tfsdk:"status"`
	Message types.String `tfsdk:"message"`
	SslCertificates types.String `tfsdk:"ssl_certificates"`
	Expiry types.Int64 `tfsdk:"expiry"`
}

type DeleteAPIModel struct {
	CertificateId string `json:"certificate_id"`
	Status string `json:"status"`
	Message string `json:"message"`
	SslCertificates string `json:"ssl_certificates"`
	Expiry string `json:"expiry"`
}

func (r *SslDeleteResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = r.TypeName
}

func (r *SslDeleteResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"certificate_id": schema.StringAttribute{
				Required: true,
				Description: "The certificate_id of the resource.",
			},
			"status": schema.StringAttribute{
				Optional: true,
				Description: "The status of the resource.",
			},
			"message": schema.StringAttribute{
				Optional: true,
				Description: "The message of the resource.",
			},
			"ssl_certificates": schema.StringAttribute{
				Optional: true,
				Description: "The ssl_certificates of the resource.",
			},
			"expiry": schema.StringAttribute{
				Optional: true,
				Description: "The expiry of the resource.",
			},
		},
		MarkdownDescription: "Manages delete in JFrog Platform.",
	}
}

func (r *SslDeleteResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.ProviderData = req.ProviderData.(util.ProviderMetadata)
}



func (r *SslDeleteResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	go util.SendUsageResourceRead(ctx, r.ProviderData.Client.R(), r.ProviderData.ProductId, r.TypeName)

	var state SslDeleteResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var result SslDeleteAPIModel

	response, err := r.ProviderData.Client.R().
		SetPathParams(map[string]string{
			"certificate_id": state.CertificateId.ValueString(),
		}).
		SetResult(&result).
		Get(DeleteEndpoint)
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




func (r *SslDeleteResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	go util.SendUsageResourceDelete(ctx, r.ProviderData.Client.R(), r.ProviderData.ProductId, r.TypeName)

	var state SslDeleteResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	response, err := r.ProviderData.Client.R().
		SetPathParams(map[string]string{
			"certificate_id": state.CertificateId.ValueString(),
		}).
		Delete(DeleteEndpoint)
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

