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
	CandidatessEndpoint = "mc/api/v1/federation/candidates"
	CandidatessEndpoint  = "mc/api/v1/federation/candidates"
)

func NewFederationCandidatesResource() resource.Resource {
	return &FederationCandidatesResource{
		TypeName: "platform_candidates",
	}
}

type FederationCandidatesResource struct {
	ProviderData util.ProviderMetadata
	TypeName     string
}

type FederationCandidatesResourceModel struct {
}

type CandidatesAPIModel struct {
}

func (r *FederationCandidatesResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = r.TypeName
}

func (r *FederationCandidatesResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
		},
		MarkdownDescription: "Manages candidates in JFrog Platform.",
	}
}

func (r *FederationCandidatesResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.ProviderData = req.ProviderData.(util.ProviderMetadata)
}



func (r *FederationCandidatesResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	go util.SendUsageResourceRead(ctx, r.ProviderData.Client.R(), r.ProviderData.ProductId, r.TypeName)

	var state FederationCandidatesResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var result FederationCandidatesAPIModel

	response, err := r.ProviderData.Client.R().
		SetPathParams(map[string]string{
		}).
		SetResult(&result).
		Get(CandidatessEndpoint)
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




