import argparse
from pyspark.sql import SparkSession

parser = argparse.ArgumentParser()
parser.add_argument("--data-bucket")
parser.add_argument("--out-format")
parser.add_argument("--out-mode")

args = parser.parse_args()
mode = args.out_mode
out_format = args.out_format
bucket = args.data_bucket

spark = SparkSession.builder.appName("PokemonPartitionByTypeAndGeneration").getOrCreate()

frame = spark.read.parquet(f"s3://{bucket}/pokemons/parquet/")

frame.write.partitionBy("type", "generation").format(out_format).mode(mode).save(f"s3://{bucket}/pokemons/delta/")
