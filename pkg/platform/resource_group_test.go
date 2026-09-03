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
	"regexp"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/jfrog/terraform-provider-shared/testutil"
	"github.com/jfrog/terraform-provider-shared/util"
)

func TestAccGroup_full(t *testing.T) {
	skipIfArtifactoryVersionBefore(t, "7.128.0")

	_, fqrn, groupName := testutil.MkNames("test-group", "platform_group")

	temp := `
		resource "platform_group" "{{ .groupName }}" {
			name                       = "{{ .groupName }}"
			description 	           = "Test group"
			external_id                = "externalID"
			auto_join                  = {{ .autoJoin }}
			admin_privileges           = false
			use_group_members_resource = false
			members                    = {{ .members }}
			reports_manager            = {{ .reportsManager }}
			watch_manager              = {{ .watchManager }}
			policy_manager             = {{ .policyManager }}
			policy_viewer              = {{ .policyViewer }}
			manage_resources           = {{ .manageResources }}
			manage_webhook             = {{ .manageWebhook }}
		}
	`

	// The Access API treats policy_manager as implying policy_viewer (a
	// manager is also a viewer). Keep the test data consistent with that
	// hierarchy: when policy_manager = true, policy_viewer must also be true,
	// otherwise the BoolImplies validator fires at plan time.
	testData := map[string]string{
		"groupName":       groupName,
		"autoJoin":        fmt.Sprintf("%t", testutil.RandBool()),
		"members":         "[\"anonymous\", \"admin\"]",
		"reportsManager":  "true",
		"watchManager":    "false",
		"policyManager":   "true",
		"policyViewer":    "true",
		"manageResources": "true",
		"manageWebhook":   "false",
	}

	config := util.ExecuteTemplate(groupName, temp, testData)

	updatedTestData := map[string]string{
		"groupName":       groupName,
		"autoJoin":        fmt.Sprintf("%t", testutil.RandBool()),
		"members":         "[\"admin\"]",
		"reportsManager":  "false",
		"watchManager":    "true",
		"policyManager":   "false",
		"policyViewer":    "true",
		"manageResources": "false",
		"manageWebhook":   "true",
	}

	updatedConfig := util.ExecuteTemplate(groupName, temp, updatedTestData)

	updated2TestData := map[string]string{
		"groupName":       groupName,
		"autoJoin":        fmt.Sprintf("%t", testutil.RandBool()),
		"adminPrivileges": fmt.Sprintf("%t", testutil.RandBool()),
		"members":         "[\"anonymous\"]",
		"reportsManager":  "true",
		"watchManager":    "true",
		"policyManager":   "false",
		"policyViewer":    "false",
		"manageResources": "true",
		"manageWebhook":   "true",
	}

	updated2Config := util.ExecuteTemplate(groupName, temp, updated2TestData)

	rolesCheck := func(td map[string]string) resource.TestCheckFunc {
		return resource.ComposeTestCheckFunc(
			resource.TestCheckResourceAttr(fqrn, "reports_manager", td["reportsManager"]),
			resource.TestCheckResourceAttr(fqrn, "watch_manager", td["watchManager"]),
			resource.TestCheckResourceAttr(fqrn, "policy_manager", td["policyManager"]),
			resource.TestCheckResourceAttr(fqrn, "policy_viewer", td["policyViewer"]),
			resource.TestCheckResourceAttr(fqrn, "manage_resources", td["manageResources"]),
			resource.TestCheckResourceAttr(fqrn, "manage_webhook", td["manageWebhook"]),
		)
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProviders(),
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(fqrn, "name", testData["groupName"]),
					resource.TestCheckResourceAttr(fqrn, "description", "Test group"),
					resource.TestCheckResourceAttr(fqrn, "external_id", "externalID"),
					resource.TestCheckResourceAttr(fqrn, "auto_join", testData["autoJoin"]),
					resource.TestCheckResourceAttr(fqrn, "admin_privileges", "false"),
					resource.TestCheckResourceAttrSet(fqrn, "realm"),
					resource.TestCheckNoResourceAttr(fqrn, "realm_attributes"),
					resource.TestCheckResourceAttr(fqrn, "members.#", "2"),
					resource.TestCheckResourceAttr(fqrn, "members.0", "admin"),
					resource.TestCheckResourceAttr(fqrn, "members.1", "anonymous"),
					rolesCheck(testData),
				),
			},
			{
				// Re-applying the same config must produce no diff. This guards
				// against drift on Computed-only attrs like realm/realm_attributes
				// and the new role booleans.
				Config: config,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
			},
			{
				Config: updatedConfig,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(fqrn, "name", updatedTestData["groupName"]),
					resource.TestCheckResourceAttr(fqrn, "description", "Test group"),
					resource.TestCheckResourceAttr(fqrn, "external_id", "externalID"),
					resource.TestCheckResourceAttr(fqrn, "auto_join", updatedTestData["autoJoin"]),
					resource.TestCheckResourceAttr(fqrn, "admin_privileges", "false"),
					resource.TestCheckResourceAttrSet(fqrn, "realm"),
					resource.TestCheckNoResourceAttr(fqrn, "realm_attributes"),
					resource.TestCheckResourceAttr(fqrn, "members.#", "1"),
					resource.TestCheckResourceAttr(fqrn, "members.0", "admin"),
					rolesCheck(updatedTestData),
				),
			},
			{
				Config: updated2Config,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(fqrn, "name", updated2TestData["groupName"]),
					resource.TestCheckResourceAttr(fqrn, "description", "Test group"),
					resource.TestCheckResourceAttr(fqrn, "external_id", "externalID"),
					resource.TestCheckResourceAttr(fqrn, "auto_join", updated2TestData["autoJoin"]),
					resource.TestCheckResourceAttr(fqrn, "admin_privileges", "false"),
					resource.TestCheckResourceAttrSet(fqrn, "realm"),
					resource.TestCheckNoResourceAttr(fqrn, "realm_attributes"),
					resource.TestCheckResourceAttr(fqrn, "members.#", "1"),
					resource.TestCheckResourceAttr(fqrn, "members.0", "anonymous"),
					rolesCheck(updated2TestData),
				),
			},
			{
				// This group manages membership inline (use_group_members_resource
				// = false), so import with the ":members" opt-in to load members
				// into state; ImportStateVerify then round-trips cleanly.
				ResourceName:                         fqrn,
				ImportState:                          true,
				ImportStateId:                        updated2TestData["groupName"] + ":members",
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "name",
				ImportStateVerifyIgnore:              []string{"use_group_members_resource"},
			},
		},
	})
}

