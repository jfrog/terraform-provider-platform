package platform

import (
	"context"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-go/tfprotov5"
	"github.com/jfrog/terraform-provider-shared/util"
)

// needs to be exported so make file can update this
var Version = "2.0.0"

var _ provider.Provider = &PlatformProvider{}

type PlatformProvider struct {
	util.JFrogProvider
	logger hclog.Logger
}

func NewProvider() func() provider.Provider {
	return func() provider.Provider {
		return &PlatformProvider{
			JFrogProvider: util.JFrogProvider{
				TypeName:  "platform",
				ProductID: "terraform-provider-platform/" + Version,
				Version:   Version,
			},
			logger: hclog.New(&hclog.LoggerOptions{
				Name:  "platform-provider",
				Level: hclog.LevelFromString("DEBUG"),
			}),
		}
	}
}

func (p *PlatformProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		// NewDataSource,
	}
}

func (p *PlatformProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewAWSIAMRoleResource,
		NewCrowdSettingsResource,
		NewLicenseResource,
		NewGlobalRoleResource,
		NewGroupResource,
		NewGroupMembersResource,
		NewHTTPSSOSettingsResource,
		NewOIDCConfigurationResource,
		NewOIDCIdentityMappingResource,
		NewMyJFrogIPAllowListResource,
		NewPermissionResource,
		NewReverseProxyResource,
		NewSAMLSettingsResource,
		NewSCIMUserResource,
		NewSCIMGroupResource,
		NewWorkerServiceResource,
	}
}

func dump(v interface{}) hclog.Format {
	return hclog.Fmt("%v", v)
}

func (s *PlatformProvider) GetResourceIdentitySchemas(ctx context.Context, req *tfprotov5.GetResourceIdentitySchemasRequest) (*tfprotov5.GetResourceIdentitySchemasResponse, error) {
	s.logger.Trace("[GetResourceIdentitySchemas][Request]\n%s\n", dump(*req))
	resp := &tfprotov5.GetResourceIdentitySchemasResponse{}
	return resp, nil
}

func (s *PlatformProvider) UpgradeResourceIdentity(ctx context.Context, req *tfprotov5.UpgradeResourceIdentityRequest) (*tfprotov5.UpgradeResourceIdentityResponse, error) {
	s.logger.Trace("[UpgradeResourceIdentity][Request]\n%s\n", dump(*req))
	resp := &tfprotov5.UpgradeResourceIdentityResponse{}
	return resp, nil
}
