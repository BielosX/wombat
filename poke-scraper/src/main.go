package main

import (
	"context"
	"fmt"
	"os"

	"github.com/BielosX/wombat/poke-scraper/src/parquet"

	"github.com/BielosX/wombat/poke-scraper/src/pokeapi"
	"github.com/aws/aws-sdk-go-v2/aws"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
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

func handleScraping(request Schedule) (string, error) {
	sugar.Infof("Starting Scrapping Handler, limit: %d offset: %d",
		request.Limit,
		request.Offset)
	limit := request.Limit
	offset := request.Offset
	client := pokeapi.NewClient(sugar)
	pokemons, err := client.ListPokemons(limit, offset)
	if err != nil {
		return "", err
	}
	firstId := offset + 1
	resultsCount := int32(len(pokemons))
	sugar.Infof("Got %d Pokemon results", resultsCount)
	if resultsCount > 0 {
		pokemonWriter, err := parquet.NewPokemonWriter()
		if err != nil {
			sugar.Errorf("Failed to create Pokemon Parquet Writer: %s", err)
			return "", err
		}
		for _, pokemon := range pokemons {
			generation, err := client.GetPokemonGeneration(pokemon.Species)
			if err != nil {
				sugar.Errorf("Failed to get Pokemon Generation: %s", err)
				return "", err
			}
			entries, err := parquet.ToPokemon(pokemon, generation)
			if err != nil {
				sugar.Errorf("Failed to parse Pokemon response: %s", err)
				return "", err
			}
			for _, entry := range entries {
				entry.Generation = generation
				sugar.Infof("Writing Pokemon %s", entry.Name)
				err = pokemonWriter.WritePokemon(&entry)
				if err != nil {
					sugar.Errorf("Error writing Pokemon to Parquet: %s", err)
					return "", err
				}
			}
		}
		if err := pokemonWriter.Finish(); err != nil {
			return "", err
		}
		fileName := fmt.Sprintf("pokemons/%d_%d.parquet", firstId, firstId+resultsCount)
		sugar.Infof("Sending parquet file of size %d to S3", pokemonWriter.Size())
		_, err = s3Client.PutObject(context.Background(), &s3.PutObjectInput{
			Bucket: aws.String(bucketName),
			Key:    aws.String(fileName),
			Body:   pokemonWriter.BufferReader(),
		})
		if err != nil {
			return "", err
		}
		return fileName, nil
	}
	return "", nil
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