func TestAccGroup_schema_migration(t *testing.T) {
	_, fqrn, groupName := testutil.MkNames("test-group", "platform_group")

	temp := `
		resource "platform_group" "{{ .groupName }}" {
			name             = "{{ .groupName }}"
			description 	 = "Test group"
			external_id      = "externalID"
			auto_join        = "true"
			admin_privileges = false
			members          = ["anonymous", "admin"]
		}
	`

	testData := map[string]string{
		"groupName": groupName,
	}

	config := util.ExecuteTemplate(groupName, temp, testData)

	resource.Test(t, resource.TestCase{
		Steps: []resource.TestStep{
			{
				ExternalProviders: map[string]resource.ExternalProvider{
					"platform": {
						Source:            "jfrog/platform",
						VersionConstraint: "2.1.0",
					},
				},
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(fqrn, "name", testData["groupName"]),
					resource.TestCheckResourceAttr(fqrn, "description", "Test group"),
					resource.TestCheckResourceAttr(fqrn, "external_id", "externalID"),
					resource.TestCheckResourceAttr(fqrn, "auto_join", "true"),
					resource.TestCheckResourceAttr(fqrn, "admin_privileges", "false"),
					resource.TestCheckNoResourceAttr(fqrn, "use_group_members_resource"),
					resource.TestCheckResourceAttrSet(fqrn, "realm"),
					resource.TestCheckNoResourceAttr(fqrn, "realm_attributes"),
					resource.TestCheckResourceAttr(fqrn, "members.#", "2"),
					resource.TestCheckResourceAttr(fqrn, "members.0", "admin"),
					resource.TestCheckResourceAttr(fqrn, "members.1", "anonymous"),
				),
			},
			{
				ProtoV6ProviderFactories: testAccProviders(),
				Config:                   config,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(fqrn, "name", testData["groupName"]),
					resource.TestCheckResourceAttr(fqrn, "description", "Test group"),
					resource.TestCheckResourceAttr(fqrn, "external_id", "externalID"),
					resource.TestCheckResourceAttr(fqrn, "auto_join", "true"),
					resource.TestCheckResourceAttr(fqrn, "admin_privileges", "false"),
					resource.TestCheckResourceAttrSet(fqrn, "realm"),
					resource.TestCheckNoResourceAttr(fqrn, "realm_attributes"),
					resource.TestCheckResourceAttr(fqrn, "members.#", "2"),
					resource.TestCheckResourceAttr(fqrn, "members.0", "admin"),
					resource.TestCheckResourceAttr(fqrn, "members.1", "anonymous"),
				),
			},
		},
	})
}

