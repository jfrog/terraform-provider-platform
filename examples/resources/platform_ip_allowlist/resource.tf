resource "platform_ip_allowlist" "myjfrog-ip-allowlist" {
  server_name = "my-jpd-sever-name"
  ips = [
    "1.1.1.7/1",
    "2.2.2.7/1",
  ]
}