resource "platform_workers_service" "my-workers-service" {
  key         = "my-workers-service"
  enabled     = true
  description = "My workers service"
  source_code = "export default async (context: PlatformContext, data: BeforeDownloadRequest): Promise<BeforeDownloadResponse> => { console.log(await context.clients.platformHttp.get('/artifactory/api/system/ping')); console.log(await axios.get('https://my.external.resource')); return { status: 'DOWNLOAD_PROCEED', message: 'proceed', } }"
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