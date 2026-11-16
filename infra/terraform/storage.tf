# Avatar images. The service signs V4 upload/read URLs, so the bucket stays
# private and the runtime service account gets objectAdmin on this bucket only.

resource "google_storage_bucket" "avatars" {
  name                        = local.avatars_bucket
  location                    = var.region
  uniform_bucket_level_access = true
  public_access_prevention    = "enforced"
  force_destroy               = true
  labels                      = local.labels

  lifecycle_rule {
    condition {
      age = 1
    }
    action {
      type = "AbortIncompleteMultipartUpload"
    }
  }

  depends_on = [google_project_service.enabled]
}
