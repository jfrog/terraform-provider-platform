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
	"os"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/jfrog/terraform-provider-shared/testutil"
	"github.com/jfrog/terraform-provider-shared/util"
)

func readLicense(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return strings.TrimSuffix(string(data), "\n"), nil
}

func TestAccLicense_full(t *testing.T) {
	_, fqrn, licenseName := testutil.MkNames("test-license", "platform_license")

	temp := `
	resource "platform_license" "{{ .name }}" {
		name = "{{ .name }}"
		key = <<EOT
{{ .key }}
EOT
	}`

	licenseFilePath1 := os.Getenv("JFROG_LICENSE_PATH_1")
	licenseFilePath2 := os.Getenv("JFROG_LICENSE_PATH_2")
	if len(licenseFilePath1) == 0 || len(licenseFilePath2) == 0 {
		t.Skip("env var JFROG_LICENSE_PATH_1 or JFROG_LICENSE_PATH_2 is not set")
	}

	licenseKey, err := readLicense(licenseFilePath1)
	if err != nil {
		t.Fatalf("failed to read license file: %v", err)
	}
	testData := map[string]string{
		"name": licenseName,
		"key":  licenseKey,
	}

	config := util.ExecuteTemplate(licenseName, temp, testData)

	licenseKey, err = readLicense(licenseFilePath2)
	if err != nil {
		t.Fatalf("failed to read license file: %v", err)
	}
	updatedTestData := map[string]string{
		"name": licenseName,
		"key":  licenseKey,
	}
	updatedConfig := util.ExecuteTemplate(licenseName, temp, updatedTestData)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProviders(),
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(fqrn, "name", testData["name"]),
					resource.TestCheckResourceAttr(fqrn, "key", fmt.Sprintf("%s\n", testData["key"])),
					resource.TestCheckResourceAttrSet(fqrn, "type"),
					resource.TestCheckResourceAttrSet(fqrn, "valid_through"),
					resource.TestCheckResourceAttrSet(fqrn, "licensed_to"),
				),
			},
			{
				Config: updatedConfig,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(fqrn, "name", updatedTestData["name"]),
					resource.TestCheckResourceAttr(fqrn, "key", fmt.Sprintf("%s\n", updatedTestData["key"])),
					resource.TestCheckResourceAttrSet(fqrn, "type"),
					resource.TestCheckResourceAttrSet(fqrn, "valid_through"),
					resource.TestCheckResourceAttrSet(fqrn, "licensed_to"),
				),
			},
		},
	})
}
