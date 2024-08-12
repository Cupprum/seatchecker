resource "aws_apigatewayv2_api" "seatchecker" {
  name          = "seatchecker_api"
  protocol_type = "HTTP"
}

resource "aws_apigatewayv2_integration" "trigger_step_function" {
  api_id                 = aws_apigatewayv2_api.seatchecker.id
  description            = "Invoke Step Functions"
  integration_type       = "AWS_PROXY"
  integration_subtype    = "StepFunctions-StartExecution"
  credentials_arn        = module.step-functions.role_arn
  payload_format_version = "1.0"
  request_parameters = {
    "StateMachineArn" = module.step-functions.state_machine_arn
    "Input"           = "$request.body",
  }
}

resource "aws_apigatewayv2_route" "trigger_step_function" {
  api_id    = aws_apigatewayv2_api.seatchecker.id
  route_key = "POST /test"
  target    = "integrations/${aws_apigatewayv2_integration.trigger_step_function.id}"
}

// TODO: logs are not configured properly, iam permissions are missing
# resource "aws_cloudwatch_log_group" "apigateway" {
#   name              = "/aws/apigateway/${aws_apigatewayv2_api.seatchecker.name}"
#   retention_in_days = 3
# }

resource "aws_apigatewayv2_stage" "deployment" {
  api_id      = aws_apigatewayv2_api.seatchecker.id
  name        = "$default"
  auto_deploy = true
  # access_log_settings {
  #   destination_arn = aws_cloudwatch_log_group.apigateway.arn
  #   format = jsonencode({
  #     "requestId" : "$context.requestId"
  #     "ip" : "$context.identity.sourceIp"
  #     "requestTime" : "$context.requestTime"
  #     "httpMethod" : "$context.httpMethod"
  #     "routeKey" : "$context.routeKey"
  #     "status" : "$context.status"
  #     "protocol" : "$context.protocol"
  #     "responseLength" : "$context.responseLength"
  #     "authorizationError" : "$context.authorizer.error"
  #   })
  # }
}
