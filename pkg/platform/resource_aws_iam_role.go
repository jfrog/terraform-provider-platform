package platform

import (
	"context"
	"fmt"
	"net/http"
	"regexp"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/jfrog/terraform-provider-shared/util"
	utilfw "github.com/jfrog/terraform-provider-shared/util/fw"
)

const (
	AWSIAMRolesEndpoint = "access/api/v1/aws/iam_role"
	AWSIAMRoleEndpoint  = "access/api/v1/aws/iam_role/{username}"
)

func NewAWSIAMRoleResource() resource.Resource {
	return &AWSIAMRoleResource{
		TypeName: "platform_aws_iam_role",
	}
}

type AWSIAMRoleResource struct {
	ProviderData util.ProviderMetadata
	TypeName     string
}

type AWSIAMRoleResourceModel struct {
	Username types.String `tfsdk:"username"`
	IAMRole  types.String `tfsdk:"iam_role"`
}

type AWSIAMRoleAPIModel struct {
	Username string `json:"username"`
	IAMRole  string `json:"iam_role"`
}

func (r *AWSIAMRoleResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = r.TypeName
}

func (r *AWSIAMRoleResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"username": schema.StringAttribute{
				Required: true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Description: "The JFrog Platform user name.",
			},
			"iam_role": schema.StringAttribute{
				Required: true,
				Validators: []validator.String{
					stringvalidator.RegexMatches(regexp.MustCompile(`^arn:aws:iam::\d{12}:role/[\w+=,.@:-]+$`), "Must follow the regex, \"^arn:aws:iam::\\d{12}:role/[\\w+=,.@:-]+$\""),
				},
				MarkdownDescription: "The AWS IAM role. Must follow the regex, \"^arn:aws:iam::\\d{12}:role/[\\w+=,.@:-]+$\"",
			},
		},
		MarkdownDescription: "Provides a resource to manage AWS IAM roles for JFrog platform users. You can use the AWS IAM roles for passwordless access to Amazon EKS. For more information, see [Passwordless Access for Amazon EKS](https://jfrog.com/help/r/jfrog-installation-setup-documentation/passwordless-access-for-amazon-eks).\n\n" +
			"->Only available for Artifactory 7.90.10 or later.",
	}
}

func (r *AWSIAMRoleResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}
	r.ProviderData = req.ProviderData.(util.ProviderMetadata)

	supported, err := util.CheckVersion(r.ProviderData.ArtifactoryVersion, "7.90.10")
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to check Artifactory version",
			err.Error(),
		)
		return
	}

	if !supported {
		resp.Diagnostics.AddError(
			"Unsupported Artifactory version",
			fmt.Sprintf("This resource is supported by Artifactory version 7.90.10 or later. Current version: %s", r.ProviderData.ArtifactoryVersion),
		)
		return
	}
}

func (r *AWSIAMRoleResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	go util.SendUsageResourceCreate(ctx, r.ProviderData.Client.R(), r.ProviderData.ProductId, r.TypeName)

	var plan AWSIAMRoleResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	role := AWSIAMRoleAPIModel{
		Username: plan.Username.ValueString(),
		IAMRole:  plan.IAMRole.ValueString(),
	}

	response, err := r.ProviderData.Client.R().
		SetBody(role).
		Put(AWSIAMRolesEndpoint)

	if err != nil {
		utilfw.UnableToCreateResourceError(resp, err.Error())
		return
	}

	if response.IsError() {
		utilfw.UnableToCreateResourceError(resp, response.String())
		return
	}

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *AWSIAMRoleResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	go util.SendUsageResourceRead(ctx, r.ProviderData.Client.R(), r.ProviderData.ProductId, r.TypeName)

	var state AWSIAMRoleResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var role AWSIAMRoleAPIModel

	response, err := r.ProviderData.Client.R().
		SetPathParam("username", state.Username.ValueString()).
		SetResult(&role).
		Get(AWSIAMRoleEndpoint)

	if err != nil {
		utilfw.UnableToRefreshResourceError(resp, err.Error())
		return
	}

	if response.IsError() {
		utilfw.UnableToRefreshResourceError(resp, response.String())
		return
	}

	// Treat HTTP 404 Not Found status as a signal to recreate resource
	// and return early
	if response.StatusCode() == http.StatusNotFound {
		resp.State.RemoveResource(ctx)
		return
	}

	// Convert from the API data model to the Terraform data model
	// and refresh any attribute values.
	state.Username = types.StringValue(role.Username)
	state.IAMRole = types.StringValue(role.IAMRole)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *AWSIAMRoleResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	go util.SendUsageResourceUpdate(ctx, r.ProviderData.Client.R(), r.ProviderData.ProductId, r.TypeName)

	var plan AWSIAMRoleResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	role := AWSIAMRoleAPIModel{
		Username: plan.Username.ValueString(),
		IAMRole:  plan.IAMRole.ValueString(),
	}

	response, err := r.ProviderData.Client.R().
		SetBody(role).
		Put(AWSIAMRolesEndpoint)

	if err != nil {
		utilfw.UnableToUpdateResourceError(resp, err.Error())
		return
	}

	if response.IsError() {
		utilfw.UnableToUpdateResourceError(resp, response.String())
		return
	}

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *AWSIAMRoleResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	go util.SendUsageResourceDelete(ctx, r.ProviderData.Client.R(), r.ProviderData.ProductId, r.TypeName)

	var state AWSIAMRoleResourceModel

	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	response, err := r.ProviderData.Client.R().
		SetPathParam("username", state.Username.ValueString()).
		Delete(AWSIAMRoleEndpoint)

	if err != nil {
		utilfw.UnableToDeleteResourceError(resp, err.Error())
		return
	}

	if response.IsError() {
		utilfw.UnableToDeleteResourceError(resp, response.String())
		return
	}

	// If the logic reaches here, it implicitly succeeded and will remove
	// the resource from state if there are no other errors.
}

func (r *AWSIAMRoleResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("username"), req, resp)
}
