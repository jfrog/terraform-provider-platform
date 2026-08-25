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
	"net/url"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/jfrog/terraform-provider-platform/v2/pkg/platform"
	"github.com/jfrog/terraform-provider-shared/testutil"
	"github.com/jfrog/terraform-provider-shared/util"
)

const testSourceCode = "export default async (context: PlatformContext, data: BeforeDownloadRequest): Promise<BeforeDownloadResponse> => { console.log(await context.clients.platformHttp.get('/artifactory/api/system/ping')); console.log(await axios.get('https://my.external.resource')); return { status: 'DOWNLOAD_PROCEED', message: 'proceed', } }"

func TestAccWorkersService_full(t *testing.T) {
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
				repo_keys = ["{{ .repoKey }}"]
			}
		}

		secrets = [
			{
				key   = "{{ .secretKey }}"
				value = "{{ .secretValue }}"
			},
			{
				key   = "{{ .secretKey2 }}"
				value = "{{ .secretValue2 }}"
			}
		]
	}`
	testData := map[string]string{
		"key":          workersServiceName,
		"enabled":      "true",
		"description":  "Description",
		"sourceCode":   testSourceCode,
		"action":       "BEFORE_DOWNLOAD",
		"repoKey":      repoKey,
		"secretKey":    "test-secret-key",
		"secretValue":  "test-secret-value",
		"secretKey2":   "test-secret-key-2",
		"secretValue2": "test-secret-value-2",
	}

	config := util.ExecuteTemplate(workersServiceName, temp, testData)

	updatedTemp := `
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
				repo_keys = ["{{ .repoKey }}"]
			}
		}

		secrets = [
			{
				key   = "{{ .secretKey }}"
				value = "{{ .secretValue }}"
			}
		]
	}`
	updatedTestData := map[string]string{
		"key":         workersServiceName,
		"enabled":     "false",
		"description": "Updated description",
		"sourceCode":  testSourceCode,
		"action":      "BEFORE_DOWNLOAD",
		"repoKey":     repoKey,
		"secretKey":   "test-secret-key",
		"secretValue": "test-secret-value",
	}
	updatedConfig := util.ExecuteTemplate(workersServiceName, updatedTemp, updatedTestData)

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
					resource.TestCheckResourceAttr(fqrn, "enabled", testData["enabled"]),
					resource.TestCheckResourceAttr(fqrn, "description", testData["description"]),
					resource.TestCheckResourceAttr(fqrn, "source_code", testData["sourceCode"]),
					resource.TestCheckResourceAttr(fqrn, "filter_criteria.artifact_filter_criteria.repo_keys.#", "1"),
					resource.TestCheckResourceAttr(fqrn, "filter_criteria.artifact_filter_criteria.repo_keys.0", testData["repoKey"]),
					resource.TestCheckResourceAttr(fqrn, "secrets.#", "2"),
					resource.TestCheckResourceAttr(fqrn, "secrets.0.key", testData["secretKey"]),
					resource.TestCheckResourceAttr(fqrn, "secrets.0.value", testData["secretValue"]),
					resource.TestCheckResourceAttr(fqrn, "secrets.1.key", testData["secretKey2"]),
					resource.TestCheckResourceAttr(fqrn, "secrets.1.value", testData["secretValue2"]),
				),
			},
			{
				Config: updatedConfig,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(fqrn, "key", workersServiceName),
					resource.TestCheckResourceAttr(fqrn, "enabled", updatedTestData["enabled"]),
					resource.TestCheckResourceAttr(fqrn, "description", updatedTestData["description"]),
					resource.TestCheckResourceAttr(fqrn, "source_code", updatedTestData["sourceCode"]),
					resource.TestCheckResourceAttr(fqrn, "filter_criteria.artifact_filter_criteria.repo_keys.#", "1"),
					resource.TestCheckResourceAttr(fqrn, "filter_criteria.artifact_filter_criteria.repo_keys.0", updatedTestData["repoKey"]),
					resource.TestCheckResourceAttr(fqrn, "secrets.#", "1"),
					resource.TestCheckResourceAttr(fqrn, "secrets.0.key", updatedTestData["secretKey"]),
					resource.TestCheckResourceAttr(fqrn, "secrets.0.value", updatedTestData["secretValue"]),
				),
			},
			{
				ResourceName:                         fqrn,
				ImportState:                          true,
				ImportStateId:                        workersServiceName,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "key",
				ImportStateVerifyIgnore:              []string{"secrets"}, // `secrets.value` attribute is not being sent via API, can't be imported
			},
		},
	})
}

func TestAccWorkersService_name_change(t *testing.T) {
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
				repo_keys = ["{{ .repoKey }}"]
			}
		}

		secrets = [
			{
				key   = "{{ .secretKey }}"
				value = "{{ .secretValue }}"
			},
			{
				key   = "{{ .secretKey2 }}"
				value = "{{ .secretValue2 }}"
			}
		]
	}`
	testData := map[string]string{
		"key":          workersServiceName,
		"enabled":      "true",
		"description":  "Description",
		"sourceCode":   testSourceCode,
		"action":       "BEFORE_DOWNLOAD",
		"repoKey":      repoKey,
		"secretKey":    "test-secret-key",
		"secretValue":  "test-secret-value",
		"secretKey2":   "test-secret-key-2",
		"secretValue2": "test-secret-value-2",
	}

	config := util.ExecuteTemplate(workersServiceName, temp, testData)

	nameChangeTemp := `
	resource "artifactory_local_generic_repository" "{{ .repoKey }}" {
		key = "{{ .repoKey }}"
	}

	resource "platform_workers_service" "{{ .key }}" {
		key         = "foobar"
		enabled     = {{ .enabled }}
		description = "{{ .description }}"
		source_code = "{{ .sourceCode }}"
		action      = "{{ .action }}"

		filter_criteria = {
			artifact_filter_criteria = {
				repo_keys = ["{{ .repoKey }}"]
			}
		}

		secrets = [
			{
				key   = "{{ .secretKey }}"
				value = "{{ .secretValue }}"
			},
			{
				key   = "{{ .secretKey2 }}"
				value = "{{ .secretValue2 }}"
			}
		]
	}`
	updatedConfig := util.ExecuteTemplate(workersServiceName, nameChangeTemp, testData)

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
					resource.TestCheckResourceAttr(fqrn, "enabled", testData["enabled"]),
					resource.TestCheckResourceAttr(fqrn, "description", testData["description"]),
					resource.TestCheckResourceAttr(fqrn, "source_code", testData["sourceCode"]),
					resource.TestCheckResourceAttr(fqrn, "filter_criteria.artifact_filter_criteria.repo_keys.#", "1"),
					resource.TestCheckResourceAttr(fqrn, "filter_criteria.artifact_filter_criteria.repo_keys.0", testData["repoKey"]),
					resource.TestCheckResourceAttr(fqrn, "secrets.#", "2"),
					resource.TestCheckResourceAttr(fqrn, "secrets.0.key", testData["secretKey"]),
					resource.TestCheckResourceAttr(fqrn, "secrets.0.value", testData["secretValue"]),
					resource.TestCheckResourceAttr(fqrn, "secrets.1.key", testData["secretKey2"]),
					resource.TestCheckResourceAttr(fqrn, "secrets.1.value", testData["secretValue2"]),
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

