package platform_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/jfrog/terraform-provider-platform/pkg/platform"
	"github.com/jfrog/terraform-provider-shared/testutil"
	"github.com/jfrog/terraform-provider-shared/util"
)

func TestAccAWSIAMRole_full(t *testing.T) {
	_, fqrn, name := testutil.MkNames("test-aws-iam-role", "platform_aws_iam_role")

	id, _, _ := testutil.MkNames("test-user-upgrade-", "artifactory_managed_user")
	username := fmt.Sprintf("dummy_user%d", id)
	email := fmt.Sprintf(username + "@test.com")

	temp := `
	resource "artifactory_managed_user" "{{ .username }}" {
		name = "{{ .username }}"
		email = "{{ .email }}"
		password = "Passsw0rd!12"
	}

	resource "platform_aws_iam_role" "{{ .name }}" {
		username = artifactory_managed_user.{{ .username }}.name
		iam_role = "{{ .iam_role }}"
	}`

	testData := map[string]string{
		"name":     name,
		"email":    email,
		"username": username,
		"iam_role": "arn:aws:iam::000000000000:role/example",
	}

	config := util.ExecuteTemplate(name, temp, testData)

	updatedTestData := map[string]string{
		"name":     name,
		"email":    email,
		"username": username,
		"iam_role": "arn:aws:iam::000000000000:role/example-2",
	}

	updatedConfig := util.ExecuteTemplate(name, temp, updatedTestData)

	resource.Test(t, resource.TestCase{
		PreCheck: func() { testAccPreCheck(t) },
		ExternalProviders: map[string]resource.ExternalProvider{
			"artifactory": {
				Source: "jfrog/artifactory",
			},
		},
		ProtoV6ProviderFactories: testAccProviders(),
		CheckDestroy:             testAccAWSIAMRoleDestroy(fqrn),
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(fqrn, "username", testData["username"]),
					resource.TestCheckResourceAttr(fqrn, "iam_role", testData["iam_role"]),
				),
			},
			{
				Config: updatedConfig,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(fqrn, "username", updatedTestData["username"]),
					resource.TestCheckResourceAttr(fqrn, "iam_role", updatedTestData["iam_role"]),
				),
			},
			{
				ResourceName:                         fqrn,
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateId:                        updatedTestData["username"],
				ImportStateVerifyIdentifierAttribute: "username",
			},
		},
	})
}

func testAccAWSIAMRoleDestroy(id string) func(*terraform.State) error {
	return func(s *terraform.State) error {
		c := TestProvider.(*platform.PlatformProvider).Meta.Client

		rs, ok := s.RootModule().Resources[id]
		if !ok {
			return fmt.Errorf("error: resource id [%s] not found", id)
		}

		var role platform.AWSIAMRoleAPIModel
		resp, err := c.R().
			SetPathParam("username", rs.Primary.Attributes["username"]).
			SetResult(&role).
			Get(platform.AWSIAMRoleEndpoint)
		if err != nil {
			return err
		}

		if resp != nil && resp.StatusCode() == http.StatusBadRequest {
			return nil
		}

		return fmt.Errorf("error: AWS IAM role for username %s still exists", rs.Primary.Attributes["username"])
	}
}
