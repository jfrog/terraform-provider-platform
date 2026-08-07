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
	"os"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/jfrog/terraform-provider-shared/testutil"
	"github.com/jfrog/terraform-provider-shared/util"
)

// TestAccWorkersService_ArtifactFilterCriteriaAnyRepo is a regression test for
// https://github.com/jfrog/terraform-provider-platform issue "Missing Fields for
// Worker creation in 'artifact_filter_criteria'". It verifies that the
// any_local / any_remote / any_federated booleans are accepted, persisted to the
// API, and read back into state (previously they were silently dropped).
func TestAccWorkersService_ArtifactFilterCriteriaAnyRepo(t *testing.T) {
	jfrogURL := os.Getenv("JFROG_URL")
	if !strings.HasSuffix(jfrogURL, "jfrog.io") {
		t.Skipf("JFROG_URL '%s' is not a cloud instance. Workers Service is only available on cloud.", jfrogURL)
	}

	_, fqrn, workersServiceName := testutil.MkNames("test-workers-service-", "platform_workers_service")
	_, _, repoKey := testutil.MkNames("test-repo-local-", "artifactory_local_generic_repository")

	temp := `
	resource "artifactory_local_generic_repository" "{{ .repoKey }}" {
		key = "{{ .repoKey }}"
	}

	resource "platform_workers_service" "{{ .key }}" {
		key         = "{{ .key }}"
		enabled     = {{ .enabled }}
		description = "{{ .description }}"
		source_code = "{{ .sourceCode }}"
		action      = "{{ .action }}"

		filter_criteria = {
			artifact_filter_criteria = {
				repo_keys     = ["{{ .repoKey }}"]
				any_local     = true
				any_remote    = true
				any_federated = false
			}
		}
	}`
	testData := map[string]string{
		"key":         workersServiceName,
		"enabled":     "false",
		"description": "Description",
		"sourceCode":  testSourceCode,
		"action":      "BEFORE_DOWNLOAD",
		"repoKey":     repoKey,
	}

	config := util.ExecuteTemplate(workersServiceName, temp, testData)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProviders(),
		ExternalProviders: map[string]resource.ExternalProvider{
			"artifactory": {
				Source:            "registry.terraform.io/jfrog/artifactory",
				VersionConstraint: "9.9.0",
			},
		},
		CheckDestroy: testAccCheckWorkersServiceDestroy(fqrn),
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(fqrn, "key", workersServiceName),
					resource.TestCheckResourceAttr(fqrn, "filter_criteria.artifact_filter_criteria.repo_keys.#", "1"),
					resource.TestCheckResourceAttr(fqrn, "filter_criteria.artifact_filter_criteria.repo_keys.0", testData["repoKey"]),
					resource.TestCheckResourceAttr(fqrn, "filter_criteria.artifact_filter_criteria.any_local", "true"),
					resource.TestCheckResourceAttr(fqrn, "filter_criteria.artifact_filter_criteria.any_remote", "true"),
					resource.TestCheckResourceAttr(fqrn, "filter_criteria.artifact_filter_criteria.any_federated", "false"),
				),
			},
			{
				ResourceName:                         fqrn,
				ImportState:                          true,
				ImportStateId:                        workersServiceName,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "key",
			},
		},
	})
}