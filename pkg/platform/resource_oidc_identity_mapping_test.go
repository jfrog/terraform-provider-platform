package platform_test

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/jfrog/terraform-provider-shared/testutil"
)

func TestAccOIDIdentityMapping_full(t *testing.T) {
	_, _, configName := testutil.MkNames("test-oidc-configuration", "platform_oidc_configuration")
	_, fqrn, identityMappingName := testutil.MkNames("test-oidc-identity-mapping", "platform_oidc_identity_mapping")

	temp := `
	resource "platform_oidc_configuration" "{{ .configName }}" {
		name          = "{{ .configName }}"
		description   = "Test description"
		issuer_url    = "{{ .issuerURL }}"
		provider_type = "{{ .providerType }}"
		audience      = "{{ .audience }}"
	}

	resource "platform_oidc_identity_mapping" "{{ .identityMappingName }}" {
		name          = "{{ .identityMappingName }}"
		description   = "Test description"
		provider_name = platform_oidc_configuration.{{ .configName }}.name
		priority      = {{ .priority }}
		claims_json   = jsonencode({
			sub = "{{ .sub }}",
			updated_at = 1490198843
		})
		token_spec = {
			username   = "{{ .username }}"
			scope      = "applied-permissions/user"
			audience   = "*@*"
			expires_in = 120
		}
	}`

	testData := map[string]string{
		"configName":          configName,
		"identityMappingName": identityMappingName,
		"issuerURL":           "https://tempurl.org",
		"providerType":        "generic",
		"audience":            "test-audience",
		"priority":            fmt.Sprintf("%d", testutil.RandomInt()),
		"sub":                 fmt.Sprintf("test-subscriber-%d", testutil.RandomInt()),
		"username":            fmt.Sprintf("test-user-%d", testutil.RandomInt()),
	}

	config := testutil.ExecuteTemplate(identityMappingName, temp, testData)

	updatedTestData := map[string]string{
		"configName":          configName,
		"identityMappingName": identityMappingName,
		"issuerURL":           "https://tempurl.org",
		"providerType":        "generic",
		"audience":            "test-audience",
		"priority":            fmt.Sprintf("%d", testutil.RandomInt()),
		"sub":                 fmt.Sprintf("test-subscriber-%d", testutil.RandomInt()),
		"username":            fmt.Sprintf("test-user-%d", testutil.RandomInt()),
	}

	updatedConfig := testutil.ExecuteTemplate(identityMappingName, temp, updatedTestData)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProviders(),
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(fqrn, "name", testData["identityMappingName"]),
					resource.TestCheckResourceAttr(fqrn, "priority", testData["priority"]),
					resource.TestCheckResourceAttr(fqrn, "claims_json", fmt.Sprintf("{\"sub\":\"%s\",\"updated_at\":1490198843}", testData["sub"])),
					resource.TestCheckResourceAttr(fqrn, "token_spec.username", testData["username"]),
					resource.TestCheckResourceAttr(fqrn, "token_spec.scope", "applied-permissions/user"),
					resource.TestCheckResourceAttr(fqrn, "token_spec.audience", "*@*"),
					resource.TestCheckResourceAttr(fqrn, "token_spec.expires_in", "120"),
				),
			},
			{
				Config: updatedConfig,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(fqrn, "name", updatedTestData["identityMappingName"]),
					resource.TestCheckResourceAttr(fqrn, "priority", updatedTestData["priority"]),
					resource.TestCheckResourceAttr(fqrn, "claims_json", fmt.Sprintf("{\"sub\":\"%s\",\"updated_at\":1490198843}", updatedTestData["sub"])),
					resource.TestCheckResourceAttr(fqrn, "token_spec.username", updatedTestData["username"]),
					resource.TestCheckResourceAttr(fqrn, "token_spec.scope", "applied-permissions/user"),
					resource.TestCheckResourceAttr(fqrn, "token_spec.audience", "*@*"),
					resource.TestCheckResourceAttr(fqrn, "token_spec.expires_in", "120"),
				),
			},
			{
				ResourceName:                         fqrn,
				ImportState:                          true,
				ImportStateId:                        fmt.Sprintf("%s:%s", identityMappingName, configName),
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "name",
			},
		},
	})
}

