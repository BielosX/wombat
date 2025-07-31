package utils

import (
	"reflect"
	"strings"
)

func GetFields(t any) []reflect.StructField {
	typeOf := reflect.TypeOf(t)
	var result []reflect.StructField
	for i := 0; i < typeOf.NumField(); i++ {
		result = append(result, typeOf.Field(i))
	}
	return result
}

func ParquetTagToKeyValue(tag string) map[string]string {
	result := make(map[string]string)
	for _, entry := range strings.Split(tag, ",") {
		keyValue := strings.Split(strings.TrimSpace(entry), "=")
		result[keyValue[0]] = keyValue[1]
	}
	return result
}
