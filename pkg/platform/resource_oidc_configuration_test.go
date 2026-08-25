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

package platform_test

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"testing"

	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/jfrog/terraform-provider-artifactory/v12/pkg/acctest"
	"github.com/jfrog/terraform-provider-platform/v2/pkg/platform"
	"github.com/jfrog/terraform-provider-shared/testutil"
	"github.com/jfrog/terraform-provider-shared/util"
)

func TestAccOIDCConfiguration_full(t *testing.T) {
	_, fqrn, configName := testutil.MkNames("test-oidc-configuration", "platform_oidc_configuration")

	temp := `
	resource "platform_oidc_configuration" "{{ .name }}" {
		name          = "{{ .name }}"
		issuer_url    = "{{ .issuerURL }}"
		provider_type = "{{ .providerType }}"
	}`

	testData := map[string]string{
		"name":         configName,
		"issuerURL":    "https://tempurl.org",
		"providerType": "generic",
	}

	config := util.ExecuteTemplate(configName, temp, testData)

	updatedTemp := `
	resource "platform_oidc_configuration" "{{ .name }}" {
		name              = "{{ .name }}"
		description       = "Test Description"
		issuer_url        = "{{ .issuerURL }}"
		provider_type     = "{{ .providerType }}"
		audience          = "{{ .audience }}"
		use_default_proxy = true
	}`

	updatedTestData := map[string]string{
		"name":         configName,
		"issuerURL":    "https://token.actions.githubusercontent.com/jfrog",
		"providerType": "GitHub",
		"audience":     "test-audience-2",
	}
	updatedConfig := util.ExecuteTemplate(configName, updatedTemp, updatedTestData)

	var onOrAfterVersion71380 = func() (bool, error) {
		return acctest.CompareAcessVersions(t, "7.138.0")
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProviders(),
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(fqrn, "name", testData["name"]),
					resource.TestCheckResourceAttr(fqrn, "issuer_url", testData["issuerURL"]),
					resource.TestCheckResourceAttr(fqrn, "provider_type", testData["providerType"]),
				),
			},
			{
				SkipFunc: onOrAfterVersion71380,
				Config:   updatedConfig,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(fqrn, "name", updatedTestData["name"]),
					resource.TestCheckResourceAttr(fqrn, "description", "Test Description"),
					resource.TestCheckResourceAttr(fqrn, "issuer_url", updatedTestData["issuerURL"]),
					resource.TestCheckResourceAttr(fqrn, "provider_type", updatedTestData["providerType"]),
					resource.TestCheckResourceAttr(fqrn, "audience", updatedTestData["audience"]),
					resource.TestCheckResourceAttr(fqrn, "use_default_proxy", "true"),
				),
			},
			{
				ResourceName:                         fqrn,
				ImportState:                          true,
				ImportStateId:                        configName,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "name",
			},
		},
	})
}

func TestAccOIDCConfiguration_with_project(t *testing.T) {
	_, _, projectName := testutil.MkNames("test-project-", "project")
	projectKey := strings.ToLower(fmt.Sprintf("proj%d", testutil.RandomInt()))
	_, fqrn, configName := testutil.MkNames("test-oidc-configuration", "platform_oidc_configuration")

	temp := `
	resource "project" "{{ .projectName }}" {
		key = "{{ .projectKey }}"
		display_name = "{{ .projectName }}"
		description = "test description"
		admin_privileges {
			manage_members = true
			manage_resources = true
			index_resources = true
		}
		max_storage_in_gibibytes = 1
		block_deployments_on_limit = true
		email_notification = false
	}

	resource "platform_oidc_configuration" "{{ .name }}" {
		name          = "{{ .name }}"
		issuer_url    = "{{ .issuerURL }}"
		provider_type = "{{ .providerType }}"
		project_key   = project.{{ .projectName }}.key
	}`

	testData := map[string]string{
		"projectName":  projectName,
		"projectKey":   projectKey,
		"name":         configName,
		"issuerURL":    "https://tempurl.org",
		"providerType": "generic",
	}

	config := util.ExecuteTemplate(configName, temp, testData)

	resource.Test(t, resource.TestCase{
		PreCheck: func() { testAccPreCheck(t) },
		ExternalProviders: map[string]resource.ExternalProvider{
			"project": {
				Source: "jfrog/project",
			},
		},
		ProtoV6ProviderFactories: testAccProviders(),
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(fqrn, "name", testData["name"]),
					resource.TestCheckResourceAttr(fqrn, "issuer_url", testData["issuerURL"]),
					resource.TestCheckResourceAttr(fqrn, "provider_type", testData["providerType"]),
				),
			},
			{
				ResourceName:                         fqrn,
				ImportState:                          true,
				ImportStateId:                        fmt.Sprintf("%s:%s", configName, projectKey),
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "name",
			},
		},
	})
}

func TestAccOIDCConfiguration_invalid_name(t *testing.T) {
	for _, invalidName := range []string{"Test", "test!@", "1test"} {
		t.Run(invalidName, func(t *testing.T) {
			_, _, configName := testutil.MkNames("test-oidc-configuration", "platform_oidc_configuration")

			temp := `
			resource "platform_oidc_configuration" "{{ .resourceName }}" {
				name          = "{{ .name }}"
				description   = "Test description"
				issuer_url    = "{{ .issuerURL }}"
				provider_type = "{{ .providerType }}"
				audience      = "{{ .audience }}"
			}`

			testData := map[string]string{
				"resourceName": configName,
				"name":         invalidName,
				"issuerURL":    "https://tempurl.org",
				"providerType": "generic",
				"audience":     "test-audience",
			}

			config := util.ExecuteTemplate(configName, temp, testData)

			resource.Test(t, resource.TestCase{
				PreCheck:                 func() { testAccPreCheck(t) },
				ProtoV6ProviderFactories: testAccProviders(),
				Steps: []resource.TestStep{
					{
						Config:      config,
						ExpectError: regexp.MustCompile(`must start with a lowercase letter and only contain lowercase`),
					},
				},
			})
		})
	}
}

