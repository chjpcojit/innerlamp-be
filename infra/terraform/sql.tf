# Cloud SQL for PostgreSQL, reached from Cloud Run through the built-in
# Cloud SQL connector (unix socket at /cloudsql/<connection name>), so the
# instance needs no authorized network for the service and no proxy sidecar.

resource "random_password" "db" {
  length  = 32
  special = false
}

resource "google_sql_database_instance" "main" {
  name                = var.sql_instance_name
  database_version    = "POSTGRES_16"
  region              = var.region
  deletion_protection = false

  settings {
    tier              = var.sql_tier
    edition           = "ENTERPRISE"
    disk_size         = 10
    disk_type         = "PD_HDD"
    availability_type = "ZONAL"
    user_labels       = local.labels

    backup_configuration {
      enabled = false
    }

    ip_configuration {
      ipv4_enabled = true
      # No authorized networks: only the Cloud SQL connector (IAM-authenticated)
      # can open a connection.
      ssl_mode = "ENCRYPTED_ONLY"
    }
  }

  depends_on = [google_project_service.enabled]
}

resource "google_sql_database" "innerlamp" {
  name     = "innerlamp"
  instance = google_sql_database_instance.main.name
}

resource "google_sql_user" "app" {
  name     = "innerlamp"
  instance = google_sql_database_instance.main.name
  password = random_password.db.result
}
