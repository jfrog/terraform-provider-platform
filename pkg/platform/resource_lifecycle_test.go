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
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/jfrog/terraform-provider-shared/testutil"
	"github.com/jfrog/terraform-provider-shared/util"
)

// checkDestroyNoOp is a CheckDestroy that always succeeds (does not verify resources are destroyed).
func checkDestroyNoOp(*terraform.State) error { return nil }

func TestAccLifecycle_full(t *testing.T) {
	_, fqrn, lifecycleName := testutil.MkNames("test-lifecycle", "platform_lifecycle")
	_, _, projectName := testutil.MkNames("test-project-", "project")
	projectKey := strings.ToLower(fmt.Sprintf("proj%d", testutil.RandomInt()))
	_, _, stage1Name := testutil.MkNames("test-stage-1", "platform_lifecycle_stage")
	_, _, stage2Name := testutil.MkNames("test-stage-2", "platform_lifecycle_stage")
	// Project-scoped stages must have project key as prefix
	fullStage1Name := fmt.Sprintf("%s-%s", projectKey, stage1Name)
	fullStage2Name := fmt.Sprintf("%s-%s", projectKey, stage2Name)

	// First create the project and stages
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

	stagesTemp := `
		resource "platform_lifecycle_stage" "{{ .stage1Name }}" {
			name        = "{{ .fullStage1Name }}"
			project_key = project.{{ .projectName }}.key
			category    = "promote"
		}

		resource "platform_lifecycle_stage" "{{ .stage2Name }}" {
			name        = "{{ .fullStage2Name }}"
			project_key = project.{{ .projectName }}.key
			category    = "promote"
			depends_on  = [platform_lifecycle_stage.{{ .stage1Name }}]
		}
	`

	stagesData := map[string]string{
		"projectName":    projectName,
		"projectKey":     projectKey,
		"stage1Name":     stage1Name,
		"stage2Name":     stage2Name,
		"fullStage1Name": fullStage1Name,
		"fullStage2Name": fullStage2Name,
	}

	projectConfig := util.ExecuteTemplate(projectKey, projectTemp, stagesData)
	stagesConfig := util.ExecuteTemplate(projectKey, stagesTemp, stagesData)

	// Then create the lifecycle
	lifecycleTemp := `
		resource "platform_lifecycle" "{{ .resourceName }}" {
			project_key    = project.{{ .projectName }}.key
			promote_stages = [
				platform_lifecycle_stage.{{ .stage1Name }}.name,
				platform_lifecycle_stage.{{ .stage2Name }}.name
			]
		}
	`

	lifecycleData := map[string]string{
		"projectName":    projectName,
		"projectKey":     projectKey,
		"resourceName":   lifecycleName,
		"stage1Name":     stage1Name,
		"stage2Name":     stage2Name,
		"fullStage1Name": fullStage1Name,
		"fullStage2Name": fullStage2Name,
	}

	lifecycleConfig := util.ExecuteTemplate(projectKey, lifecycleTemp, lifecycleData)

	config := projectConfig + "\n" + stagesConfig + "\n" + lifecycleConfig

	// Update lifecycle with different order
	updatedLifecycleTemp := `
		resource "platform_lifecycle" "{{ .resourceName }}" {
			project_key    = project.{{ .projectName }}.key
			promote_stages = [
				platform_lifecycle_stage.{{ .stage2Name }}.name,
				platform_lifecycle_stage.{{ .stage1Name }}.name
			]
		}
	`

	updatedConfig := projectConfig + "\n" + stagesConfig + "\n" + util.ExecuteTemplate(projectKey, updatedLifecycleTemp, lifecycleData)

	// Teardown: clear promote_stages so stage destroy can succeed (no API clear in resource Delete)
	lifecycleEmptyTemp := `
		resource "platform_lifecycle" "{{ .resourceName }}" {
			project_key    = project.{{ .projectName }}.key
			promote_stages = []
		}
	`
	teardownConfig := projectConfig + "\n" + stagesConfig + "\n" + util.ExecuteTemplate(projectKey, lifecycleEmptyTemp, lifecycleData)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		CheckDestroy: checkDestroyNoOp,
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
					resource.TestCheckResourceAttr(fqrn, "project_key", projectKey),
					resource.TestCheckResourceAttr(fqrn, "promote_stages.#", "2"),
					resource.TestCheckResourceAttr(fqrn, "promote_stages.0", fullStage1Name),
					resource.TestCheckResourceAttr(fqrn, "promote_stages.1", fullStage2Name),
					resource.TestCheckResourceAttrSet(fqrn, "release_stage"),
					resource.TestCheckResourceAttrSet(fqrn, "categories.#"),
				),
			},
			{
				Config: updatedConfig,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(fqrn, "project_key", projectKey),
					resource.TestCheckResourceAttr(fqrn, "promote_stages.#", "2"),
					resource.TestCheckResourceAttr(fqrn, "promote_stages.0", fullStage2Name),
					resource.TestCheckResourceAttr(fqrn, "promote_stages.1", fullStage1Name),
				),
			},
			{
				ResourceName:                         fqrn,
				ImportState:                          true,
				ImportStateId:                        projectKey,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "project_key",
			},
			{
				Config: teardownConfig,
				Check:  resource.TestCheckResourceAttr(fqrn, "promote_stages.#", "0"),
			},
		},
	})
}

