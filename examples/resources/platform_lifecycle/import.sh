#!/bin/bash

# Import a global lifecycle
terraform import platform_lifecycle.global ""

# Import a project-level lifecycle
terraform import platform_lifecycle.project my-project

