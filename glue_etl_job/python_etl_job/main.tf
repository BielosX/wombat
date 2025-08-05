locals {
  arguments = var.requirements_key == "" ? var.arguments : merge(var.arguments, {
    "--python-modules-installer-option" = "-r"
    "--additional-python-modules"       = "s3://${var.script_bucket}/${var.requirements_key}"
  })
}

resource "aws_glue_job" "etl_partition_job" {
  name              = var.name
  role_arn          = var.role_arn
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
    script_location = "s3://${var.script_bucket}/${var.script_key}"
    name            = "glueetl"
    python_version  = "3"
  }

  default_arguments = local.arguments
}