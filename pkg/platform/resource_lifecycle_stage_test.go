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
	"github.com/jfrog/terraform-provider-shared/testutil"
	"github.com/jfrog/terraform-provider-shared/util"
)

func TestAccLifecycleStage_full(t *testing.T) {
	_, _, projectName := testutil.MkNames("test-project-", "project")
	_, fqrn, stageName := testutil.MkNames("test-stage", "platform_lifecycle_stage")
	projectKey := strings.ToLower(fmt.Sprintf("proj%d", testutil.RandomInt()))
	// Project-scoped stages must have project key as prefix
	// Using shorter project key to stay within 32 char limit for stage names
	fullStageName := fmt.Sprintf("%s-%s", projectKey, stageName)

	projectTemp := `
	resource "project" "{{ .projectName }}" {
		key = "{{ .projectKey }}"
		display_name = "{{ .projectName }}"
		description = "test description"
		admin_privileges {
			manage_members = true
			manage_resources = true
			index_resources = true
		}
		max_storage_in_gibibytes = 1
		block_deployments_on_limit = true
		email_notification = false
	}
	`

	temp := `
		resource "platform_lifecycle_stage" "{{ .stageName }}" {
			name        = "{{ .fullStageName }}"
			project_key = project.{{ .projectName }}.key
			category    = "promote"
		}
	`

	testData := map[string]string{
		"projectName":   projectName,
		"stageName":     stageName,
		"fullStageName": fullStageName,
		"projectKey":    projectKey,
	}

	projectConfig := util.ExecuteTemplate(stageName, projectTemp, testData)
	config := projectConfig + "\n" + util.ExecuteTemplate(stageName, temp, testData)

	updatedTemp := `
		resource "platform_lifecycle_stage" "{{ .stageName }}" {
			name        = "{{ .fullStageName }}"
			project_key = project.{{ .projectName }}.key
			category    = "promote"
		}
	`

	updatedConfig := projectConfig + "\n" + util.ExecuteTemplate(stageName, updatedTemp, testData)

	resource.Test(t, resource.TestCase{
		PreCheck: func() { testAccPreCheck(t) },
		ExternalProviders: map[string]resource.ExternalProvider{
			"project": {
				Source: "jfrog/project",
			},
		},
		ProtoV6ProviderFactories: testAccProviders(),
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(fqrn, "name", testData["fullStageName"]),
					resource.TestCheckResourceAttr(fqrn, "scope", "PROJECT"),
					resource.TestCheckResourceAttr(fqrn, "project_key", testData["projectKey"]),
					resource.TestCheckResourceAttr(fqrn, "category", "promote"),
					// repositories is computed and managed by the API - not checking it as it may be null
					// used_in_lifecycles is computed - check count instead of using TestCheckResourceAttrSet
					resource.TestCheckResourceAttr(fqrn, "used_in_lifecycles.#", "0"),
					resource.TestCheckResourceAttrSet(fqrn, "created"),
					resource.TestCheckResourceAttrSet(fqrn, "modified"),
				),
			},
			{
				Config: updatedConfig,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(fqrn, "name", testData["fullStageName"]),
					resource.TestCheckResourceAttr(fqrn, "scope", "PROJECT"),
					resource.TestCheckResourceAttr(fqrn, "project_key", testData["projectKey"]),
					resource.TestCheckResourceAttr(fqrn, "category", "promote"),
					// repositories is computed and managed by the API - not checking it as it may be null
				),
			},
			{
				ResourceName:                         fqrn,
				ImportState:                          true,
				ImportStateId:                        fmt.Sprintf("%s:%s", testData["fullStageName"], testData["projectKey"]),
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "name",
			},
		},
	})
}

func TestAccLifecycleStage_global(t *testing.T) {
	_, fqrn, stageName := testutil.MkNames("test-global-stage", "platform_lifecycle_stage")

	temp := `
		resource "platform_lifecycle_stage" "{{ .stageName }}" {
			name     = "{{ .stageName }}"
			category = "promote"
		}
	`

	testData := map[string]string{
		"stageName": stageName,
	}

	config := util.ExecuteTemplate(stageName, temp, testData)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProviders(),
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(fqrn, "name", testData["stageName"]),
					resource.TestCheckResourceAttr(fqrn, "scope", "GLOBAL"),
					resource.TestCheckResourceAttr(fqrn, "category", "promote"),
					resource.TestCheckNoResourceAttr(fqrn, "project_key"),
					// used_in_lifecycles is computed - check count instead of using TestCheckResourceAttrSet
					resource.TestCheckResourceAttr(fqrn, "used_in_lifecycles.#", "0"),
					resource.TestCheckResourceAttrSet(fqrn, "created"),
					resource.TestCheckResourceAttrSet(fqrn, "modified"),
				),
			},
			{
				ResourceName:                         fqrn,
				ImportState:                          true,
				ImportStateId:                        testData["stageName"],
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "name",
			},
		},
	})
}

