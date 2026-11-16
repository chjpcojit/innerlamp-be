locals {
  avatars_bucket = var.avatars_bucket != "" ? var.avatars_bucket : "${var.project_id}-avatars"

  # External Postgres (Neon). Empty host keeps the old Cloud SQL path during
  # the transition; the config template always reads db_*.
  db_host     = var.neon_host != "" ? var.neon_host : "127.0.0.1"
  db_port     = var.neon_port
  db_user     = var.neon_user
  db_name     = var.neon_database
  db_ssl_mode = var.neon_ssl_mode
  labels = {
    app        = "innerlamp"
    env        = var.environment
    managed_by = "terraform"
  }

  # Everything the stack needs. Enabling from Terraform keeps a fresh project
  # reproducible; disable_on_destroy = false so a destroy never breaks a
  # project other resources still use.
  required_services = [
    "artifactregistry.googleapis.com",
    "cloudresourcemanager.googleapis.com",
    "cloudtrace.googleapis.com",
    "iam.googleapis.com",
    "iamcredentials.googleapis.com",
    "logging.googleapis.com",
    "monitoring.googleapis.com",
    "run.googleapis.com",
    "secretmanager.googleapis.com",
    "sqladmin.googleapis.com",
    "storage.googleapis.com",
  ]
}

resource "google_project_service" "enabled" {
  for_each           = toset(local.required_services)
  service            = each.value
  disable_on_destroy = false
}

resource "google_artifact_registry_repository" "app" {
  location      = var.region
  repository_id = "innerlamp"
  description   = "InnerLamp container images (appService, migrations)"
  format        = "DOCKER"
  labels        = local.labels

  depends_on = [google_project_service.enabled]
}
