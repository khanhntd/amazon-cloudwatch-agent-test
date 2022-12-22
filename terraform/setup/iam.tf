// Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: MIT

resource "aws_iam_role" "cwagent_role" {
  name = module.common.cwa_iam_role

  assume_role_policy = <<EOF
{
 "Version": "2012-10-17",
 "Statement": [
   {
     "Action": "sts:AssumeRole",
     "Principal": {
       "Service": "ec2.amazonaws.com",
       "Service": "ecs-tasks.amazonaws.com"
     },
     "Effect": "Allow",
     "Sid": ""
   }
 ]
}
EOF
}

data "aws_iam_policy_document" "user-managed-policy-document" {
  statement {
    actions = [
      "cloudwatch:GetMetricData",
      "cloudwatch:PutMetricData",
      "cloudwatch:ListMetrics",
      "cloudwatch:GetMetricStatistics",
      "ec2:DescribeVolumes",
      "ec2:DescribeTags",
      "ec2:DescribeInstances",
      "logs:DescribeLogStreams",
      "logs:DescribeLogGroups",
      "logs:CreateLogStream",
      "logs:CreateLogGroup",
      "logs:DeleteLogGroup",
      "logs:DeleteLogStream",
      "logs:PutRetentionPolicy",
      "logs:GetLogEvents",
      "logs:PutLogEvents",
      "dynamodb:DescribeTable",
      "dynamodb:PutItem",
      "dynamodb:CreateTable",
      "dynamodb:Query",
      "dynamodb:UpdateItem",
      "ecs:DescribeTasks",
      "ecs:ListTasks",
      "ecs:DescribeContainerInstances",
      "ecs:DescribeServices",
      "ecs:ListServices",
      "ecs:DescribeTaskDefinition",
      "ecr:GetAuthorizationToken",
      "ecr:BatchCheckLayerAvailability",
      "ecr:GetDownloadUrlForLayer",
      "ecr:BatchGetImage",
      "ssm:Describe*",
      "ssm:Get*",
      "ssm:List*",
      "s3:*",
    ]
    resources = ["*"]
  }
}

resource "aws_iam_policy" "cwagent_iam_policy" {
  name   = module.common.cwa_iam_policy
  policy = data.aws_iam_policy_document.user-managed-policy-document.json
}

resource "aws_iam_role_policy_attachment" "cwagent_server_policy_attachment" {
  role       = aws_iam_role.cwagent_role.name
  policy_arn = aws_iam_policy.cwagent_iam_policy.arn
}

resource "aws_iam_instance_profile" "cwagent_instance_profile" {
  name = module.common.cwa_iam_instance_profile
  role = aws_iam_role.cwagent_role.name
}
