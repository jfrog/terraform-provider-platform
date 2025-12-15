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
