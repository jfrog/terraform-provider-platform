resource "platform_saml_settings" "my-okta-saml-settings" {
  name                         = "my-okta-saml-settings"
  enable                       = true
  certificate                  = "MIICTjCCA...gPRXbm49Mz4o1nbwH"
  email_attribute              = "email"
  group_attribute              = "group"
  name_id_attribute            = "id"
  login_url                    = "http://tempurl.org/saml"
  logout_url                   = "https://myaccount.okta.com"
  no_auto_user_creation        = false
  service_provider_name        = "okta"
  allow_user_to_access_profile = true
  auto_redirect                = true
  sync_groups                  = true
  verify_audience_restriction  = true
  use_encrypted_assertion      = false
}