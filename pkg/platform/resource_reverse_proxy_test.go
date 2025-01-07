package platform_test

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/jfrog/terraform-provider-shared/testutil"
	"github.com/jfrog/terraform-provider-shared/util"
)

func TestAccReverseProxy_full(t *testing.T) {
	_, fqrn, reverseProxyName := testutil.MkNames("test-reverse-proxy", "platform_reverse_proxy")

	temp := `
	resource "platform_reverse_proxy" "{{ .name }}" {
		docker_reverse_proxy_method = "{{ .dockerProxyMethod }}"
		server_provider             = "{{ .serverProvider }}"
		public_server_name          = "{{ .serverName }}"
		internal_hostname           = "localhost"
	}`

	testData := map[string]string{
		"name":              reverseProxyName,
		"dockerProxyMethod": "SUBDOMAIN",
		"serverProvider":    "NGINX",
		"serverName":        "tempurl.org",
	}

	config := util.ExecuteTemplate(reverseProxyName, temp, testData)

	updatedTemp := `
	resource "platform_reverse_proxy" "{{ .name }}" {
		docker_reverse_proxy_method = "{{ .dockerProxyMethod }}"
		server_provider             = "{{ .serverProvider }}"
		public_server_name          = "{{ .serverName }}"
		internal_hostname           = "localhost"
		http_port                   = {{ .httpPort }}
		use_https                   = {{ .useHttps }}
		https_port                  = {{ .httpsPort }}
		ssl_key_path                = "{{ .sslKeyPath }}"
		ssl_certificate_path        = "{{ .sslCertPath }}"
	}`

	updatedTestData := map[string]string{
		"name":              reverseProxyName,
		"dockerProxyMethod": "REPOPATHPREFIX",
		"serverProvider":    "NGINX",
		"serverName":        "tempurl.org",
		"httpPort":          "88",
		"useHttps":          "true",
		"httpsPort":         "666",
		"sslKeyPath":        "foo/bar.key",
		"sslCertPath":       "foo/bar.crt",
	}
	updatedConfig := util.ExecuteTemplate(reverseProxyName, updatedTemp, updatedTestData)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProviders(),
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(fqrn, "docker_reverse_proxy_method", testData["dockerProxyMethod"]),
					resource.TestCheckResourceAttr(fqrn, "server_provider", testData["serverProvider"]),
					resource.TestCheckResourceAttr(fqrn, "public_server_name", testData["serverName"]),
					resource.TestCheckResourceAttr(fqrn, "internal_hostname", "localhost"),
					resource.TestCheckResourceAttr(fqrn, "use_https", "false"),
					resource.TestCheckResourceAttr(fqrn, "http_port", "80"),
					resource.TestCheckResourceAttr(fqrn, "https_port", "443"),
					resource.TestCheckNoResourceAttr(fqrn, "ssl_key_path"),
					resource.TestCheckNoResourceAttr(fqrn, "ssl_certificate_path"),
				),
			},
			{
				Config: updatedConfig,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(fqrn, "docker_reverse_proxy_method", updatedTestData["dockerProxyMethod"]),
					resource.TestCheckResourceAttr(fqrn, "server_provider", updatedTestData["serverProvider"]),
					resource.TestCheckResourceAttr(fqrn, "public_server_name", updatedTestData["serverName"]),
					resource.TestCheckResourceAttr(fqrn, "internal_hostname", "localhost"),
					resource.TestCheckResourceAttr(fqrn, "use_https", updatedTestData["useHttps"]),
					resource.TestCheckResourceAttr(fqrn, "http_port", updatedTestData["httpPort"]),
					resource.TestCheckResourceAttr(fqrn, "https_port", updatedTestData["httpsPort"]),
					resource.TestCheckResourceAttr(fqrn, "ssl_key_path", updatedTestData["sslKeyPath"]),
					resource.TestCheckResourceAttr(fqrn, "ssl_certificate_path", updatedTestData["sslCertPath"]),
				),
			},
			{
				ResourceName:                         fqrn,
				ImportState:                          true,
				ImportStateId:                        reverseProxyName,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "server_provider",
			},
		},
	})
}

func TestAccReverseProxy_invalid_server_provider(t *testing.T) {
	for _, serverProvider := range []string{"NGINX", "APACHE"} {
		t.Run(serverProvider, func(t *testing.T) {
			resource.Test(testAccReverseProxy_missing_internal_hostname(t, serverProvider))
			resource.Test(testAccReverseProxy_missing_public_server_name(t, serverProvider))
		})
	}
}

