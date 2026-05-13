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
      repo_keys = ["my-repo-key"]
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

# Worker triggerd by schedule
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
