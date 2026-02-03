# Global lifecycle stage
resource "platform_lifecycle_stage" "dev" {
  name     = "dev"
  category = "promote"
}

# Project-level lifecycle stage
# Note: Project-scoped stage names must be prefixed with the project_key
resource "platform_lifecycle_stage" "staging" {
  name        = "my-project-staging"
  project_key = "my-project"
  category    = "promote"
}

# Code category stage
resource "platform_lifecycle_stage" "qa" {
  name        = "my-project-qa"
  project_key = "my-project"
  category    = "code"
}

# Minimal example (category defaults to "promote")
resource "platform_lifecycle_stage" "test" {
  name        = "my-project-test"
  project_key = "my-project"
}