func testAccReverseProxy_missing_internal_hostname(t *testing.T, serverProvider string) (*testing.T, resource.TestCase) {
	_, _, reverseProxyName := testutil.MkNames("test-reverse-proxy", "platform_reverse_proxy")

	temp := `
	resource "platform_reverse_proxy" "{{ .name }}" {
		docker_reverse_proxy_method = "SUBDOMAIN"
		server_provider             = "{{ .serverProvider }}"
		public_server_name          = "{{ .serverName }}"
	}`

	testData := map[string]string{
		"name":              reverseProxyName,
		"dockerProxyMethod": "SUBDOMAIN",
		"serverProvider":    serverProvider,
		"serverName":        "tempurl.org",
	}

	config := util.ExecuteTemplate(reverseProxyName, temp, testData)

	return t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProviders(),
		Steps: []resource.TestStep{
			{
				Config:      config,
				ExpectError: regexp.MustCompile(fmt.Sprintf(`.*internal_hostname must be configured when server_provider is set to '%s'.*`, serverProvider)),
			},
		},
	}
}

func testAccReverseProxy_missing_public_server_name(t *testing.T, serverProvider string) (*testing.T, resource.TestCase) {
	_, _, reverseProxyName := testutil.MkNames("test-reverse-proxy", "platform_reverse_proxy")

	temp := `
	resource "platform_reverse_proxy" "{{ .name }}" {
		docker_reverse_proxy_method = "SUBDOMAIN"
		server_provider             = "{{ .serverProvider }}"
		internal_hostname           = "localhost"
	}`

	testData := map[string]string{
		"name":              reverseProxyName,
		"dockerProxyMethod": "SUBDOMAIN",
		"serverProvider":    serverProvider,
	}

	config := util.ExecuteTemplate(reverseProxyName, temp, testData)

	return t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProviders(),
		Steps: []resource.TestStep{
			{
				Config:      config,
				ExpectError: regexp.MustCompile(`.*public_server_name must be configured when server_provider is set to.*`),
			},
		},
	}
}

func TestAccReverseProxy_invalid_https(t *testing.T) {
	for _, serverProvider := range []string{"NGINX", "APACHE"} {
		t.Run(serverProvider, func(t *testing.T) {
			resource.Test(testAccReverseProxy_missing_ssl_key_path(t, serverProvider))
			resource.Test(testAccReverseProxy_missing_ssl_certificate_path(t, serverProvider))
		})
	}
}

func testAccReverseProxy_missing_ssl_key_path(t *testing.T, serverProvider string) (*testing.T, resource.TestCase) {
	_, _, reverseProxyName := testutil.MkNames("test-reverse-proxy", "platform_reverse_proxy")

	temp := `
	resource "platform_reverse_proxy" "{{ .name }}" {
		docker_reverse_proxy_method = "SUBDOMAIN"
		server_provider             = "NGINX"
		public_server_name          = "{{ .serverName }}"
		internal_hostname           = "localhost"
		use_https                   = true
		ssl_certificate_path        = "/foo/bar.crt"
	}`

	testData := map[string]string{
		"name":              reverseProxyName,
		"dockerProxyMethod": "SUBDOMAIN",
		"serverProvider":    serverProvider,
		"serverName":        "tempurl.org",
	}

	config := util.ExecuteTemplate(reverseProxyName, temp, testData)

	return t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProviders(),
		Steps: []resource.TestStep{
			{
				Config:      config,
				ExpectError: regexp.MustCompile(`ssl_key_path must be configured when use_https is set to 'true'.`),
			},
		},
	}
}

func testAccReverseProxy_missing_ssl_certificate_path(t *testing.T, serverProvider string) (*testing.T, resource.TestCase) {
	_, _, reverseProxyName := testutil.MkNames("test-reverse-proxy", "platform_reverse_proxy")

	temp := `
	resource "platform_reverse_proxy" "{{ .name }}" {
		docker_reverse_proxy_method = "SUBDOMAIN"
		server_provider             = "NGINX"
		public_server_name          = "{{ .serverName }}"
		internal_hostname           = "localhost"
		use_https                   = true
		ssl_key_path                = "/foo/bar.key"
	}`

	testData := map[string]string{
		"name":              reverseProxyName,
		"dockerProxyMethod": "SUBDOMAIN",
		"serverProvider":    serverProvider,
		"serverName":        "tempurl.org",
	}

	config := util.ExecuteTemplate(reverseProxyName, temp, testData)

	return t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProviders(),
		Steps: []resource.TestStep{
			{
				Config:      config,
				ExpectError: regexp.MustCompile(`ssl_certificate_path must be configured when use_https is set to 'true'.`),
			},
		},
	}
}
