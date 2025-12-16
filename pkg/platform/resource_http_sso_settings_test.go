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

func TestAccHTTPSSOSettings_full(t *testing.T) {
	_, fqrn, name := testutil.MkNames("test-http-sso-settings", "platform_http_sso_settings")

	temp := `
	resource "platform_http_sso_settings" "{{ .name }}" {
		proxied                      = {{ .proxied }}
		auto_create_user             = {{ .autoCreateUser }}
		allow_user_to_access_profile = {{ .allowUserToAccessProfile }}
		remote_user_request_variable = "{{ .remoteUserRequestVariable }}"
		sync_ldap_groups             = {{ .syncLDAPGroups }}
	}`

	testData := map[string]string{
		"name":                      name,
		"proxied":                   "true",
		"autoCreateUser":            fmt.Sprintf("%t", testutil.RandBool()),
		"allowUserToAccessProfile":  fmt.Sprintf("%t", testutil.RandBool()),
		"remoteUserRequestVariable": "TEST",
		"syncLDAPGroups":            fmt.Sprintf("%t", testutil.RandBool()),
	}

	config := util.ExecuteTemplate(name, temp, testData)

	updatedTestData := map[string]string{
		"name":                      name,
		"proxied":                   "false",
		"autoCreateUser":            fmt.Sprintf("%t", testutil.RandBool()),
		"allowUserToAccessProfile":  fmt.Sprintf("%t", testutil.RandBool()),
		"remoteUserRequestVariable": "TEST",
		"syncLDAPGroups":            fmt.Sprintf("%t", testutil.RandBool()),
	}

	updatedConfig := util.ExecuteTemplate(name, temp, updatedTestData)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProviders(),
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(fqrn, "proxied", testData["proxied"]),
					resource.TestCheckResourceAttr(fqrn, "auto_create_user", testData["autoCreateUser"]),
					resource.TestCheckResourceAttr(fqrn, "allow_user_to_access_profile", testData["allowUserToAccessProfile"]),
					resource.TestCheckResourceAttr(fqrn, "remote_user_request_variable", testData["remoteUserRequestVariable"]),
					resource.TestCheckResourceAttr(fqrn, "sync_ldap_groups", testData["syncLDAPGroups"]),
				),
			},
			{
				Config: updatedConfig,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(fqrn, "proxied", updatedTestData["proxied"]),
					resource.TestCheckResourceAttr(fqrn, "auto_create_user", updatedTestData["autoCreateUser"]),
					resource.TestCheckResourceAttr(fqrn, "allow_user_to_access_profile", updatedTestData["allowUserToAccessProfile"]),
					resource.TestCheckResourceAttr(fqrn, "remote_user_request_variable", updatedTestData["remoteUserRequestVariable"]),
					resource.TestCheckResourceAttr(fqrn, "sync_ldap_groups", updatedTestData["syncLDAPGroups"]),
				),
			},
			{
				ResourceName:                         fqrn,
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateId:                        name,
				ImportStateVerifyIdentifierAttribute: "remote_user_request_variable",
			},
		},
	})
}
