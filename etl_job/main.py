import sys
from awsglue.utils import getResolvedOptions
from awsglue.context import GlueContext
from awsglue.job import Job
from pyspark import SparkContext

JOB_NAME = 'JOB_NAME'
GLUE_DB_NAME = 'glue_db_name'
GLUE_TABLE_NAME = 'glue_table_name'
TABLE_FORMAT = 'glue_table_format'
OUT_BUCKET = "output_bucket"
WRITER_MODE = "writer_mode"

args = getResolvedOptions(sys.argv, [JOB_NAME, GLUE_DB_NAME, GLUE_TABLE_NAME, TABLE_FORMAT, OUT_BUCKET, WRITER_MODE])
job_name = args[JOB_NAME]
db_name = args[GLUE_DB_NAME]
table_name = args[GLUE_TABLE_NAME]
out_bucket = args[OUT_BUCKET]
table_format = args[TABLE_FORMAT]
writer_mode = args[WRITER_MODE]
spark_ctx = SparkContext()
glue_ctx = GlueContext(spark_ctx)
job = Job(glue_ctx)
job.init(job_name, args)

frame = glue_ctx.create_dynamic_frame_from_catalog(database=db_name, table_name=table_name)
data_frame = frame.toDF()

data_frame.write.partitionBy("type", "generation") \
    .format(table_format).mode(writer_mode).save(f"s3://{out_bucket}/pokemons/partitioned/{table_format}/")

job.commit()