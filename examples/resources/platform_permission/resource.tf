resource "platform_permission" "my-permission" {
  name = "my-permission-name"

  artifact = {
    actions = {
      users = [
        {
          name = "my-user"
          permissions = ["READ", "WRITE"]
        }
      ]
    }

    targets = [
      {
        name = "my-docker-local"
        include_patterns = ["**"]
      },
      {
        name = "ALL-LOCAL"
        include_patterns = ["**", "*.js"]
      },
      {
        name = "ALL-REMOTE"
        include_patterns = ["**", "*.js"]
      },
      {
        name = "ALL-DISTRIBUTION"
        include_patterns = ["**", "*.js"]
      }
    ]
  }

  build = {
    targets = [
      {
        name = "artifactory-build-info"
        include_patterns = ["**"]
        exclude_patterns = ["*.js"]
      }
    ] 
  }

  release_bundle = {
    actions = {
      users = [
        {
          name = "my-user"
          permissions = ["READ", "WRITE"]
        }
      ]

      groups = [
        {
          name = "my-group"
          permissions = ["READ", "ANNOTATE"]
        }
      ]
    }

    targets = [
      {
        name = "release-bundle"
        include_patterns = ["**"]
      }
    ]
  }

  destination = {
    actions = {
      groups = [
        {
          name = "my-group"
          permissions = ["READ", "ANNOTATE"]
        }
      ]
    }

    targets = [
      {
        name = "*"
        include_patterns = ["**"]
      }
    ]
  }

  pipeline_source = {
    actions = {
      groups = [
        {
          name = "my-group"
          permissions = ["READ", "ANNOTATE"]
        }
      ]
    }

    targets = [
      {
        name = "*"
        include_patterns = ["**"]
      }
    ]
  }
}