func TestAccLifecycle_single_stage(t *testing.T) {
	_, fqrn, lifecycleName := testutil.MkNames("test-lifecycle", "platform_lifecycle")
	_, _, projectName := testutil.MkNames("test-project-", "project")
	projectKey := strings.ToLower(fmt.Sprintf("proj%d", testutil.RandomInt()))
	_, _, stageName := testutil.MkNames("s1", "platform_lifecycle_stage")
	// Project-scoped stages must have project key as prefix
	// Using shorter names to stay within 32 char limit
	fullStageName := fmt.Sprintf("%s-%s", projectKey, stageName)

	// Create project and stage
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

	stagesTemp := `
		resource "platform_lifecycle_stage" "{{ .stageName }}" {
			name        = "{{ .fullStageName }}"
			project_key = project.{{ .projectName }}.key
			category    = "promote"
		}
	`

	stagesData := map[string]string{
		"projectName":   projectName,
		"projectKey":    projectKey,
		"stageName":     stageName,
		"fullStageName": fullStageName,
	}

	projectConfig := util.ExecuteTemplate(projectKey, projectTemp, stagesData)
	stagesConfig := util.ExecuteTemplate(projectKey, stagesTemp, stagesData)

	// Create lifecycle with single stage
	lifecycleTemp := `
		resource "platform_lifecycle" "{{ .resourceName }}" {
			project_key    = project.{{ .projectName }}.key
			promote_stages = [
				platform_lifecycle_stage.{{ .stageName }}.name
			]
		}
	`

	lifecycleData := map[string]string{
		"projectName":   projectName,
		"projectKey":    projectKey,
		"resourceName":  lifecycleName,
		"stageName":     stageName,
		"fullStageName": fullStageName,
	}

	lifecycleConfig := util.ExecuteTemplate(projectKey, lifecycleTemp, lifecycleData)

	config := projectConfig + "\n" + stagesConfig + "\n" + lifecycleConfig

	// Teardown: clear promote_stages so stage destroy can succeed
	lifecycleEmptyTemp := `
		resource "platform_lifecycle" "{{ .resourceName }}" {
			project_key    = project.{{ .projectName }}.key
			promote_stages = []
		}
	`
	teardownConfig := projectConfig + "\n" + stagesConfig + "\n" + util.ExecuteTemplate(projectKey, lifecycleEmptyTemp, lifecycleData)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		CheckDestroy: checkDestroyNoOp,
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
					resource.TestCheckResourceAttr(fqrn, "project_key", projectKey),
					resource.TestCheckResourceAttr(fqrn, "promote_stages.#", "1"),
					resource.TestCheckResourceAttr(fqrn, "promote_stages.0", fullStageName),
					resource.TestCheckResourceAttrSet(fqrn, "release_stage"),
					resource.TestCheckResourceAttrSet(fqrn, "categories.#"),
				),
			},
			{
				Config: teardownConfig,
				Check:  resource.TestCheckResourceAttr(fqrn, "promote_stages.#", "0"),
			},
		},
	})
}

