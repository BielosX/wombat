resource "aws_s3_bucket" "output" {
  bucket_prefix = "athena-output-"
  force_destroy = true
}

resource "aws_s3_bucket_public_access_block" "output" {
  bucket                  = aws_s3_bucket.output.id
  block_public_acls       = true
  block_public_policy     = true
  ignore_public_acls      = true
  restrict_public_buckets = true
}

resource "aws_s3_bucket_ownership_controls" "output_ownership" {
  bucket = aws_s3_bucket.output.id
  rule {
    object_ownership = "BucketOwnerEnforced"
  }
}

resource "aws_athena_workgroup" "workgroup" {
  name          = "pokemon-workgroup"
  force_destroy = true
  configuration {
    enforce_workgroup_configuration = true
    result_configuration {
      output_location = "s3://${aws_s3_bucket.output.bucket}/"
    }
  }
}