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

	"github.com/hashicorp/terraform-plugin-framework/datasource"
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