func TestAccGroup_no_members(t *testing.T) {
	_, fqrn, groupName := testutil.MkNames("test-group", "platform_group")

	temp := `
		resource "platform_group" "{{ .groupName }}" {
			name                       = "{{ .groupName }}"
			description 	           = "Test group"
			external_id                = "externalID"
			auto_join                  = {{ .autoJoin }}
			admin_privileges           = false
			use_group_members_resource = false
		}

		resource "artifactory_managed_user" "test-user" {
			name     = "test-user"
			password = "Password1!"
			email    = "test@tempurl.org"
			groups   = [platform_group.{{ .groupName }}.name]
		}
	`

	testData := map[string]string{
		"groupName": groupName,
		"autoJoin":  fmt.Sprintf("%t", testutil.RandBool()),
	}

	config := util.ExecuteTemplate(groupName, temp, testData)

	resource.Test(t, resource.TestCase{
		PreCheck: func() { testAccPreCheck(t) },
		ExternalProviders: map[string]resource.ExternalProvider{
			"artifactory": {
				Source: "jfrog/artifactory",
			},
		},
		ProtoV6ProviderFactories: testAccProviders(),
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(fqrn, "name", testData["groupName"]),
					resource.TestCheckResourceAttr(fqrn, "description", "Test group"),
					resource.TestCheckResourceAttr(fqrn, "external_id", "externalID"),
					resource.TestCheckResourceAttr(fqrn, "auto_join", testData["autoJoin"]),
					resource.TestCheckResourceAttr(fqrn, "admin_privileges", "false"),
					resource.TestCheckResourceAttrSet(fqrn, "realm"),
					resource.TestCheckNoResourceAttr(fqrn, "realm_attributes"),
					resource.TestCheckNoResourceAttr(fqrn, "members"),
				),
				ExpectNonEmptyPlan: true,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPostRefresh: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(fqrn, plancheck.ResourceActionUpdate),
					},
				},
			},
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(fqrn, "name", testData["groupName"]),
					resource.TestCheckResourceAttr(fqrn, "description", "Test group"),
					resource.TestCheckResourceAttr(fqrn, "external_id", "externalID"),
					resource.TestCheckResourceAttr(fqrn, "auto_join", testData["autoJoin"]),
					resource.TestCheckResourceAttr(fqrn, "admin_privileges", "false"),
					resource.TestCheckResourceAttrSet(fqrn, "realm"),
					resource.TestCheckNoResourceAttr(fqrn, "realm_attributes"),
					resource.TestCheckResourceAttr(fqrn, "members.#", "0"),
				),
				ExpectNonEmptyPlan: true,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPostRefresh: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(fqrn, plancheck.ResourceActionNoop),
						plancheck.ExpectResourceAction("artifactory_managed_user.test-user", plancheck.ResourceActionUpdate),
					},
				},
			},
		},
	})
}

func TestAccGroup_auto_join_conflict(t *testing.T) {
	_, _, groupName := testutil.MkNames("test-group", "platform_group")
	temp := `
		resource "platform_group" "{{ .groupName }}" {
			name             = "{{ .groupName }}"
			description 	 = "Test group"
			external_id      = "externalID"
			auto_join        = true
			admin_privileges = true
		}
	`

	config := util.ExecuteTemplate(groupName, temp, map[string]string{"groupName": groupName})

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProviders(),
		Steps: []resource.TestStep{
			{
				Config:      config,
				ExpectError: regexp.MustCompile(".*can not be set to.*"),
			},
		},
	})
}