func TestAccOIDCConfiguration_invalid_issuer_url(t *testing.T) {
	_, _, configName := testutil.MkNames("test-oidc-configuration", "platform_oidc_configuration")

	temp := `
	resource "platform_oidc_configuration" "{{ .name }}" {
		name          = "{{ .name }}"
		description   = "Test description"
		issuer_url    = "{{ .issuerURL }}"
		provider_type = "{{ .providerType }}"
		audience      = "{{ .audience }}"
	}`

	testData := map[string]string{
		"name":         configName,
		"issuerURL":    "http://tempurl.org",
		"providerType": "generic",
		"audience":     "test-audience",
	}

	config := util.ExecuteTemplate(configName, temp, testData)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProviders(),
		Steps: []resource.TestStep{
			{
				Config:      config,
				ExpectError: regexp.MustCompile(`issuer_url must use https protocol`),
			},
		},
	})
}

func TestAccOIDCConfiguration_invalid_provider_type_issuer_url(t *testing.T) {
	_, _, configName := testutil.MkNames("test-oidc-configuration", "platform_oidc_configuration")

	temp := `
	resource "platform_oidc_configuration" "{{ .name }}" {
		name          = "{{ .name }}"
		description   = "Test description"
		issuer_url    = "{{ .issuerURL }}"
		provider_type = "{{ .providerType }}"
		audience      = "{{ .audience }}"
	}`

	testData := map[string]string{
		"name":         configName,
		"issuerURL":    "https://tempurl.org",
		"providerType": "GitHub",
		"audience":     "test-audience",
	}

	var onOrAfterVersion71380 = func() (bool, error) {
		return acctest.CompareAcessVersions(t, "7.138.0")
	}

	config := util.ExecuteTemplate(configName, temp, testData)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProviders(),
		Steps: []resource.TestStep{
			{
				SkipFunc:    onOrAfterVersion71380,
				Config:      config,
				ExpectError: regexp.MustCompile(`.*must start with https:\/\/token\.actions\.githubusercontent\.com[^\/].*`),
			},
		},
	})
}

func TestAccOIDCConfiguration_custom_provider_type_issuer_url_with_org(t *testing.T) {
	_, fqrn, configName := testutil.MkNames("test-oidc-configuration", "platform_oidc_configuration")

	temp := `
	resource "platform_oidc_configuration" "{{ .name }}" {
		name          = "{{ .name }}"
		description   = "Test description"
		issuer_url    = "{{ .issuerURL }}"
		provider_type = "{{ .providerType }}"
		audience      = "{{ .audience }}"
		organization      = "{{ .organization }}"
	}`

	testData := map[string]string{
		"name":         configName,
		"issuerURL":    "https://token.actions.githubusercontent.com/jfrog",
		"providerType": "GitHub",
		"audience":     "test-audience",
		"organization": "test-organization",
	}

	config := util.ExecuteTemplate(configName, temp, testData)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProviders(),
		Steps: []resource.TestStep{
			{
				Config: config,
				Check:  resource.TestCheckResourceAttr(fqrn, "organization", testData["organization"])},
		},
	})
}

func TestAccOIDCConfiguration_custom_provider_type_enable_premissive_configuration(t *testing.T) {
	_, fqrn, configName := testutil.MkNames("test-oidc-configuration", "platform_oidc_configuration")

	temp := `
	resource "platform_oidc_configuration" "{{ .name }}" {
		name          = "{{ .name }}"
		description   = "Test description"
		issuer_url    = "{{ .issuerURL }}"
		provider_type = "{{ .providerType }}"
		audience      = "{{ .audience }}"
		enable_permissive_configuration = true
	}`

	testData := map[string]string{
		"name":         configName,
		"issuerURL":    "https://token.actions.githubusercontent.com/jfrog",
		"providerType": "GitHub",
		"audience":     "test-audience",
	}

	config := util.ExecuteTemplate(configName, temp, testData)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProviders(),
		Steps: []resource.TestStep{
			{
				Config: config,
				Check:  resource.TestCheckResourceAttr(fqrn, "enable_permissive_configuration", "true")},
		},
	})
}

