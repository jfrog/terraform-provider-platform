package platform_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/jfrog/terraform-provider-platform/pkg/platform"
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
		no_auto_user_creation        = false
		service_provider_name        = "okta"
		allow_user_to_access_profile = true
		auto_redirect                = true
		sync_groups                  = true
		verify_audience_restriction  = true
		use_encrypted_assertion      = false
	}`

	testData := map[string]string{
		"name":              name,
		"certificate":       "MIICTjCCAbegAwIBAgIBADANBgkqhkiG9w0BAQ0FADBEMQswCQYDVQQGEwJ1czELMAkGA1UECAwCQ0ExFjAUBgNVBAoMDUpGcm9nIFRlc3RpbmcxEDAOBgNVBAMMB1Rlc3RpbmcwHhcNMjQwODA4MTgzNjMxWhcNMjUwODA4MTgzNjMxWjBEMQswCQYDVQQGEwJ1czELMAkGA1UECAwCQ0ExFjAUBgNVBAoMDUpGcm9nIFRlc3RpbmcxEDAOBgNVBAMMB1Rlc3RpbmcwgZ8wDQYJKoZIhvcNAQEBBQADgY0AMIGJAoGBAOPwKU3SxuRaJply2by60NxYmbIPfelhM6sObgPRXbm49Mz4o1nbwH/vwhz1K+klVO4hOiKc5aP5GtQEoBejZbxOXlYlf8YirNqbtEXlIattvZA3tlC8O9oNOzBuT6tRdAA9CvN035p17fN0tpejz7Ptn1G1yUAt9klTUBBZ8eERAgMBAAGjUDBOMB0GA1UdDgQWBBR2y2SefjbqeSHTj+URrKc540YkGTAfBgNVHSMEGDAWgBR2y2SefjbqeSHTj+URrKc540YkGTAMBgNVHRMEBTADAQH/MA0GCSqGSIb3DQEBDQUAA4GBAKxnkFRgLZnQ4U6fWjfuJnx29cKbIq4oBr9RuWEKH2Hhx+jWy/3baNrxE0AsNWTLX6gGVd2qJbfae803AN6ZLx+VrLCWKl+c5MTTZBhuX6G/JvWviavE44P1U4cl2c6w4qvAmY+SY0cnJeWGLCBJ2vJ/fauXS/TIr0IfziSRcVYY",
		"email_attribute":   "email",
		"group_attribute":   "group",
		"name_id_attribute": "name",
	}

	config := util.ExecuteTemplate(name, temp, testData)

	updatedTestData := map[string]string{
		"name":              name,
		"certificate":       "MIICTjCCAbegAwIBAgIBADANBgkqhkiG9w0BAQ0FADBEMQswCQYDVQQGEwJ1czELMAkGA1UECAwCQ0ExFjAUBgNVBAoMDUpGcm9nIFRlc3RpbmcxEDAOBgNVBAMMB1Rlc3RpbmcwHhcNMjQwODA4MTgzNjMxWhcNMjUwODA4MTgzNjMxWjBEMQswCQYDVQQGEwJ1czELMAkGA1UECAwCQ0ExFjAUBgNVBAoMDUpGcm9nIFRlc3RpbmcxEDAOBgNVBAMMB1Rlc3RpbmcwgZ8wDQYJKoZIhvcNAQEBBQADgY0AMIGJAoGBAOPwKU3SxuRaJply2by60NxYmbIPfelhM6sObgPRXbm49Mz4o1nbwH/vwhz1K+klVO4hOiKc5aP5GtQEoBejZbxOXlYlf8YirNqbtEXlIattvZA3tlC8O9oNOzBuT6tRdAA9CvN035p17fN0tpejz7Ptn1G1yUAt9klTUBBZ8eERAgMBAAGjUDBOMB0GA1UdDgQWBBR2y2SefjbqeSHTj+URrKc540YkGTAfBgNVHSMEGDAWgBR2y2SefjbqeSHTj+URrKc540YkGTAMBgNVHRMEBTADAQH/MA0GCSqGSIb3DQEBDQUAA4GBAKxnkFRgLZnQ4U6fWjfuJnx29cKbIq4oBr9RuWEKH2Hhx+jWy/3baNrxE0AsNWTLX6gGVd2qJbfae803AN6ZLx+VrLCWKl+c5MTTZBhuX6G/JvWviavE44P1U4cl2c6w4qvAmY+SY0cnJeWGLCBJ2vJ/fauXS/TIr0IfziSRcVYY",
		"email_attribute":   "email2",
		"group_attribute":   "group2",
		"name_id_attribute": "name2",
	}

	updatedConfig := util.ExecuteTemplate(name, temp, updatedTestData)

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
					resource.TestCheckResourceAttr(fqrn, "no_auto_user_creation", "false"),
					resource.TestCheckResourceAttr(fqrn, "service_provider_name", "okta"),
					resource.TestCheckResourceAttr(fqrn, "allow_user_to_access_profile", "true"),
					resource.TestCheckResourceAttr(fqrn, "auto_redirect", "true"),
					resource.TestCheckResourceAttr(fqrn, "sync_groups", "true"),
					resource.TestCheckResourceAttr(fqrn, "verify_audience_restriction", "true"),
					resource.TestCheckResourceAttr(fqrn, "use_encrypted_assertion", "false"),
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
					resource.TestCheckResourceAttr(fqrn, "no_auto_user_creation", "false"),
					resource.TestCheckResourceAttr(fqrn, "service_provider_name", "okta"),
					resource.TestCheckResourceAttr(fqrn, "allow_user_to_access_profile", "true"),
					resource.TestCheckResourceAttr(fqrn, "auto_redirect", "true"),
					resource.TestCheckResourceAttr(fqrn, "sync_groups", "true"),
					resource.TestCheckResourceAttr(fqrn, "verify_audience_restriction", "true"),
					resource.TestCheckResourceAttr(fqrn, "use_encrypted_assertion", "false"),
				),
			},
			{
				ResourceName:                         fqrn,
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateId:                        name,
				ImportStateVerifyIdentifierAttribute: "name",
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
			Get(platform.SAMLSettingEndpoint)
		if err != nil {
			return err
		}

		if resp != nil && resp.StatusCode() == http.StatusNotFound {
			return nil
		}

		return fmt.Errorf("error: SAML Settings %s still exists", rs.Primary.Attributes["name"])
	}
}
