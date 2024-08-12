module "step-functions" {
  source      = "terraform-aws-modules/step-functions/aws"
  version     = "4.2.0"
  definition  = templatefile("./seatchecker.asl.json", {})
  name        = "seatchecker_stepfunction"
  create_role = true
  service_integrations = { # will automatically create policies to attach to the role 
    lambda = {
      lambda = [
        aws_lambda_function.seatchecker.arn,
        "${aws_lambda_function.seatchecker.arn}:*",
      ]
    }
  }
}