# One-shot Cloud Run job that applies migrations/*.sql in filename order and
# records each filename in schema_migrations, so a rerun skips what is applied.
resource "google_cloud_run_v2_job" "migrate" {
  count = var.deploy_migrate_job ? 1 : 0

  name                = "innerlamp-migrate"
  location            = var.region
  labels              = local.labels
  deletion_protection = false

  template {
    template {
      service_account = google_service_account.run.email
      timeout         = "600s"
      max_retries     = 0

      containers {
        image = var.migrate_image

        env {
          name  = "PGHOST"
          value = "/cloudsql/${local.sql_connection_name}"
        }
        env {
          name  = "PGPORT"
          value = "5432"
        }
        env {
          name  = "PGUSER"
          value = "innerlamp"
        }
        env {
          name  = "PGDATABASE"
          value = "innerlamp"
        }
        env {
          name  = "PGSSLMODE"
          value = "disable"
        }
        env {
          name = "PGPASSWORD"
          value_source {
            secret_key_ref {
              secret  = google_secret_manager_secret.db_password.secret_id
              version = "latest"
            }
          }
        }

        volume_mounts {
          name       = "cloudsql"
          mount_path = "/cloudsql"
        }
      }

      volumes {
        name = "cloudsql"
        cloud_sql_instance {
          instances = [local.sql_connection_name]
        }
      }
    }
  }

  depends_on = [
    google_project_service.enabled,
    google_project_iam_member.run,
    google_secret_manager_secret_version.db_password,
  ]
}
