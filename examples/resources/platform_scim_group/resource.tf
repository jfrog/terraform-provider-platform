resource "platform_scim_group" "my-scim-group" {
  id = "my-scim-group"
  display_name = "my-scim-group"
  members = [{
    value = "test@tempurl.org"
    display = "test@tempurl.org"
  }, {
    value = "anonymous"
    display = "anonymous"
  }]
}