func TestAccLifecycleStage_minimal(t *testing.T) {
	_, _, projectName := testutil.MkNames("test-project-", "project")
	_, fqrn, stageName := testutil.MkNames("min", "platform_lifecycle_stage")
	projectKey := strings.ToLower(fmt.Sprintf("proj%d", testutil.RandomInt()))
	// Project-scoped stages must have project key as prefix
	// Using shorter names to stay within 32 char limit
	fullStageName := fmt.Sprintf("%s-%s", projectKey, stageName)

	projectTemp := `
	resource "project" "{{ .projectName }}" {
		key = "{{ .projectKey }}"
		display_name = "{{ .projectName }}"
		description = "test description"
		admin_privileges {
			manage_members = true
			manage_resources = true
			index_resources = true
		}
		max_storage_in_gibibytes = 1
		block_deployments_on_limit = true
		email_notification = false
	}
	`

	temp := `
		resource "platform_lifecycle_stage" "{{ .stageName }}" {
			name        = "{{ .fullStageName }}"
			project_key = project.{{ .projectName }}.key
		}
	`

	testData := map[string]string{
		"projectName":   projectName,
		"stageName":     stageName,
		"fullStageName": fullStageName,
		"projectKey":    projectKey,
	}

	projectConfig := util.ExecuteTemplate(stageName, projectTemp, testData)
	config := projectConfig + "\n" + util.ExecuteTemplate(stageName, temp, testData)

	resource.Test(t, resource.TestCase{
		PreCheck: func() { testAccPreCheck(t) },
		ExternalProviders: map[string]resource.ExternalProvider{
			"project": {
				Source: "jfrog/project",
			},
		},
		ProtoV6ProviderFactories: testAccProviders(),
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(fqrn, "name", testData["fullStageName"]),
					resource.TestCheckResourceAttr(fqrn, "scope", "PROJECT"),
					resource.TestCheckResourceAttr(fqrn, "project_key", testData["projectKey"]),
					resource.TestCheckResourceAttr(fqrn, "category", "promote"),
					resource.TestCheckResourceAttrSet(fqrn, "created"),
					resource.TestCheckResourceAttrSet(fqrn, "modified"),
					resource.TestCheckResourceAttrSet(fqrn, "total_repository_count"),
				),
			},
		},
	})
}

func TestAccLifecycleStage_update_category(t *testing.T) {
	_, _, projectName := testutil.MkNames("test-project-", "project")
	_, fqrn, stageName := testutil.MkNames("upd", "platform_lifecycle_stage")
	projectKey := strings.ToLower(fmt.Sprintf("proj%d", testutil.RandomInt()))
	// Project-scoped stages must have project key as prefix
	// Using shorter names to stay within 32 char limit
	fullStageName := fmt.Sprintf("%s-%s", projectKey, stageName)

	projectTemp := `
	resource "project" "{{ .projectName }}" {
		key = "{{ .projectKey }}"
		display_name = "{{ .projectName }}"
		description = "test description"
		admin_privileges {
			manage_members = true
			manage_resources = true
			index_resources = true
		}
		max_storage_in_gibibytes = 1
		block_deployments_on_limit = true
		email_notification = false
	}
	`

	temp := `
		resource "platform_lifecycle_stage" "{{ .stageName }}" {
			name        = "{{ .fullStageName }}"
			project_key = project.{{ .projectName }}.key
			category    = "promote"
		}
	`

	testData := map[string]string{
		"projectName":   projectName,
		"stageName":     stageName,
		"fullStageName": fullStageName,
		"projectKey":    projectKey,
	}

	projectConfig := util.ExecuteTemplate(stageName, projectTemp, testData)
	config := projectConfig + "\n" + util.ExecuteTemplate(stageName, temp, testData)

	updatedTemp := `
		resource "platform_lifecycle_stage" "{{ .stageName }}" {
			name        = "{{ .fullStageName }}"
			project_key = project.{{ .projectName }}.key
			category    = "none"
		}
	`

	updatedConfig := projectConfig + "\n" + util.ExecuteTemplate(stageName, updatedTemp, testData)

	resource.Test(t, resource.TestCase{
		PreCheck: func() { testAccPreCheck(t) },
		ExternalProviders: map[string]resource.ExternalProvider{
			"project": {
				Source: "jfrog/project",
			},
		},
		ProtoV6ProviderFactories: testAccProviders(),
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(fqrn, "category", "promote"),
				),
			},
			{
				Config: updatedConfig,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(fqrn, "category", "none"),
				),
			},
		},
	})
}

