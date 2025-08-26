package platform_test

import (
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/jfrog/terraform-provider-artifactory/v12/pkg/acctest"
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
  use_default_proxy = true
}`

	updatedTestData := map[string]string{
		"name":         configName,
		"issuerURL":    "https://sts.windows.net/your-tenant-id/",
		"providerType": "Azure",
		"audience":     "azure-audience",
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
