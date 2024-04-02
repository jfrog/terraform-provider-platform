package platform_test

import (
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/jfrog/terraform-provider-shared/testutil"
)

func TestAccIPAllowlist_full(t *testing.T) {
	jfrogURL := os.Getenv("JFROG_URL")
	if !strings.HasSuffix(jfrogURL, "jfrog.io") {
		t.Skipf("env var JFROG_URL '%s' is not a cloud instance. MyJFrog features are only available on cloud.", jfrogURL)
	}
	myJFrogAPIToken := os.Getenv("JFROG_MYJFROG_API_TOKEN")
	if len(myJFrogAPIToken) == 0 {
		t.Fatalf("env var JFROG_MYJFROG_API_TOKEN is not set. Please create a API token in MyJFrog portal")
	}

	_, fqrn, allowlistName := testutil.MkNames("test-myjfrog-ip-allowlist", "platform_myjfrog_ip_allowlist")

	re := regexp.MustCompile(`^https://(\w+)\.jfrog\.io$`)
	matches := re.FindStringSubmatch(jfrogURL)
	if len(matches) < 2 {
		t.Fatalf("can't find server name from JFROG_URL %s", jfrogURL)
	}
	serverName := matches[1]

	temp := `
	resource "platform_myjfrog_ip_allowlist" "{{ .name }}" {
		server_name = "{{ .serverName }}"
		ips = {{ .ips }}
	}`

	testData := map[string]string{
		"name":       allowlistName,
		"serverName": serverName,
		"ips":        `["1.1.1.7/1"]`,
	}

	config := testutil.ExecuteTemplate(allowlistName, temp, testData)

	updatedTestData := map[string]string{
		"name":       allowlistName,
		"serverName": serverName,
		"ips":        `["1.1.1.7/1", "2.2.2.7/1"]`,
	}
	updatedConfig := testutil.ExecuteTemplate(allowlistName, temp, updatedTestData)

	updatedTestData2 := map[string]string{
		"name":       allowlistName,
		"serverName": serverName,
		"ips":        `["2.2.2.7/1"]`,
	}
	updatedConfig2 := testutil.ExecuteTemplate(allowlistName, temp, updatedTestData2)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProviders(),
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(fqrn, "server_name", testData["serverName"]),
					resource.TestCheckResourceAttr(fqrn, "ips.#", "1"),
					resource.TestCheckResourceAttr(fqrn, "ips.0", "1.1.1.7/1"),
				),
			},
			{
				Config: updatedConfig,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(fqrn, "server_name", updatedTestData["serverName"]),
					resource.TestCheckResourceAttr(fqrn, "ips.#", "2"),
					resource.TestCheckTypeSetElemAttr(fqrn, "ips.*", "1.1.1.7/1"),
					resource.TestCheckTypeSetElemAttr(fqrn, "ips.*", "2.2.2.7/1"),
				),
			},
			{
				Config: updatedConfig2,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(fqrn, "server_name", testData["serverName"]),
					resource.TestCheckResourceAttr(fqrn, "ips.#", "1"),
					resource.TestCheckResourceAttr(fqrn, "ips.0", "2.2.2.7/1"),
				),
			},
			{
				ResourceName:                         fqrn,
				ImportState:                          true,
				ImportStateId:                        allowlistName,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "server_name",
			},
		},
	})
}