// enable_permissive_configuration = false is rejected at plan time for provider types
// where the platform does not enforce restrictive authentication.
func TestAccOIDCConfiguration_enable_permissive_configuration_false_rejected_for_non_github(t *testing.T) {
	for _, testCase := range []struct {
		providerType string
		issuerURL    string
		extraAttrs   string
	}{
		{providerType: "generic", issuerURL: "https://tempurl.org"},
		{providerType: "Azure", issuerURL: "https://sts.windows.net/your-tenant-id/"},
		{
			providerType: "GitHubEnterprise",
			issuerURL:    "https://token.actions.githubusercontent.com/jfrog",
			extraAttrs:   `organization = "test-org"`,
		},
	} {
		t.Run(testCase.providerType, func(t *testing.T) {
			_, _, configName := testutil.MkNames("test-oidc-permissive-false", "platform_oidc_configuration")

			temp := `
	resource "platform_oidc_configuration" "{{ .name }}" {
		name          = "{{ .name }}"
		description   = "Test description"
		issuer_url    = "{{ .issuerURL }}"
		provider_type = "{{ .providerType }}"
		audience      = "{{ .audience }}"
		{{ .extraAttrs }}
		enable_permissive_configuration = false
	}`

			testData := map[string]string{
				"name":         configName,
				"issuerURL":    testCase.issuerURL,
				"providerType": testCase.providerType,
				"audience":     "test-audience",
				"extraAttrs":   testCase.extraAttrs,
			}

			config := util.ExecuteTemplate(configName, temp, testData)

			resource.Test(t, resource.TestCase{
				PreCheck:                 func() { testAccPreCheck(t) },
				ProtoV6ProviderFactories: testAccProviders(),
				Steps: []resource.TestStep{
					{
						Config:      config,
						ExpectError: regexp.MustCompile(`(?s)enable_permissive_configuration = false is only applicable when provider_type[\s\n]+is set to 'GitHub'`),
					},
				},
			})
		})
	}
}

func TestAccOIDCConfiguration_github_enterprise(t *testing.T) {
	_, fqrn, configName := testutil.MkNames("test-oidc-ge-configuration", "platform_oidc_configuration")

	temp := `
resource "platform_oidc_configuration" "{{ .name }}" {
  name          = "{{ .name }}"
  issuer_url    = "{{ .issuerURL }}"
  provider_type = "{{ .providerType }}"
  organization  = "{{ .organization }}"
}`

	testData := map[string]string{
		"name":         configName,
		"issuerURL":    "https://token.actions.githubusercontent.com/jfrog",
		"providerType": "GitHubEnterprise",
		"organization": "test-org-ge",
	}

	config := util.ExecuteTemplate(configName, temp, testData)

	updatedTemp := `
resource "platform_oidc_configuration" "{{ .name }}" {
  name              = "{{ .name }}"
  description       = "GitHub Enterprise OIDC"
  issuer_url        = "{{ .issuerURL }}"
  provider_type     = "{{ .providerType }}"
  audience          = "{{ .audience }}"
  organization      = "{{ .organization }}"
  use_default_proxy = true
  enable_permissive_configuration = true
}`

	updatedTestData := map[string]string{
		"name":         configName,
		"issuerURL":    "https://token.actions.githubusercontent.com/jfrog",
		"providerType": "GitHubEnterprise",
		"audience":     "ge-audience",
		"organization": "test-org-ge",
	}
	updatedConfig := util.ExecuteTemplate(configName, updatedTemp, updatedTestData)

	var onOrAfterVersion71440 = func() (bool, error) {
		return acctest.CompareAcessVersions(t, "7.144.0")
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProviders(),
		Steps: []resource.TestStep{
			{
				SkipFunc: onOrAfterVersion71440,
				Config:   config,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(fqrn, "name", testData["name"]),
					resource.TestCheckResourceAttr(fqrn, "issuer_url", testData["issuerURL"]),
					resource.TestCheckResourceAttr(fqrn, "provider_type", testData["providerType"]),
					resource.TestCheckResourceAttr(fqrn, "organization", testData["organization"]),
				),
			},
			{
				SkipFunc: onOrAfterVersion71440,
				Config:   updatedConfig,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(fqrn, "name", updatedTestData["name"]),
					resource.TestCheckResourceAttr(fqrn, "description", "GitHub Enterprise OIDC"),
					resource.TestCheckResourceAttr(fqrn, "issuer_url", updatedTestData["issuerURL"]),
					resource.TestCheckResourceAttr(fqrn, "provider_type", updatedTestData["providerType"]),
					resource.TestCheckResourceAttr(fqrn, "audience", updatedTestData["audience"]),
					resource.TestCheckResourceAttr(fqrn, "organization", updatedTestData["organization"]),
					resource.TestCheckResourceAttr(fqrn, "use_default_proxy", "true"),
					resource.TestCheckResourceAttr(fqrn, "enable_permissive_configuration", "true"),
				),
			},
			{
				SkipFunc:                             onOrAfterVersion71440,
				ResourceName:                         fqrn,
				ImportState:                          true,
				ImportStateId:                        configName,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "name",
			},
		},
	})
}

func TestAccOIDCConfiguration_with_token_issuer(t *testing.T) {
	_, fqrn, configName := testutil.MkNames("test-oidc-token-issuer", "platform_oidc_configuration")

	temp := `
resource "platform_oidc_configuration" "{{ .name }}" {
  name          = "{{ .name }}"
  issuer_url    = "{{ .issuerURL }}"
  provider_type = "{{ .providerType }}"
  token_issuer  = "{{ .tokenIssuer }}"
}`

	testData := map[string]string{
		"name":         configName,
		"issuerURL":    "https://tempurl.org",
		"providerType": "generic",
		"tokenIssuer":  "https://tempurl.org",
	}

	config := util.ExecuteTemplate(configName, temp, testData)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProviders(),
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(fqrn, "name", testData["name"]),
					resource.TestCheckResourceAttr(fqrn, "provider_type", testData["providerType"]),
					resource.TestCheckResourceAttr(fqrn, "token_issuer", testData["tokenIssuer"]),
				),
			},
			{
				ResourceName:                         fqrn,
				ImportState:                          true,
				ImportStateId:                        configName,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "name",
			},
		},
	})
}

