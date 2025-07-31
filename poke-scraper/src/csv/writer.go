package csv

import (
	"encoding/csv"
	"fmt"
	"io"
	"reflect"

	"github.com/BielosX/wombat/poke-scraper/src/parquet"
	"github.com/BielosX/wombat/poke-scraper/src/utils"
	"github.com/xitongsys/parquet-go-source/buffer"
)

type PokemonWriter struct {
	buffer *buffer.BufferFile
	writer *csv.Writer
	fields []reflect.StructField
}

const InitialCapacity = 16 * 1024 * 1024

func NewPokemonWriter() PokemonWriter {
	bufferFile := buffer.NewBufferFileCapacity(InitialCapacity)
	writer := csv.NewWriter(bufferFile)
	pokemon := parquet.Pokemon{}
	fields := utils.GetFields(pokemon)
	return PokemonWriter{
		buffer: bufferFile,
		writer: writer,
		fields: fields,
	}
}

func (w *PokemonWriter) WriteHeader() error {
	var parquetNames []string
	for _, field := range w.fields {
		tag := field.Tag.Get("parquet")
		properties := utils.ParquetTagToKeyValue(tag)
		parquetNames = append(parquetNames, properties["name"])
	}
	return w.writer.Write(parquetNames)
}

func (w *PokemonWriter) Write(pokemon parquet.Pokemon) error {
	value := reflect.ValueOf(pokemon)
	var converted []string
	for _, field := range w.fields {
		value.FieldByName(field.Name)
		fieldValue := fmt.Sprint(value.FieldByName(field.Name).Interface())
		converted = append(converted, fieldValue)
	}
	return w.writer.Write(converted)
}

func (w *PokemonWriter) Finish() error {
	w.writer.Flush()
	_, err := w.buffer.Seek(0, io.SeekStart)
	return err
}

func (w *PokemonWriter) Size() int {
	return len(w.buffer.Bytes())
}

func (w *PokemonWriter) BufferReader() io.Reader {
	return w.buffer
}
