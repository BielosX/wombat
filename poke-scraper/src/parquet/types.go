package parquet

type Pokemon struct {
	Name   string `parquet:"name=name, type=BYTE_ARRAY, convertedtype=UTF8"`
	Weight int32  `parquet:"name=weight, type=INT32"`
	Height int32  `parquet:"name=height, type=INT32"`
	Type   string `parquet:"name=type, type=BYTE_ARRAY, convertedtype=UTF8"`
}
