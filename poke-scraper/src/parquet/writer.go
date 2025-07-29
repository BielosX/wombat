package parquet

import (
	"io"

	"go.uber.org/zap"

	"github.com/xitongsys/parquet-go-source/buffer"
	"github.com/xitongsys/parquet-go/writer"
)

type PokemonWriter struct {
	buffer *buffer.BufferFile
	writer *writer.ParquetWriter
}

const InitialCapacity = 16 * 1024 * 1024

func NewPokemonWriter(sugar *zap.SugaredLogger) (*PokemonWriter, error) {
	bufferFile := buffer.NewBufferFileCapacity(InitialCapacity)
	w, err := writer.NewParquetWriter(bufferFile, new(Pokemon), 10)
	if err != nil {
		return nil, err
	}
	return &PokemonWriter{
		buffer: bufferFile,
		writer: w,
	}, nil
}

func (w *PokemonWriter) WritePokemon(pokemon *Pokemon) error {
	return w.writer.Write(pokemon)
}

func (w *PokemonWriter) Finish() error {
	err := w.writer.WriteStop()
	if err != nil {
		return err
	}
	err = w.writer.Flush(false)
	if err != nil {
		return err
	}
	_, err = w.buffer.Seek(0, io.SeekStart)
	return err
}

func (w *PokemonWriter) Size() int {
	return len(w.buffer.Bytes())
}

func (w *PokemonWriter) BufferReader() io.Reader {
	return w.buffer
}
