resource "platform_reverse_proxy" "my-reverse-proxy" {
  docker_reverse_proxy_method = "SUBDOMAIN"
  server_name_expression      = "*.jfrog.com"
  server_provider             = "NGINX"
  public_server_name          = "jfrog.com"
  internal_hostname           = "localhost"
  use_https                   = true
  http_port                   = 80
  https_port                  = 443
  ssl_key_path                = "/etc/ssl/private/myserver.key"
  ssl_certificate_path        = "/etc/ssl/certs/myserver.crt"
}