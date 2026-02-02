## Description
JTFPR-55 - Updated SAML settings documentation to support activation on SaaS.

## Changes
- **resource/platform_saml_settings**: Documentation and schema description updated to state that the resource supports both JFrog SaaS and Self-Hosted instances. For SaaS, the `enable` parameter must be activated via a manual PATCH to the Access API after Terraform apply.
- **docs/resources/saml_settings.md**: Replaced "Only available for self-hosted instances" with SaaS/Self-Hosted support note and added "Activate SAML on SaaS" section with curl example.
- **templates/resources/saml_settings.md.tmpl**: Aligned template with new documentation.
- **pkg/platform/resource_saml_settings.go**: Updated resource `MarkdownDescription` to match new docs.
- **PROJECT_SUMMARY.md**: Updated platform_saml_settings summary from "Self-hosted instances only" to reflect SaaS support.

## Testing
- [ ] Documentation reviewed for accuracy.
- [ ] No behavioral code changes; documentation only.
