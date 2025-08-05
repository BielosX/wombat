resource "aws_s3_bucket" "jobs" {
  bucket_prefix = "glue-etl-jobs-"
}

resource "aws_s3_bucket_public_access_block" "jobs" {
  bucket                  = aws_s3_bucket.jobs.id
  block_public_acls       = true
  block_public_policy     = true
  ignore_public_acls      = true
  restrict_public_buckets = true
}

resource "aws_s3_bucket_ownership_controls" "jobs_ownership" {
  bucket = aws_s3_bucket.jobs.id
  rule {
    object_ownership = "BucketOwnerEnforced"
  }
}

variable "partition_by_path" {
  type    = string
  default = ""
}

variable "report_path" {
  type    = string
  default = ""
}

variable "report_requirements_path" {
  type    = string
  default = ""
}

resource "aws_s3_object" "partition_by" {
  bucket      = aws_s3_bucket.jobs.id
  key         = "partition_by.py"
  source      = var.partition_by_path
  source_hash = filemd5(var.partition_by_path)
}

resource "aws_s3_object" "report" {
  bucket      = aws_s3_bucket.jobs.id
  key         = "report.py"
  source      = var.report_path
  source_hash = filemd5(var.report_path)
}

resource "aws_s3_object" "report_requirements" {
  bucket      = aws_s3_bucket.jobs.id
  key         = "requirements.txt"
  source      = var.report_requirements_path
  source_hash = filemd5(var.report_requirements_path)
}

data "aws_iam_policy_document" "assume_role" {
  statement {
    effect  = "Allow"
    actions = ["sts:AssumeRole"]
    principals {
      identifiers = ["glue.amazonaws.com"]
      type        = "Service"
    }
  }
}

resource "aws_iam_role" "job_role" {
  assume_role_policy = data.aws_iam_policy_document.assume_role.json
}

data "aws_iam_policy_document" "s3_access" {
  statement {
    effect = "Allow"
    actions = [
      "s3:PutObject",
      "s3:GetObject",
      "s3:ListBucket"
    ]
    resources = [
      local.bucket_arn,
      "${local.bucket_arn}/*",
      aws_s3_bucket.jobs.arn,
      "${aws_s3_bucket.jobs.arn}/*"
    ]
  }
}

resource "aws_iam_policy" "s3_access" {
  policy = data.aws_iam_policy_document.s3_access.json
}

resource "aws_iam_role_policy_attachment" "glue_service_role" {
  policy_arn = "arn:aws:iam::aws:policy/service-role/AWSGlueServiceRole"
  role       = aws_iam_role.job_role.id
}

resource "aws_iam_role_policy_attachment" "glue_s3_access" {
  policy_arn = aws_iam_policy.s3_access.arn
  role       = aws_iam_role.job_role.id
}

locals {
  table_name_to_format = {
    "csv"     = "csv"
    "parquet" = "parquet"
  }
}

module "elt_partition_job" {
  for_each = local.table_name_to_format
  source   = "./python_etl_job"
  arguments = {
    "--job-language"                     = "python"
    "--enable-spark-ui"                  = "true"
    "--enable-continuous-cloudwatch-log" = "true"
    "--glue_db_name"                     = local.catalog_db_name
    "--glue_table_name"                  = each.key
    "--glue_table_format"                = each.value
    "--output_bucket"                    = local.bucket_name
    "--writer_mode"                      = "append"
  }
  name          = "pokemon-etl-${each.key}-partition-job"
  role_arn      = aws_iam_role.job_role.arn
  script_bucket = aws_s3_object.partition_by.bucket
  script_key    = aws_s3_object.partition_by.key
}

module "elt_report_job" {
  source = "./python_etl_job"
  arguments = {
    "--job-language"                     = "python"
    "--enable-spark-ui"                  = "true"
    "--enable-continuous-cloudwatch-log" = "true"
    "--glue_db_name"                     = local.catalog_db_name
    "--glue_table_name"                  = local.table_name_to_format["parquet"]
    "--output_bucket"                    = local.bucket_name
  }
  name             = "pokemon-etl-report"
  role_arn         = aws_iam_role.job_role.arn
  script_bucket    = aws_s3_object.report.bucket
  script_key       = aws_s3_object.report.key
  requirements_key = aws_s3_object.report_requirements.key
}
