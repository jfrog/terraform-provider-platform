package platform_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/jfrog/terraform-provider-platform/v2/pkg/platform"
	"github.com/jfrog/terraform-provider-shared/testutil"
	"github.com/jfrog/terraform-provider-shared/util"
)

func TestAccSAMLSettings_full(t *testing.T) {
	_, fqrn, name := testutil.MkNames("test-saml-settings", "platform_saml_settings")

	temp := `
	resource "platform_saml_settings" "{{ .name }}" {
		name                         = "{{ .name }}"
		enable                       = true
		certificate                  = "{{ .certificate }}"
		email_attribute              = "{{ .email_attribute }}"
		group_attribute              = "{{ .group_attribute }}"
		name_id_attribute            = "{{ .name_id_attribute }}"
		login_url                    = "http://tempurl.org/login"
		logout_url                   = "http://tempurl.org/logout"
		auto_user_creation           = {{ .auto_user_creation }}
		service_provider_name        = "okta"
		allow_user_to_access_profile = true
		auto_redirect                = true
		sync_groups                  = true
		verify_audience_restriction  = true
		use_encrypted_assertion      = false
		ldap_group_settings          = {{ .ldap_groups }}
	}`

	testData := map[string]string{
		"name":               name,
		"certificate":        "MIICTjCCAbegAwIBAgIBADANBgkqhkiG9w0BAQ0FADBEMQswCQYDVQQGEwJ1czELMAkGA1UECAwCQ0ExFjAUBgNVBAoMDUpGcm9nIFRlc3RpbmcxEDAOBgNVBAMMB1Rlc3RpbmcwHhcNMjQwODA4MTgzNjMxWhcNMjUwODA4MTgzNjMxWjBEMQswCQYDVQQGEwJ1czELMAkGA1UECAwCQ0ExFjAUBgNVBAoMDUpGcm9nIFRlc3RpbmcxEDAOBgNVBAMMB1Rlc3RpbmcwgZ8wDQYJKoZIhvcNAQEBBQADgY0AMIGJAoGBAOPwKU3SxuRaJply2by60NxYmbIPfelhM6sObgPRXbm49Mz4o1nbwH/vwhz1K+klVO4hOiKc5aP5GtQEoBejZbxOXlYlf8YirNqbtEXlIattvZA3tlC8O9oNOzBuT6tRdAA9CvN035p17fN0tpejz7Ptn1G1yUAt9klTUBBZ8eERAgMBAAGjUDBOMB0GA1UdDgQWBBR2y2SefjbqeSHTj+URrKc540YkGTAfBgNVHSMEGDAWgBR2y2SefjbqeSHTj+URrKc540YkGTAMBgNVHRMEBTADAQH/MA0GCSqGSIb3DQEBDQUAA4GBAKxnkFRgLZnQ4U6fWjfuJnx29cKbIq4oBr9RuWEKH2Hhx+jWy/3baNrxE0AsNWTLX6gGVd2qJbfae803AN6ZLx+VrLCWKl+c5MTTZBhuX6G/JvWviavE44P1U4cl2c6w4qvAmY+SY0cnJeWGLCBJ2vJ/fauXS/TIr0IfziSRcVYY",
		"email_attribute":    "email",
		"group_attribute":    "group",
		"name_id_attribute":  "name",
		"auto_user_creation": "false",
		"ldap_groups":        "[\"test-group-1\"]",
	}

	config := util.ExecuteTemplate(name, temp, testData)

	updatedTestData := map[string]string{
		"name":               name,
		"certificate":        "MIICTjCCAbegAwIBAgIBADANBgkqhkiG9w0BAQ0FADBEMQswCQYDVQQGEwJ1czELMAkGA1UECAwCQ0ExFjAUBgNVBAoMDUpGcm9nIFRlc3RpbmcxEDAOBgNVBAMMB1Rlc3RpbmcwHhcNMjQwODA4MTgzNjMxWhcNMjUwODA4MTgzNjMxWjBEMQswCQYDVQQGEwJ1czELMAkGA1UECAwCQ0ExFjAUBgNVBAoMDUpGcm9nIFRlc3RpbmcxEDAOBgNVBAMMB1Rlc3RpbmcwgZ8wDQYJKoZIhvcNAQEBBQADgY0AMIGJAoGBAOPwKU3SxuRaJply2by60NxYmbIPfelhM6sObgPRXbm49Mz4o1nbwH/vwhz1K+klVO4hOiKc5aP5GtQEoBejZbxOXlYlf8YirNqbtEXlIattvZA3tlC8O9oNOzBuT6tRdAA9CvN035p17fN0tpejz7Ptn1G1yUAt9klTUBBZ8eERAgMBAAGjUDBOMB0GA1UdDgQWBBR2y2SefjbqeSHTj+URrKc540YkGTAfBgNVHSMEGDAWgBR2y2SefjbqeSHTj+URrKc540YkGTAMBgNVHRMEBTADAQH/MA0GCSqGSIb3DQEBDQUAA4GBAKxnkFRgLZnQ4U6fWjfuJnx29cKbIq4oBr9RuWEKH2Hhx+jWy/3baNrxE0AsNWTLX6gGVd2qJbfae803AN6ZLx+VrLCWKl+c5MTTZBhuX6G/JvWviavE44P1U4cl2c6w4qvAmY+SY0cnJeWGLCBJ2vJ/fauXS/TIr0IfziSRcVYY",
		"email_attribute":    "email2",
		"group_attribute":    "group2",
		"name_id_attribute":  "name2",
		"auto_user_creation": "true",
		"ldap_groups":        "[\"test-group-1\", \"test-group-2\"]",
	}

	updatedConfig := util.ExecuteTemplate(name, temp, updatedTestData)

	temp2 := `
	resource "platform_saml_settings" "{{ .name }}" {
		name                         = "{{ .name }}"
		enable                       = true
		certificate                  = "{{ .certificate }}"
		email_attribute              = "{{ .email_attribute }}"
		group_attribute              = "{{ .group_attribute }}"
		name_id_attribute            = "{{ .name_id_attribute }}"
		login_url                    = "http://tempurl.org/login"
		logout_url                   = "http://tempurl.org/logout"
		auto_user_creation           = {{ .auto_user_creation }}
		service_provider_name        = "okta"
		allow_user_to_access_profile = true
		auto_redirect                = true
		sync_groups                  = true
		verify_audience_restriction  = true
		use_encrypted_assertion      = false
	}`

	updatedConfig2 := util.ExecuteTemplate(name, temp2, updatedTestData)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProviders(),
		CheckDestroy:             testAccSamlSettingsDestroy(fqrn),
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(fqrn, "name", testData["name"]),
					resource.TestCheckResourceAttr(fqrn, "enable", "true"),
					resource.TestCheckResourceAttr(fqrn, "certificate", testData["certificate"]),
					resource.TestCheckResourceAttr(fqrn, "email_attribute", testData["email_attribute"]),
					resource.TestCheckResourceAttr(fqrn, "group_attribute", testData["group_attribute"]),
					resource.TestCheckResourceAttr(fqrn, "name_id_attribute", testData["name_id_attribute"]),
					resource.TestCheckResourceAttr(fqrn, "login_url", "http://tempurl.org/login"),
					resource.TestCheckResourceAttr(fqrn, "logout_url", "http://tempurl.org/logout"),
					resource.TestCheckResourceAttr(fqrn, "auto_user_creation", testData["auto_user_creation"]),
					resource.TestCheckResourceAttr(fqrn, "service_provider_name", "okta"),
					resource.TestCheckResourceAttr(fqrn, "allow_user_to_access_profile", "true"),
					resource.TestCheckResourceAttr(fqrn, "auto_redirect", "true"),
					resource.TestCheckResourceAttr(fqrn, "sync_groups", "true"),
					resource.TestCheckResourceAttr(fqrn, "verify_audience_restriction", "true"),
					resource.TestCheckResourceAttr(fqrn, "use_encrypted_assertion", "false"),
					resource.TestCheckResourceAttr(fqrn, "ldap_group_settings.#", "1"),
					resource.TestCheckResourceAttr(fqrn, "ldap_group_settings.0", "test-group-1"),
				),
			},
			{
				Config: updatedConfig,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(fqrn, "name", updatedTestData["name"]),
					resource.TestCheckResourceAttr(fqrn, "enable", "true"),
					resource.TestCheckResourceAttr(fqrn, "certificate", updatedTestData["certificate"]),
					resource.TestCheckResourceAttr(fqrn, "email_attribute", updatedTestData["email_attribute"]),
					resource.TestCheckResourceAttr(fqrn, "group_attribute", updatedTestData["group_attribute"]),
					resource.TestCheckResourceAttr(fqrn, "name_id_attribute", updatedTestData["name_id_attribute"]),
					resource.TestCheckResourceAttr(fqrn, "login_url", "http://tempurl.org/login"),
					resource.TestCheckResourceAttr(fqrn, "logout_url", "http://tempurl.org/logout"),
					resource.TestCheckResourceAttr(fqrn, "auto_user_creation", updatedTestData["auto_user_creation"]),
					resource.TestCheckResourceAttr(fqrn, "service_provider_name", "okta"),
					resource.TestCheckResourceAttr(fqrn, "allow_user_to_access_profile", "true"),
					resource.TestCheckResourceAttr(fqrn, "auto_redirect", "true"),
					resource.TestCheckResourceAttr(fqrn, "sync_groups", "true"),
					resource.TestCheckResourceAttr(fqrn, "verify_audience_restriction", "true"),
					resource.TestCheckResourceAttr(fqrn, "use_encrypted_assertion", "false"),
					resource.TestCheckResourceAttr(fqrn, "ldap_group_settings.#", "2"),
					resource.TestCheckResourceAttr(fqrn, "ldap_group_settings.0", "test-group-1"),
					resource.TestCheckResourceAttr(fqrn, "ldap_group_settings.1", "test-group-2"),
				),
			},
			{
				Config: updatedConfig2,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(fqrn, "name", updatedTestData["name"]),
					resource.TestCheckResourceAttr(fqrn, "enable", "true"),
					resource.TestCheckResourceAttr(fqrn, "certificate", updatedTestData["certificate"]),
					resource.TestCheckResourceAttr(fqrn, "email_attribute", updatedTestData["email_attribute"]),
					resource.TestCheckResourceAttr(fqrn, "group_attribute", updatedTestData["group_attribute"]),
					resource.TestCheckResourceAttr(fqrn, "name_id_attribute", updatedTestData["name_id_attribute"]),
					resource.TestCheckResourceAttr(fqrn, "login_url", "http://tempurl.org/login"),
					resource.TestCheckResourceAttr(fqrn, "logout_url", "http://tempurl.org/logout"),
					resource.TestCheckResourceAttr(fqrn, "auto_user_creation", updatedTestData["auto_user_creation"]),
					resource.TestCheckResourceAttr(fqrn, "service_provider_name", "okta"),
					resource.TestCheckResourceAttr(fqrn, "allow_user_to_access_profile", "true"),
					resource.TestCheckResourceAttr(fqrn, "auto_redirect", "true"),
					resource.TestCheckResourceAttr(fqrn, "sync_groups", "true"),
					resource.TestCheckResourceAttr(fqrn, "verify_audience_restriction", "true"),
					resource.TestCheckResourceAttr(fqrn, "use_encrypted_assertion", "false"),
					resource.TestCheckNoResourceAttr(fqrn, "ldap_group_settings"),
				),
			},
			{
				ResourceName:                         fqrn,
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateId:                        name,
				ImportStateVerifyIdentifierAttribute: "name",
				ImportStateVerifyIgnore:              []string{"no_auto_user_creation"},
			},
		},
	})
}

