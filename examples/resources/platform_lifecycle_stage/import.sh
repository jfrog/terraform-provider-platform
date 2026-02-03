#!/bin/bash

# Import a global lifecycle stage
terraform import platform_lifecycle_stage.production PROD

# Import a project-level lifecycle stage
terraform import platform_lifecycle_stage.staging staging-us-east:my-project

