resource "platform_ldap_group" "my-ldap-group" {
  name               = "my-ldap-group"
  group_dn           = "cn=my-ldap-group,ou=Groups,dc=example,dc=com"
  ldap_group_setting = "my-ldap-setting"
}
