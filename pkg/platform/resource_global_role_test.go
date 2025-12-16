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
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/jfrog/terraform-provider-shared/testutil"
	"github.com/jfrog/terraform-provider-shared/util"
)

func TestAccGlobalRole_full(t *testing.T) {
	_, fqrn, roleName := testutil.MkNames("test-global-role", "platform_global_role")

	temp := `
	resource "platform_global_role" "{{ .name }}" {
		name         = "{{ .name }}"
		description  = "Test description"
		type         = "{{ .type }}"
		environments = ["{{ .environment }}"]
		actions      = ["{{ .action }}"]
	}`

	testData := map[string]string{
		"name":        roleName,
		"type":        "CUSTOM_GLOBAL",
		"environment": "DEV",
		"action":      "READ_REPOSITORY",
	}

	config := util.ExecuteTemplate(roleName, temp, testData)

	updatedTemp := `
	resource "platform_global_role" "{{ .name }}" {
		name         = "{{ .name }}"
		description  = "Test description"
		type         = "{{ .type }}"
		environments = ["{{ .environment }}", "{{ .environment2 }}"]
		actions      = ["{{ .action }}", "{{ .action2 }}"]
	}`

	updatedTestData := map[string]string{
		"name":         roleName,
		"type":         "CUSTOM_GLOBAL",
		"environment":  "DEV",
		"environment2": "PROD",
		"action":       "READ_REPOSITORY",
		"action2":      "READ_BUILD",
	}
	updatedConfig := util.ExecuteTemplate(roleName, updatedTemp, updatedTestData)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProviders(),
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(fqrn, "name", testData["name"]),
					resource.TestCheckResourceAttr(fqrn, "type", testData["type"]),
					resource.TestCheckResourceAttr(fqrn, "environments.#", "1"),
					resource.TestCheckResourceAttr(fqrn, "environments.0", "DEV"),
					resource.TestCheckResourceAttr(fqrn, "actions.#", "1"),
					resource.TestCheckResourceAttr(fqrn, "actions.0", "READ_REPOSITORY"),
				),
			},
			{
				Config: updatedConfig,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(fqrn, "name", testData["name"]),
					resource.TestCheckResourceAttr(fqrn, "type", testData["type"]),
					resource.TestCheckResourceAttr(fqrn, "environments.#", "2"),
					resource.TestCheckTypeSetElemAttr(fqrn, "environments.*", "DEV"),
					resource.TestCheckTypeSetElemAttr(fqrn, "environments.*", "PROD"),
					resource.TestCheckResourceAttr(fqrn, "actions.#", "2"),
					resource.TestCheckTypeSetElemAttr(fqrn, "actions.*", "READ_REPOSITORY"),
					resource.TestCheckTypeSetElemAttr(fqrn, "actions.*", "READ_BUILD"),
				),
			},
			{
				ResourceName:                         fqrn,
				ImportState:                          true,
				ImportStateId:                        roleName,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "name",
			},
		},
	})
}

func TestAccGlobalRole_name_change(t *testing.T) {
	_, fqrn, roleName := testutil.MkNames("test-global-role", "platform_global_role")

	temp := `
	resource "platform_global_role" "{{ .name }}" {
		name         = "{{ .name }}"
		description  = "Test description"
		type         = "{{ .type }}"
		environments = ["{{ .environment }}"]
		actions      = ["{{ .action }}"]
	}`

	testData := map[string]string{
		"name":        roleName,
		"type":        "CUSTOM_GLOBAL",
		"environment": "DEV",
		"action":      "READ_REPOSITORY",
	}

	config := util.ExecuteTemplate(roleName, temp, testData)

	nameChangeTemp := `
	resource "platform_global_role" "{{ .name }}" {
		name         = "foobar"
		description  = "Test description"
		type         = "{{ .type }}"
		environments = ["{{ .environment }}"]
		actions      = ["{{ .action }}"]
	}`

	updatedConfig := util.ExecuteTemplate(roleName, nameChangeTemp, testData)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProviders(),
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(fqrn, "name", testData["name"]),
					resource.TestCheckResourceAttr(fqrn, "type", testData["type"]),
					resource.TestCheckResourceAttr(fqrn, "environments.#", "1"),
					resource.TestCheckResourceAttr(fqrn, "environments.0", "DEV"),
					resource.TestCheckResourceAttr(fqrn, "actions.#", "1"),
					resource.TestCheckResourceAttr(fqrn, "actions.0", "READ_REPOSITORY"),
				),
			},
			{
				Config: updatedConfig,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(fqrn, plancheck.ResourceActionDestroyBeforeCreate),
					},
				},
			},
		},
	})
}
