# Every secret is generated here and never written to a file or to git:
#   innerlamp-db-password      Cloud SQL password (also what Terraform set on the user)
#   innerlamp-jwt-secret-key   HS256 signing key for appService tokens
#   innerlamp-web-api-key      shared X-API-Key the website's server sends
#   innerlamp-app-config       rendered config.cloud.yaml (no secret values inside)

resource "random_password" "jwt" {
  length  = 48
  special = false
}

resource "random_password" "web_api_key" {
  length  = 32
  special = false
}

resource "google_secret_manager_secret" "db_password" {
  secret_id = "innerlamp-db-password"
  labels    = local.labels

  replication {
    auto {}
  }

  depends_on = [google_project_service.enabled]
}

resource "google_secret_manager_secret_version" "db_password" {
  secret      = google_secret_manager_secret.db_password.id
  secret_data = var.neon_password
}

resource "google_secret_manager_secret" "jwt" {
  secret_id = "innerlamp-jwt-secret-key"
  labels    = local.labels

  replication {
    auto {}
  }

  depends_on = [google_project_service.enabled]
}

resource "google_secret_manager_secret_version" "jwt" {
  secret      = google_secret_manager_secret.jwt.id
  secret_data = random_password.jwt.result
}

resource "google_secret_manager_secret" "web_api_key" {
  secret_id = "innerlamp-web-api-key"
  labels    = local.labels

  replication {
    auto {}
  }

  depends_on = [google_project_service.enabled]
}

resource "google_secret_manager_secret_version" "web_api_key" {
  secret      = google_secret_manager_secret.web_api_key.id
  secret_data = random_password.web_api_key.result
}

resource "google_secret_manager_secret" "app_config" {
  secret_id = "innerlamp-app-config"
  labels    = local.labels

  replication {
    auto {}
  }

  depends_on = [google_project_service.enabled]
}

resource "google_secret_manager_secret_version" "app_config" {
  secret = google_secret_manager_secret.app_config.id
  secret_data = templatefile("${path.module}/templates/config.cloud.yaml.tftpl", {
    project_id     = var.project_id
    region         = var.region
    avatars_bucket = local.avatars_bucket
    db_host        = local.db_host
    db_port        = local.db_port
    db_user        = local.db_user
    db_name        = local.db_name
    db_ssl_mode    = local.db_ssl_mode
  })
}
