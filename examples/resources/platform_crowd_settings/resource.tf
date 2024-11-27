resource "platform_crowd_settings" "my-crowd-settings" {
  enable                         = true
  server_url                     = "http://tempurl.org"
  application_name               = "my-crowd-settings"
  password                       = "my-password"
  session_validation_interval    = 5
  use_default_proxy              = false
  auto_user_creation             = true
  allow_user_to_access_profile   = false
  direct_authentication          = true
  override_all_groups_upon_login = false
}