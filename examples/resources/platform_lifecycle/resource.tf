# First, create the lifecycle stages
resource "platform_lifecycle_stage" "dev" {
  name     = "dev"
  category = "promote"
}

resource "platform_lifecycle_stage" "qa" {
  name     = "qa"
  category = "promote"
}

# Global lifecycle
# Note: PROD, PR, and COMMIT are system-managed stages and should not be included in promote_stages
resource "platform_lifecycle" "global" {
  promote_stages = [
    platform_lifecycle_stage.dev.name,
    platform_lifecycle_stage.qa.name
  ]
}

# Project-level lifecycle
# Note: Project-scoped stages must be prefixed with project_key
resource "platform_lifecycle_stage" "project_dev" {
  name        = "my-project-dev"
  project_key = "my-project"
  category    = "promote"
}

resource "platform_lifecycle_stage" "project_staging" {
  name        = "my-project-staging"
  project_key = "my-project"
  category    = "promote"
}

resource "platform_lifecycle" "project" {
  project_key = "my-project"
  promote_stages = [
    platform_lifecycle_stage.project_dev.name,
    platform_lifecycle_stage.project_staging.name
  ]
}
