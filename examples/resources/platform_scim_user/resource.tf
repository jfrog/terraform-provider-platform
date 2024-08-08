resource "platform_scim_user" "my-scim-user" {
  username = "test@tempurl.org"
  active   = true
  emails = [{
    value = "test@tempurl.org"
    primary = true
  }]
}