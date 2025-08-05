import sys
import pandas
import boto3
from awsglue.utils import getResolvedOptions
from awsglue.context import GlueContext
from awsglue.job import Job
from pyspark import SparkContext
from pyspark.sql.functions import col
from datetime import datetime
from io import BytesIO

def group_and_count(frame, column):
    return frame.groupBy(column).count().sort(col("count").desc()).toPandas()

def write_cell(worksheet, col_index, rows, frame, even_format, odd_format):
    for row_num in range(0, rows):
        cell_value = frame.iloc[row_num, col_num]
        row_index = row_num + 1
        if row_num % 2 == 0:
            worksheet.write(row_index, col_index, cell_value, even_format)
        else:
            worksheet.write(row_index, col_index, cell_value, odd_format)

JOB_NAME = "JOB_NAMEJ"
GLUE_DB_NAME = 'glue_db_name'
GLUE_TABLE_NAME = 'glue_table_name'
OUT_BUCKET = "output_bucket"

client = boto3.client('s3')
args = getResolvedOptions(sys.argv, [JOB_NAME, GLUE_DB_NAME, GLUE_TABLE_NAME, OUT_BUCKET])
job_name = args[JOB_NAME]
db_name = args[GLUE_DB_NAME]
table_name = args[GLUE_TABLE_NAME]
out_bucket = args[OUT_BUCKET]

spark_ctx = SparkContext()
glue_ctx = GlueContext(spark_ctx)
job = Job(glue_ctx)
job.init(job_name, args)

dynamic_frame = glue_ctx.create_dynamic_frame_from_catalog(database=db_name, table_name=table_name)
frame = dynamic_frame.toDF()

by_type = group_and_count(frame, "type")
by_generation = group_and_count(frame, "generation")

buffer = BytesIO()

by_type_columns = len(by_type.columns)
by_generation_columns = len(by_generation.columns)
with pandas.ExcelWriter(buffer, engine="xlsxwriter") as writer:
    by_type.to_excel(writer, sheet_name="Report", startcol=0, index=False, header=True)
    start_col = by_type_columns+1
    by_generation.to_excel(writer, sheet_name="Report", startcol=start_col, index=False, header=True)

    workbook  = writer.book
    worksheet = writer.sheets["Report"]

    header_format = workbook.add_format({
        'bold': True,
        'font_color': 'white',
        'bg_color': '#4F81BD',
        'align': 'center'
    })

    data_commons = {
        'align': 'center',
        'valign': 'vcenter'
    }

    even_cell = workbook.add_format({
        'bg_color': "#8BB7ED",
    } | data_commons)

    odd_cell = workbook.add_format({
        'bg_color': "#BAD0EC",
    } | data_commons)

    rows = len(by_type.index)
    for col_num, value in enumerate(by_type.columns):
        worksheet.write(0, col_num, value, header_format)
        write_cell(worksheet, col_num, rows, by_type, even_cell, odd_cell)

    rows = len(by_generation.index)
    for col_num, value in enumerate(by_generation.columns):
        col_index = start_col+col_num
        worksheet.write(0, col_index, value, header_format)
        write_cell(worksheet, col_index, rows, by_generation, even_cell, odd_cell)


timestamp = datetime.now().isoformat()
buffer.seek(0)

client.upload_fileobj(buffer, Bucket=out_bucket, Key=f"reports/report-{timestamp}.xlsx")

job.commit()