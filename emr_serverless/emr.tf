resource "aws_emrserverless_application" "emr" {
  name          = "pokemon-emr"
  release_label = "emr-7.9.0"
  type          = "spark"

  maximum_capacity {
    cpu    = "3 vCPU"
    memory = "6 GB"
  }

  initial_capacity {
    initial_capacity_type = "Driver"
    initial_capacity_config {
      worker_count = 1
      worker_configuration {
        cpu    = "1 vCPU"
        memory = "2 GB"
      }
    }
  }

  initial_capacity {
    initial_capacity_type = "Executor"
    initial_capacity_config {
      worker_count = 2
      worker_configuration {
        cpu    = "1 vCPU"
        memory = "2 GB"
      }
    }
  }
}

data "aws_caller_identity" "current" {}

locals {
  account_id        = data.aws_caller_identity.current.account_id
  deployment_prefix = "emr-jobs-"
  data_bucket_arn   = data.terraform_remote_state.poke_scraper.outputs.data_bucket_arn
  data_bucket_name  = data.terraform_remote_state.poke_scraper.outputs.data_bucket_name
}

data "aws_iam_policy_document" "assume_role" {
  statement {
    effect  = "Allow"
    actions = ["sts:AssumeRole"]
    principals {
      identifiers = ["emr-serverless.amazonaws.com"]
      type        = "Service"
    }
  }
}

resource "aws_iam_role" "execution_role" {
  assume_role_policy = data.aws_iam_policy_document.assume_role.json
}

resource "aws_cloudwatch_log_group" "job_log_group" {
  name_prefix = "/pokemon/emr/"
}

data "aws_iam_policy_document" "policy" {
  statement {
    effect  = "Allow"
    actions = ["s3:PutObject", "s3:ListBucket", "s3:GetObject"]
    resources = [
      local.data_bucket_arn,
      "${local.data_bucket_arn}/*"
    ]
  }
  statement {
    effect  = "Allow"
    actions = ["s3:ListBucket", "s3:GetObject"]
    resources = [
      module.bucket.bucket_arn,
      "${module.bucket.bucket_arn}/*"
    ]
  }
  statement {
    effect    = "Allow"
    actions   = ["logs:DescribeLogGroups"]
    resources = ["arn:aws:logs:${var.region}:${local.account_id}:*"]
  }
  statement {
    effect = "Allow"
    actions = [
      "logs:PutLogEvents",
      "logs:CreateLogStream",
      "logs:DescribeLogStreams"
    ]
    resources = ["${aws_cloudwatch_log_group.job_log_group.arn}:*"]
  }
}

resource "aws_iam_policy" "policy" {
  policy = data.aws_iam_policy_document.policy.json
}

resource "aws_iam_role_policy_attachment" "policy_attachment" {
  policy_arn = aws_iam_policy.policy.arn
  role       = aws_iam_role.execution_role.id
}

module "bucket" {
  source        = "../private_bucket"
  name_prefix   = local.deployment_prefix
  force_destroy = true
}

output "deployment_bucket_name" {
  value = module.bucket.bucket_name
}

output "execution_role_arn" {
  value = aws_iam_role.execution_role.arn
}

output "data_bucket_name" {
  value = local.data_bucket_name
}

output "job_log_group_name" {
  value = aws_cloudwatch_log_group.job_log_group.name
}

output "application_id" {
  value = aws_emrserverless_application.emr.id
}