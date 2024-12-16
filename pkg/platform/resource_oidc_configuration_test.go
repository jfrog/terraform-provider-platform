package platform_test

import (
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
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
				Config: updatedConfig,
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

	config := util.ExecuteTemplate(configName, temp, testData)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProviders(),
		Steps: []resource.TestStep{
			{
				Config:      config,
				ExpectError: regexp.MustCompile(`.*must start with https:\/\/token\.actions\.githubusercontent\.com[^\/].*`),
			},
		},
	})
}

func TestAccOIDCConfiguration_custom_provider_type_issuer_url(t *testing.T) {
	_, fqrn, configName := testutil.MkNames("test-oidc-configuration", "platform_oidc_configuration")

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
				Check:  resource.TestCheckResourceAttr(fqrn, "issuer_url", testData["issuerURL"])},
		},
	})
}