func TestAccLifecycle_update_stages(t *testing.T) {
	_, fqrn, lifecycleName := testutil.MkNames("test-lifecycle", "platform_lifecycle")
	_, _, projectName := testutil.MkNames("test-project-", "project")
	projectKey := strings.ToLower(fmt.Sprintf("proj%d", testutil.RandomInt()))
	_, _, stage1Name := testutil.MkNames("test-stage-1", "platform_lifecycle_stage")
	_, _, stage2Name := testutil.MkNames("test-stage-2", "platform_lifecycle_stage")
	_, _, stage3Name := testutil.MkNames("test-stage-3", "platform_lifecycle_stage")
	// Project-scoped stages must have project key as prefix
	fullStage1Name := fmt.Sprintf("%s-%s", projectKey, stage1Name)
	fullStage2Name := fmt.Sprintf("%s-%s", projectKey, stage2Name)
	fullStage3Name := fmt.Sprintf("%s-%s", projectKey, stage3Name)

	// Create project and all stages
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

	stagesTemp := `
		resource "platform_lifecycle_stage" "{{ .stage1Name }}" {
			name        = "{{ .fullStage1Name }}"
			project_key = project.{{ .projectName }}.key
			category    = "promote"
		}

		resource "platform_lifecycle_stage" "{{ .stage2Name }}" {
			name        = "{{ .fullStage2Name }}"
			project_key = project.{{ .projectName }}.key
			category    = "promote"
			depends_on  = [platform_lifecycle_stage.{{ .stage1Name }}]
		}

		resource "platform_lifecycle_stage" "{{ .stage3Name }}" {
			name        = "{{ .fullStage3Name }}"
			project_key = project.{{ .projectName }}.key
			category    = "promote"
			depends_on  = [platform_lifecycle_stage.{{ .stage2Name }}]
		}
	`

	stagesData := map[string]string{
		"projectName":    projectName,
		"projectKey":     projectKey,
		"stage1Name":     stage1Name,
		"stage2Name":     stage2Name,
		"stage3Name":     stage3Name,
		"fullStage1Name": fullStage1Name,
		"fullStage2Name": fullStage2Name,
		"fullStage3Name": fullStage3Name,
	}

	projectConfig := util.ExecuteTemplate(projectKey, projectTemp, stagesData)
	stagesConfig := util.ExecuteTemplate(projectKey, stagesTemp, stagesData)

	// Initial lifecycle with 2 stages
	lifecycleTemp1 := `
		resource "platform_lifecycle" "{{ .resourceName }}" {
			project_key    = project.{{ .projectName }}.key
			promote_stages = [
				platform_lifecycle_stage.{{ .stage1Name }}.name,
				platform_lifecycle_stage.{{ .stage2Name }}.name
			]
		}
	`

	lifecycleData := map[string]string{
		"projectName":    projectName,
		"projectKey":     projectKey,
		"resourceName":   lifecycleName,
		"stage1Name":     stage1Name,
		"stage2Name":     stage2Name,
		"stage3Name":     stage3Name,
		"fullStage1Name": fullStage1Name,
		"fullStage2Name": fullStage2Name,
		"fullStage3Name": fullStage3Name,
	}

	config := projectConfig + "\n" + stagesConfig + "\n" + util.ExecuteTemplate(projectKey, lifecycleTemp1, lifecycleData)

	// Updated lifecycle with 3 stages
	lifecycleTemp2 := `
		resource "platform_lifecycle" "{{ .resourceName }}" {
			project_key    = project.{{ .projectName }}.key
			promote_stages = [
				platform_lifecycle_stage.{{ .stage1Name }}.name,
				platform_lifecycle_stage.{{ .stage2Name }}.name,
				platform_lifecycle_stage.{{ .stage3Name }}.name
			]
		}
	`

	updatedConfig := projectConfig + "\n" + stagesConfig + "\n" + util.ExecuteTemplate(projectKey, lifecycleTemp2, lifecycleData)

	// Updated lifecycle with different order
	lifecycleTemp3 := `
		resource "platform_lifecycle" "{{ .resourceName }}" {
			project_key    = project.{{ .projectName }}.key
			promote_stages = [
				platform_lifecycle_stage.{{ .stage3Name }}.name,
				platform_lifecycle_stage.{{ .stage1Name }}.name
			]
		}
	`

	updatedConfig2 := projectConfig + "\n" + stagesConfig + "\n" + util.ExecuteTemplate(projectKey, lifecycleTemp3, lifecycleData)

	// Teardown: clear promote_stages so stage destroy can succeed
	lifecycleEmptyTemp := `
		resource "platform_lifecycle" "{{ .resourceName }}" {
			project_key    = project.{{ .projectName }}.key
			promote_stages = []
		}
	`
	teardownConfig := projectConfig + "\n" + stagesConfig + "\n" + util.ExecuteTemplate(projectKey, lifecycleEmptyTemp, lifecycleData)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		CheckDestroy: checkDestroyNoOp,
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
					resource.TestCheckResourceAttr(fqrn, "promote_stages.#", "2"),
					resource.TestCheckResourceAttr(fqrn, "promote_stages.0", fullStage1Name),
					resource.TestCheckResourceAttr(fqrn, "promote_stages.1", fullStage2Name),
				),
			},
			{
				Config: updatedConfig,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(fqrn, "promote_stages.#", "3"),
					resource.TestCheckResourceAttr(fqrn, "promote_stages.0", fullStage1Name),
					resource.TestCheckResourceAttr(fqrn, "promote_stages.1", fullStage2Name),
					resource.TestCheckResourceAttr(fqrn, "promote_stages.2", fullStage3Name),
				),
			},
			{
				Config: updatedConfig2,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(fqrn, "promote_stages.#", "2"),
					resource.TestCheckResourceAttr(fqrn, "promote_stages.0", fullStage3Name),
					resource.TestCheckResourceAttr(fqrn, "promote_stages.1", fullStage1Name),
				),
			},
			{
				Config: teardownConfig,
				Check:  resource.TestCheckResourceAttr(fqrn, "promote_stages.#", "0"),
			},
		},
	})
}