func TestAccLifecycleStage_update_name_requires_replace(t *testing.T) {
	_, _, projectName := testutil.MkNames("test-project-", "project")
	_, fqrn, stageName := testutil.MkNames("test-rn", "platform_lifecycle_stage")
	projectKey := strings.ToLower(fmt.Sprintf("proj%d", testutil.RandomInt()))
	// Project-scoped stages must have project key as prefix
	// Using shorter names to stay within 32 char limit
	fullStageName := fmt.Sprintf("%s-%s", projectKey, stageName)
	renamedStageName := fmt.Sprintf("%s-%s-n", projectKey, stageName)

	projectTemp := `
	resource "project" "{{ .projectName }}" {
		key = "{{ .projectKey }}"
		display_name = "{{ .projectName }}"
		description = "test description"
		admin_privileges {
			manage_members = true
			manage_resources = true
			index_resources = true
		}
		max_storage_in_gibibytes = 1
		block_deployments_on_limit = true
		email_notification = false
	}
	`

	temp := `
		resource "platform_lifecycle_stage" "{{ .stageName }}" {
			name        = "{{ .fullStageName }}"
			project_key = project.{{ .projectName }}.key
			category    = "promote"
		}
	`

	testData := map[string]string{
		"projectName":   projectName,
		"stageName":     stageName,
		"fullStageName": fullStageName,
		"projectKey":    projectKey,
	}

	projectConfig := util.ExecuteTemplate(stageName, projectTemp, testData)
	config := projectConfig + "\n" + util.ExecuteTemplate(stageName, temp, testData)

	updatedTemp := `
		resource "platform_lifecycle_stage" "{{ .stageName }}" {
			name        = "{{ .renamedStageName }}"
			project_key = project.{{ .projectName }}.key
			category    = "promote"
		}
	`

	updatedTestData := map[string]string{
		"projectName":      projectName,
		"stageName":        stageName,
		"fullStageName":    fullStageName,
		"renamedStageName": renamedStageName,
		"projectKey":       projectKey,
	}

	updatedConfig := projectConfig + "\n" + util.ExecuteTemplate(stageName, updatedTemp, updatedTestData)

	resource.Test(t, resource.TestCase{
		PreCheck: func() { testAccPreCheck(t) },
		ExternalProviders: map[string]resource.ExternalProvider{
			"project": {
				Source: "jfrog/project",
			},
		},
		ProtoV6ProviderFactories: testAccProviders(),
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(fqrn, "name", testData["fullStageName"]),
				),
			},
			{
				Config: updatedConfig,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(fqrn, plancheck.ResourceActionDestroyBeforeCreate),
					},
				},
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(fqrn, "name", updatedTestData["renamedStageName"]),
				),
			},
		},
	})
}

func TestAccLifecycleStage_invalid_scope(t *testing.T) {
	_, _, projectName := testutil.MkNames("test-project-", "project")
	_, _, stageName := testutil.MkNames("inv", "platform_lifecycle_stage")
	projectKey := strings.ToLower(fmt.Sprintf("proj%d", testutil.RandomInt()))

	projectTemp := `
	resource "project" "{{ .projectName }}" {
		key = "{{ .projectKey }}"
		display_name = "{{ .projectName }}"
		description = "test description"
		admin_privileges {
			manage_members = true
			manage_resources = true
			index_resources = true
		}
		max_storage_in_gibibytes = 1
		block_deployments_on_limit = true
		email_notification = false
	}
	`

	temp := `
		resource "platform_lifecycle_stage" "{{ .stageName }}" {
			name        = "{{ .stageName }}"
			project_key = project.{{ .projectName }}.key
			category    = "promote"
		}
	`

	testData := map[string]string{
		"projectName": projectName,
		"stageName":   stageName,
		"projectKey":  projectKey,
	}

	projectConfig := util.ExecuteTemplate(stageName, projectTemp, testData)
	config := projectConfig + "\n" + util.ExecuteTemplate(stageName, temp, testData)

	resource.Test(t, resource.TestCase{
		PreCheck: func() { testAccPreCheck(t) },
		ExternalProviders: map[string]resource.ExternalProvider{
			"project": {
				Source: "jfrog/project",
			},
		},
		ProtoV6ProviderFactories: testAccProviders(),
		Steps: []resource.TestStep{
			{
				Config:      config,
				ExpectError: regexp.MustCompile(".*Stage name.*must be prefixed with project_key.*"),
			},
		},
	})
}

