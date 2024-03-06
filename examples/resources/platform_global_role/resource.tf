resource "platform_global_role" "my-global-role" {
  name         = "my-global-role"
  description  = "My custom global role"
  type         = "CUSTOM_GLOBAL"
  environments = ["DEV", "PROD"]
  actions      = ["READ_REPOSITORY", "READ_BUILD"]
}