func TestAccLifecycle_empty_stages(t *testing.T) {
	_, fqrn, lifecycleName := testutil.MkNames("test-lifecycle", "platform_lifecycle")
	_, _, projectName := testutil.MkNames("test-project-", "project")
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

	lifecycleTemp := `
		resource "platform_lifecycle" "{{ .resourceName }}" {
			project_key    = project.{{ .projectName }}.key
			promote_stages = []
		}
	`

	testData := map[string]string{
		"projectName":  projectName,
		"projectKey":   projectKey,
		"resourceName": lifecycleName,
	}

	projectConfig := util.ExecuteTemplate(projectKey, projectTemp, testData)
	config := projectConfig + "\n" + util.ExecuteTemplate(projectKey, lifecycleTemp, testData)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		CheckDestroy: checkDestroyNoOp,
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
					resource.TestCheckResourceAttr(fqrn, "project_key", projectKey),
					resource.TestCheckResourceAttr(fqrn, "promote_stages.#", "0"),
					resource.TestCheckResourceAttrSet(fqrn, "release_stage"),
					resource.TestCheckResourceAttrSet(fqrn, "categories.#"),
				),
			},
		},
	})
}

func TestAccLifecycle_update_empty_stages(t *testing.T) {
	_, fqrn, lifecycleName := testutil.MkNames("test-lifecycle", "platform_lifecycle")
	_, _, projectName := testutil.MkNames("test-project-", "project")
	projectKey := strings.ToLower(fmt.Sprintf("proj%d", testutil.RandomInt()))
	_, _, stageName := testutil.MkNames("s1", "platform_lifecycle_stage")
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

	stagesTemp := `
		resource "platform_lifecycle_stage" "{{ .stageName }}" {
			name        = "{{ .fullStageName }}"
			project_key = project.{{ .projectName }}.key
			category    = "promote"
		}
	`

	lifecycleTemp1 := `
		resource "platform_lifecycle" "{{ .resourceName }}" {
			project_key    = project.{{ .projectName }}.key
			promote_stages = [
				platform_lifecycle_stage.{{ .stageName }}.name
			]
		}
	`

	lifecycleTemp2 := `
		resource "platform_lifecycle" "{{ .resourceName }}" {
			project_key    = project.{{ .projectName }}.key
			promote_stages = []
		}
	`

	testData := map[string]string{
		"projectName":   projectName,
		"projectKey":    projectKey,
		"resourceName":  lifecycleName,
		"stageName":     stageName,
		"fullStageName": fullStageName,
	}

	projectConfig := util.ExecuteTemplate(projectKey, projectTemp, testData)
	stagesConfig := util.ExecuteTemplate(projectKey, stagesTemp, testData)
	config := projectConfig + "\n" + stagesConfig + "\n" + util.ExecuteTemplate(projectKey, lifecycleTemp1, testData)
	updatedConfig := projectConfig + "\n" + stagesConfig + "\n" + util.ExecuteTemplate(projectKey, lifecycleTemp2, testData)
	// Last step (updatedConfig) already has promote_stages = [], so teardown/destroy can succeed

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		CheckDestroy: checkDestroyNoOp,
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
					resource.TestCheckResourceAttr(fqrn, "promote_stages.#", "1"),
				),
			},
			{
				Config: updatedConfig,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(fqrn, "promote_stages.#", "0"),
					// PROD will remain as it's system-managed
					resource.TestCheckResourceAttrSet(fqrn, "release_stage"),
				),
			},
		},
	})
}

