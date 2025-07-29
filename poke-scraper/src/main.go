package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"sync"

	"github.com/aws/aws-sdk-go-v2/aws"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/google/uuid"
	"github.com/xitongsys/parquet-go-source/buffer"
	"github.com/xitongsys/parquet-go/writer"
	"go.uber.org/zap"
)

var sugar *zap.SugaredLogger
var s3Client *s3.Client
var bucketName string

type ScheduleRequest struct {
	PageSize    int32 `json:"pageSize"`
	StartOffset int32 `json:"startOffset"`
	PageCount   int32 `json:"pageCount"`
}

type Schedule struct {
	Limit  int32 `json:"limit"`
	Offset int32 `json:"offset"`
}

type PokemonTypeEntry struct {
	Name string `json:"name"`
}

type PokemonType struct {
	Slot int32            `json:"slot"`
	Type PokemonTypeEntry `json:"type"`
}

type PokemonResponse struct {
	Name   string        `json:"name"`
	Weight int32         `json:"weight"`
	Height int32         `json:"height"`
	Types  []PokemonType `json:"types"`
}

func scheduleTasks(request ScheduleRequest) ([]Schedule, error) {
	sugar.Infof("Starting Schedule Tasks Handler, pageSize: %d, startOffset: %d, pageCount: %d",
		request.PageSize,
		request.StartOffset,
		request.PageCount)
	result := make([]Schedule, 0, request.PageCount)
	for i := int32(0); i < request.PageCount; i++ {
		pageSize := request.PageSize
		offset := request.StartOffset
		result = append(result, Schedule{Limit: pageSize, Offset: offset + i*pageSize})
	}
	return result, nil
}

func fetchPokemon(url string,
	client *http.Client,
	errChan chan<- error,
	resultChan chan<- PokemonResponse,
	waitGroup *sync.WaitGroup) {
	defer waitGroup.Done()
	sugar.Infof("Fetching PokemonResponse %s", url)
	response, err := client.Get(url)
	if err != nil {
		errChan <- err
		return
	}
	var pokemon PokemonResponse
	decoder := json.NewDecoder(response.Body)
	err = decoder.Decode(&pokemon)
	if err != nil {
		errChan <- err
		return
	}
	resultChan <- pokemon
	err = response.Body.Close()
	if err != nil {
		errChan <- err
		return
	}
}

type PokemonListResultEntry struct {
	Url string `json:"url"`
}

type PokemonListResult struct {
	Results []PokemonListResultEntry `json:"results"`
}

type Pokemon struct {
	Name   string `parquet:"name=name, type=BYTE_ARRAY, convertedtype=UTF8"`
	Weight int32  `parquet:"name=weight, type=INT32"`
	Height int32  `parquet:"name=height, type=INT32"`
	Type   string `parquet:"name=type, type=BYTE_ARRAY, convertedtype=UTF8"`
}

func handleScraping(request Schedule) (string, error) {
	sugar.Infof("Starting Scrapping Handler, limit: %d offset: %d",
		request.Limit,
		request.Offset)
	client := http.Client{}
	limit := request.Limit
	offset := request.Offset
	url := fmt.Sprintf("https://pokeapi.co/api/v2/pokemon?limit=%d&offset=%d", limit, offset)
	response, err := client.Get(url)
	if err != nil {
		return "", err
	}
	var result PokemonListResult
	decoder := json.NewDecoder(response.Body)
	err = decoder.Decode(&result)
	if err != nil {
		return "", err
	}
	err = response.Body.Close()
	if err != nil {
		return "", err
	}
	var waitGroup sync.WaitGroup
	errChan := make(chan error, len(result.Results))
	resultChan := make(chan PokemonResponse, len(result.Results))
	bufferFile := buffer.NewBufferFile()
	parquetWriter, err := writer.NewParquetWriter(bufferFile, new(Pokemon), 10)
	if err != nil {
		return "", err
	}
	fileName := fmt.Sprintf("%s.parquet", uuid.New().String())
	for _, entry := range result.Results {
		waitGroup.Add(1)
		go fetchPokemon(entry.Url, &client, errChan, resultChan, &waitGroup)
	}
	waitGroup.Wait()
	close(errChan)
	close(resultChan)
	sugar.Infof("Workers finished")
	var errs []error
	for e := range errChan {
		if e != nil {
			errs = append(errs, e)
		}
	}
	if len(errs) > 0 {
		return "", errors.Join(errs...)
	}
	for pokemon := range resultChan {
		for _, pokemonType := range pokemon.Types {
			entry := Pokemon{
				Name:   pokemon.Name,
				Weight: pokemon.Weight,
				Height: pokemon.Height,
				Type:   pokemonType.Type.Name,
			}
			sugar.Infof("Writing Pokemon %s", entry.Name)
			err = parquetWriter.Write(entry)
			if err != nil {
				sugar.Errorf("Error writing Pokemon to Parquet: %s", err)
				return "", err
			}
		}
	}
	err = parquetWriter.WriteStop()
	if err != nil {
		return "", err
	}
	err = parquetWriter.Flush(false)
	if err != nil {
		return "", err
	}
	_, err = bufferFile.Seek(0, io.SeekStart)
	if err != nil {
		return "", err
	}
	sugar.Infof("Sending parquet file of size %d to S3", len(bufferFile.Bytes()))
	_, err = s3Client.PutObject(context.Background(), &s3.PutObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(fileName),
		Body:   bufferFile,
	})
	if err != nil {
		return "", err
	}
	return fileName, nil
}

func syncLogger() {
	_ = sugar.Sync()
}

func main() {
	logger, _ := zap.NewDevelopment(zap.AddStacktrace(zap.FatalLevel))
	sugar = logger.Sugar()
	defer syncLogger()
	region := os.Getenv("AWS_REGION")
	bucketName = os.Getenv("BUCKET_NAME")
	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion(os.Getenv(region)))
	if err != nil {
		sugar.Fatal("Failed to load SDK config")
	}
	s3Client = s3.NewFromConfig(cfg)
	handler := os.Getenv("_HANDLER")
	switch handler {
	case "scraper":
		lambda.Start(handleScraping)
	case "scheduler":
		lambda.Start(scheduleTasks)
	default:
		sugar.Fatal("Unknown Handler %s", handler)
	}
}
