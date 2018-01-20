### providers

provider "aws" {
  region = "eu-west-1"
}

### variables

variable "expiration" {
  default = 90
}

variable "function_name" {
  default = "aws-rotate-access-keys"
}

variable "slack_token" {}

### resources

resource "aws_iam_role" "this" {
  name = "${var.function_name}-lambda"

  assume_role_policy = <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Action": "sts:AssumeRole",
      "Principal": {
        "Service": "lambda.amazonaws.com"
      },
      "Effect": "Allow",
      "Sid": ""
    }
  ]
}
EOF
}

resource "aws_iam_role_policy_attachment" "this" {
  role       = "${aws_iam_role.this.name}"
  policy_arn = "arn:aws:iam::aws:policy/service-role/AWSLambdaBasicExecutionRole"
}

resource "aws_iam_role_policy" "this" {
  name = "IAM"
  role = "${aws_iam_role.this.id}"

  policy = <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Action": [
        "iam:CreateAccessKey",
        "iam:DeleteAccessKey",
        "iam:ListAccessKeys",
        "iam:ListUsers",
        "iam:UpdateAccessKey"
      ],
      "Effect": "Allow",
      "Resource": "*"
    }
  ]
}
EOF
}

resource "aws_cloudwatch_log_group" "this" {
  name              = "/aws/lambda/${var.function_name}"
  retention_in_days = 90
}

resource "aws_cloudwatch_event_rule" "this" {
  name                = "${var.function_name}-scheduler"
  schedule_expression = "cron(0 8 * * ? *)"
  is_enabled          = true
}

resource "aws_lambda_function" "this" {
  function_name    = "${var.function_name}"
  filename         = "../${var.function_name}.zip"
  role             = "${aws_iam_role.this.arn}"
  handler          = "main"
  source_code_hash = "${base64sha256(file("../${var.function_name}.zip"))}"
  runtime          = "go1.x"
  timeout          = 60

  environment {
    variables = {
      endpoint   = "https://slack.com/api/chat.postMessage"
      token      = "${var.slack_token}"
      expiration = "${var.expiration}"
    }
  }
}

resource "aws_lambda_permission" "this" {
  statement_id  = "AllowExecutionFromCloudWatchEvents"
  action        = "lambda:InvokeFunction"
  function_name = "${aws_lambda_function.this.function_name}"
  principal     = "events.amazonaws.com"
  source_arn    = "${aws_cloudwatch_event_rule.this.arn}"
}

resource "aws_cloudwatch_event_target" "this" {
  target_id = "${var.function_name}"
  rule      = "${aws_cloudwatch_event_rule.this.name}"
  arn       = "${aws_lambda_function.this.arn}"
}

### outputs

output "iam_role_arn" {
  value = "${aws_iam_role.this.arn}"
}

output "lambda_function_arn" {
  value = "${aws_lambda_function.this.arn}"
}