func TestAccOIDCConfiguration_token_issuer_invalid_for_github(t *testing.T) {
	_, _, configName := testutil.MkNames("test-oidc-token-issuer-invalid", "platform_oidc_configuration")

	temp := `
resource "platform_oidc_configuration" "{{ .name }}" {
  name          = "{{ .name }}"
  issuer_url    = "{{ .issuerURL }}"
  provider_type = "{{ .providerType }}"
  organization  = "{{ .organization }}"
  token_issuer  = "{{ .tokenIssuer }}"
}`

	for _, providerType := range []string{"GitHub", "GitHubEnterprise"} {
		issuerURL := "https://token.actions.githubusercontent.com"
		if providerType == "GitHubEnterprise" {
			issuerURL = "https://token.actions.githubusercontent.com/jfrog"
		}
		testData := map[string]string{
			"name":         configName,
			"issuerURL":    issuerURL,
			"providerType": providerType,
			"organization": "test-org",
			"tokenIssuer":  "https://token.actions.githubusercontent.com/test-org",
		}
		config := util.ExecuteTemplate(configName, temp, testData)

		resource.Test(t, resource.TestCase{
			PreCheck:                 func() { testAccPreCheck(t) },
			ProtoV6ProviderFactories: testAccProviders(),
			Steps: []resource.TestStep{
				{
					Config:      config,
					ExpectError: regexp.MustCompile(`token_issuer is not allowed when provider_type is set to`),
				},
			},
		})
	}
}

func TestAccOIDCConfiguration_organization_invalid_for_non_github(t *testing.T) {
	_, _, configName := testutil.MkNames("test-oidc-organization-invalid", "platform_oidc_configuration")

	temp := `
resource "platform_oidc_configuration" "{{ .name }}" {
  name          = "{{ .name }}"
  issuer_url    = "{{ .issuerURL }}"
  provider_type = "{{ .providerType }}"
  organization  = "{{ .organization }}"
}`

	for _, providerType := range []string{"generic", "Azure"} {
		testData := map[string]string{
			"name":         configName,
			"issuerURL":    "https://tempurl.org/oidc",
			"providerType": providerType,
			"organization": "test-org",
		}
		config := util.ExecuteTemplate(configName, temp, testData)

		resource.Test(t, resource.TestCase{
			PreCheck:                 func() { testAccPreCheck(t) },
			ProtoV6ProviderFactories: testAccProviders(),
			Steps: []resource.TestStep{
				{
					Config:      config,
					ExpectError: regexp.MustCompile(`organization is only applicable when provider_type is set to`),
				},
			},
		})
	}
}

func TestAccOIDCConfiguration_azure(t *testing.T) {
	_, fqrn, configName := testutil.MkNames("test-oidc-azure-configuration", "platform_oidc_configuration")

	temp := `
resource "platform_oidc_configuration" "{{ .name }}" {
  name          = "{{ .name }}"
  issuer_url    = "{{ .issuerURL }}"
  provider_type = "{{ .providerType }}"
}`

	testData := map[string]string{
		"name":         configName,
		"issuerURL":    "https://sts.windows.net/your-tenant-id/",
		"providerType": "Azure",
	}

	config := util.ExecuteTemplate(configName, temp, testData)

	updatedTemp := `
resource "platform_oidc_configuration" "{{ .name }}" {
  name              = "{{ .name }}"
  description       = "Azure OIDC"
  issuer_url        = "{{ .issuerURL }}"
  provider_type     = "{{ .providerType }}"
  audience          = "{{ .audience }}"
  azure_app_id      = "{{ .azureAppId }}"
  use_default_proxy = true
}`

	updatedTestData := map[string]string{
		"name":         configName,
		"issuerURL":    "https://sts.windows.net/your-tenant-id/",
		"providerType": "Azure",
		"audience":     "azure-audience",
		"azureAppId":   "test-azure-app-id",
	}
	updatedConfig := util.ExecuteTemplate(configName, updatedTemp, updatedTestData)

	var onOrAfterVersion7731 = func() (bool, error) {
		skiptest, err := acctest.CompareArtifactoryVersions(t, "7.73.1")
		return !skiptest, err
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProviders(),
		Steps: []resource.TestStep{
			{
				SkipFunc: onOrAfterVersion7731,
				Config:   config,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(fqrn, "name", testData["name"]),
					resource.TestCheckResourceAttr(fqrn, "issuer_url", testData["issuerURL"]),
					resource.TestCheckResourceAttr(fqrn, "provider_type", testData["providerType"]),
				),
			},
			{
				SkipFunc: onOrAfterVersion7731,
				Config:   updatedConfig,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(fqrn, "name", updatedTestData["name"]),
					resource.TestCheckResourceAttr(fqrn, "description", "Azure OIDC"),
					resource.TestCheckResourceAttr(fqrn, "issuer_url", updatedTestData["issuerURL"]),
					resource.TestCheckResourceAttr(fqrn, "provider_type", updatedTestData["providerType"]),
					resource.TestCheckResourceAttr(fqrn, "audience", updatedTestData["audience"]),
					resource.TestCheckResourceAttr(fqrn, "azure_app_id", updatedTestData["azureAppId"]),
					resource.TestCheckResourceAttr(fqrn, "use_default_proxy", "true"),
				),
			},
			{
				SkipFunc:                             onOrAfterVersion7731,
				ResourceName:                         fqrn,
				ImportState:                          true,
				ImportStateId:                        configName,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "name",
			},
		},
	})
}

