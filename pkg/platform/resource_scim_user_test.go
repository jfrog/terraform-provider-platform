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
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/jfrog/terraform-provider-platform/v2/pkg/platform"
	"github.com/jfrog/terraform-provider-shared/testutil"
	"github.com/jfrog/terraform-provider-shared/util"
)

func TestAccSCIMUser_full(t *testing.T) {
	_, fqrn, name := testutil.MkNames("test-scim-user", "platform_scim_user")

	temp := `
	resource "platform_scim_user" "{{ .name }}" {
		username = "{{ .email }}"
		active   = {{ .active }}
		emails = [{
			value = "{{ .email }}"
			primary = true
		}]
	}`

	testData := map[string]string{
		"name":   name,
		"email":  "test@tempurl.org",
		"active": "true",
	}

	config := util.ExecuteTemplate(name, temp, testData)

	updatedTestData := map[string]string{
		"name":   name,
		"email":  "test@tempurl.org",
		"active": "false",
	}

	updatedConfig := util.ExecuteTemplate(name, temp, updatedTestData)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProviders(),
		CheckDestroy:             testAccSCIMUserDestroy(fqrn),
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(fqrn, "username", testData["email"]),
					resource.TestCheckResourceAttr(fqrn, "active", "true"),
					resource.TestCheckResourceAttr(fqrn, "emails.#", "1"),
					resource.TestCheckResourceAttr(fqrn, "emails.0.value", testData["email"]),
					resource.TestCheckResourceAttr(fqrn, "emails.0.primary", "true"),
					resource.TestCheckResourceAttr(fqrn, "groups.#", "1"),
					resource.TestCheckResourceAttr(fqrn, "groups.0.value", "readers"),
					resource.TestCheckResourceAttr(fqrn, "meta.%", "1"),
					resource.TestCheckResourceAttr(fqrn, "meta.resourceType", "User"),
				),
			},
			{
				Config: updatedConfig,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(fqrn, "username", updatedTestData["email"]),
					resource.TestCheckResourceAttr(fqrn, "active", "false"),
					resource.TestCheckResourceAttr(fqrn, "emails.#", "1"),
					resource.TestCheckResourceAttr(fqrn, "emails.0.value", updatedTestData["email"]),
					resource.TestCheckResourceAttr(fqrn, "emails.0.primary", "true"),
					resource.TestCheckResourceAttr(fqrn, "groups.#", "1"),
					resource.TestCheckResourceAttr(fqrn, "groups.0.value", "readers"),
					resource.TestCheckResourceAttr(fqrn, "meta.%", "1"),
					resource.TestCheckResourceAttr(fqrn, "meta.resourceType", "User"),
				),
			},
			{
				ResourceName:                         fqrn,
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateId:                        updatedTestData["email"],
				ImportStateVerifyIdentifierAttribute: "username",
			},
		},
	})
}

func testAccSCIMUserDestroy(id string) func(*terraform.State) error {
	return func(s *terraform.State) error {
		c := TestProvider.(*platform.PlatformProvider).Meta.Client

		rs, ok := s.RootModule().Resources[id]
		if !ok {
			return fmt.Errorf("error: resource id [%s] not found", id)
		}

		var user platform.SCIMUserAPIModel
		resp, err := c.R().
			SetPathParam("id", rs.Primary.Attributes["username"]).
			SetResult(&user).
			Get(platform.SCIMUserEndpoint)
		if err != nil {
			return err
		}

		if resp != nil && resp.StatusCode() == http.StatusNotFound {
			return nil
		}

		return fmt.Errorf("error: SCIM user %s still exists", rs.Primary.Attributes["username"])
	}
}