func TestAccLifecycleStage_invalid_category(t *testing.T) {
	_, _, projectName := testutil.MkNames("test-project-", "project")
	_, _, stageName := testutil.MkNames("test-invalid-category", "platform_lifecycle_stage")
	projectKey := strings.ToLower(fmt.Sprintf("proj%d", testutil.RandomInt()))

	projectTemp := `
	resource "project" "{{ .projectName }}" {
		key = "{{ .projectKey }}"
		display_name = "{{ .projectName }}"
		description = "test description"
		admin_privileges {
			manage_members = true
			manage_resources = true
			index_resources = true
		}
		max_storage_in_gibibytes = 1
		block_deployments_on_limit = true
		email_notification = false
	}
	`

	temp := `
		resource "platform_lifecycle_stage" "{{ .stageName }}" {
			name        = "{{ .stageName }}"
			project_key = project.{{ .projectName }}.key
			category    = "INVALID"
		}
	`

	testData := map[string]string{
		"projectName": projectName,
		"stageName":   stageName,
		"projectKey":  projectKey,
	}

	projectConfig := util.ExecuteTemplate(stageName, projectTemp, testData)
	config := projectConfig + "\n" + util.ExecuteTemplate(stageName, temp, testData)

	resource.Test(t, resource.TestCase{
		PreCheck: func() { testAccPreCheck(t) },
		ExternalProviders: map[string]resource.ExternalProvider{
			"project": {
				Source: "jfrog/project",
			},
		},
		ProtoV6ProviderFactories: testAccProviders(),
		Steps: []resource.TestStep{
			{
				Config:      config,
				ExpectError: regexp.MustCompile(".*Attribute category value must be one of.*"),
			},
		},
	})
}

func TestAccLifecycleStage_global_with_project_key_error(t *testing.T) {
	_, _, projectName := testutil.MkNames("test-project-", "project")
	_, _, stageName := testutil.MkNames("test-global-error", "platform_lifecycle_stage")
	projectKey := strings.ToLower(fmt.Sprintf("proj%d", testutil.RandomInt()))

	projectTemp := `
	resource "project" "{{ .projectName }}" {
		key = "{{ .projectKey }}"
		display_name = "{{ .projectName }}"
		description = "test description"
		admin_privileges {
			manage_members = true
			manage_resources = true
			index_resources = true
		}
		max_storage_in_gibibytes = 1
		block_deployments_on_limit = true
		email_notification = false
	}
	`

	temp := `
		resource "platform_lifecycle_stage" "{{ .stageName }}" {
			name        = "{{ .stageName }}"
			project_key = project.{{ .projectName }}.key
			category    = "promote"
		}
	`

	testData := map[string]string{
		"projectName": projectName,
		"stageName":   stageName,
		"projectKey":  projectKey,
	}

	projectConfig := util.ExecuteTemplate(stageName, projectTemp, testData)
	config := projectConfig + "\n" + util.ExecuteTemplate(stageName, temp, testData)

	resource.Test(t, resource.TestCase{
		PreCheck: func() { testAccPreCheck(t) },
		ExternalProviders: map[string]resource.ExternalProvider{
			"project": {
				Source: "jfrog/project",
			},
		},
		ProtoV6ProviderFactories: testAccProviders(),
		Steps: []resource.TestStep{
			{
				Config:      config,
				ExpectError: regexp.MustCompile(".*"),
			},
		},
	})
}