// TestOIDCConfigurationValidateConfig_enable_permissive_configuration exercises
// ValidateConfig directly for the plan-time rejection of enable_permissive_configuration
// = false on provider types where the platform does not enforce it.
func TestOIDCConfigurationValidateConfig_enable_permissive_configuration(t *testing.T) {
	ctx := context.Background()

	const (
		gitHubProviderType   = "GitHub"
		githubEnterpriseType = "GitHubEnterprise"
		azureProviderType    = "Azure"
		gitHubProviderURL    = "https://token.actions.githubusercontent.com"
	)

	r, ok := platform.NewOIDCConfigurationResource().(interface {
		fwresource.ResourceWithConfigure
		fwresource.ResourceWithValidateConfig
	})
	if !ok {
		t.Fatal("expected the OIDC configuration resource to implement Configure and ValidateConfig")
	}

	configureResp := fwresource.ConfigureResponse{}
	r.Configure(ctx, fwresource.ConfigureRequest{
		ProviderData: util.ProviderMetadata{AccessVersion: "7.176.15"},
	}, &configureResp)
	if configureResp.Diagnostics.HasError() {
		t.Fatalf("expected the resource to configure cleanly, got: %v", configureResp.Diagnostics.Errors())
	}

	schemaResp := fwresource.SchemaResponse{}
	r.Schema(ctx, fwresource.SchemaRequest{}, &schemaResp)

	objectType, ok := schemaResp.Schema.Type().TerraformType(ctx).(tftypes.Object)
	if !ok {
		t.Fatalf("expected schema to be an object type, got %T", schemaResp.Schema.Type().TerraformType(ctx))
	}

	testCases := []struct {
		name                          string
		providerType                  interface{}
		enablePermissiveConfiguration interface{}
		expectError                   bool
		expectedDetailSubstrings      []string
	}{
		{
			name:                          "GitHub honours false so it is accepted",
			providerType:                  gitHubProviderType,
			enablePermissiveConfiguration: false,
		},
		{
			name:                          "GitHub honours true so it is accepted",
			providerType:                  gitHubProviderType,
			enablePermissiveConfiguration: true,
		},
		{
			name:                          "GitHubEnterprise false is rejected",
			providerType:                  githubEnterpriseType,
			enablePermissiveConfiguration: false,
			expectError:                   true,
			expectedDetailSubstrings: []string{
				"enable_permissive_configuration = false is only applicable when provider_type is set to 'GitHub'",
				"does not enforce restrictive (non-permissive) authentication for provider_type 'GitHubEnterprise'",
				"permissive authentication remains enabled regardless of this setting",
				"set provider_type to 'GitHub' if you need to restrict authentication",
			},
		},
		{
			name:                          "GitHubEnterprise true is accepted silently",
			providerType:                  githubEnterpriseType,
			enablePermissiveConfiguration: true,
		},
		{
			name:                          "generic false is rejected",
			providerType:                  "generic",
			enablePermissiveConfiguration: false,
			expectError:                   true,
			expectedDetailSubstrings: []string{
				"enable_permissive_configuration = false is only applicable when provider_type is set to 'GitHub'",
				"does not enforce restrictive (non-permissive) authentication for provider_type 'generic'",
			},
		},
		{
			name:                          "generic true is accepted silently",
			providerType:                  "generic",
			enablePermissiveConfiguration: true,
		},
		{
			name:                          "Azure false is rejected",
			providerType:                  azureProviderType,
			enablePermissiveConfiguration: false,
			expectError:                   true,
			expectedDetailSubstrings: []string{
				"enable_permissive_configuration = false is only applicable when provider_type is set to 'GitHub'",
				"does not enforce restrictive (non-permissive) authentication for provider_type 'Azure'",
			},
		},
		{
			name:                          "Azure true is accepted silently",
			providerType:                  azureProviderType,
			enablePermissiveConfiguration: true,
		},
		{
			name:                          "an unknown provider_type defers validation instead of failing the plan",
			providerType:                  tftypes.UnknownValue,
			enablePermissiveConfiguration: false,
		},
		{
			name:                          "an unknown value defers validation instead of failing the plan",
			providerType:                  "generic",
			enablePermissiveConfiguration: tftypes.UnknownValue,
		},
		{
			name:                          "an unset attribute is never rejected",
			providerType:                  "generic",
			enablePermissiveConfiguration: nil,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			attributeValues := map[string]tftypes.Value{}
			for attributeName, attributeType := range objectType.AttributeTypes {
				attributeValues[attributeName] = tftypes.NewValue(attributeType, nil)
			}

			issuerURL := "https://tempurl.org/oidc"
			if testCase.providerType == gitHubProviderType || testCase.providerType == githubEnterpriseType {
				issuerURL = gitHubProviderURL
				attributeValues["organization"] = tftypes.NewValue(tftypes.String, "test-org")
			}

			attributeValues["name"] = tftypes.NewValue(tftypes.String, "test-oidc-configuration")
			attributeValues["issuer_url"] = tftypes.NewValue(tftypes.String, issuerURL)
			attributeValues["provider_type"] = tftypes.NewValue(tftypes.String, testCase.providerType)
			attributeValues["use_default_proxy"] = tftypes.NewValue(tftypes.Bool, false)
			attributeValues["enable_permissive_configuration"] = tftypes.NewValue(tftypes.Bool, testCase.enablePermissiveConfiguration)

			req := fwresource.ValidateConfigRequest{
				Config: tfsdk.Config{
					Raw:    tftypes.NewValue(objectType, attributeValues),
					Schema: schemaResp.Schema,
				},
			}
			resp := fwresource.ValidateConfigResponse{}

			r.ValidateConfig(ctx, req, &resp)

			errors := resp.Diagnostics.Errors()

			if !testCase.expectError {
				if len(errors) != 0 {
					t.Fatalf("expected no errors, got: %v", errors)
				}
				if len(resp.Diagnostics.Warnings()) != 0 {
					t.Fatalf("expected no warnings, got: %v", resp.Diagnostics.Warnings())
				}
				return
			}

			if len(errors) != 1 {
				t.Fatalf("expected 1 error, got %d: %v", len(errors), errors)
			}

			if errors[0].Summary() != "Invalid Attribute Configuration" {
				t.Errorf("unexpected error summary: %q", errors[0].Summary())
			}

			for _, substring := range testCase.expectedDetailSubstrings {
				if !strings.Contains(errors[0].Detail(), substring) {
					t.Errorf("expected error detail to contain %q, got: %s", substring, errors[0].Detail())
				}
			}
		})
	}
}

