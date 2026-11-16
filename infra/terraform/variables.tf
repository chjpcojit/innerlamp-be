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

# ---------------------------------------------------------------------------
# Postgres bên ngoài (Neon). Cloud SQL chỉ còn trong lúc chuyển đổi; xem
# README "Chuyển sang Neon".
# ---------------------------------------------------------------------------
variable "neon_host" {
  description = "Postgres host (Neon: ep-...ap-southeast-1.aws.neon.tech)."
  type        = string
  default     = ""
}

variable "neon_port" {
  description = "Postgres port."
  type        = number
  default     = 5432
}

variable "neon_user" {
  description = "Postgres role the service connects as."
  type        = string
  default     = "neondb_owner"
}

variable "neon_database" {
  description = "Postgres database name."
  type        = string
  default     = "neondb"
}

variable "neon_password" {
  description = "Postgres password. Stored in Secret Manager, never in the repo."
  type        = string
  sensitive   = true
  default     = ""
}

variable "neon_ssl_mode" {
  description = "libpq sslmode for the external database (Neon requires TLS)."
  type        = string
  default     = "require"
}
