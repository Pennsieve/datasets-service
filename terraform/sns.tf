resource "aws_sns_topic" "create_manifest_sns_topic" {
  name         = "${var.environment_name}-${var.service_name}-create-manifest-file-${data.terraform_remote_state.region.outputs.aws_region_shortname}"
  display_name = "${var.environment_name}-${var.service_name}-create-manifest-file-${data.terraform_remote_state.region.outputs.aws_region_shortname}"
}

resource "aws_sns_topic_subscription" "create_manifest_subscription" {
  topic_arn = aws_sns_topic.create_manifest_sns_topic.arn
  protocol  = "lambda"
  endpoint  = aws_lambda_function.manifest_worker_lambda.arn
}

resource "aws_lambda_permission" "lambda_with_sns" {
  statement_id  = "AllowExecutionFromSNS"
  action        = "lambda:InvokeFunction"
  function_name = aws_lambda_function.manifest_worker_lambda.function_name
  principal     = "sns.amazonaws.com"
  source_arn    = aws_sns_topic.create_manifest_sns_topic.arn
}