func testAccCheckWorkersServiceDestroy(id string) func(*terraform.State) error {
	return func(s *terraform.State) error {
		client := TestProvider.(*platform.PlatformProvider).Meta.Client

		rs, ok := s.RootModule().Resources[id]
		if !ok {
			return fmt.Errorf("err: Resource id[%s] not found", id)
		}

		var workersService platform.WorkersServiceAPIModel
		url, err := url.JoinPath(platform.WorkersServiceEndpoint, rs.Primary.Attributes["key"])
		if err != nil {
			return err
		}

		resp, err := client.R().
			SetResult(&workersService).
			Get(url)
		if err != nil {
			return err
		}

		if resp.StatusCode() == http.StatusNotFound {
			return nil
		}

		return fmt.Errorf("error: Workers Service %s still exists", rs.Primary.Attributes["key"])
	}
}

const testBeforePropertyCreate = "export default async (context: PlatformContext, data: BeforePropertyCreateRequest): Promise<BeforePropertyCreateResponse> => { console.log(await context.clients.platformHttp.get('/artifactory/api/system/ping')); console.log(await axios.get('https://my.external.resource')); return { status: 'BEFORE_PROPERTY_CREATE_PROCEED', message: 'proceed', } }"

func TestAccWorkersService_BeforePropertyCreate(t *testing.T) {
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
				repo_keys = ["{{ .repoKey }}"]
			}
		}

		secrets = [
			{
				key   = "{{ .secretKey }}"
				value = "{{ .secretValue }}"
			},
			{
				key   = "{{ .secretKey2 }}"
				value = "{{ .secretValue2 }}"
			}
		]
	}`
	testData := map[string]string{
		"key":          workersServiceName,
		"enabled":      "true",
		"description":  "Description",
		"sourceCode":   testBeforePropertyCreate,
		"action":       "BEFORE_PROPERTY_CREATE",
		"repoKey":      repoKey,
		"secretKey":    "test-secret-key",
		"secretValue":  "test-secret-value",
		"secretKey2":   "test-secret-key-2",
		"secretValue2": "test-secret-value-2",
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
					resource.TestCheckResourceAttr(fqrn, "enabled", testData["enabled"]),
					resource.TestCheckResourceAttr(fqrn, "description", testData["description"]),
					resource.TestCheckResourceAttr(fqrn, "source_code", testData["sourceCode"]),
					resource.TestCheckResourceAttr(fqrn, "filter_criteria.artifact_filter_criteria.repo_keys.#", "1"),
					resource.TestCheckResourceAttr(fqrn, "filter_criteria.artifact_filter_criteria.repo_keys.0", testData["repoKey"]),
					resource.TestCheckResourceAttr(fqrn, "secrets.#", "2"),
					resource.TestCheckResourceAttr(fqrn, "secrets.0.key", testData["secretKey"]),
					resource.TestCheckResourceAttr(fqrn, "secrets.0.value", testData["secretValue"]),
					resource.TestCheckResourceAttr(fqrn, "secrets.1.key", testData["secretKey2"]),
					resource.TestCheckResourceAttr(fqrn, "secrets.1.value", testData["secretValue2"]),
				),
			},
			{
				ResourceName:                         fqrn,
				ImportState:                          true,
				ImportStateId:                        workersServiceName,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "key",
				ImportStateVerifyIgnore:              []string{"secrets"}, // `secrets.value` attribute is not being sent via API, can't be imported
			},
		},
	})
}

const testBeforePropertyDelete = "export default async (context: PlatformContext, data: BeforePropertyDeleteRequest): Promise<BeforePropertyDeleteResponse> => { console.log(await context.clients.platformHttp.get('/artifactory/api/system/ping')); console.log(await axios.get('https://my.external.resource')); return { status: 'BEFORE_PROPERTY_DELETE_PROCEED', message: 'proceed', } }"

func TestAccWorkersService_BeforePropertyDelete(t *testing.T) {
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
				repo_keys = ["{{ .repoKey }}"]
			}
		}

		secrets = [
			{
				key   = "{{ .secretKey }}"
				value = "{{ .secretValue }}"
			},
			{
				key   = "{{ .secretKey2 }}"
				value = "{{ .secretValue2 }}"
			}
		]
	}`
	testData := map[string]string{
		"key":          workersServiceName,
		"enabled":      "true",
		"description":  "Description",
		"sourceCode":   testBeforePropertyDelete,
		"action":       "BEFORE_PROPERTY_DELETE",
		"repoKey":      repoKey,
		"secretKey":    "test-secret-key",
		"secretValue":  "test-secret-value",
		"secretKey2":   "test-secret-key-2",
		"secretValue2": "test-secret-value-2",
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
					resource.TestCheckResourceAttr(fqrn, "enabled", testData["enabled"]),
					resource.TestCheckResourceAttr(fqrn, "description", testData["description"]),
					resource.TestCheckResourceAttr(fqrn, "source_code", testData["sourceCode"]),
					resource.TestCheckResourceAttr(fqrn, "filter_criteria.artifact_filter_criteria.repo_keys.#", "1"),
					resource.TestCheckResourceAttr(fqrn, "filter_criteria.artifact_filter_criteria.repo_keys.0", testData["repoKey"]),
					resource.TestCheckResourceAttr(fqrn, "secrets.#", "2"),
					resource.TestCheckResourceAttr(fqrn, "secrets.0.key", testData["secretKey"]),
					resource.TestCheckResourceAttr(fqrn, "secrets.0.value", testData["secretValue"]),
					resource.TestCheckResourceAttr(fqrn, "secrets.1.key", testData["secretKey2"]),
					resource.TestCheckResourceAttr(fqrn, "secrets.1.value", testData["secretValue2"]),
				),
			},
			{
				ResourceName:                         fqrn,
				ImportState:                          true,
				ImportStateId:                        workersServiceName,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "key",
				ImportStateVerifyIgnore:              []string{"secrets"}, // `secrets.value` attribute is not being sent via API, can't be imported
			},
		},
	})
}

