resource "aws_lambda_function" "service_lambda" {
  description   = "Lambda Function which handles requests for serverless datasets-service"
  function_name = "${var.environment_name}-${var.service_name}-lambda-${data.terraform_remote_state.region.outputs.aws_region_shortname}"
  handler       = "datasets_service"
  runtime       = "provided.al2"
  architectures = ["arm64"]
  role          = aws_iam_role.datasets_service_lambda_role.arn
  timeout       = 300
  memory_size   = 1536
  s3_bucket     = var.lambda_bucket
  s3_key        = "${var.service_name}/${var.service_name}-${var.image_tag}.zip"

  vpc_config {
    subnet_ids         = tolist(data.terraform_remote_state.vpc.outputs.private_subnet_ids)
    security_group_ids = [data.terraform_remote_state.platform_infrastructure.outputs.upload_v2_security_group_id]
  }

  environment {
    variables = {
      ENV                = var.environment_name
      PENNSIEVE_DOMAIN   = data.terraform_remote_state.account.outputs.domain_name,
      REGION             = var.aws_region
      RDS_PROXY_ENDPOINT = data.terraform_remote_state.pennsieve_postgres.outputs.rds_proxy_endpoint,
      LOG_LEVEL = "INFO"
    }
  }
}