// TestOIDCConfigurationValidateConfig_organization covers the plan-time rejection of
// `organization` for provider types that discard it. It exercises ValidateConfig directly
// so that an unknown `provider_type`, which cannot be expressed in the static
// configuration of an acceptance test, is covered alongside the concrete provider types.
func TestOIDCConfigurationValidateConfig_organization(t *testing.T) {
	ctx := context.Background()

	const (
		gitHubProviderType   = "GitHub"
		githubEnterpriseType = "GitHubEnterprise"
		azureProviderType    = "Azure"
		gitHubProviderURL    = "https://token.actions.githubusercontent.com"
	)

	r, ok := platform.NewOIDCConfigurationResource().(interface {
		fwresource.ResourceWithConfigure
		fwresource.ResourceWithValidateConfig
	})
	if !ok {
		t.Fatal("expected the OIDC configuration resource to implement Configure and ValidateConfig")
	}

	configureResp := fwresource.ConfigureResponse{}
	r.Configure(ctx, fwresource.ConfigureRequest{
		ProviderData: util.ProviderMetadata{AccessVersion: "7.176.15"},
	}, &configureResp)
	if configureResp.Diagnostics.HasError() {
		t.Fatalf("expected the resource to configure cleanly, got: %v", configureResp.Diagnostics.Errors())
	}

	schemaResp := fwresource.SchemaResponse{}
	r.Schema(ctx, fwresource.SchemaRequest{}, &schemaResp)

	objectType, ok := schemaResp.Schema.Type().TerraformType(ctx).(tftypes.Object)
	if !ok {
		t.Fatalf("expected schema to be an object type, got %T", schemaResp.Schema.Type().TerraformType(ctx))
	}

	testCases := []struct {
		name                     string
		providerType             interface{}
		organization             interface{}
		expectError              bool
		expectedDetailSubstrings []string
	}{
		{
			name:         "GitHub stores the organization so it is accepted",
			providerType: gitHubProviderType,
			organization: "test-org",
		},
		{
			name:         "GitHubEnterprise stores the organization so it is accepted",
			providerType: githubEnterpriseType,
			organization: "test-org",
		},
		{
			name:         "generic discards the organization so it is rejected",
			providerType: "generic",
			organization: "test-org",
			expectError:  true,
			expectedDetailSubstrings: []string{
				"organization is only applicable when provider_type is set to 'GitHub' or 'GitHubEnterprise'",
				"discards it for provider_type 'generic'",
				"Remove the attribute from your configuration.",
			},
		},
		{
			name:         "Azure discards the organization so it is rejected",
			providerType: azureProviderType,
			organization: "test-org",
			expectError:  true,
			expectedDetailSubstrings: []string{
				"organization is only applicable when provider_type is set to 'GitHub' or 'GitHubEnterprise'",
				"discards it for provider_type 'Azure'",
			},
		},
		{
			name:         "generic without an organization is unaffected",
			providerType: "generic",
		},
		{
			name:         "Azure without an organization is unaffected",
			providerType: azureProviderType,
		},
		{
			name:         "an unknown provider_type defers validation instead of failing the plan",
			providerType: tftypes.UnknownValue,
			organization: "test-org",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			attributeValues := map[string]tftypes.Value{}
			for attributeName, attributeType := range objectType.AttributeTypes {
				attributeValues[attributeName] = tftypes.NewValue(attributeType, nil)
			}

			issuerURL := "https://tempurl.org/oidc"
			if testCase.providerType == gitHubProviderType || testCase.providerType == githubEnterpriseType {
				issuerURL = gitHubProviderURL
			}

			attributeValues["name"] = tftypes.NewValue(tftypes.String, "test-oidc-configuration")
			attributeValues["issuer_url"] = tftypes.NewValue(tftypes.String, issuerURL)
			attributeValues["provider_type"] = tftypes.NewValue(tftypes.String, testCase.providerType)
			attributeValues["organization"] = tftypes.NewValue(tftypes.String, testCase.organization)
			attributeValues["use_default_proxy"] = tftypes.NewValue(tftypes.Bool, false)

			req := fwresource.ValidateConfigRequest{
				Config: tfsdk.Config{
					Raw:    tftypes.NewValue(objectType, attributeValues),
					Schema: schemaResp.Schema,
				},
			}
			resp := fwresource.ValidateConfigResponse{}

			r.ValidateConfig(ctx, req, &resp)

			errors := resp.Diagnostics.Errors()

			if !testCase.expectError {
				if len(errors) != 0 {
					t.Fatalf("expected no errors, got: %v", errors)
				}
				return
			}

			if len(errors) != 1 {
				t.Fatalf("expected 1 error, got %d: %v", len(errors), errors)
			}

			if errors[0].Summary() != "Invalid Attribute Configuration" {
				t.Errorf("unexpected error summary: %q", errors[0].Summary())
			}

			for _, substring := range testCase.expectedDetailSubstrings {
				if !strings.Contains(errors[0].Detail(), substring) {
					t.Errorf("expected error detail to contain %q, got: %s", substring, errors[0].Detail())
				}
			}
		})
	}
}