const testAfterPropertyCreate = "export default async (context: PlatformContext, data: AfterPropertyCreateRequest): Promise<AfterPropertyCreateResponse> => { console.log(await context.clients.platformHttp.get('/artifactory/api/system/ping')); console.log(await axios.get('https://my.external.resource')); return { status: 'AFTER_PROPERTY_CREATE_PROCEED', message: 'proceed', } }"

func TestAccWorkersService_AfterPropertyCreate(t *testing.T) {
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
				repo_keys = ["{{ .repoKey }}"]
			}
		}

		secrets = [
			{
				key   = "{{ .secretKey }}"
				value = "{{ .secretValue }}"
			},
			{
				key   = "{{ .secretKey2 }}"
				value = "{{ .secretValue2 }}"
			}
		]
	}`
	testData := map[string]string{
		"key":          workersServiceName,
		"enabled":      "true",
		"description":  "Description",
		"sourceCode":   testAfterPropertyCreate,
		"action":       "AFTER_PROPERTY_CREATE",
		"repoKey":      repoKey,
		"secretKey":    "test-secret-key",
		"secretValue":  "test-secret-value",
		"secretKey2":   "test-secret-key-2",
		"secretValue2": "test-secret-value-2",
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
					resource.TestCheckResourceAttr(fqrn, "enabled", testData["enabled"]),
					resource.TestCheckResourceAttr(fqrn, "description", testData["description"]),
					resource.TestCheckResourceAttr(fqrn, "source_code", testData["sourceCode"]),
					resource.TestCheckResourceAttr(fqrn, "filter_criteria.artifact_filter_criteria.repo_keys.#", "1"),
					resource.TestCheckResourceAttr(fqrn, "filter_criteria.artifact_filter_criteria.repo_keys.0", testData["repoKey"]),
					resource.TestCheckResourceAttr(fqrn, "secrets.#", "2"),
					resource.TestCheckResourceAttr(fqrn, "secrets.0.key", testData["secretKey"]),
					resource.TestCheckResourceAttr(fqrn, "secrets.0.value", testData["secretValue"]),
					resource.TestCheckResourceAttr(fqrn, "secrets.1.key", testData["secretKey2"]),
					resource.TestCheckResourceAttr(fqrn, "secrets.1.value", testData["secretValue2"]),
				),
			},
			{
				ResourceName:                         fqrn,
				ImportState:                          true,
				ImportStateId:                        workersServiceName,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "key",
				ImportStateVerifyIgnore:              []string{"secrets"}, // `secrets.value` attribute is not being sent via API, can't be imported
			},
		},
	})
}

const testAfterPropertyDelete = "export default async (context: PlatformContext, data: AfterPropertyDeleteRequest): Promise<AfterPropertyDeleteResponse> => { console.log(await context.clients.platformHttp.get('/artifactory/api/system/ping')); console.log(await axios.get('https://my.external.resource')); return { status: 'AFTER_PROPERTY_DELETE_PROCEED', message: 'proceed', } }"

func TestAccWorkersService_AfterPropertyDelete(t *testing.T) {
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
				repo_keys = ["{{ .repoKey }}"]
			}
		}

		secrets = [
			{
				key   = "{{ .secretKey }}"
				value = "{{ .secretValue }}"
			},
			{
				key   = "{{ .secretKey2 }}"
				value = "{{ .secretValue2 }}"
			}
		]
	}`
	testData := map[string]string{
		"key":          workersServiceName,
		"enabled":      "true",
		"description":  "Description",
		"sourceCode":   testAfterPropertyDelete,
		"action":       "AFTER_PROPERTY_DELETE",
		"repoKey":      repoKey,
		"secretKey":    "test-secret-key",
		"secretValue":  "test-secret-value",
		"secretKey2":   "test-secret-key-2",
		"secretValue2": "test-secret-value-2",
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
					resource.TestCheckResourceAttr(fqrn, "enabled", testData["enabled"]),
					resource.TestCheckResourceAttr(fqrn, "description", testData["description"]),
					resource.TestCheckResourceAttr(fqrn, "source_code", testData["sourceCode"]),
					resource.TestCheckResourceAttr(fqrn, "filter_criteria.artifact_filter_criteria.repo_keys.#", "1"),
					resource.TestCheckResourceAttr(fqrn, "filter_criteria.artifact_filter_criteria.repo_keys.0", testData["repoKey"]),
					resource.TestCheckResourceAttr(fqrn, "secrets.#", "2"),
					resource.TestCheckResourceAttr(fqrn, "secrets.0.key", testData["secretKey"]),
					resource.TestCheckResourceAttr(fqrn, "secrets.0.value", testData["secretValue"]),
					resource.TestCheckResourceAttr(fqrn, "secrets.1.key", testData["secretKey2"]),
					resource.TestCheckResourceAttr(fqrn, "secrets.1.value", testData["secretValue2"]),
				),
			},
			{
				ResourceName:                         fqrn,
				ImportState:                          true,
				ImportStateId:                        workersServiceName,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "key",
				ImportStateVerifyIgnore:              []string{"secrets"}, // `secrets.value` attribute is not being sent via API, can't be imported
			},
		},
	})
}

