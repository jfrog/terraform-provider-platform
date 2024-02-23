package platform_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/jfrog/terraform-provider-shared/testutil"
)

func TestAccPermission_full(t *testing.T) {
	_, fqrn, permissionName := testutil.MkNames("test-permission", "platform_permission")
	_, _, userName := testutil.MkNames("test-user", "artifactory_managed_user")
	_, _, groupName := testutil.MkNames("test-group", "artifactory_group")
	_, _, repoName := testutil.MkNames("test-local-repo", "artifactory_local_generic_repository")

	temp := `
	resource "artifactory_managed_user" "{{ .userName }}" {
		name = "{{ .userName }}"
		email = "{{ .userName }}@tempurl.org"
	}

	resource "artifactory_group" "{{ .groupName }}" {
		name = "{{ .groupName }}"
	}

	resource "artifactory_local_generic_repository" "{{ .repoName }}" {
		key = "{{ .repoName }}"
	}

	resource "platform_permission" "{{ .name }}" {
		name = "{{ .name }}"

		artifact = {
			actions = {
				users  = {
					(artifactory_managed_user.{{ .userName }}.name) = ["READ"]
				}

				groups = {
					(artifactory_group.{{ .groupName }}.name) = ["READ"]
				}
			}

			targets = {
				(artifactory_local_generic_repository.{{ .repoName }}.key) = {
					include_patterns = ["**"]
					exclude_patterns = ["{{ .excludePattern }}"]
				}
			}
		}
	}`

	updatedTemp := `
	resource "artifactory_managed_user" "{{ .userName }}" {
		name = "{{ .userName }}"
		email = "{{ .userName }}@tempurl.org"
	}

	resource "artifactory_group" "{{ .groupName }}" {
		name = "{{ .groupName }}"
	}

	resource "artifactory_local_generic_repository" "{{ .repoName }}" {
		key = "{{ .repoName }}"
	}

	resource "platform_permission" "{{ .name }}" {
		name = "{{ .name }}"

		artifact = {
			actions = {
				users  = {
					(artifactory_managed_user.{{ .userName }}.name) = ["READ", "WRITE"]
				}

				groups = {
					(artifactory_group.{{ .groupName }}.name) = ["READ", "ANNOTATE"]
				}
			}

			targets = {
				(artifactory_local_generic_repository.{{ .repoName }}.key) = {
					include_patterns = ["**", "*.js"]
					exclude_patterns = ["{{ .excludePattern }}", "bar"]
				}
			}
		}

		build = {
			targets = {
				artifactory-build-info = {
					include_patterns = ["**"]
					exclude_patterns = ["{{ .excludePattern }}"]
				}
			}
		}
	}`

	testData := map[string]string{
		"name":           permissionName,
		"userName":       userName,
		"groupName":      groupName,
		"repoName":       repoName,
		"excludePattern": "foo",
	}

	config := testutil.ExecuteTemplate(permissionName, temp, testData)
	updatedConfig := testutil.ExecuteTemplate(permissionName, updatedTemp, testData)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProviders(),
		ExternalProviders: map[string]resource.ExternalProvider{
			"artifactory": {
				Source: "jfrog/artifactory",
			},
		},
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(fqrn, "name", testData["name"]),
					resource.TestCheckResourceAttr(fqrn, fmt.Sprintf("artifact.actions.users.%s.#", testData["userName"]), "1"),
					resource.TestCheckResourceAttr(fqrn, fmt.Sprintf("artifact.actions.users.%s.0", testData["userName"]), "READ"),
					resource.TestCheckResourceAttr(fqrn, fmt.Sprintf("artifact.actions.groups.%s.#", testData["groupName"]), "1"),
					resource.TestCheckResourceAttr(fqrn, fmt.Sprintf("artifact.actions.groups.%s.0", testData["groupName"]), "READ"),
					resource.TestCheckResourceAttr(fqrn, fmt.Sprintf("artifact.targets.%s.include_patterns.#", testData["repoName"]), "1"),
					resource.TestCheckResourceAttr(fqrn, fmt.Sprintf("artifact.targets.%s.include_patterns.0", testData["repoName"]), "**"),
					resource.TestCheckResourceAttr(fqrn, fmt.Sprintf("artifact.targets.%s.exclude_patterns.#", testData["repoName"]), "1"),
					resource.TestCheckResourceAttr(fqrn, fmt.Sprintf("artifact.targets.%s.exclude_patterns.0", testData["repoName"]), testData["excludePattern"]),
				),
			},
			{
				Config: updatedConfig,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(fqrn, "name", testData["name"]),
					resource.TestCheckResourceAttr(fqrn, fmt.Sprintf("artifact.actions.users.%s.#", testData["userName"]), "2"),
					resource.TestCheckTypeSetElemAttr(fqrn, fmt.Sprintf("artifact.actions.users.%s.*", testData["userName"]), "READ"),
					resource.TestCheckTypeSetElemAttr(fqrn, fmt.Sprintf("artifact.actions.users.%s.*", testData["userName"]), "WRITE"),
					resource.TestCheckResourceAttr(fqrn, fmt.Sprintf("artifact.actions.groups.%s.#", testData["groupName"]), "2"),
					resource.TestCheckTypeSetElemAttr(fqrn, fmt.Sprintf("artifact.actions.groups.%s.*", testData["groupName"]), "READ"),
					resource.TestCheckTypeSetElemAttr(fqrn, fmt.Sprintf("artifact.actions.groups.%s.*", testData["groupName"]), "ANNOTATE"),
					resource.TestCheckResourceAttr(fqrn, fmt.Sprintf("artifact.targets.%s.include_patterns.#", testData["repoName"]), "2"),
					resource.TestCheckTypeSetElemAttr(fqrn, fmt.Sprintf("artifact.targets.%s.include_patterns.*", testData["repoName"]), "**"),
					resource.TestCheckTypeSetElemAttr(fqrn, fmt.Sprintf("artifact.targets.%s.include_patterns.*", testData["repoName"]), "*.js"),
					resource.TestCheckResourceAttr(fqrn, fmt.Sprintf("artifact.targets.%s.exclude_patterns.#", testData["repoName"]), "2"),
					resource.TestCheckTypeSetElemAttr(fqrn, fmt.Sprintf("artifact.targets.%s.exclude_patterns.*", testData["repoName"]), testData["excludePattern"]),
					resource.TestCheckTypeSetElemAttr(fqrn, fmt.Sprintf("artifact.targets.%s.exclude_patterns.*", testData["repoName"]), "bar"),
					resource.TestCheckNoResourceAttr(fqrn, "build.actions"),
					resource.TestCheckResourceAttr(fqrn, "build.targets.artifactory-build-info.include_patterns.#", "1"),
					resource.TestCheckResourceAttr(fqrn, "build.targets.artifactory-build-info.include_patterns.0", "**"),
					resource.TestCheckResourceAttr(fqrn, "build.targets.artifactory-build-info.exclude_patterns.#", "1"),
					resource.TestCheckResourceAttr(fqrn, "build.targets.artifactory-build-info.exclude_patterns.0", testData["excludePattern"]),
				),
			},
			{
				ResourceName:                         fqrn,
				ImportState:                          true,
				ImportStateId:                        permissionName,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "name",
			},
		},
	})
}
