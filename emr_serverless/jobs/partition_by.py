import argparse

from pyspark.sql import SparkSession, Column
from pyspark.sql.functions import (
    max,
    col,
    struct,
    array,
    lit,
    array_sort,
    element_at,
    when,
)

parser = argparse.ArgumentParser()
parser.add_argument("--data-bucket")
parser.add_argument("--out-format")
parser.add_argument("--out-mode")

args = parser.parse_args()
mode = args.out_mode
out_format = args.out_format
bucket = args.data_bucket

spark = SparkSession.builder.appName(
    "PokemonPartitionByTypeAndGeneration"
).getOrCreate()

stat_columns = [
    "basehp",
    "baseattack",
    "basedefense",
    "basespecialattack",
    "basespecialdefense",
    "basespeed",
]
max_expressions = [max(c).alias(c) for c in stat_columns]
frame = spark.read.parquet(f"s3://{bucket}/pokemons/parquet/")
max_values = frame.agg(*max_expressions).collect()[0]
structs = []
sorted_pairs = "sorted_pairs"
min_pair = "min_pair"
max_pair = "max_pair"
strongest = "strongest"
weakest = "weakest"
col_name = "col_name"
value_name = "col_value"
for name in stat_columns:
    max_value = max_values[name]
    normalized_value = col(name) / max_value
    structs.append(
        struct(lit(name).alias(col_name), lit(normalized_value).alias(value_name))
    )
    frame = frame.withColumn(f"{name}_norm", normalized_value)
pairs = array(*structs)


def comparator(a: Column, b: Column) -> Column:
    # noinspection PyTypeChecker
    return (
        when(a[value_name] < b[value_name], -1)
        .when(a[value_name] > b[value_name], 1)
        .otherwise(0)
    )


frame = (
    frame.withColumn(sorted_pairs, array_sort(pairs, comparator))
    .withColumn(max_pair, element_at(col(sorted_pairs), -1))
    .withColumn(min_pair, element_at(col(sorted_pairs), 1))
    .withColumn(weakest, col(min_pair)[col_name])
    .withColumn(strongest, col(max_pair)[col_name])
    .drop(sorted_pairs, min_pair, max_pair)
)

frame.write.partitionBy("type", "generation", strongest, weakest).format(
    out_format
).mode(mode).save(f"s3://{bucket}/pokemons/${out_format}/")
