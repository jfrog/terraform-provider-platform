# Default: adopt the group without loading its membership into state
# (use_group_members_resource is set to true).
terraform import platform_group.my-group my-group

# Opt in to managing membership inline on this resource: the ":members" suffix
# sets use_group_members_resource to false and loads the current members into
# state. Use this only when membership is NOT managed via platform_group_members.
terraform import platform_group.my-group my-group:members
