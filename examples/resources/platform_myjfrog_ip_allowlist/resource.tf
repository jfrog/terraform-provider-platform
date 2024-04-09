resource "platform_myjfrog_ip_allowlist" "myjfrog-ip-allowlist" {
  server_name = "my-jpd-server-name"
  ips = [
    "1.1.1.7/1",
    "2.2.2.7/1",
  ]
}