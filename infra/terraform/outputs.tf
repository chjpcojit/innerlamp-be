output "service_url" {
  description = "Public URL of appService."
  value       = google_cloud_run_v2_service.app.uri
}

output "artifact_registry" {
  description = "Docker repository to tag and push images to."
  value       = "${var.region}-docker.pkg.dev/${var.project_id}/${google_artifact_registry_repository.app.repository_id}"
}

output "avatars_bucket" {
  description = "GCS bucket that stores avatar images."
  value       = google_storage_bucket.avatars.name
}

output "app_config_secret" {
  description = "Secret Manager secret holding the rendered config.cloud.yaml."
  value       = google_secret_manager_secret.app_config.secret_id
}

output "migrate_job" {
  description = "Cloud Run job that applies migrations (empty when disabled)."
  value       = var.deploy_migrate_job ? google_cloud_run_v2_job.migrate[0].name : ""
}

output "runtime_service_account" {
  description = "Identity Cloud Run runs as."
  value       = google_service_account.run.email
}
