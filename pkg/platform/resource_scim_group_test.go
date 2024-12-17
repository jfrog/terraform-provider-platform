package platform_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/jfrog/terraform-provider-platform/v2/pkg/platform"
	"github.com/jfrog/terraform-provider-shared/testutil"
	"github.com/jfrog/terraform-provider-shared/util"
)

func TestAccSCIMGroup_full(t *testing.T) {
	_, _, username := testutil.MkNames("test-scim-user", "platform_scim_user")
	_, fqrn, name := testutil.MkNames("test-scim-group", "platform_scim_group")

	temp := `
	resource "platform_scim_user" "{{ .username }}" {
		username = "{{ .email }}"
		active   = true
		emails = [{
			value = "{{ .email }}"
			primary = true
		}]
	}

	resource "platform_scim_group" "{{ .name }}" {
		id = "{{ .name }}"
		display_name = "{{ .name }}"
		members = [{
			value = platform_scim_user.{{ .username }}.username
			display = platform_scim_user.{{ .username }}.username
		}]
	}`

	testData := map[string]string{
		"username": username,
		"email":    "test@tempurl.org",
		"name":     name,
	}

	config := util.ExecuteTemplate(name, temp, testData)

	updatedTemp := `
	resource "platform_scim_user" "{{ .username }}" {
		username = "{{ .email }}"
		active   = true
		emails = [{
			value = "{{ .email }}"
			primary = true
		}]
	}

	resource "platform_scim_group" "{{ .name }}" {
		id = "{{ .name }}"
		display_name = "{{ .name }}"
		members = [{
			value = platform_scim_user.{{ .username }}.username
			display = platform_scim_user.{{ .username }}.username
		}, {
			value = "anonymous"
			display = "anonymous"
		}]
	}`

	updatedTestData := map[string]string{
		"username": username,
		"email":    "test@tempurl.org",
		"name":     name,
	}

	updatedConfig := util.ExecuteTemplate(name, updatedTemp, updatedTestData)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProviders(),
		CheckDestroy:             testAccSCIMGroupDestroy(fqrn),
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(fqrn, "id", testData["name"]),
					resource.TestCheckResourceAttr(fqrn, "display_name", testData["name"]),
					resource.TestCheckResourceAttr(fqrn, "members.#", "1"),
					resource.TestCheckResourceAttr(fqrn, "members.0.value", testData["email"]),
					resource.TestCheckResourceAttr(fqrn, "members.0.display", testData["email"]),
					resource.TestCheckResourceAttr(fqrn, "meta.%", "1"),
					resource.TestCheckResourceAttr(fqrn, "meta.resourceType", "Group"),
				),
			},
			{
				Config: updatedConfig,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(fqrn, "id", testData["name"]),
					resource.TestCheckResourceAttr(fqrn, "display_name", testData["name"]),
					resource.TestCheckResourceAttr(fqrn, "members.#", "2"),
					resource.TestCheckTypeSetElemNestedAttrs(fqrn, "members.*", map[string]string{
						"value":   testData["email"],
						"display": testData["email"],
					}),
					resource.TestCheckTypeSetElemNestedAttrs(fqrn, "members.*", map[string]string{
						"value":   "anonymous",
						"display": "anonymous",
					}),
					resource.TestCheckResourceAttr(fqrn, "meta.%", "1"),
					resource.TestCheckResourceAttr(fqrn, "meta.resourceType", "Group"),
				),
			},
			{
				ResourceName:                         fqrn,
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateId:                        updatedTestData["name"],
				ImportStateVerifyIdentifierAttribute: "id",
			},
		},
	})
}

func testAccSCIMGroupDestroy(id string) func(*terraform.State) error {
	return func(s *terraform.State) error {
		c := TestProvider.(*platform.PlatformProvider).Meta.Client

		rs, ok := s.RootModule().Resources[id]
		if !ok {
			return fmt.Errorf("error: resource id [%s] not found", id)
		}

		var group platform.SCIMGroupAPIModel
		resp, err := c.R().
			SetPathParam("name", rs.Primary.Attributes["id"]).
			SetResult(&group).
			Get(platform.SCIMGroupEndpoint)
		if err != nil {
			return err
		}

		if resp != nil && resp.StatusCode() == http.StatusNotFound {
			return nil
		}

		return fmt.Errorf("error: SCIM group %s still exists", rs.Primary.Attributes["id"])
	}
}
