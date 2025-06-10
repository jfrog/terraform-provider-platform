package platform

import (
	"context"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/jfrog/terraform-provider-shared/util"
)

// needs to be exported so make file can update this
var Version = "2.0.0"

var _ provider.Provider = &PlatformProvider{}

type PlatformProvider struct {
	util.JFrogProvider
}

func NewProvider() func() provider.Provider {
	return func() provider.Provider {
		return &PlatformProvider{
			JFrogProvider: util.JFrogProvider{
				TypeName:  "platform",
				ProductID: "terraform-provider-platform/" + Version,
				Version:   Version,
			},
		}
	}
}

func (p *PlatformProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		// NewDataSource,
	}
}

func (p *PlatformProvider) GetResourceIdentitySchemas(ctx context.Context) (map[string]interface{}, diag.Diagnostics) {
	return map[string]interface{}{
		"platform_group": map[string]string{
			"type_name": "platform_group",
			"id_attr":   "name",
		},
		"platform_permission": map[string]string{
			"type_name": "platform_permission",
			"id_attr":   "name",
		},
	}, nil
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
