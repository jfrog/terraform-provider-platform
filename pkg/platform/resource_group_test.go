package platform_test

import (
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/jfrog/terraform-provider-shared/testutil"
	"github.com/jfrog/terraform-provider-shared/util"
)

func TestAccGroup_full(t *testing.T) {
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
		}
	`

	testData := map[string]string{
		"groupName": groupName,
		"autoJoin":  fmt.Sprintf("%t", testutil.RandBool()),
		"members":   "[\"anonymous\", \"admin\"]",
	}

	config := util.ExecuteTemplate(groupName, temp, testData)

	updatedTestData := map[string]string{
		"groupName": groupName,
		"autoJoin":  fmt.Sprintf("%t", testutil.RandBool()),
		"members":   "[\"admin\"]",
	}

	updatedConfig := util.ExecuteTemplate(groupName, temp, updatedTestData)

	updated2TestData := map[string]string{
		"groupName":       groupName,
		"autoJoin":        fmt.Sprintf("%t", testutil.RandBool()),
		"adminPrivileges": fmt.Sprintf("%t", testutil.RandBool()),
		"members":         "[\"anonymous\"]",
	}

	updated2Config := util.ExecuteTemplate(groupName, temp, updated2TestData)

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
				),
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
				),
			},
			{
				ResourceName:                         fqrn,
				ImportState:                          true,
				ImportStateId:                        updated2TestData["groupName"],
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

	// Step 1: Create with old provider
	resource.Test(t, resource.TestCase{
		ExternalProviders: map[string]resource.ExternalProvider{
			"platform": {
				Source:            "jfrog/platform",
				VersionConstraint: "2.1.0",
			},
		},
		Steps: []resource.TestStep{
			{
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
		},
	})

	// Step 2: Upgrade with local provider
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProviders(),
		Steps: []resource.TestStep{
			{
				Config: config,
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
