resource "google_cloud_run_v2_service" "app" {
  name                = "innerlamp-app"
  location            = var.region
  ingress             = "INGRESS_TRAFFIC_ALL"
  deletion_protection = false
  labels              = local.labels

  template {
    service_account = google_service_account.run.email

    scaling {
      min_instance_count = 0
      max_instance_count = var.max_instances
    }

    containers {
      image = var.app_image

      ports {
        container_port = 8080
      }

      env {
        name  = "CONFIG_PATH"
        value = "/app/config/config.cloud.yaml"
      }

      # main.go reads secrets through Secret Manager when this is "gcp".
      env {
        name  = "SECRETS_PROVIDER"
        value = "gcp"
      }

      env {
        name  = "GOOGLE_CLOUD_PROJECT"
        value = var.project_id
      }

      resources {
        # Request-based billing: CPU is allocated only while a request runs.
        # Without this the provider leaves cpu_idle = false and every live
        # instance bills CPU for its whole keep-alive window.
        cpu_idle = true

        limits = {
          cpu    = "1"
          memory = "512Mi"
        }
      }

      volume_mounts {
        name       = "config"
        mount_path = "/app/config"
      }

      volume_mounts {
        name       = "cloudsql"
        mount_path = "/cloudsql"
      }
    }

    volumes {
      name = "config"
      secret {
        secret = google_secret_manager_secret.app_config.secret_id
        items {
          version = "latest"
          path    = "config.cloud.yaml"
        }
      }
    }

    volumes {
      name = "cloudsql"
      cloud_sql_instance {
        instances = [local.sql_connection_name]
      }
    }
  }

  depends_on = [
    google_project_service.enabled,
    google_secret_manager_secret_version.app_config,
    google_project_iam_member.run,
    google_storage_bucket_iam_member.run_avatars,
  ]
}

resource "google_cloud_run_v2_service_iam_member" "public" {
  count = var.allow_unauthenticated ? 1 : 0

  name     = google_cloud_run_v2_service.app.name
  location = google_cloud_run_v2_service.app.location
  role     = "roles/run.invoker"
  member   = "allUsers"
}
