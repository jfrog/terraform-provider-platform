package platform

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

var Version = "0.0.1"

// needs to be exported so make file can update this
var productId = "terraform-provider-platform/" + Version

var _ provider.Provider = (*platformProvider)(nil)

type platformProvider struct{}

func New() func() provider.Provider {
	return func() provider.Provider {
		return &platformProvider{}
	}
}

func (p *platformProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
}

func (p *platformProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "platform"
}

func (p *platformProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		// NewDataSource,
	}
}

func (p *platformProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		// NewResourceWorkerService,
	}
}

func (p *platformProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
}
