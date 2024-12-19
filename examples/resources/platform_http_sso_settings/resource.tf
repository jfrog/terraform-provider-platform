resource "platform_http_sso_settings" "my-http-sso-settings" {
	proxied                      = true
	auto_create_user             = true
	allow_user_to_access_profile = true
	remote_user_request_variable = "MY_REMOTE_USER"
	sync_ldap_groups             = false
}