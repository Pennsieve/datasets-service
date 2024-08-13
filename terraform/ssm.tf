resource "aws_ssm_parameter" "manifest_file_bucket" {
  name  = "/${var.environment_name}/${var.service_name}/manifest-bucket"
  type  = "String"
  value = aws_s3_bucket.manifest_file_bucket.id
}