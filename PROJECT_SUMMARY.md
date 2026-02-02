# Terraform Provider Platform - Project Summary

## Overview

Terraform Provider for JFrog Platform, providing resources to manage platform-level configurations including permissions, authentication providers, OIDC integrations, groups, roles, and more.

## Features Implemented

### Resources

1. **platform_permission**
   - Manages JFrog permissions for users and groups
   - Supports artifact, build, release_bundle, destination, and pipeline_source targets
   - Next-generation permissions model compatible with legacy `artifactory_permission_target`

2. **platform_oidc_configuration**
   - Manages OIDC provider configurations
   - Supports GitHub, GitHub Enterprise, Azure, and generic OIDC providers

3. **platform_oidc_identity_mapping**
   - Manages OIDC identity mappings for configurations
   - Supports user, group, and admin scope mappings

4. **platform_group**
   - Creates and manages groups for RBAC
   - Supports auto-join and admin privileges

5. **platform_group_members**
   - Manages group membership separately from group resource
   - Supports import functionality

6. **platform_global_role**
   - Creates custom global roles
   - Configurable actions and environments

7. **platform_workers_service**
   - Manages Workers Service for custom automation
   - Supports multiple trigger actions (BEFORE_DOWNLOAD, AFTER_UPLOAD, etc.)

8. **platform_aws_iam_role**
   - Manages AWS IAM roles for passwordless EKS access
   - Requires Artifactory 7.90.10+

9. **platform_saml_settings**
   - Configures SAML SSO settings
   - This resource supports both JFrog SaaS and Self-Hosted instances. For SaaS instances, the `enable` parameter must currently be activated via a manual API call after the Terraform apply is complete.

10. **platform_http_sso_settings**
    - Configures HTTP-based SSO
    - Compatible with Apache mod_auth_* modules

11. **platform_crowd_settings**
    - Configures Atlassian Crowd/JIRA integration
    - Supports SSO delegation

12. **platform_license**
    - Installs/updates JFrog licenses
    - Self-hosted instances only

13. **platform_reverse_proxy**
    - Configures reverse proxy settings
    - Supports NGINX, Apache, and direct modes

14. **platform_scim_user**
    - Manages users via SCIM protocol

15. **platform_scim_group**
    - Manages groups via SCIM protocol

16. **platform_myjfrog_ip_allowlist** (Deprecated)
    - Manages IP allowlist for MyJFrog
    - Being moved to jfrog/myjfrog provider

## Project Structure

```
terraform-provider-platform/
├── main.go                           # Provider entry point
├── go.mod                            # Go module definition
├── go.sum                            # Go module checksums
├── GNUmakefile                       # Build and test automation
├── LICENSE                           # Apache 2.0 license
├── NOTICE                            # Third-party attributions
├── README.md                         # User documentation
├── CHANGELOG.md                      # Version history
├── CODEOWNERS                        # Code ownership
├── CONTRIBUTING.md                   # Contribution guidelines
├── CONTRIBUTIONS.md                  # Contributor list
├── releasePlatformProvider.sh        # Release automation script
├── sample.tf                         # Sample Terraform configuration
├── terraform-registry-manifest.json  # Terraform registry metadata
├── pkg/platform/
│   ├── provider.go                   # Provider implementation
│   ├── resource_permission.go        # Permission resource
│   ├── resource_permission_test.go   # Permission tests
│   ├── resource_oidc_*.go            # OIDC resources
│   ├── resource_group*.go            # Group resources
│   ├── resource_*_test.go            # Test files
│   └── ...                           # Other resources
├── docs/
│   ├── index.md                      # Provider documentation
│   └── resources/                    # Resource documentation
│       └── *.md
├── examples/
│   ├── provider/
│   │   └── provider.tf               # Provider configuration examples
│   └── resources/
│       └── platform_*/               # Resource examples
├── templates/
│   ├── index.md.tmpl                 # Provider doc template
│   └── resources/                    # Resource doc templates
│       └── *.md.tmpl
├── scripts/                          # Development scripts
└── tools/
    └── tools.go                      # Build tools
```

## Provider Configuration

The provider supports multiple authentication methods:

1. **Access Token** - Via configuration or `JFROG_ACCESS_TOKEN` environment variable
2. **OIDC** - Via OIDC provider name
3. **Terraform Cloud Workload Identity** - Automatic when running in TFC

Example configuration:

```terraform
terraform {
  required_providers {
    platform = {
      source  = "jfrog/platform"
      version = "~> 2.0"
    }
  }
}

provider "platform" {
  url          = "https://myinstance.jfrog.io"
  access_token = var.jfrog_access_token
}
```

## Building the Provider

```bash
# Initialize dependencies
go mod tidy

# Build the provider
make build

# Install locally for testing
make install

# Run tests
make test

# Run acceptance tests
make acceptance

# Generate documentation
make generate
```

## Key Dependencies

| Dependency | Version | Purpose |
|------------|---------|---------|
| terraform-plugin-framework | v1.17.0 | Terraform provider framework |
| terraform-plugin-framework-validators | v0.19.0 | Schema validators |
| terraform-plugin-testing | v1.14.0 | Acceptance testing |
| terraform-provider-shared | v1.30.7 | JFrog shared utilities |
| go-resty/resty | v2.17.1 | HTTP client |
| samber/lo | v1.52.0 | Go utilities |

## OpenTofu Support

This provider is fully tested and compatible with OpenTofu. Releases are published to both:
- Terraform Registry: `registry.terraform.io/jfrog/platform`
- OpenTofu Registry: `registry.opentofu.org/jfrog/platform`

## Development Notes

- Built with Terraform Plugin Framework
- Uses JFrog shared library for common functionality
- Compatible with Go 1.25.5+
- Supports Terraform 1.0+ and OpenTofu 1.0+
- All source files include Apache 2.0 copyright headers

## Current Version

See [CHANGELOG.md](./CHANGELOG.md) for version history.

## License

Apache 2.0 - Copyright (c) 2025 JFrog Ltd