func TestAccOIDIdentityMapping_invalid_name(t *testing.T) {
	for _, invalidName := range []string{"invalid name", "invalid!name"} {
		t.Run(invalidName, func(t *testing.T) {
			_, _, configName := testutil.MkNames("test-oidc-configuration", "platform_oidc_configuration")
			_, _, identityMappingName := testutil.MkNames("test-oidc-identity-mapping", "platform_oidc_identity_mapping")

			temp := `
			resource "platform_oidc_configuration" "{{ .configName }}" {
				name          = "{{ .configName }}"
				description   = "Test description"
				issuer_url    = "{{ .issuerURL }}"
				provider_type = "{{ .providerType }}"
				audience      = "{{ .audience }}"
			}

			resource "platform_oidc_identity_mapping" "{{ .identityMappingName }}" {
				name          = "{{ .invalidName }}"
				description   = "Test description"
				provider_name = platform_oidc_configuration.{{ .configName }}.name
				priority      = {{ .priority }}
				claims_json   = jsonencode({
					sub = "test-subscriber",
					updated_at = 1490198843
				})
				token_spec = {
					username   = "{{ .username }}"
					scope      = "applied-permissions/user"
					audience   = "*@*"
					expires_in = 120
				}
			}`

			testData := map[string]string{
				"configName":          configName,
				"identityMappingName": identityMappingName,
				"invalidName":         invalidName,
				"issuerURL":           "https://tempurl.org",
				"providerType":        "generic",
				"audience":            "test-audience",
				"priority":            fmt.Sprintf("%d", testutil.RandomInt()),
				"username":            fmt.Sprintf("test-user-%d", testutil.RandomInt()),
			}

			config := testutil.ExecuteTemplate(identityMappingName, temp, testData)

			resource.Test(t, resource.TestCase{
				PreCheck:                 func() { testAccPreCheck(t) },
				ProtoV6ProviderFactories: testAccProviders(),
				Steps: []resource.TestStep{
					{
						Config:      config,
						ExpectError: regexp.MustCompile(`.*name cannot contain spaces or special characters.*`),
					},
				},
			})
		})
	}
}

func TestAccOIDIdentityMapping_invalid_provider_name(t *testing.T) {
	for _, invalidName := range []string{"Test", "test!@", "1test"} {
		t.Run(invalidName, func(t *testing.T) {
			_, _, identityMappingName := testutil.MkNames("test-oidc-identity-mapping", "platform_oidc_identity_mapping")

			temp := `
			resource "platform_oidc_identity_mapping" "{{ .identityMappingName }}" {
				name          = "{{ .identityMappingName }}"
				description   = "Test description"
				provider_name = "{{ .invalidName }}"
				priority      = {{ .priority }}
				claims_json   = jsonencode({
					sub = "{{ .sub }}",
					workflow_ref = "{{ .workflowRef }}"
				})
				token_spec = {
					username   = "{{ .username }}"
					scope      = "applied-permissions/user"
					audience   = "*@*"
					expires_in = 120
				}
			}`

			testData := map[string]string{
				"identityMappingName": identityMappingName,
				"invalidName":         invalidName,
				"priority":            fmt.Sprintf("%d", testutil.RandomInt()),
				"sub":                 "repo:humpty/access-oidc-poc:ref:refs/heads/main",
				"workflowRef":         "humpty/access-oidc-poc/.github/workflows/job.yaml@refs/heads/main",
				"username":            fmt.Sprintf("test-user-%d", testutil.RandomInt()),
			}

			config := testutil.ExecuteTemplate(identityMappingName, temp, testData)

			resource.Test(t, resource.TestCase{
				PreCheck:                 func() { testAccPreCheck(t) },
				ProtoV6ProviderFactories: testAccProviders(),
				Steps: []resource.TestStep{
					{
						Config:      config,
						ExpectError: regexp.MustCompile(`.*must start with a lowercase letter and only contain.*`),
					},
				},
			})
		})
	}
}