// TestOIDCConfigurationValidateConfig_azure_app_id covers the plan-time rejection of
// `azure_app_id` for provider types other than `Azure`. It exercises ValidateConfig
// directly so that an unknown `provider_type`, which cannot be expressed in the static
// configuration of an acceptance test, is covered alongside the concrete provider types.
func TestOIDCConfigurationValidateConfig_azure_app_id(t *testing.T) {
	ctx := context.Background()

	const (
		gitHubProviderType   = "GitHub"
		githubEnterpriseType = "GitHubEnterprise"
		azureProviderType    = "Azure"
		gitHubProviderURL    = "https://token.actions.githubusercontent.com"
	)

	r, ok := platform.NewOIDCConfigurationResource().(interface {
		fwresource.ResourceWithConfigure
		fwresource.ResourceWithValidateConfig
	})
	if !ok {
		t.Fatal("expected the OIDC configuration resource to implement Configure and ValidateConfig")
	}

	configureResp := fwresource.ConfigureResponse{}
	r.Configure(ctx, fwresource.ConfigureRequest{
		ProviderData: util.ProviderMetadata{AccessVersion: "7.176.15"},
	}, &configureResp)
	if configureResp.Diagnostics.HasError() {
		t.Fatalf("expected the resource to configure cleanly, got: %v", configureResp.Diagnostics.Errors())
	}

	schemaResp := fwresource.SchemaResponse{}
	r.Schema(ctx, fwresource.SchemaRequest{}, &schemaResp)

	objectType, ok := schemaResp.Schema.Type().TerraformType(ctx).(tftypes.Object)
	if !ok {
		t.Fatalf("expected schema to be an object type, got %T", schemaResp.Schema.Type().TerraformType(ctx))
	}

	testCases := []struct {
		name         string
		providerType interface{}
		azureAppId   interface{}
		expectError  bool
	}{
		{
			name:         "Azure accepts the azure_app_id",
			providerType: azureProviderType,
			azureAppId:   "00000000-0000-0000-0000-000000000001",
		},
		{
			name:         "generic rejects the azure_app_id",
			providerType: "generic",
			azureAppId:   "00000000-0000-0000-0000-000000000001",
			expectError:  true,
		},
		{
			name:         "GitHub rejects the azure_app_id",
			providerType: gitHubProviderType,
			azureAppId:   "00000000-0000-0000-0000-000000000001",
			expectError:  true,
		},
		{
			name:         "GitHubEnterprise rejects the azure_app_id",
			providerType: githubEnterpriseType,
			azureAppId:   "00000000-0000-0000-0000-000000000001",
			expectError:  true,
		},
		{
			name:         "an unknown provider_type defers validation instead of failing the plan",
			providerType: tftypes.UnknownValue,
			azureAppId:   "00000000-0000-0000-0000-000000000001",
		},
		{
			name:         "generic without an azure_app_id is unaffected",
			providerType: "generic",
		},
		{
			name:         "an unknown provider_type without an azure_app_id is unaffected",
			providerType: tftypes.UnknownValue,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			attributeValues := map[string]tftypes.Value{}
			for attributeName, attributeType := range objectType.AttributeTypes {
				attributeValues[attributeName] = tftypes.NewValue(attributeType, nil)
			}

			issuerURL := "https://tempurl.org/oidc"
			if testCase.providerType == gitHubProviderType || testCase.providerType == githubEnterpriseType {
				issuerURL = gitHubProviderURL
				attributeValues["organization"] = tftypes.NewValue(tftypes.String, "test-org")
			}

			attributeValues["name"] = tftypes.NewValue(tftypes.String, "test-oidc-configuration")
			attributeValues["issuer_url"] = tftypes.NewValue(tftypes.String, issuerURL)
			attributeValues["provider_type"] = tftypes.NewValue(tftypes.String, testCase.providerType)
			attributeValues["azure_app_id"] = tftypes.NewValue(tftypes.String, testCase.azureAppId)
			attributeValues["use_default_proxy"] = tftypes.NewValue(tftypes.Bool, false)

			req := fwresource.ValidateConfigRequest{
				Config: tfsdk.Config{
					Raw:    tftypes.NewValue(objectType, attributeValues),
					Schema: schemaResp.Schema,
				},
			}
			resp := fwresource.ValidateConfigResponse{}

			r.ValidateConfig(ctx, req, &resp)

			errors := resp.Diagnostics.Errors()

			if !testCase.expectError {
				if len(errors) != 0 {
					t.Fatalf("expected no errors, got: %v", errors)
				}
				return
			}

			if len(errors) != 1 {
				t.Fatalf("expected 1 error, got %d: %v", len(errors), errors)
			}

			if errors[0].Summary() != "Invalid Attribute Configuration" {
				t.Errorf("unexpected error summary: %q", errors[0].Summary())
			}

			expectedDetail := "azure_app_id is only applicable when provider_type is set to 'Azure'."
			if errors[0].Detail() != expectedDetail {
				t.Errorf("expected error detail %q, got: %s", expectedDetail, errors[0].Detail())
			}
		})
	}
}