// TestAccGroup_policy_manager_implies_viewer asserts that the BoolImplies
// validator on policy_manager rejects the only invalid combination at plan
// time: policy_manager = true together with policy_viewer = false. The Access
// API would otherwise silently coerce policy_viewer back to true, surfacing as
// a "Provider produced inconsistent result after apply" error.
func TestAccGroup_policy_manager_implies_viewer(t *testing.T) {
	skipIfArtifactoryVersionBefore(t, "7.128.0")

	_, _, groupName := testutil.MkNames("test-group", "platform_group")
	temp := `
		resource "platform_group" "{{ .groupName }}" {
			name           = "{{ .groupName }}"
			description    = "Test group"
			policy_manager = true
			policy_viewer  = false
		}
	`

	config := util.ExecuteTemplate(groupName, temp, map[string]string{"groupName": groupName})

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProviders(),
		Steps: []resource.TestStep{
			{
				Config:      config,
				ExpectError: regexp.MustCompile(".*can not be set to.*"),
			},
		},
	})
}

// TestAccGroup_description_empty verifies that the validator on the
// `description` attribute rejects empty strings at plan time. The Access
// service normalizes empty descriptions to null on read, so allowing `""`
// would cause perpetual plan drift after import / first apply (see issue #265).
func TestAccGroup_description_empty(t *testing.T) {
	_, _, groupName := testutil.MkNames("test-group", "platform_group")

	temp := `
		resource "platform_group" "{{ .groupName }}" {
			name        = "{{ .groupName }}"
			description = ""
		}
	`
	config := util.ExecuteTemplate(groupName, temp, map[string]string{"groupName": groupName})

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProviders(),
		Steps: []resource.TestStep{
			{
				Config:      config,
				ExpectError: regexp.MustCompile(`(?i)attribute description string length must be at least 1`),
			},
		},
	})
}

// TestAccGroup_import_no_description guards against the regression in issue
// #265: a group whose description was never set on the server (the GET
// response omits the field entirely) must be importable, must not show drift
// on subsequent plans, and must round-trip through ImportStateVerify.
func TestAccGroup_import_no_description(t *testing.T) {
	_, fqrn, groupName := testutil.MkNames("test-group", "platform_group")

	temp := `
		resource "platform_group" "{{ .groupName }}" {
			name = "{{ .groupName }}"
		}
	`
	config := util.ExecuteTemplate(groupName, temp, map[string]string{"groupName": groupName})

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProviders(),
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(fqrn, "name", groupName),
					resource.TestCheckNoResourceAttr(fqrn, "description"),
				),
			},
			{
				// Re-applying the same config must produce no diff, even
				// though the GET response omits the description field.
				Config: config,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
			},
			{
				ResourceName:                         fqrn,
				ImportState:                          true,
				ImportStateId:                        groupName,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "name",
				ImportStateVerifyIgnore:              []string{"use_group_members_resource"},
			},
		},
	})
}

// TestAccGroup_import_skips_members guards against issue #250: importing a
// platform_group must not pull the group's membership into state. Import seeds
// use_group_members_resource = true and leaves members null, so the follow-up
// Read skips the members block and membership stays owned by
// platform_group_members. This keeps state small and plans fast when adopting
// large AD/LDAP-synced groups.
func TestAccGroup_import_skips_members(t *testing.T) {
	_, fqrn, groupName := testutil.MkNames("test-group", "platform_group")

	temp := `
		resource "platform_group" "{{ .groupName }}" {
			name                       = "{{ .groupName }}"
			description                = "Test group"
			use_group_members_resource = false
			members                    = ["anonymous", "admin"]
		}
	`
	config := util.ExecuteTemplate(groupName, temp, map[string]string{"groupName": groupName})

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProviders(),
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(fqrn, "name", groupName),
					resource.TestCheckResourceAttr(fqrn, "members.#", "2"),
					resource.TestCheckResourceAttr(fqrn, "use_group_members_resource", "false"),
				),
			},
			{
				// Default import ("<name>") must NOT capture members into state and
				// must set use_group_members_resource = true. ImportStateVerify is
				// not used here because the imported state intentionally differs
				// from the applied state (members dropped, flag flipped).
				ResourceName:  fqrn,
				ImportState:   true,
				ImportStateId: groupName,
				ImportStateCheck: func(states []*terraform.InstanceState) error {
					if len(states) != 1 {
						return fmt.Errorf("expected 1 imported instance, got %d", len(states))
					}
					attrs := states[0].Attributes

					if got := attrs["use_group_members_resource"]; got != "true" {
						return fmt.Errorf("expected use_group_members_resource = true after import, got %q", got)
					}

					if got := attrs["members.#"]; got != "" && got != "0" {
						return fmt.Errorf("expected no members in imported state, got members.# = %q", got)
					}

					return nil
				},
			},
		},
	})
}