func TestAccLifecycle_many_stages(t *testing.T) {
	_, fqrn, lifecycleName := testutil.MkNames("test-lifecycle", "platform_lifecycle")
	_, _, projectName := testutil.MkNames("test-project-", "project")
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

	// Create 5 stages in order so project environments stay deterministic (API ordering requirement)
	stagesTemp := `
		resource "platform_lifecycle_stage" "s1" {
			name        = "{{ .projectKey }}-s1"
			project_key = project.{{ .projectName }}.key
			category    = "promote"
		}
		resource "platform_lifecycle_stage" "s2" {
			name        = "{{ .projectKey }}-s2"
			project_key = project.{{ .projectName }}.key
			category    = "promote"
			depends_on  = [platform_lifecycle_stage.s1]
		}
		resource "platform_lifecycle_stage" "s3" {
			name        = "{{ .projectKey }}-s3"
			project_key = project.{{ .projectName }}.key
			category    = "promote"
			depends_on  = [platform_lifecycle_stage.s2]
		}
		resource "platform_lifecycle_stage" "s4" {
			name        = "{{ .projectKey }}-s4"
			project_key = project.{{ .projectName }}.key
			category    = "promote"
			depends_on  = [platform_lifecycle_stage.s3]
		}
		resource "platform_lifecycle_stage" "s5" {
			name        = "{{ .projectKey }}-s5"
			project_key = project.{{ .projectName }}.key
			category    = "promote"
			depends_on  = [platform_lifecycle_stage.s4]
		}
	`

	lifecycleTemp := `
		resource "platform_lifecycle" "{{ .resourceName }}" {
			project_key    = project.{{ .projectName }}.key
			promote_stages = [
				platform_lifecycle_stage.s1.name,
				platform_lifecycle_stage.s2.name,
				platform_lifecycle_stage.s3.name,
				platform_lifecycle_stage.s4.name,
				platform_lifecycle_stage.s5.name
			]
		}
	`

	testData := map[string]string{
		"projectName":  projectName,
		"projectKey":   projectKey,
		"resourceName": lifecycleName,
	}

	projectConfig := util.ExecuteTemplate(projectKey, projectTemp, testData)
	stagesConfig := util.ExecuteTemplate(projectKey, stagesTemp, testData)
	config := projectConfig + "\n" + stagesConfig + "\n" + util.ExecuteTemplate(projectKey, lifecycleTemp, testData)

	// Teardown: clear promote_stages so stage destroy can succeed
	lifecycleEmptyTemp := `
		resource "platform_lifecycle" "{{ .resourceName }}" {
			project_key    = project.{{ .projectName }}.key
			promote_stages = []
		}
	`
	teardownConfig := projectConfig + "\n" + stagesConfig + "\n" + util.ExecuteTemplate(projectKey, lifecycleEmptyTemp, testData)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		CheckDestroy: checkDestroyNoOp,
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
					resource.TestCheckResourceAttr(fqrn, "promote_stages.#", "5"),
					resource.TestCheckResourceAttr(fqrn, "promote_stages.0", fmt.Sprintf("%s-s1", projectKey)),
					resource.TestCheckResourceAttr(fqrn, "promote_stages.4", fmt.Sprintf("%s-s5", projectKey)),
				),
			},
			{
				Config: teardownConfig,
				Check:  resource.TestCheckResourceAttr(fqrn, "promote_stages.#", "0"),
			},
		},
	})
}