// TestOIDCConfigurationValidateConfig_token_issuer covers the plan-time rejection of
// `token_issuer` for the GitHub provider types. It exercises ValidateConfig directly so
// that an unknown `provider_type`, which cannot be expressed in the static configuration
// of an acceptance test, is covered alongside the concrete provider types.
func TestOIDCConfigurationValidateConfig_token_issuer(t *testing.T) {
	ctx := context.Background()

	const (
		gitHubProviderType   = "GitHub"
		githubEnterpriseType = "GitHubEnterprise"
		azureProviderType    = "Azure"
		gitHubProviderURL    = "https://token.actions.githubusercontent.com"
	)

	r, ok := platform.NewOIDCConfigurationResource().(interface {
		fwresource.ResourceWithConfigure
		fwresource.ResourceWithValidateConfig
	})
	if !ok {
		t.Fatal("expected the OIDC configuration resource to implement Configure and ValidateConfig")
	}

	configureResp := fwresource.ConfigureResponse{}
	r.Configure(ctx, fwresource.ConfigureRequest{
		ProviderData: util.ProviderMetadata{AccessVersion: "7.176.15"},
	}, &configureResp)
	if configureResp.Diagnostics.HasError() {
		t.Fatalf("expected the resource to configure cleanly, got: %v", configureResp.Diagnostics.Errors())
	}

	schemaResp := fwresource.SchemaResponse{}
	r.Schema(ctx, fwresource.SchemaRequest{}, &schemaResp)

	objectType, ok := schemaResp.Schema.Type().TerraformType(ctx).(tftypes.Object)
	if !ok {
		t.Fatalf("expected schema to be an object type, got %T", schemaResp.Schema.Type().TerraformType(ctx))
	}

	testCases := []struct {
		name         string
		providerType interface{}
		tokenIssuer  interface{}
		expectError  bool
	}{
		{
			name:         "generic accepts the token_issuer",
			providerType: "generic",
			tokenIssuer:  "https://tempurl.org/token",
		},
		{
			name:         "Azure accepts the token_issuer",
			providerType: azureProviderType,
			tokenIssuer:  "https://tempurl.org/token",
		},
		{
			name:         "GitHub rejects the token_issuer",
			providerType: gitHubProviderType,
			tokenIssuer:  "https://tempurl.org/token",
			expectError:  true,
		},
		{
			name:         "GitHubEnterprise rejects the token_issuer",
			providerType: githubEnterpriseType,
			tokenIssuer:  "https://tempurl.org/token",
			expectError:  true,
		},
		{
			name:         "an unknown provider_type defers validation instead of failing the plan",
			providerType: tftypes.UnknownValue,
			tokenIssuer:  "https://tempurl.org/token",
		},
		{
			name:         "an unknown provider_type without a token_issuer is unaffected",
			providerType: tftypes.UnknownValue,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			attributeValues := map[string]tftypes.Value{}
			for attributeName, attributeType := range objectType.AttributeTypes {
				attributeValues[attributeName] = tftypes.NewValue(attributeType, nil)
			}

			issuerURL := "https://tempurl.org/oidc"
			if testCase.providerType == gitHubProviderType || testCase.providerType == githubEnterpriseType {
				issuerURL = gitHubProviderURL
				attributeValues["organization"] = tftypes.NewValue(tftypes.String, "test-org")
			}

			attributeValues["name"] = tftypes.NewValue(tftypes.String, "test-oidc-configuration")
			attributeValues["issuer_url"] = tftypes.NewValue(tftypes.String, issuerURL)
			attributeValues["provider_type"] = tftypes.NewValue(tftypes.String, testCase.providerType)
			attributeValues["token_issuer"] = tftypes.NewValue(tftypes.String, testCase.tokenIssuer)
			attributeValues["use_default_proxy"] = tftypes.NewValue(tftypes.Bool, false)

			req := fwresource.ValidateConfigRequest{
				Config: tfsdk.Config{
					Raw:    tftypes.NewValue(objectType, attributeValues),
					Schema: schemaResp.Schema,
				},
			}
			resp := fwresource.ValidateConfigResponse{}

			r.ValidateConfig(ctx, req, &resp)

			errors := resp.Diagnostics.Errors()

			if !testCase.expectError {
				if len(errors) != 0 {
					t.Fatalf("expected no errors, got: %v", errors)
				}
				return
			}

			if len(errors) != 1 {
				t.Fatalf("expected 1 error, got %d: %v", len(errors), errors)
			}

			if errors[0].Summary() != "Invalid Attribute Configuration" {
				t.Errorf("unexpected error summary: %q", errors[0].Summary())
			}

			expectedDetail := "token_issuer is not allowed when provider_type is set to 'GitHub' or 'GitHubEnterprise'."
			if errors[0].Detail() != expectedDetail {
				t.Errorf("expected error detail %q, got: %s", expectedDetail, errors[0].Detail())
			}
		})
	}
}
