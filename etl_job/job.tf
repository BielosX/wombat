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

variable "script_path" {
  type    = string
  default = ""
}

resource "aws_s3_object" "etl" {
  bucket      = aws_s3_bucket.jobs.id
  key         = "main.py"
  source      = var.script_path
  source_hash = filemd5(var.script_path)
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

resource "aws_glue_job" "etl_job" {
  for_each          = local.table_name_to_format
  name              = "pokemon-etl-${each.key}-job"
  role_arn          = aws_iam_role.job_role.arn
  glue_version      = "5.0"
  worker_type       = "G.1X"
  number_of_workers = 2
  execution_class   = "STANDARD"
  timeout           = 60 * 5
  max_retries       = 0

  execution_property {
    max_concurrent_runs = 2
  }

  command {
    script_location = "s3://${aws_s3_object.etl.bucket}/${aws_s3_object.etl.key}"
    name            = "glueetl"
    python_version  = "3"
  }

  default_arguments = {
    "--job-language"                     = "python"
    "--enable-spark-ui"                  = "true"
    "--enable-continuous-cloudwatch-log" = "true"
    "--glue_db_name"                     = local.catalog_db_name
    "--glue_table_name"                  = each.key
    "--glue_table_format"                = each.value
    "--output_bucket"                    = local.bucket_name
    "--writer_mode"                      = "append"
  }
}