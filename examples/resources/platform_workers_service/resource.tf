# Worker triggered by BEFORE_DOWNLOAD
resource "platform_workers_service" "my-workers-service" {
  key         = "my-workers-service"
  enabled     = true
  description = "My workers service"
  source_code = <<EOT
export default async (context: PlatformContext, data: BeforeDownloadRequest): Promise<BeforeDownloadResponse> => {
  console.log(await context.clients.platformHttp.get('/artifactory/api/system/ping'));
  console.log(await axios.get('https://my.external.resource'));
  return {
    status: 'DOWNLOAD_PROCEED',
    message: 'proceed',
  }
}
EOT
  action      = "BEFORE_DOWNLOAD"

  filter_criteria = {
    artifact_filter_criteria = {
      repo_keys        = ["my-repo-key"]
      include_patterns = ["**/*.jar"]
      exclude_patterns = ["**/*.txt"]
    }
  }

  secrets = [
    {
      key   = "my-secret-key-1"
      value = "my-secret-value-1"
    },
    {
      key   = "my-secret-key-2"
      value = "my-secret-value-2"
    }
  ]
}

# Worker triggered by every local and federated repository, without naming any
# repository explicitly. At least one of `repo_keys`, `any_local`, `any_remote` or
# `any_federated` must be set.
resource "platform_workers_service" "my-any-repo-workers-service" {
  key         = "my-any-repo-workers-service"
  enabled     = true
  description = "My workers service for every local and federated repository"
  source_code = <<EOT
export default async (context: PlatformContext, data: BeforeDownloadRequest): Promise<BeforeDownloadResponse> => {
  console.log(await context.clients.platformHttp.get('/artifactory/api/system/ping'));
  return {
    status: 'DOWNLOAD_PROCEED',
    message: 'proceed',
  }
}
EOT
  action      = "BEFORE_DOWNLOAD"

  filter_criteria = {
    artifact_filter_criteria = {
      any_local     = true
      any_federated = true
    }
  }
}

# Worker triggered by an action that does not accept a filter. `filter_criteria` must
# be omitted entirely, otherwise the JFrog platform rejects the request.
resource "platform_workers_service" "my-build-info-workers-service" {
  key         = "my-build-info-workers-service"
  enabled     = true
  description = "My workers service triggered after build info is saved"
  source_code = <<EOT
export default async (context: PlatformContext, data: AfterBuildInfoSaveRequest): Promise<AfterBuildInfoSaveResponse> => {
  console.log(await context.clients.platformHttp.get('/artifactory/api/system/ping'));
  return {
    message: 'proceed',
  }
}
EOT
  action      = "AFTER_BUILD_INFO_SAVE"
}

# Worker triggered by schedule
resource "platform_workers_service" "my-scheduled-workers-service" {
  key         = "my-scheduled-workers-service"
  enabled     = true
  description = "My Scheduled workers service"
  source_code = <<EOT
export default async (context: PlatformContext, data: BeforeDownloadRequest): Promise<BeforeDownloadResponse> => {
  console.log(await context.clients.platformHttp.get('/artifactory/api/system/ping'));
  console.log(await axios.get('https://my.external.resource'));
  return {
    message: 'Request is successful',
  }
}
EOT
  action      = "SCHEDULED_EVENT"

  filter_criteria = {
    schedule = {
      cron     = "*/2 * * * *"
      timezone = "UTC"
    }
  }

  secrets = [
    {
      key   = "my-secret-key-1"
      value = "my-secret-value-1"
    },
    {
      key   = "my-secret-key-2"
      value = "my-secret-value-2"
    }
  ]
}
