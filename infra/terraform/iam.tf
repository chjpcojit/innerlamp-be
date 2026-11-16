# One runtime identity for every workload in this stack. It is not a deployer:
# humans/CI deploy with their own account and impersonate nothing.

resource "google_service_account" "run" {
  account_id   = "innerlamp-run"
  display_name = "InnerLamp Cloud Run runtime"
  description  = "Runs appService and the migration job; reads config and secrets, talks to Cloud SQL and GCS."
}

locals {
  run_project_roles = [
    "roles/cloudtrace.agent",
    "roles/logging.logWriter",
    "roles/monitoring.metricWriter",
  ]
}

resource "google_project_iam_member" "run" {
  for_each = toset(local.run_project_roles)

  project = var.project_id
  role    = each.value
  member  = "serviceAccount:${google_service_account.run.email}"
}

resource "google_storage_bucket_iam_member" "run_avatars" {
  bucket = google_storage_bucket.avatars.name
  role   = "roles/storage.objectAdmin"
  member = "serviceAccount:${google_service_account.run.email}"
}

# Signed URLs: the client library signs with the runtime service account's
# private key through the IAM signBlob API, so the account can act as itself.
resource "google_service_account_iam_member" "run_signs_itself" {
  service_account_id = google_service_account.run.name
  role               = "roles/iam.serviceAccountTokenCreator"
  member             = "serviceAccount:${google_service_account.run.email}"
}

# Secret access is granted per secret, not project-wide: the runtime identity
# can read the four secrets this stack creates and nothing else in the project.
locals {
  app_secret_ids = {
    db_password = google_secret_manager_secret.db_password.secret_id
    jwt         = google_secret_manager_secret.jwt.secret_id
    web_api_key = google_secret_manager_secret.web_api_key.secret_id
    config      = google_secret_manager_secret.app_config.secret_id
    seed_config = google_secret_manager_secret.seed_config.secret_id
  }
}

resource "google_secret_manager_secret_iam_member" "run_reads" {
  for_each = local.app_secret_ids

  project   = var.project_id
  secret_id = each.value
  role      = "roles/secretmanager.secretAccessor"
  member    = "serviceAccount:${google_service_account.run.email}"
}