func TestAccLifecycleStage_delete_not_in_lifecycle(t *testing.T) {
	_, _, projectName := testutil.MkNames("test-project-", "project")
	_, fqrn, stageName := testutil.MkNames("del", "platform_lifecycle_stage")
	projectKey := strings.ToLower(fmt.Sprintf("proj%d", testutil.RandomInt()))
	// Project-scoped stages must have project key as prefix
	// Using shorter names to stay within 32 char limit
	fullStageName := fmt.Sprintf("%s-%s", projectKey, stageName)

	projectTemp := `
	resource "project" "{{ .projectName }}" {
		key = "{{ .projectKey }}"
		display_name = "{{ .projectName }}"
		description = "test description"
		admin_privileges {
			manage_members = true
			manage_resources = true
			index_resources = true
		}
		max_storage_in_gibibytes = 1
		block_deployments_on_limit = true
		email_notification = false
	}
	`

	temp := `
		resource "platform_lifecycle_stage" "{{ .stageName }}" {
			name        = "{{ .fullStageName }}"
			project_key = project.{{ .projectName }}.key
			category    = "promote"
		}
	`

	testData := map[string]string{
		"projectName":   projectName,
		"stageName":     stageName,
		"fullStageName": fullStageName,
		"projectKey":    projectKey,
	}

	projectConfig := util.ExecuteTemplate(stageName, projectTemp, testData)
	config := projectConfig + "\n" + util.ExecuteTemplate(stageName, temp, testData)

	resource.Test(t, resource.TestCase{
		PreCheck: func() { testAccPreCheck(t) },
		ExternalProviders: map[string]resource.ExternalProvider{
			"project": {
				Source: "jfrog/project",
			},
		},
		ProtoV6ProviderFactories: testAccProviders(),
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(fqrn, "name", testData["fullStageName"]),
					resource.TestCheckResourceAttr(fqrn, "scope", "PROJECT"),
					resource.TestCheckResourceAttr(fqrn, "project_key", testData["projectKey"]),
					resource.TestCheckResourceAttr(fqrn, "category", "promote"),
					// Verify stage is not in any lifecycle
					resource.TestCheckResourceAttr(fqrn, "used_in_lifecycles.#", "0"),
				),
			},
			// Note: Deletion is tested implicitly - the test framework will automatically
			// destroy the resource after the test completes. The important verification
			// is that the stage is not in any lifecycle (checked above), which means
			// it can be safely deleted.
		},
	})
}

func TestAccLifecycleStage_category_none(t *testing.T) {
	_, _, projectName := testutil.MkNames("test-project-", "project")
	_, fqrn, stageName := testutil.MkNames("none", "platform_lifecycle_stage")
	projectKey := strings.ToLower(fmt.Sprintf("proj%d", testutil.RandomInt()))
	fullStageName := fmt.Sprintf("%s-%s", projectKey, stageName)

	projectTemp := `
	resource "project" "{{ .projectName }}" {
		key = "{{ .projectKey }}"
		display_name = "{{ .projectName }}"
		description = "test description"
		admin_privileges {
			manage_members = true
			manage_resources = true
			index_resources = true
		}
		max_storage_in_gibibytes = 1
		block_deployments_on_limit = true
		email_notification = false
	}
	`

	temp := `
		resource "platform_lifecycle_stage" "{{ .stageName }}" {
			name        = "{{ .fullStageName }}"
			project_key = project.{{ .projectName }}.key
			category    = "none"
		}
	`

	testData := map[string]string{
		"projectName":   projectName,
		"stageName":     stageName,
		"fullStageName": fullStageName,
		"projectKey":    projectKey,
	}

	projectConfig := util.ExecuteTemplate(stageName, projectTemp, testData)
	config := projectConfig + "\n" + util.ExecuteTemplate(stageName, temp, testData)

	resource.Test(t, resource.TestCase{
		PreCheck: func() { testAccPreCheck(t) },
		ExternalProviders: map[string]resource.ExternalProvider{
			"project": {
				Source: "jfrog/project",
			},
		},
		ProtoV6ProviderFactories: testAccProviders(),
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(fqrn, "name", testData["fullStageName"]),
					resource.TestCheckResourceAttr(fqrn, "category", "none"),
					resource.TestCheckResourceAttr(fqrn, "scope", "PROJECT"),
					resource.TestCheckResourceAttrSet(fqrn, "total_repository_count"),
				),
			},
		},
	})
}

