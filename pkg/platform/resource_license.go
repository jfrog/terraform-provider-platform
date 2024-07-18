package platform

import (
	"context"
	"net/http"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/jfrog/terraform-provider-shared/util"
	utilfw "github.com/jfrog/terraform-provider-shared/util/fw"
	"github.com/samber/lo"
)

const (
	licensePostEndpoint = "/artifactory/api/system/licenses"
	licenseGetEndpoint  = "/artifactory/api/system/license"
)

var _ resource.Resource = (*licenseResource)(nil)

type licenseResource struct {
	ProviderData PlatformProviderMetadata
	TypeName     string
}

func NewLicenseResource() resource.Resource {
	return &licenseResource{
		TypeName: "platform_license",
	}
}

func (r *licenseResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = r.TypeName
}

func (r *licenseResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"key": schema.StringAttribute{
				Required:    true,
				Description: "License key. Any newline characters must be represented by escape sequence `\n`",
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"name": schema.StringAttribute{
				Optional:    true,
				Description: "Name of the license",
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"type": schema.StringAttribute{
				Computed:    true,
				Description: "Type of the license.",
			},
			"valid_through": schema.StringAttribute{
				Computed:    true,
				Description: "Date of the license is valid through.",
			},
			"licensed_to": schema.StringAttribute{
				Computed:    true,
				Description: "Customer name the license belongs to.",
			},
		},
		MarkdownDescription: "Provides a JFrog [license](https://jfrog.com/help/r/jfrog-platform-administration-documentation/managing-licenses) resource to install/update license.\n\n~>Only available for self-hosted instances.",
	}
}

type licenseResourceModel struct {
	Key          types.String `tfsdk:"key"`
	Name         types.String `tfsdk:"name"`
	Type         types.String `tfsdk:"type"`
	ValidThrough types.String `tfsdk:"valid_through"`
	LicensedTo   types.String `tfsdk:"licensed_to"`
}

func (r *licenseResourceModel) fromAPIModel(_ context.Context, apiModel *licenseAPIGetModel) (ds diag.Diagnostics) {
	r.Type = types.StringValue(apiModel.Type)
	r.ValidThrough = types.StringValue(apiModel.ValidThrough)
	r.LicensedTo = types.StringValue(apiModel.LicensedTo)

	return
}

type licenseAPIPostRequestModel struct {
	Key string `json:"licenseKey"`
}

type licenseAPIPostResonseModel struct {
	Status   int               `json:"status"`
	Messages map[string]string `json:"messages"`
}

type licenseAPIGetModel struct {
	Type         string `json:"type"`
	ValidThrough string `json:"validThrough"`
	LicensedTo   string `json:"licensedTo"`
}

func (r *licenseResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}
	r.ProviderData = req.ProviderData.(PlatformProviderMetadata)
}

func (r *licenseResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	go util.SendUsageResourceCreate(ctx, r.ProviderData.Client.R(), r.ProviderData.ProductId, r.TypeName)

	var plan licenseResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	license := licenseAPIPostRequestModel{
		Key: plan.Key.ValueString(),
	}

	var errorResult licenseAPIPostResonseModel

	response, err := r.ProviderData.Client.R().
		SetBody(&license).
		SetError(&errorResult).
		Post(licensePostEndpoint)

	if err != nil {
		utilfw.UnableToCreateResourceError(resp, err.Error())
		return
	}

	if response.IsError() && !(response.StatusCode() == http.StatusBadRequest &&
		errorResult.Messages[license.Key] == "License already exists.") {
		messages := lo.Values[string, string](errorResult.Messages)
		utilfw.UnableToCreateResourceError(resp, strings.Join(messages, ","))
		return
	}

	var licenseGet licenseAPIGetModel

	response, err = r.ProviderData.Client.R().
		SetResult(&licenseGet).
		Get(licenseGetEndpoint)

	if err != nil {
		utilfw.UnableToCreateResourceError(resp, err.Error())
		return
	}

	if response.IsError() {
		utilfw.UnableToCreateResourceError(resp, response.String())
		return
	}

	// Convert from the API data model to the Terraform data model
	// and refresh any attribute values.
	resp.Diagnostics.Append(plan.fromAPIModel(ctx, &licenseGet)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *licenseResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	go util.SendUsageResourceRead(ctx, r.ProviderData.Client.R(), r.ProviderData.ProductId, r.TypeName)

	var state licenseResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var license licenseAPIGetModel

	response, err := r.ProviderData.Client.R().
		SetResult(&license).
		Get(licenseGetEndpoint)

	if err != nil {
		utilfw.UnableToRefreshResourceError(resp, err.Error())
		return
	}

	// Treat HTTP 404 Not Found status as a signal to recreate resource
	// and return early
	if response.StatusCode() == http.StatusNotFound {
		resp.State.RemoveResource(ctx)
		return
	}

	if response.IsError() {
		utilfw.UnableToRefreshResourceError(resp, response.String())
		return
	}

	// Convert from the API data model to the Terraform data model
	// and refresh any attribute values.
	resp.Diagnostics.Append(state.fromAPIModel(ctx, &license)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *licenseResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	go util.SendUsageResourceUpdate(ctx, r.ProviderData.Client.R(), r.ProviderData.ProductId, r.TypeName)

	var plan licenseResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	license := licenseAPIPostRequestModel{
		Key: plan.Key.ValueString(),
	}

	var errorResult licenseAPIPostResonseModel

	response, err := r.ProviderData.Client.R().
		SetBody(&license).
		SetError(&errorResult).
		Post(licensePostEndpoint)

	if err != nil {
		utilfw.UnableToUpdateResourceError(resp, err.Error())
		return
	}

	if response.IsError() && !(response.StatusCode() == http.StatusBadRequest &&
		errorResult.Messages[license.Key] == "License already exists.") {
		messages := lo.Values[string, string](errorResult.Messages)
		utilfw.UnableToUpdateResourceError(resp, strings.Join(messages, ","))
		return
	}

	var licenseGet licenseAPIGetModel

	response, err = r.ProviderData.Client.R().
		SetResult(&licenseGet).
		Get(licenseGetEndpoint)

	if err != nil {
		utilfw.UnableToUpdateResourceError(resp, err.Error())
		return
	}

	if response.IsError() {
		utilfw.UnableToUpdateResourceError(resp, response.String())
		return
	}

	// Convert from the API data model to the Terraform data model
	// and refresh any attribute values.
	resp.Diagnostics.Append(plan.fromAPIModel(ctx, &licenseGet)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *licenseResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	go util.SendUsageResourceDelete(ctx, r.ProviderData.Client.R(), r.ProviderData.ProductId, r.TypeName)

	resp.Diagnostics.AddWarning(
		"Unable to Delete Resource",
		"License cannot be deleted.",
	)
}