const testSchedule = "export default async (context: PlatformContext, data: ScheduledEventRequest): Promise<ScheduledEventResponse> => { console.log(await context.clients.platformHttp.get('/artifactory/api/system/ping')); console.log(await axios.get('https://my.external.resource')); return { message: 'proceed', } }"

func TestAccWorkersService_Schedule(t *testing.T) {
	jfrogURL := os.Getenv("JFROG_URL")
	if !strings.HasSuffix(jfrogURL, "jfrog.io") {
		t.Skipf("JFROG_URL '%s' is not a cloud instance. Workers Service is only available on cloud.", jfrogURL)
	}

	_, fqrn, workersServiceName := testutil.MkNames("test-workers-service-", "platform_workers_service")

	temp := `
	resource "platform_workers_service" "{{ .key }}" {
		key         = "{{ .key }}"
		enabled     = {{ .enabled }}
		description = "{{ .description }}"
		source_code = "{{ .sourceCode }}"
		action      = "{{ .action }}"

		filter_criteria = {
			schedule = {
				cron     = "{{ .cron }}"
				timezone = "{{ .timezone }}"
			}
		}

		secrets = [
			{
				key   = "{{ .secretKey }}"
				value = "{{ .secretValue }}"
			},
			{
				key   = "{{ .secretKey2 }}"
				value = "{{ .secretValue2 }}"
			}
		]
	}`
	testData := map[string]string{
		"key":          workersServiceName,
		"enabled":      "true",
		"description":  "Description",
		"sourceCode":   testSchedule,
		"action":       "SCHEDULED_EVENT",
		"cron":         "*/2 * * * *",
		"timezone":     "UTC",
		"secretKey":    "test-secret-key",
		"secretValue":  "test-secret-value",
		"secretKey2":   "test-secret-key-2",
		"secretValue2": "test-secret-value-2",
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
					resource.TestCheckResourceAttr(fqrn, "enabled", testData["enabled"]),
					resource.TestCheckResourceAttr(fqrn, "description", testData["description"]),
					resource.TestCheckResourceAttr(fqrn, "source_code", testData["sourceCode"]),
					resource.TestCheckResourceAttr(fqrn, "filter_criteria.schedule.cron", "*/2 * * * *"),
					resource.TestCheckResourceAttr(fqrn, "filter_criteria.schedule.timezone", "UTC"),
					resource.TestCheckResourceAttr(fqrn, "secrets.#", "2"),
					resource.TestCheckResourceAttr(fqrn, "secrets.0.key", testData["secretKey"]),
					resource.TestCheckResourceAttr(fqrn, "secrets.0.value", testData["secretValue"]),
					resource.TestCheckResourceAttr(fqrn, "secrets.1.key", testData["secretKey2"]),
					resource.TestCheckResourceAttr(fqrn, "secrets.1.value", testData["secretValue2"]),
				),
			},
			{
				ResourceName:                         fqrn,
				ImportState:                          true,
				ImportStateId:                        workersServiceName,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "key",
				ImportStateVerifyIgnore:              []string{"secrets"}, // `secrets.value` attribute is not being sent via API, can't be imported
			},
		},
	})
}

