variable "project_id" {
  description = "GCP project that hosts the InnerLamp dev stack."
  type        = string
  default     = "innerlamp-dev"
}

variable "region" {
  description = "Region for Cloud Run, Cloud SQL, Artifact Registry and the bucket."
  type        = string
  default     = "asia-southeast1"
}

variable "environment" {
  description = "Environment label applied to every resource."
  type        = string
  default     = "dev"

  validation {
    condition     = contains(["dev", "staging", "prod"], var.environment)
    error_message = "environment must be dev, staging or prod."
  }
}

variable "app_image" {
  description = "Container image of appService, built from the repo Dockerfile and pushed to Artifact Registry."
  type        = string
  default     = "asia-southeast1-docker.pkg.dev/innerlamp-dev/innerlamp/app:latest"
}

variable "migrate_image" {
  description = "Container image that applies migrations/*.sql (migrations/Dockerfile)."
  type        = string
  default     = "asia-southeast1-docker.pkg.dev/innerlamp-dev/innerlamp/migrate:latest"
}

variable "sql_instance_name" {
  description = "Cloud SQL instance name. The connection name is project:region:instance."
  type        = string
  default     = "innerlamp-pg"
}

variable "sql_tier" {
  description = "Cloud SQL machine tier. db-f1-micro is the cheapest shared-core tier for dev."
  type        = string
  default     = "db-f1-micro"
}

variable "avatars_bucket" {
  description = "GCS bucket for avatar images. Empty = <project_id>-avatars."
  type        = string
  default     = ""
}

variable "max_instances" {
  description = "Cloud Run maximum instances. Keep it low while rate limits use the in-memory store."
  type        = number
  default     = 2
}

variable "allow_unauthenticated" {
  description = "Allow anonymous invocations of the dev API. Only for a dev project with no real users."
  type        = bool
  default     = true
}

variable "deploy_migrate_job" {
  description = "Create the Cloud Run job that applies migrations/*.sql."
  type        = bool
  default     = true
}
