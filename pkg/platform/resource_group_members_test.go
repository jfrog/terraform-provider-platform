package platform_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/jfrog/terraform-provider-shared/testutil"
	"github.com/jfrog/terraform-provider-shared/util"
)

func TestAccGroupMembers_full(t *testing.T) {
	_, fqrn, groupName := testutil.MkNames("test-group-members", "platform_group_members")
	id, _, name := testutil.MkNames("test-user-", "artifactory_managed_user")

	email := fmt.Sprintf("dummy_user%d@test.com", id)

	temp := `
	resource "artifactory_managed_user" "{{ .name }}" {
		name     = "{{ .name }}"
		email 	 = "{{ .email }}"
		password = "Passsw0rd!12"
	}

	resource "platform_group" "{{ .groupName }}" {
		name             = "{{ .groupName }}"
		description 	 = "Test group"
		external_id      = "externalID"
		auto_join        = true
		admin_privileges = false
	}

	resource "platform_group_members" "{{ .groupName }}" {
		name    = platform_group.{{ .groupName }}.name
		members = [
			"anonymous",
			"admin",
			artifactory_managed_user.{{ .name }}.name,
		]
	}`

	testData := map[string]string{
		"name":      name,
		"email":     email,
		"groupName": groupName,
	}

	config := util.ExecuteTemplate(groupName, temp, testData)

	updatedTemp := `
	resource "artifactory_managed_user" "{{ .name }}" {
		name     = "{{ .name }}"
		email 	 = "{{ .email }}"
		password = "Passsw0rd!12"
	}

	resource "platform_group" "{{ .groupName }}" {
		name             = "{{ .groupName }}"
		description 	 = "Test group"
		external_id      = "externalID"
		auto_join        = true
		admin_privileges = false
	}

	resource "platform_group_members" "{{ .groupName }}" {
		name    = platform_group.{{ .groupName }}.name
		members = [
			"anonymous",
			artifactory_managed_user.{{ .name }}.name,
		]
	}`

	updatedConfig := util.ExecuteTemplate(groupName, updatedTemp, testData)

	updatedTemp2 := `
	resource "artifactory_managed_user" "{{ .name }}" {
		name     = "{{ .name }}"
		email 	 = "{{ .email }}"
		password = "Passsw0rd!12"
	}

	resource "platform_group" "{{ .groupName }}" {
		name             = "{{ .groupName }}"
		description 	 = "Test group"
		external_id      = "externalID"
		auto_join        = true
		admin_privileges = false
	}

	resource "platform_group_members" "{{ .groupName }}" {
		name    = platform_group.{{ .groupName }}.name
		members = [
			"anonymous",
		]
	}`

	updated2Config := util.ExecuteTemplate(groupName, updatedTemp2, testData)

	resource.Test(t, resource.TestCase{
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
					resource.TestCheckResourceAttr(fqrn, "members.#", "3"),
					resource.TestCheckResourceAttr(fqrn, "members.0", "admin"),
					resource.TestCheckResourceAttr(fqrn, "members.1", "anonymous"),
					resource.TestCheckResourceAttr(fqrn, "members.2", testData["name"]),
				),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPostRefresh: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(fqrn, plancheck.ResourceActionNoop),
					},
				},
			},
			{
				Config: updatedConfig,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(fqrn, "name", testData["groupName"]),
					resource.TestCheckResourceAttr(fqrn, "members.#", "2"),
					resource.TestCheckResourceAttr(fqrn, "members.0", "anonymous"),
					resource.TestCheckResourceAttr(fqrn, "members.1", testData["name"]),
				),
			},
			{
				Config: updated2Config,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(fqrn, "name", testData["groupName"]),
					resource.TestCheckResourceAttr(fqrn, "members.#", "1"),
					resource.TestCheckResourceAttr(fqrn, "members.0", "anonymous"),
				),
			},
		},
	})
}
