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

// enable_permissive_configuration is not applicable to non-GitHub provider types, but
// setting it must not block the plan — provider versions <= 2.2.10 accepted it, so
// erroring out would lock existing workspaces on upgrade. It is ignored instead.
func TestAccOIDCConfiguration_enable_permissive_configuration_ignored_for_non_github(t *testing.T) {
	for _, testCase := range []struct {
		providerType string
		issuerURL    string
	}{
		{providerType: "generic", issuerURL: "https://tempurl.org"},
		{providerType: "Azure", issuerURL: "https://sts.windows.net/your-tenant-id/"},
	} {
		t.Run(testCase.providerType, func(t *testing.T) {
			_, fqrn, configName := testutil.MkNames("test-oidc-permissive-ignored", "platform_oidc_configuration")

			temp := `
	resource "platform_oidc_configuration" "{{ .name }}" {
		name          = "{{ .name }}"
		description   = "Test description"
		issuer_url    = "{{ .issuerURL }}"
		provider_type = "{{ .providerType }}"
		audience      = "{{ .audience }}"
		enable_permissive_configuration = false
	}`

			testData := map[string]string{
				"name":         configName,
				"issuerURL":    testCase.issuerURL,
				"providerType": testCase.providerType,
				"audience":     "test-audience",
			}

			config := util.ExecuteTemplate(configName, temp, testData)

			resource.Test(t, resource.TestCase{
				PreCheck:                 func() { testAccPreCheck(t) },
				ProtoV6ProviderFactories: testAccProviders(),
				Steps: []resource.TestStep{
					{
						Config: config,
						Check: resource.ComposeTestCheckFunc(
							resource.TestCheckResourceAttr(fqrn, "provider_type", testCase.providerType),
							resource.TestCheckResourceAttr(fqrn, "enable_permissive_configuration", "false"),
						),
					},
					{
						Config:   config,
						PlanOnly: true,
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
// ValidateConfig directly because the plan-time diagnostic under test is a warning, and
// the acceptance test harness (terraform-plugin-testing v1.15.0) can only match errors.
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
		providerType                  string
		enablePermissiveConfiguration interface{}
		expectWarning                 bool
		expectedDetailSubstrings      []string
	}{
		{
			name:                          "GitHub honours the attribute so false is not warned about",
			providerType:                  gitHubProviderType,
			enablePermissiveConfiguration: false,
		},
		{
			name:                          "GitHub honours the attribute so true is not warned about",
			providerType:                  gitHubProviderType,
			enablePermissiveConfiguration: true,
		},
		{
			name:                          "GitHubEnterprise false warns that authentication is not restricted",
			providerType:                  githubEnterpriseType,
			enablePermissiveConfiguration: false,
			expectWarning:                 true,
			expectedDetailSubstrings: []string{
				"Security impact",
				"does NOT restrict authentication when provider_type is 'GitHubEnterprise'",
				"only enforces this attribute for provider_type 'GitHub'",
				"It is sent to the JFrog platform, which ignores it.",
				"keeps reporting enable_permissive_configuration as true",
				"Terraform state records false and does not reflect the platform",
			},
		},
		{
			name:                          "GitHubEnterprise true warns that the value is not enforced",
			providerType:                  githubEnterpriseType,
			enablePermissiveConfiguration: true,
			expectWarning:                 true,
			expectedDetailSubstrings: []string{
				"only enforced when provider_type is 'GitHub'",
				"It is sent to the JFrog platform, which ignores it.",
				"regardless of the configured value",
			},
		},
		{
			name:                          "generic false warns that authentication is not restricted",
			providerType:                  "generic",
			enablePermissiveConfiguration: false,
			expectWarning:                 true,
			expectedDetailSubstrings: []string{
				"Security impact",
				"does NOT restrict authentication when provider_type is 'generic'",
				"It is not sent to the JFrog platform for provider_type 'generic'.",
			},
		},
		{
			name:                          "generic true warns that the value is not enforced",
			providerType:                  "generic",
			enablePermissiveConfiguration: true,
			expectWarning:                 true,
			expectedDetailSubstrings: []string{
				"only enforced when provider_type is 'GitHub'",
				"It is not sent to the JFrog platform for provider_type 'generic'.",
			},
		},
		{
			name:                          "Azure false warns that authentication is not restricted",
			providerType:                  azureProviderType,
			enablePermissiveConfiguration: false,
			expectWarning:                 true,
			expectedDetailSubstrings: []string{
				"Security impact",
				"does NOT restrict authentication when provider_type is 'Azure'",
				"It is not sent to the JFrog platform for provider_type 'Azure'.",
			},
		},
		{
			name:                          "Azure true warns that the value is not enforced",
			providerType:                  azureProviderType,
			enablePermissiveConfiguration: true,
			expectWarning:                 true,
			expectedDetailSubstrings: []string{
				"only enforced when provider_type is 'GitHub'",
				"It is not sent to the JFrog platform for provider_type 'Azure'.",
			},
		},
		{
			name:                          "an unknown value still warns, without claiming which value was configured",
			providerType:                  "generic",
			enablePermissiveConfiguration: tftypes.UnknownValue,
			expectWarning:                 true,
			expectedDetailSubstrings: []string{
				"only enforced when provider_type is 'GitHub'",
				"regardless of the configured value",
			},
		},
		{
			name:                          "an unset attribute is never warned about",
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

			if resp.Diagnostics.HasError() {
				t.Fatalf("expected no errors, got: %v", resp.Diagnostics.Errors())
			}

			warnings := resp.Diagnostics.Warnings()

			if !testCase.expectWarning {
				if len(warnings) != 0 {
					t.Fatalf("expected no warnings, got: %v", warnings)
				}
				return
			}

			if len(warnings) != 1 {
				t.Fatalf("expected 1 warning, got %d: %v", len(warnings), warnings)
			}

			if warnings[0].Summary() != "Unenforced Attribute Configuration" {
				t.Errorf("unexpected warning summary: %q", warnings[0].Summary())
			}

			for _, substring := range testCase.expectedDetailSubstrings {
				if !strings.Contains(warnings[0].Detail(), substring) {
					t.Errorf("expected warning detail to contain %q, got: %s", substring, warnings[0].Detail())
				}
			}
		})
	}
}
