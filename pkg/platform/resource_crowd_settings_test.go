package platform_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/jfrog/terraform-provider-shared/testutil"
	"github.com/jfrog/terraform-provider-shared/util"
)

func TestAccCrowdSettings_full(t *testing.T) {
	_, fqrn, name := testutil.MkNames("test-crowd-settings", "platform_crowd_settings")

	temp := `
	resource "platform_crowd_settings" "{{ .name }}" {
		enable                         = true
		server_url                     = "{{ .serverURL }}"
		application_name               = "{{ .name }}"
		password                       = "Password1!"
		session_validation_interval    = {{ .sessionValidationInterval }}
		use_default_proxy              = false
		auto_user_creation             = {{ .autoUserCreation }}
		allow_user_to_access_profile   = false
		direct_authentication          = true
		override_all_groups_upon_login = false
	}`

	testData := map[string]string{
		"name":                      name,
		"serverURL":                 "http://tempurl.org",
		"sessionValidationInterval": "1",
		"autoUserCreation":          fmt.Sprintf("%t", testutil.RandBool()),
	}

	config := util.ExecuteTemplate(name, temp, testData)

	updatedTestData := map[string]string{
		"name":                      name,
		"serverURL":                 "http://tempurl.org",
		"sessionValidationInterval": "5",
		"autoUserCreation":          fmt.Sprintf("%t", testutil.RandBool()),
	}

	updatedConfig := util.ExecuteTemplate(name, temp, updatedTestData)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProviders(),
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(fqrn, "enable", "true"),
					resource.TestCheckResourceAttr(fqrn, "server_url", "http://tempurl.org"),
					resource.TestCheckResourceAttr(fqrn, "application_name", testData["name"]),
					resource.TestCheckResourceAttr(fqrn, "session_validation_interval", testData["sessionValidationInterval"]),
					resource.TestCheckResourceAttr(fqrn, "use_default_proxy", "false"),
					resource.TestCheckResourceAttr(fqrn, "auto_user_creation", testData["autoUserCreation"]),
					resource.TestCheckResourceAttr(fqrn, "allow_user_to_access_profile", "false"),
					resource.TestCheckResourceAttr(fqrn, "direct_authentication", "true"),
					resource.TestCheckResourceAttr(fqrn, "override_all_groups_upon_login", "false"),
				),
			},
			{
				Config: updatedConfig,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(fqrn, "enable", "true"),
					resource.TestCheckResourceAttr(fqrn, "server_url", "http://tempurl.org"),
					resource.TestCheckResourceAttr(fqrn, "application_name", updatedTestData["name"]),
					resource.TestCheckResourceAttr(fqrn, "session_validation_interval", updatedTestData["sessionValidationInterval"]),
					resource.TestCheckResourceAttr(fqrn, "use_default_proxy", "false"),
					resource.TestCheckResourceAttr(fqrn, "auto_user_creation", updatedTestData["autoUserCreation"]),
					resource.TestCheckResourceAttr(fqrn, "allow_user_to_access_profile", "false"),
					resource.TestCheckResourceAttr(fqrn, "direct_authentication", "true"),
					resource.TestCheckResourceAttr(fqrn, "override_all_groups_upon_login", "false"),
				),
			},
			{
				ResourceName:                         fqrn,
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateId:                        name,
				ImportStateVerifyIdentifierAttribute: "server_url",
				ImportStateVerifyIgnore:              []string{"password"},
			},
		},
	})
}
