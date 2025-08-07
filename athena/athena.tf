module "bucket" {
  source        = "../private_bucket"
  name_prefix   = "athena-output-"
  force_destroy = true
}

resource "aws_athena_workgroup" "workgroup" {
  name          = "pokemon-workgroup"
  force_destroy = true
  configuration {
    enforce_workgroup_configuration = true
    result_configuration {
      output_location = "s3://${module.bucket.bucket_name}/"
    }
  }
}