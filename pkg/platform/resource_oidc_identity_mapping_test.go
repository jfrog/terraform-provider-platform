package platform_test

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/jfrog/terraform-provider-shared/testutil"
	"github.com/jfrog/terraform-provider-shared/util"
)

func TestAccOIDCIdentityMapping_full(t *testing.T) {
	_, _, configName := testutil.MkNames("test-oidc-configuration", "platform_oidc_configuration")
	_, fqrn, identityMappingName := testutil.MkNames("test-oidc-identity-mapping", "platform_oidc_identity_mapping")

	temp := `
	resource "platform_oidc_configuration" "{{ .configName }}" {
		name          = "{{ .configName }}"
		issuer_url    = "{{ .issuerURL }}"
		provider_type = "{{ .providerType }}"
		audience      = "{{ .audience }}"
	}

	resource "platform_oidc_identity_mapping" "{{ .identityMappingName }}" {
		name          = "{{ .identityMappingName }}"
		provider_name = platform_oidc_configuration.{{ .configName }}.name
		priority      = {{ .priority }}
		claims_json   = jsonencode({
			sub = "{{ .sub }}",
			updated_at = 1490198843
		})
		token_spec = {
			username   = "{{ .username }}"
			scope      = "applied-permissions/user"
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

	config := util.ExecuteTemplate(identityMappingName, temp, testData)

	updatedTemp := `
	resource "platform_oidc_configuration" "{{ .configName }}" {
		name          = "{{ .configName }}"
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
			audience   = "jfrt@* jfac@* jfmc@* jfmd@* jfevt@* jfxfer@* jflnk@* jfint@* jfwks@*"
			expires_in = 120
		}
	}`

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

	updatedConfig := util.ExecuteTemplate(identityMappingName, updatedTemp, updatedTestData)

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
					resource.TestCheckResourceAttr(fqrn, "token_spec.expires_in", "60"),
				),
			},
			{
				Config: updatedConfig,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(fqrn, "name", updatedTestData["identityMappingName"]),
					resource.TestCheckResourceAttr(fqrn, "description", "Test description"),
					resource.TestCheckResourceAttr(fqrn, "priority", updatedTestData["priority"]),
					resource.TestCheckResourceAttr(fqrn, "claims_json", fmt.Sprintf("{\"sub\":\"%s\",\"updated_at\":1490198843}", updatedTestData["sub"])),
					resource.TestCheckResourceAttr(fqrn, "token_spec.username", updatedTestData["username"]),
					resource.TestCheckResourceAttr(fqrn, "token_spec.scope", "applied-permissions/user"),
					resource.TestCheckResourceAttr(fqrn, "token_spec.audience", "jfrt@* jfac@* jfmc@* jfmd@* jfevt@* jfxfer@* jflnk@* jfint@* jfwks@*"),
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

func TestAccOIDCIdentityMapping_groups_scope(t *testing.T) {
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
			scope      = "applied-permissions/groups:\"readers\",\"test\""
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
		"scope":               "applied-permissions/groups:\"readers\",\"test\"",
	}

	config := util.ExecuteTemplate(identityMappingName, temp, testData)

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
					resource.TestCheckResourceAttr(fqrn, "token_spec.scope", "applied-permissions/groups:\"readers\",\"test\""),
					resource.TestCheckResourceAttr(fqrn, "token_spec.audience", "*@*"),
					resource.TestCheckResourceAttr(fqrn, "token_spec.expires_in", "120"),
				),
			},
		},
	})
}

func TestAccOIDCIdentityMapping_invalid_name(t *testing.T) {
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

			config := util.ExecuteTemplate(identityMappingName, temp, testData)

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

func TestAccOIDCIdentityMapping_invalid_provider_name(t *testing.T) {
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

			config := util.ExecuteTemplate(identityMappingName, temp, testData)

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

func TestAccOIDCIdentityMapping_invalid_scope(t *testing.T) {
	for _, invalidScope := range []string{"invalid-scope", "applied-permissions/group", "applied-permissions/groups"} {
		t.Run(invalidScope, func(t *testing.T) {
			_, _, configName := testutil.MkNames("test-oidc-configuration", "platform_oidc_configuration")
			_, _, identityMappingName := testutil.MkNames("test-oidc-identity-mapping", "platform_oidc_identity_mapping")

			temp := `
			resource "platform_oidc_identity_mapping" "{{ .identityMappingName }}" {
				name          = "{{ .identityMappingName }}"
				description   = "Test description"
				provider_name = "{{ .configName }}"
				priority      = {{ .priority }}
				claims_json   = jsonencode({
					sub = "{{ .sub }}",
					workflow_ref = "{{ .workflowRef }}"
				})
				token_spec = {
					username   = "{{ .username }}"
					scope      = "{{ .invalidScope }}"
					audience   = "*@*"
					expires_in = 120
				}
			}`

			testData := map[string]string{
				"identityMappingName": identityMappingName,
				"configName":          configName,
				"priority":            fmt.Sprintf("%d", testutil.RandomInt()),
				"sub":                 "repo:humpty/access-oidc-poc:ref:refs/heads/main",
				"workflowRef":         "humpty/access-oidc-poc/.github/workflows/job.yaml@refs/heads/main",
				"username":            fmt.Sprintf("test-user-%d", testutil.RandomInt()),
				"invalidScope":        invalidScope,
			}

			config := util.ExecuteTemplate(identityMappingName, temp, testData)

			resource.Test(t, resource.TestCase{
				PreCheck:                 func() { testAccPreCheck(t) },
				ProtoV6ProviderFactories: testAccProviders(),
				Steps: []resource.TestStep{
					{
						Config:      config,
						ExpectError: regexp.MustCompile(`.*must start with either.*`),
					},
				},
			})
		})
	}
}