func TestAccLifecycle_global_lifecycle(t *testing.T) {
	_, fqrn, lifecycleName := testutil.MkNames("test-global-lifecycle", "platform_lifecycle")
	_, _, stageName1 := testutil.MkNames("g1", "platform_lifecycle_stage")
	_, _, stageName2 := testutil.MkNames("g2", "platform_lifecycle_stage")

	stagesTemp := `
		resource "platform_lifecycle_stage" "{{ .stageName1 }}" {
			name     = "{{ .stageName1 }}"
			category = "promote"
		}
		resource "platform_lifecycle_stage" "{{ .stageName2 }}" {
			name     = "{{ .stageName2 }}"
			category = "promote"
		}
	`

	lifecycleTemp := `
		resource "platform_lifecycle" "{{ .resourceName }}" {
			promote_stages = [
				platform_lifecycle_stage.{{ .stageName1 }}.name,
				platform_lifecycle_stage.{{ .stageName2 }}.name
			]
		}
	`

	testData := map[string]string{
		"resourceName": lifecycleName,
		"stageName1":   stageName1,
		"stageName2":   stageName2,
	}

	config := util.ExecuteTemplate(lifecycleName, stagesTemp, testData) + "\n" + util.ExecuteTemplate(lifecycleName, lifecycleTemp, testData)

	// Teardown: clear promote_stages (global lifecycle) so stage destroy can succeed
	lifecycleEmptyTemp := `
		resource "platform_lifecycle" "{{ .resourceName }}" {
			promote_stages = []
		}
	`
	teardownConfig := util.ExecuteTemplate(lifecycleName, stagesTemp, testData) + "\n" + util.ExecuteTemplate(lifecycleName, lifecycleEmptyTemp, testData)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		CheckDestroy:             checkDestroyNoOp,
		ProtoV6ProviderFactories: testAccProviders(),
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckNoResourceAttr(fqrn, "project_key"),
					resource.TestCheckResourceAttr(fqrn, "promote_stages.#", "2"),
					resource.TestCheckResourceAttr(fqrn, "promote_stages.0", testData["stageName1"]),
					resource.TestCheckResourceAttr(fqrn, "promote_stages.1", testData["stageName2"]),
				),
			},
			{
				Config: teardownConfig,
				Check:  resource.TestCheckResourceAttr(fqrn, "promote_stages.#", "0"),
			},
		},
	})
}