// `timezone` is Optional+Computed with a "UTC" default, so a schedule that omits it
// gets "UTC" from the framework in the plan while the configuration value stays null.
// Create and Update therefore have to read the plan rather than the configuration,
// otherwise the null travels into state and the apply fails with "Provider produced
// inconsistent result after apply". TestAccWorkersService_Schedule always sets
// `timezone`, so only this test exercises the defaulted path.
func TestAccWorkersService_Schedule_default_timezone(t *testing.T) {
	jfrogURL := os.Getenv("JFROG_URL")
	if !strings.HasSuffix(jfrogURL, "jfrog.io") {
		t.Skipf("JFROG_URL '%s' is not a cloud instance. Workers Service is only available on cloud.", jfrogURL)
	}

	_, fqrn, workersServiceName := testutil.MkNames("test-workers-service-", "platform_workers_service")

	temp := `
	resource "platform_workers_service" "{{ .key }}" {
		key         = "{{ .key }}"
		enabled     = {{ .enabled }}
		description = "{{ .description }}"
		source_code = "{{ .sourceCode }}"
		action      = "{{ .action }}"

		filter_criteria = {
			schedule = {
				cron = "{{ .cron }}"
			}
		}
	}`
	testData := map[string]string{
		"key":         workersServiceName,
		"enabled":     "true",
		"description": "Description",
		"sourceCode":  testSchedule,
		"action":      "SCHEDULED_EVENT",
		"cron":        "*/2 * * * *",
	}

	config := util.ExecuteTemplate(workersServiceName, temp, testData)

	updatedTestData := map[string]string{
		"key":         workersServiceName,
		"enabled":     "true",
		"description": "Updated description",
		"sourceCode":  testSchedule,
		"action":      "SCHEDULED_EVENT",
		"cron":        "*/5 * * * *",
	}
	updatedConfig := util.ExecuteTemplate(workersServiceName, temp, updatedTestData)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProviders(),
		CheckDestroy:             testAccCheckWorkersServiceDestroy(fqrn),
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(fqrn, "key", workersServiceName),
					resource.TestCheckResourceAttr(fqrn, "action", testData["action"]),
					resource.TestCheckResourceAttr(fqrn, "filter_criteria.schedule.cron", testData["cron"]),
					resource.TestCheckResourceAttr(fqrn, "filter_criteria.schedule.timezone", "UTC"),
				),
			},
			{
				RefreshState: true,
				Check:        resource.TestCheckResourceAttr(fqrn, "filter_criteria.schedule.timezone", "UTC"),
				RefreshPlanChecks: resource.RefreshPlanChecks{
					PostRefresh: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
			},
			{
				// Update reads the plan too, so the default has to survive a change
				// that leaves `timezone` omitted.
				Config: updatedConfig,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(fqrn, "description", updatedTestData["description"]),
					resource.TestCheckResourceAttr(fqrn, "filter_criteria.schedule.cron", updatedTestData["cron"]),
					resource.TestCheckResourceAttr(fqrn, "filter_criteria.schedule.timezone", "UTC"),
				),
			},
			{
				RefreshState: true,
				RefreshPlanChecks: resource.RefreshPlanChecks{
					PostRefresh: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
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

const testAfterBuildInfoSave = "export default async (context: PlatformContext, data: AfterBuildInfoSaveRequest): Promise<AfterBuildInfoSaveResponse> => { console.log(await context.clients.platformHttp.get('/artifactory/api/system/ping')); return { message: 'proceed', } }"

// AFTER_BUILD_INFO_SAVE rejects a filter, so `filter_criteria` has to be omittable.
// The refresh step covers the other half: the API answers with an empty
// `filterCriteria` object, and reading that back as anything other than null would
// leave the resource permanently drifting against a configuration that has no filter.
func TestAccWorkersService_AfterBuildInfoSave_no_filter_criteria(t *testing.T) {
	jfrogURL := os.Getenv("JFROG_URL")
	if !strings.HasSuffix(jfrogURL, "jfrog.io") {
		t.Skipf("JFROG_URL '%s' is not a cloud instance. Workers Service is only available on cloud.", jfrogURL)
	}

	_, fqrn, workersServiceName := testutil.MkNames("test-workers-service-", "platform_workers_service")

	temp := `
	resource "platform_workers_service" "{{ .key }}" {
		key         = "{{ .key }}"
		enabled     = {{ .enabled }}
		description = "{{ .description }}"
		source_code = "{{ .sourceCode }}"
		action      = "{{ .action }}"

		secrets = [
			{
				key   = "{{ .secretKey }}"
				value = "{{ .secretValue }}"
			}
		]
	}`
	testData := map[string]string{
		"key":         workersServiceName,
		"enabled":     "true",
		"description": "Description",
		"sourceCode":  testAfterBuildInfoSave,
		"action":      "AFTER_BUILD_INFO_SAVE",
		"secretKey":   "test-secret-key",
		"secretValue": "test-secret-value",
	}

	config := util.ExecuteTemplate(workersServiceName, temp, testData)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProviders(),
		CheckDestroy:             testAccCheckWorkersServiceDestroy(fqrn),
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(fqrn, "key", workersServiceName),
					resource.TestCheckResourceAttr(fqrn, "enabled", testData["enabled"]),
					resource.TestCheckResourceAttr(fqrn, "action", testData["action"]),
					resource.TestCheckNoResourceAttr(fqrn, "filter_criteria"),
				),
			},
			{
				RefreshState: true,
				Check:        resource.TestCheckNoResourceAttr(fqrn, "filter_criteria"),
				RefreshPlanChecks: resource.RefreshPlanChecks{
					PostRefresh: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
			},
			{
				ResourceName:                         fqrn,
				ImportState:                          true,
				ImportStateId:                        workersServiceName,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "key",
				ImportStateVerifyIgnore:              []string{"secrets"}, // `secrets.value` attribute is not being sent via API, can't be imported
			},
		},
	})
}

