resource "aws_cloudwatch_log_group" "datasets_service_lambda_loggroup" {
  name              = "/aws/lambda/${aws_lambda_function.service_lambda.function_name}"
  retention_in_days = 30
  tags = local.common_tags
}
resource "aws_cloudwatch_log_subscription_filter" "datasets_service_log_group_subscription" {
  name            = "${aws_cloudwatch_log_group.datasets_service_lambda_loggroup.name}-subscription"
  log_group_name  = aws_cloudwatch_log_group.datasets_service_lambda_loggroup.name
  filter_pattern  = ""
  destination_arn = data.terraform_remote_state.region.outputs.datadog_delivery_stream_arn
  role_arn        = data.terraform_remote_state.region.outputs.cw_logs_to_datadog_logs_firehose_role_arn
}

resource "aws_cloudwatch_log_group" "manifest_worker_lambda_loggroup" {
  name              = "/aws/lambda/${aws_lambda_function.manifest_worker_lambda.function_name}"
  retention_in_days = 30
  tags = local.common_tags
}
resource "aws_cloudwatch_log_subscription_filter" "datasets_service_log_group_subscription" {
  name            = "${aws_cloudwatch_log_group.manifest_worker_lambda_loggroup.name}-subscription"
  log_group_name  = aws_cloudwatch_log_group.manifest_worker_lambda_loggroup.name
  filter_pattern  = ""
  destination_arn = data.terraform_remote_state.region.outputs.datadog_delivery_stream_arn
  role_arn        = data.terraform_remote_state.region.outputs.cw_logs_to_datadog_logs_firehose_role_arn
}