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
	RerunEndpoint = "worker/api/v1/rerun/{workerKey}"
	RerunEndpoint  = "worker/api/v1/rerun/{workerKey}"
)

func NewRerunResource() resource.Resource {
	return &RerunResource{
		TypeName: "platform_rerun",
	}
}

type RerunResource struct {
	ProviderData util.ProviderMetadata
	TypeName     string
}

type RerunResourceModel struct {
	WorkerKey types.String `tfsdk:"workerKey"`
}

type RerunRequestAPIModel struct {
	WorkerKey string `json:"workerKey"`
}

type RerunAPIModel struct {
	WorkerKey string `json:"workerKey"`
}

func (r *RerunResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = r.TypeName
}

func (r *RerunResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"workerKey": schema.StringAttribute{
				Required: true,
				Description: "The workerKey of the resource.",
			},
		},
		MarkdownDescription: "Manages rerun in JFrog Platform.",
	}
}

func (r *RerunResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.ProviderData = req.ProviderData.(util.ProviderMetadata)
}


func (r *RerunResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	go util.SendUsageResourceCreate(ctx, r.ProviderData.Client.R(), r.ProviderData.ProductId, r.TypeName)

	var plan RerunResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	requestBody := RerunRequestAPIModel{
		WorkerKey: plan.WorkerKey.ValueString(),
	}

	var result RerunAPIModel

	response, err := r.ProviderData.Client.R().
		SetBody(requestBody).
		SetResult(&result).
		Post(RerunEndpoint)
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


func (r *RerunResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	go util.SendUsageResourceRead(ctx, r.ProviderData.Client.R(), r.ProviderData.ProductId, r.TypeName)

	var state RerunResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var result RerunAPIModel

	response, err := r.ProviderData.Client.R().
		SetPathParams(map[string]string{
			"workerKey": state.WorkerKey.ValueString(),
		}).
		SetResult(&result).
		Get(RerunEndpoint)
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