// `repo_keys` is only optional because the API accepts an artifact filter that names
// no repositories at all, as long as one of the `any_*` flags is set. The second
// configuration then adds `repo_keys` alongside the flags, which is the combination
// the API actually documents, and pins an explicit `false`: because the flags are sent
// as pointers, a `false` that the API declines to echo back would surface here as a
// non-empty plan rather than as silent drift.
func TestAccWorkersService_any_flags_with_and_without_repo_keys(t *testing.T) {
	jfrogURL := os.Getenv("JFROG_URL")
	if !strings.HasSuffix(jfrogURL, "jfrog.io") {
		t.Skipf("JFROG_URL '%s' is not a cloud instance. Workers Service is only available on cloud.", jfrogURL)
	}

	_, fqrn, workersServiceName := testutil.MkNames("test-workers-service-", "platform_workers_service")
	_, _, repoKey := testutil.MkNames("test-repo-local-", "artifactory_local_generic_repository")

	temp := `
	resource "platform_workers_service" "{{ .key }}" {
		key         = "{{ .key }}"
		enabled     = {{ .enabled }}
		description = "{{ .description }}"
		source_code = "{{ .sourceCode }}"
		action      = "{{ .action }}"

		filter_criteria = {
			artifact_filter_criteria = {
				any_local = {{ .anyLocal }}
			}
		}
	}`
	testData := map[string]string{
		"key":         workersServiceName,
		"enabled":     "true",
		"description": "Description",
		"sourceCode":  testSourceCode,
		"action":      "BEFORE_DOWNLOAD",
		"anyLocal":    "true",
	}

	config := util.ExecuteTemplate(workersServiceName, temp, testData)

	withRepoKeysTemp := `
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
				any_local     = {{ .anyLocal }}
				any_remote    = {{ .anyRemote }}
				any_federated = {{ .anyFederated }}
			}
		}
	}`
	withRepoKeysTestData := map[string]string{
		"key":          workersServiceName,
		"enabled":      "true",
		"description":  "Description",
		"sourceCode":   testSourceCode,
		"action":       "BEFORE_DOWNLOAD",
		"repoKey":      repoKey,
		"anyLocal":     "true",
		"anyRemote":    "true",
		"anyFederated": "false",
	}
	withRepoKeysConfig := util.ExecuteTemplate(workersServiceName, withRepoKeysTemp, withRepoKeysTestData)

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
					resource.TestCheckResourceAttr(fqrn, "enabled", testData["enabled"]),
					resource.TestCheckResourceAttr(fqrn, "filter_criteria.artifact_filter_criteria.any_local", testData["anyLocal"]),
					resource.TestCheckNoResourceAttr(fqrn, "filter_criteria.artifact_filter_criteria.repo_keys"),
				),
			},
			{
				RefreshState: true,
				RefreshPlanChecks: resource.RefreshPlanChecks{
					PostRefresh: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
			},
			{
				Config: withRepoKeysConfig,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(fqrn, "filter_criteria.artifact_filter_criteria.repo_keys.#", "1"),
					resource.TestCheckResourceAttr(fqrn, "filter_criteria.artifact_filter_criteria.repo_keys.0", withRepoKeysTestData["repoKey"]),
					resource.TestCheckResourceAttr(fqrn, "filter_criteria.artifact_filter_criteria.any_local", withRepoKeysTestData["anyLocal"]),
					resource.TestCheckResourceAttr(fqrn, "filter_criteria.artifact_filter_criteria.any_remote", withRepoKeysTestData["anyRemote"]),
					resource.TestCheckResourceAttr(fqrn, "filter_criteria.artifact_filter_criteria.any_federated", withRepoKeysTestData["anyFederated"]),
				),
			},
			{
				RefreshState: true,
				RefreshPlanChecks: resource.RefreshPlanChecks{
					PostRefresh: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
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

// Guards the pre-existing artifact filter shape against the `filter_criteria` Read
// guard: a populated filter must survive a refresh untouched, and the `any_*` flags
// must stay absent for workers that never set them.
func TestAccWorkersService_artifact_filter_criteria_survives_refresh(t *testing.T) {
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
				repo_keys = ["{{ .repoKey }}"]
			}
		}
	}`
	testData := map[string]string{
		"key":         workersServiceName,
		"enabled":     "true",
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
					resource.TestCheckResourceAttr(fqrn, "filter_criteria.artifact_filter_criteria.repo_keys.#", "1"),
					resource.TestCheckResourceAttr(fqrn, "filter_criteria.artifact_filter_criteria.repo_keys.0", testData["repoKey"]),
				),
			},
			{
				RefreshState: true,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(fqrn, "filter_criteria.artifact_filter_criteria.repo_keys.#", "1"),
					resource.TestCheckResourceAttr(fqrn, "filter_criteria.artifact_filter_criteria.repo_keys.0", testData["repoKey"]),
					resource.TestCheckNoResourceAttr(fqrn, "filter_criteria.artifact_filter_criteria.any_local"),
					resource.TestCheckNoResourceAttr(fqrn, "filter_criteria.artifact_filter_criteria.any_remote"),
					resource.TestCheckNoResourceAttr(fqrn, "filter_criteria.artifact_filter_criteria.any_federated"),
				),
				RefreshPlanChecks: resource.RefreshPlanChecks{
					PostRefresh: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
			},
		},
	})
}

// The four tests below exercise ValidateConfig, which reports the filter
// combinations the JFrog platform rejects. They stop at plan time, so unlike the
// tests above they run against any instance rather than only a cloud one, and they
// never reach the Workers API. Terraform line-wraps diagnostics at an unpredictable
// column, so every space in an expected message has to tolerate a newline.

// Every action other than AFTER_BUILD_INFO_SAVE is `mandatoryFilter` on
// `GET /worker/api/v2/actions`, so an enabled worker without a filter is rejected
// by the platform with "Filter must be set if worker is enabled".
func TestAccWorkersService_validate_artifact_action_requires_filter(t *testing.T) {
	_, _, workersServiceName := testutil.MkNames("test-workers-service-", "platform_workers_service")

	temp := `
	resource "platform_workers_service" "{{ .key }}" {
		key         = "{{ .key }}"
		enabled     = true
		description = "Description"
		source_code = "{{ .sourceCode }}"
		action      = "BEFORE_DOWNLOAD"
	}`
	config := util.ExecuteTemplate(workersServiceName, temp, map[string]string{
		"key":        workersServiceName,
		"sourceCode": testSourceCode,
	})

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProviders(),
		Steps: []resource.TestStep{
			{
				Config:      config,
				PlanOnly:    true,
				ExpectError: regexp.MustCompile(`filter_criteria\.artifact_filter_criteria\s+must\s+be\s+configured\s+when\s+action\s+is\s+set\s+to\s+'BEFORE_DOWNLOAD'`),
			},
		},
	})
}

// SCHEDULED_EVENT has `filterType: SCHEDULE`, so an artifact filter does not satisfy
// its mandatory filter even though the artifact filter is itself well formed.
func TestAccWorkersService_validate_scheduled_event_requires_schedule(t *testing.T) {
	_, _, workersServiceName := testutil.MkNames("test-workers-service-", "platform_workers_service")

	temp := `
	resource "platform_workers_service" "{{ .key }}" {
		key         = "{{ .key }}"
		enabled     = true
		description = "Description"
		source_code = "{{ .sourceCode }}"
		action      = "SCHEDULED_EVENT"

		filter_criteria = {
			artifact_filter_criteria = {
				any_local = true
			}
		}
	}`
	config := util.ExecuteTemplate(workersServiceName, temp, map[string]string{
		"key":        workersServiceName,
		"sourceCode": testSchedule,
	})

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProviders(),
		Steps: []resource.TestStep{
			{
				Config:      config,
				PlanOnly:    true,
				ExpectError: regexp.MustCompile(`filter_criteria\.schedule\s+must\s+be\s+configured\s+when\s+action\s+is\s+set\s+to\s+'SCHEDULED_EVENT'`),
			},
		},
	})
}

// AFTER_BUILD_INFO_SAVE is the inverse case: it is the only action with no
// `mandatoryFilter` and no `filterType`, and the platform rejects a filter for it
// with "Filter must not be set for this action".
func TestAccWorkersService_validate_after_build_info_save_rejects_filter(t *testing.T) {
	_, _, workersServiceName := testutil.MkNames("test-workers-service-", "platform_workers_service")

	temp := `
	resource "platform_workers_service" "{{ .key }}" {
		key         = "{{ .key }}"
		enabled     = true
		description = "Description"
		source_code = "{{ .sourceCode }}"
		action      = "AFTER_BUILD_INFO_SAVE"

		filter_criteria = {
			artifact_filter_criteria = {
				any_local = true
			}
		}
	}`
	config := util.ExecuteTemplate(workersServiceName, temp, map[string]string{
		"key":        workersServiceName,
		"sourceCode": testAfterBuildInfoSave,
	})

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProviders(),
		Steps: []resource.TestStep{
			{
				Config:      config,
				PlanOnly:    true,
				ExpectError: regexp.MustCompile(`filter_criteria\s+must\s+be\s+omitted\s+when\s+action\s+is\s+set\s+to\s+'AFTER_BUILD_INFO_SAVE'`),
			},
		},
	})
}

// The two configurations the platform accepts have to keep planning. The first is
// AFTER_BUILD_INFO_SAVE with no filter, the only shape that action allows. The
// second is an enabled=false worker missing a mandatory filter, which the platform
// accepts because it enforces the filter rules only for enabled workers: that case
// must produce a warning rather than an error. `ExpectNonEmptyPlan` is what lets a
// `PlanOnly` step assert "plans cleanly" for a resource that does not exist yet.
//
// Unlike the three tests above this one cannot stop at validation: asserting that a
// configuration is accepted means letting the plan run, which configures the provider
// against the platform. It therefore carries the same cloud guard as the acceptance
// tests earlier in this file. TestValidateFilterCriteria covers the same assertion
// for every action without needing an instance, and is what protects these rules in
// CI.
func TestAccWorkersService_validate_accepted_configurations_still_plan(t *testing.T) {
	jfrogURL := os.Getenv("JFROG_URL")
	if !strings.HasSuffix(jfrogURL, "jfrog.io") {
		t.Skipf("JFROG_URL '%s' is not a cloud instance. Workers Service is only available on cloud.", jfrogURL)
	}

	_, _, noFilterName := testutil.MkNames("test-workers-service-", "platform_workers_service")
	_, _, disabledName := testutil.MkNames("test-workers-service-", "platform_workers_service")

	noFilterTemp := `
	resource "platform_workers_service" "{{ .key }}" {
		key         = "{{ .key }}"
		enabled     = true
		description = "Description"
		source_code = "{{ .sourceCode }}"
		action      = "AFTER_BUILD_INFO_SAVE"
	}`
	noFilterConfig := util.ExecuteTemplate(noFilterName, noFilterTemp, map[string]string{
		"key":        noFilterName,
		"sourceCode": testAfterBuildInfoSave,
	})

	disabledTemp := `
	resource "platform_workers_service" "{{ .key }}" {
		key         = "{{ .key }}"
		enabled     = false
		description = "Description"
		source_code = "{{ .sourceCode }}"
		action      = "BEFORE_DOWNLOAD"
	}`
	disabledConfig := util.ExecuteTemplate(disabledName, disabledTemp, map[string]string{
		"key":        disabledName,
		"sourceCode": testSourceCode,
	})

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProviders(),
		Steps: []resource.TestStep{
			{
				Config:             noFilterConfig,
				PlanOnly:           true,
				ExpectNonEmptyPlan: true,
			},
			{
				Config:             disabledConfig,
				PlanOnly:           true,
				ExpectNonEmptyPlan: true,
			},
		},
	})
}