// TestAccGroup_import_members_mode covers the ":members" import opt-in: the
// group manages membership inline (use_group_members_resource = false), so
// importing with "<name>:members" must seed the flag false and load the current
// members into state.
func TestAccGroup_import_members_mode(t *testing.T) {
	_, fqrn, groupName := testutil.MkNames("test-group", "platform_group")

	temp := `
		resource "platform_group" "{{ .groupName }}" {
			name                       = "{{ .groupName }}"
			description                = "Test group"
			use_group_members_resource = false
			members                    = ["anonymous", "admin"]
		}
	`
	config := util.ExecuteTemplate(groupName, temp, map[string]string{"groupName": groupName})

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProviders(),
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(fqrn, "members.#", "2"),
					resource.TestCheckResourceAttr(fqrn, "use_group_members_resource", "false"),
				),
			},
			{
				// "<name>:members" opt-in: flag seeded false, members loaded.
				ResourceName:  fqrn,
				ImportState:   true,
				ImportStateId: groupName + ":members",
				ImportStateCheck: func(states []*terraform.InstanceState) error {
					if len(states) != 1 {
						return fmt.Errorf("expected 1 imported instance, got %d", len(states))
					}
					attrs := states[0].Attributes

					if got := attrs["use_group_members_resource"]; got != "false" {
						return fmt.Errorf("expected use_group_members_resource = false after :members import, got %q", got)
					}

					if got := attrs["members.#"]; got != "2" {
						return fmt.Errorf("expected 2 members in imported state, got members.# = %q", got)
					}

					return nil
				},
			},
			{
				// An unknown mode suffix must be rejected.
				ResourceName:  fqrn,
				ImportState:   true,
				ImportStateId: groupName + ":bogus",
				ExpectError:   regexp.MustCompile(`(?i)invalid import id`),
			},
		},
	})
}

// TestAccGroup_name_invalid_chars verifies the name validator rejects the
// characters the Access service forbids in group names at plan time.
func TestAccGroup_name_invalid_chars(t *testing.T) {
	_, _, groupName := testutil.MkNames("test-group", "platform_group")

	temp := `
		resource "platform_group" "{{ .groupName }}" {
			name = "{{ .groupName }}:bad"
		}
	`
	config := util.ExecuteTemplate(groupName, temp, map[string]string{"groupName": groupName})

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProviders(),
		Steps: []resource.TestStep{
			{
				Config:      config,
				ExpectError: regexp.MustCompile(`(?i)must not contain any of the following characters`),
			},
		},
	})
}

func TestAccGroup_name_too_long(t *testing.T) {
	_, _, groupName := testutil.MkNames("test-group", "platform_group")

	groupName = fmt.Sprintf("%s%s", groupName, strings.Repeat("X", 60))
	temp := `
		resource "platform_group" "{{ .groupName }}" {
			name             = "{{ .groupName }}"
			description 	 = "Test group"
			external_id      = "externalID"
			auto_join        = true
			admin_privileges = false
		}
	`

	config := util.ExecuteTemplate(groupName, temp, map[string]string{"groupName": groupName})

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProviders(),
		Steps: []resource.TestStep{
			{
				Config:      config,
				ExpectError: regexp.MustCompile(".*Attribute name string length must be between 1 and 64.*"),
			},
		},
	})
}

func TestAccGroup_update_name(t *testing.T) {
	_, fqrn, groupName := testutil.MkNames("test-group-name-", "platform_group")

	temp := `
		resource "platform_group" "{{ .groupName }}" {
			name = "{{ .groupName }}"
		}
	`
	config := util.ExecuteTemplate(groupName, temp, map[string]string{"groupName": groupName})

	updatedTemp := `
		resource "platform_group" "{{ .groupName }}" {
			name = "{{ .groupName }}-updated"
		}
	`

	updatedConfig := util.ExecuteTemplate(groupName, updatedTemp, map[string]string{"groupName": groupName})

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProviders(),
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(fqrn, "name", groupName),
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
