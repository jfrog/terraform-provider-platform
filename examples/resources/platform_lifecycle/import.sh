#!/bin/bash

# Import a global lifecycle
# "_global_" is the special import id for the global lifecycle.
terraform import platform_lifecycle.global _global_

# Import a project-level lifecycle
terraform import platform_lifecycle.project my-project

