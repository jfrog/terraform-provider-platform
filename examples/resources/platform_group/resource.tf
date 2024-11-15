resource "platform_group" "my-group" {
  name = "my-group"
  description = "My group"
  external_id = "My Azure ID"
  auto_join = true
  admin_privileges = false
  members = [
    "admin"
  ]
}