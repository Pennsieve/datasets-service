# S3 Bucket for Temporary Manifest Files
resource "aws_s3_bucket" "manifest_file_bucket" {
  bucket = "pennsieve-${var.environment_name}-manifest-files-${data.terraform_remote_state.region.outputs.aws_region_shortname}"

  lifecycle {
    prevent_destroy = true
  }

  tags = merge(
    local.common_tags,
    {
      "Name"         = "${var.environment_name}-manifest-files-${data.terraform_remote_state.region.outputs.aws_region_shortname}"
      "name"         = "${var.environment_name}-manifest-files-${data.terraform_remote_state.region.outputs.aws_region_shortname}"
      "service_name" = "upload-service-v2"
      "tier"         = "s3"
    },
  )
}

// Remove files from bucket after 5 days.
resource "aws_s3_bucket_lifecycle_configuration" "l1" {
  bucket = aws_s3_bucket.manifest_file_bucket.id
  rule {
    status = "Enabled"
    id     = "expire_all_files"
    expiration {
      days = 5
    }
  }
}