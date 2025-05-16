package platform_test

import (
	"os"
	"sync"
	"testing"

	"github.com/go-resty/resty/v2"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/jfrog/terraform-provider-platform/v2/pkg/platform"
	"github.com/jfrog/terraform-provider-shared/client"
)

// TestProvider PreCheck(t) must be called before using this provider instance.
var TestProvider provider.Provider

// testAccProviderConfigure ensures Provider is only configured once
//
// The PreCheck(t) function is invoked for every test and this prevents
// extraneous reconfiguration to the same values each time. However, this does
// not prevent reconfiguration that may happen should the address of
// Provider be errantly reused in ProviderFactories.
var testAccProviderConfigure sync.Once

// testAccPreCheck This function should be present in every acceptance test.
func testAccPreCheck(t *testing.T) {
	// Since we are outside the scope of the Terraform configuration we must
	// call Configure() to properly initialize the provider configuration.
	testAccProviderConfigure.Do(func() {
		restyClient := getTestResty(t)

		platformUrl := getPlatformUrl(t)
		// Set custom base URL so repos that relies on it will work
		// https://www.jfrog.com/confluence/display/JFROG/Artifactory+REST+API#ArtifactoryRESTAPI-UpdateCustomURLBase
		_, err := restyClient.R().
			SetBody(platformUrl).
			SetHeader("Content-Type", "text/plain").
			Put("/artifactory/api/system/configuration/baseUrl")
		if err != nil {
			t.Fatalf("failed to set custom base URL: %v", err)
		}
	})
}

func getTestResty(t *testing.T) *resty.Client {
	var ok bool

	platformUrl := getPlatformUrl(t)

	restyClient, err := client.Build(platformUrl, "")
	if err != nil {
		t.Fatal(err)
	}
	restyClient.SetRetryCount(5)
	var accessToken string
	if accessToken, ok = os.LookupEnv("JFROG_ACCESS_TOKEN"); !ok {
		t.Fatal("JFROG_ACCESS_TOKEN must be set for acceptance tests")
	}
	restyClient, err = client.AddAuth(restyClient, "", accessToken)
	if err != nil {
		t.Fatal(err)
	}

	return restyClient
}

func getPlatformUrl(t *testing.T) string {
	platformUrl, ok := os.LookupEnv("JFROG_URL")
	if !ok {
		t.Fatal("JFROG_URL must be set for acceptance tests")
	}

	return platformUrl
}

func testAccProviders() map[string]func() (tfprotov6.ProviderServer, error) {
	TestProvider = platform.NewProvider()()

	return map[string]func() (tfprotov6.ProviderServer, error){
		"platform": providerserver.NewProtocol6WithError(TestProvider),
	}
}