func TestAccSAMLSettings_schema_migrate_from_v1_to_v2(t *testing.T) {
	_, fqrn, name := testutil.MkNames("test-saml-settings", "platform_saml_settings")

	tempV1 := `
	resource "platform_saml_settings" "{{ .name }}" {
		name                         = "{{ .name }}"
		enable                       = true
		certificate                  = "{{ .certificate }}"
		email_attribute              = "{{ .email_attribute }}"
		group_attribute              = "{{ .group_attribute }}"
		name_id_attribute            = "{{ .name_id_attribute }}"
		login_url                    = "http://tempurl.org/login"
		logout_url                   = "http://tempurl.org/logout"
		no_auto_user_creation        = true
		service_provider_name        = "okta"
		allow_user_to_access_profile = true
		auto_redirect                = true
		sync_groups                  = true
		verify_audience_restriction  = true
		use_encrypted_assertion      = false
		ldap_group_settings          = {{ .ldap_groups }}
	}`

	testDataV1 := map[string]string{
		"name":              name,
		"certificate":       "MIICTjCCAbegAwIBAgIBADANBgkqhkiG9w0BAQ0FADBEMQswCQYDVQQGEwJ1czELMAkGA1UECAwCQ0ExFjAUBgNVBAoMDUpGcm9nIFRlc3RpbmcxEDAOBgNVBAMMB1Rlc3RpbmcwHhcNMjQwODA4MTgzNjMxWhcNMjUwODA4MTgzNjMxWjBEMQswCQYDVQQGEwJ1czELMAkGA1UECAwCQ0ExFjAUBgNVBAoMDUpGcm9nIFRlc3RpbmcxEDAOBgNVBAMMB1Rlc3RpbmcwgZ8wDQYJKoZIhvcNAQEBBQADgY0AMIGJAoGBAOPwKU3SxuRaJply2by60NxYmbIPfelhM6sObgPRXbm49Mz4o1nbwH/vwhz1K+klVO4hOiKc5aP5GtQEoBejZbxOXlYlf8YirNqbtEXlIattvZA3tlC8O9oNOzBuT6tRdAA9CvN035p17fN0tpejz7Ptn1G1yUAt9klTUBBZ8eERAgMBAAGjUDBOMB0GA1UdDgQWBBR2y2SefjbqeSHTj+URrKc540YkGTAfBgNVHSMEGDAWgBR2y2SefjbqeSHTj+URrKc540YkGTAMBgNVHRMEBTADAQH/MA0GCSqGSIb3DQEBDQUAA4GBAKxnkFRgLZnQ4U6fWjfuJnx29cKbIq4oBr9RuWEKH2Hhx+jWy/3baNrxE0AsNWTLX6gGVd2qJbfae803AN6ZLx+VrLCWKl+c5MTTZBhuX6G/JvWviavE44P1U4cl2c6w4qvAmY+SY0cnJeWGLCBJ2vJ/fauXS/TIr0IfziSRcVYY",
		"email_attribute":   "email",
		"group_attribute":   "group",
		"name_id_attribute": "name",
		"ldap_groups":       "[\"test-group-1\"]",
	}

	configV1 := util.ExecuteTemplate(name, tempV1, testDataV1)

	tempV2 := `
	resource "platform_saml_settings" "{{ .name }}" {
		name                         = "{{ .name }}"
		enable                       = true
		certificate                  = "{{ .certificate }}"
		email_attribute              = "{{ .email_attribute }}"
		group_attribute              = "{{ .group_attribute }}"
		name_id_attribute            = "{{ .name_id_attribute }}"
		login_url                    = "http://tempurl.org/login"
		logout_url                   = "http://tempurl.org/logout"
		service_provider_name        = "okta"
		allow_user_to_access_profile = true
		auto_redirect                = true
		sync_groups                  = true
		verify_audience_restriction  = true
		use_encrypted_assertion      = false
		ldap_group_settings          = {{ .ldap_groups }}
	}`

	testDataV2 := map[string]string{
		"name":              name,
		"certificate":       "MIICTjCCAbegAwIBAgIBADANBgkqhkiG9w0BAQ0FADBEMQswCQYDVQQGEwJ1czELMAkGA1UECAwCQ0ExFjAUBgNVBAoMDUpGcm9nIFRlc3RpbmcxEDAOBgNVBAMMB1Rlc3RpbmcwHhcNMjQwODA4MTgzNjMxWhcNMjUwODA4MTgzNjMxWjBEMQswCQYDVQQGEwJ1czELMAkGA1UECAwCQ0ExFjAUBgNVBAoMDUpGcm9nIFRlc3RpbmcxEDAOBgNVBAMMB1Rlc3RpbmcwgZ8wDQYJKoZIhvcNAQEBBQADgY0AMIGJAoGBAOPwKU3SxuRaJply2by60NxYmbIPfelhM6sObgPRXbm49Mz4o1nbwH/vwhz1K+klVO4hOiKc5aP5GtQEoBejZbxOXlYlf8YirNqbtEXlIattvZA3tlC8O9oNOzBuT6tRdAA9CvN035p17fN0tpejz7Ptn1G1yUAt9klTUBBZ8eERAgMBAAGjUDBOMB0GA1UdDgQWBBR2y2SefjbqeSHTj+URrKc540YkGTAfBgNVHSMEGDAWgBR2y2SefjbqeSHTj+URrKc540YkGTAMBgNVHRMEBTADAQH/MA0GCSqGSIb3DQEBDQUAA4GBAKxnkFRgLZnQ4U6fWjfuJnx29cKbIq4oBr9RuWEKH2Hhx+jWy/3baNrxE0AsNWTLX6gGVd2qJbfae803AN6ZLx+VrLCWKl+c5MTTZBhuX6G/JvWviavE44P1U4cl2c6w4qvAmY+SY0cnJeWGLCBJ2vJ/fauXS/TIr0IfziSRcVYY",
		"email_attribute":   "email2",
		"group_attribute":   "group2",
		"name_id_attribute": "name2",
		"ldap_groups":       "[\"test-group-1\", \"test-group-2\"]",
	}

	configV2 := util.ExecuteTemplate(name, tempV2, testDataV2)

	resource.Test(t, resource.TestCase{
		PreCheck: func() { testAccPreCheck(t) },
		ExternalProviders: map[string]resource.ExternalProvider{
			"platform": {
				Source:            "jfrog/platform",
				VersionConstraint: "1.19.0",
			},
		},
		Steps: []resource.TestStep{
			{
				Config: configV1,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(fqrn, "name", testDataV1["name"]),
					resource.TestCheckResourceAttr(fqrn, "enable", "true"),
					resource.TestCheckResourceAttr(fqrn, "certificate", testDataV1["certificate"]),
					resource.TestCheckResourceAttr(fqrn, "email_attribute", testDataV1["email_attribute"]),
					resource.TestCheckResourceAttr(fqrn, "group_attribute", testDataV1["group_attribute"]),
					resource.TestCheckResourceAttr(fqrn, "name_id_attribute", testDataV1["name_id_attribute"]),
					resource.TestCheckResourceAttr(fqrn, "login_url", "http://tempurl.org/login"),
					resource.TestCheckResourceAttr(fqrn, "logout_url", "http://tempurl.org/logout"),
					resource.TestCheckResourceAttr(fqrn, "no_auto_user_creation", "true"),
					resource.TestCheckResourceAttr(fqrn, "service_provider_name", "okta"),
					resource.TestCheckResourceAttr(fqrn, "allow_user_to_access_profile", "true"),
					resource.TestCheckResourceAttr(fqrn, "auto_redirect", "true"),
					resource.TestCheckResourceAttr(fqrn, "sync_groups", "true"),
					resource.TestCheckResourceAttr(fqrn, "verify_audience_restriction", "true"),
					resource.TestCheckResourceAttr(fqrn, "use_encrypted_assertion", "false"),
					resource.TestCheckResourceAttr(fqrn, "ldap_group_settings.#", "1"),
					resource.TestCheckResourceAttr(fqrn, "ldap_group_settings.0", "test-group-1"),
				),
			},
		},
	})
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		CheckDestroy:             testAccSamlSettingsDestroy(fqrn),
		ProtoV6ProviderFactories: testAccProviders(),
		Steps: []resource.TestStep{
			{
				Config: configV2,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(fqrn, "name", testDataV2["name"]),
					resource.TestCheckResourceAttr(fqrn, "enable", "true"),
					resource.TestCheckResourceAttr(fqrn, "certificate", testDataV2["certificate"]),
					resource.TestCheckResourceAttr(fqrn, "email_attribute", testDataV2["email_attribute"]),
					resource.TestCheckResourceAttr(fqrn, "group_attribute", testDataV2["group_attribute"]),
					resource.TestCheckResourceAttr(fqrn, "name_id_attribute", testDataV2["name_id_attribute"]),
					resource.TestCheckResourceAttr(fqrn, "login_url", "http://tempurl.org/login"),
					resource.TestCheckResourceAttr(fqrn, "logout_url", "http://tempurl.org/logout"),
					resource.TestCheckResourceAttr(fqrn, "auto_user_creation", "true"),
					resource.TestCheckResourceAttr(fqrn, "service_provider_name", "okta"),
					resource.TestCheckResourceAttr(fqrn, "allow_user_to_access_profile", "true"),
					resource.TestCheckResourceAttr(fqrn, "auto_redirect", "true"),
					resource.TestCheckResourceAttr(fqrn, "sync_groups", "true"),
					resource.TestCheckResourceAttr(fqrn, "verify_audience_restriction", "true"),
					resource.TestCheckResourceAttr(fqrn, "use_encrypted_assertion", "false"),
					resource.TestCheckResourceAttr(fqrn, "ldap_group_settings.#", "2"),
					resource.TestCheckResourceAttr(fqrn, "ldap_group_settings.0", "test-group-1"),
					resource.TestCheckResourceAttr(fqrn, "ldap_group_settings.1", "test-group-2"),
				),
			},
		},
	})
}

func testAccSamlSettingsDestroy(id string) func(*terraform.State) error {
	return func(s *terraform.State) error {
		c := TestProvider.(*platform.PlatformProvider).Meta.Client

		rs, ok := s.RootModule().Resources[id]
		if !ok {
			return fmt.Errorf("error: resource id [%s] not found", id)
		}

		var samlSettings platform.SAMLSettingsAPIModel
		resp, err := c.R().
			SetPathParam("name", rs.Primary.Attributes["name"]).
			SetResult(&samlSettings).
			Get("access/api/v1/saml/{name}")
		if err != nil {
			return err
		}

		if resp != nil && resp.StatusCode() == http.StatusNotFound {
			return nil
		}

		return fmt.Errorf("error: SAML Settings %s still exists", rs.Primary.Attributes["name"])
	}
}
