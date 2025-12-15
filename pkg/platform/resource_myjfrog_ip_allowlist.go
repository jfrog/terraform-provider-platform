// Copyright (c) JFrog Ltd. (2025)
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package platform

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/jfrog/terraform-provider-shared/util"
)

var _ resource.Resource = (*ipAllowListResource)(nil)

type ipAllowListResource struct {
	ProviderData util.ProviderMetadata
}

func NewMyJFrogIPAllowListResource() resource.Resource {
	return &ipAllowListResource{}
}

func (r *ipAllowListResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_myjfrog_ip_allowlist"
}

func (r *ipAllowListResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}
	r.ProviderData = req.ProviderData.(util.ProviderMetadata)
}

func (r *ipAllowListResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"server_name": schema.StringAttribute{
				Required: true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				MarkdownDescription: "Name of the server. If your JFrog URL is `myserver.jfrog.io`, the `server_name` is `myserver`.",
			},
			"ips": schema.SetAttribute{
				ElementType: types.StringType,
				Required:    true,
				Description: "List of IPs for the JPD allowlist",
			},
		},
		MarkdownDescription: "Provides a MyJFrog [IP allowlist](https://jfrog.com/help/r/jfrog-hosting-models-documentation/configure-the-ip/cidr-allowlist) resource to manage list of allow IP/CIDR addresses. " +
			"To use this resource, you need an access token. Only a Primary Admin can generate MyJFrog tokens. For more information, see [Generate a Token in MyJFrog](https://jfrog.com/help/r/jfrog-hosting-models-documentation/generate-a-token-in-myjfrog).\n\n" +
			"->This resource is supported only on the Cloud (SaaS) platform.\n\n" +
			"~>The rate limit is **5 times per hour** for actions that result in a successful outcome (for Create, Update, and Delete actions). See [Allowlist REST API](https://jfrog.com/help/r/jfrog-rest-apis/allowlist-rest-api) for full list of limitations.\n\n" +
			"!>This resource is being deprecated and moved to the new provider [jfrog/myjfrog](https://registry.terraform.io/providers/jfrog/myjfrog). Use `myjfrog_ip_allowlist` resource there instead.",
		DeprecationMessage: "This resource is being deprecated and moved to the new provider 'jfrog/myjfrog'. Use 'myjfrog_ip_allowlist' resource there instead.",
	}
}

func (r *ipAllowListResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	resp.Diagnostics.Append(
		diag.NewErrorDiagnostic("platform_myjfrog_ip_allowlist resource deprecated", "use myjfrog_ip_allowlist resource instead"),
	)
}

func (r *ipAllowListResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	resp.Diagnostics.Append(
		diag.NewErrorDiagnostic("platform_myjfrog_ip_allowlist resource deprecated", "use myjfrog_ip_allowlist resource instead"),
	)
}

func (r *ipAllowListResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.Append(
		diag.NewErrorDiagnostic("platform_myjfrog_ip_allowlist resource deprecated", "use myjfrog_ip_allowlist resource instead"),
	)
}

func (r *ipAllowListResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	resp.Diagnostics.Append(
		diag.NewErrorDiagnostic("platform_myjfrog_ip_allowlist resource deprecated", "use myjfrog_ip_allowlist resource instead"),
	)
}

func (r *ipAllowListResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(
		diag.NewErrorDiagnostic("platform_myjfrog_ip_allowlist resource deprecated", "use myjfrog_ip_allowlist resource instead"),
	)
}
