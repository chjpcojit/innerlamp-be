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
          value = var.neon_host
        }
        env {
          name  = "PGPORT"
          value = tostring(var.neon_port)
        }
        env {
          name  = "PGUSER"
          value = var.neon_user
        }
        env {
          name  = "PGDATABASE"
          value = var.neon_database
        }
        env {
          name  = "PGSSLMODE"
          value = var.neon_ssl_mode
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
      }
    }
  }

  depends_on = [
    google_project_service.enabled,
    google_project_iam_member.run,
    google_secret_manager_secret_version.db_password,
  ]
}

# One-shot seed job: loads every catalogue in the order the Makefile uses
# (admin units first because temples check province/ward codes).
resource "google_cloud_run_v2_job" "seed" {
  count = var.deploy_seed_job ? 1 : 0

  name                = "innerlamp-seed"
  location            = var.region
  labels              = local.labels
  deletion_protection = false

  template {
    template {
      service_account = google_service_account.run.email
      timeout         = "1800s"
      max_retries     = 0

      containers {
        image = var.seed_image

        env {
          name  = "CONFIG_PATH"
          value = "/app/config/seed.yaml"
        }

        volume_mounts {
          name       = "config"
          mount_path = "/app/config"
        }
      }

      volumes {
        name = "config"
        secret {
          secret = google_secret_manager_secret.seed_config.secret_id
          items {
            version = "latest"
            path    = "seed.yaml"
          }
        }
      }
    }
  }

  depends_on = [
    google_project_service.enabled,
    google_project_iam_member.run,
    google_secret_manager_secret_version.seed_config,
  ]
}
