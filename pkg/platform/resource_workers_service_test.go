package platform_test

import (
	"fmt"
	"net/http"
	"net/url"
	"os"
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
