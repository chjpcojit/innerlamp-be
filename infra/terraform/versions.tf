terraform {
  required_version = ">= 1.9.0"

  required_providers {
    google = {
      source  = "hashicorp/google"
      version = "~> 8.0"
    }
    random = {
      source  = "hashicorp/random"
      version = "~> 3.7"
    }
  }

  # Dev keeps state next to this folder (git-ignored). A shared/production
  # environment must use a GCS backend so two people never apply in parallel:
  # backend "gcs" { bucket = "<state-bucket>" prefix = "innerlamp/dev" }
}

provider "google" {
  project = var.project_id
  region  = var.region
}