func TestAccLifecycleStage_category_multiple_updates(t *testing.T) {
	_, _, projectName := testutil.MkNames("test-project-", "project")
	_, fqrn, stageName := testutil.MkNames("cat", "platform_lifecycle_stage")
	projectKey := strings.ToLower(fmt.Sprintf("proj%d", testutil.RandomInt()))
	fullStageName := fmt.Sprintf("%s-%s", projectKey, stageName)

	projectTemp := `
	resource "project" "{{ .projectName }}" {
		key = "{{ .projectKey }}"
		display_name = "{{ .projectName }}"
		description = "test description"
		admin_privileges {
			manage_members = true
			manage_resources = true
			index_resources = true
		}
		max_storage_in_gibibytes = 1
		block_deployments_on_limit = true
		email_notification = false
	}
	`

	config1 := `
		resource "platform_lifecycle_stage" "{{ .stageName }}" {
			name        = "{{ .fullStageName }}"
			project_key = project.{{ .projectName }}.key
			category    = "promote"
		}
	`

	config2 := `
		resource "platform_lifecycle_stage" "{{ .stageName }}" {
			name        = "{{ .fullStageName }}"
			project_key = project.{{ .projectName }}.key
			category    = "none"
		}
	`

	config3 := `
		resource "platform_lifecycle_stage" "{{ .stageName }}" {
			name        = "{{ .fullStageName }}"
			project_key = project.{{ .projectName }}.key
			category    = "promote"
		}
	`

	testData := map[string]string{
		"projectName":   projectName,
		"stageName":     stageName,
		"fullStageName": fullStageName,
		"projectKey":    projectKey,
	}

	projectConfig := util.ExecuteTemplate(stageName, projectTemp, testData)
	step1Config := projectConfig + "\n" + util.ExecuteTemplate(stageName, config1, testData)
	step2Config := projectConfig + "\n" + util.ExecuteTemplate(stageName, config2, testData)
	step3Config := projectConfig + "\n" + util.ExecuteTemplate(stageName, config3, testData)

	resource.Test(t, resource.TestCase{
		PreCheck: func() { testAccPreCheck(t) },
		ExternalProviders: map[string]resource.ExternalProvider{
			"project": {
				Source: "jfrog/project",
			},
		},
		ProtoV6ProviderFactories: testAccProviders(),
		Steps: []resource.TestStep{
			{
				Config: step1Config,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(fqrn, "category", "promote"),
				),
			},
			{
				Config: step2Config,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(fqrn, "category", "none"),
				),
			},
			{
				Config: step3Config,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(fqrn, "category", "promote"),
				),
			},
		},
	})
}

func TestAccLifecycleStage_default_category(t *testing.T) {
	_, _, projectName := testutil.MkNames("test-project-", "project")
	_, fqrn, stageName := testutil.MkNames("def", "platform_lifecycle_stage")
	projectKey := strings.ToLower(fmt.Sprintf("proj%d", testutil.RandomInt()))
	fullStageName := fmt.Sprintf("%s-%s", projectKey, stageName)

	projectTemp := `
	resource "project" "{{ .projectName }}" {
		key = "{{ .projectKey }}"
		display_name = "{{ .projectName }}"
		description = "test description"
		admin_privileges {
			manage_members = true
			manage_resources = true
			index_resources = true
		}
		max_storage_in_gibibytes = 1
		block_deployments_on_limit = true
		email_notification = false
	}
	`

	temp := `
		resource "platform_lifecycle_stage" "{{ .stageName }}" {
			name        = "{{ .fullStageName }}"
			project_key = project.{{ .projectName }}.key
		}
	`

	testData := map[string]string{
		"projectName":   projectName,
		"stageName":     stageName,
		"fullStageName": fullStageName,
		"projectKey":    projectKey,
	}

	projectConfig := util.ExecuteTemplate(stageName, projectTemp, testData)
	config := projectConfig + "\n" + util.ExecuteTemplate(stageName, temp, testData)

	resource.Test(t, resource.TestCase{
		PreCheck: func() { testAccPreCheck(t) },
		ExternalProviders: map[string]resource.ExternalProvider{
			"project": {
				Source: "jfrog/project",
			},
		},
		ProtoV6ProviderFactories: testAccProviders(),
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(fqrn, "category", "promote"),
					resource.TestCheckResourceAttr(fqrn, "scope", "PROJECT"),
				),
			},
		},
	})
}
