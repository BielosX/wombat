package main

import (
	"context"
	"fmt"
	"os"

	"github.com/BielosX/wombat/poke-scraper/src/csv"

	"github.com/BielosX/wombat/poke-scraper/src/parquet"

	"github.com/BielosX/wombat/poke-scraper/src/pokeapi"
	"github.com/BielosX/wombat/poke-scraper/src/s3"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
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

type ScraperResult struct {
	ParquetFileName string `json:"parquetFileName"`
	CsvFileName     string `json:"csvFileName"`
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

func handleScraping(request Schedule) (*ScraperResult, error) {
	sugar.Infof("Starting Scrapping Handler, limit: %d offset: %d",
		request.Limit,
		request.Offset)
	limit := request.Limit
	offset := request.Offset
	client := pokeapi.NewClient(sugar)
	pokemons, err := client.ListPokemons(limit, offset)
	if err != nil {
		return nil, err
	}
	firstId := offset + 1
	resultsCount := int32(len(pokemons))
	sugar.Infof("Got %d Pokemon results", resultsCount)
	if resultsCount > 0 {
		pokemonWriter, err := parquet.NewPokemonWriter()
		if err != nil {
			sugar.Errorf("Failed to create Pokemon Parquet Writer: %s", err)
			return nil, err
		}
		csvWriter := csv.NewPokemonWriter()
		if err := csvWriter.WriteHeader(); err != nil {
			return nil, err
		}
		for _, pokemon := range pokemons {
			generation, err := client.GetPokemonGeneration(pokemon.Species)
			if err != nil {
				sugar.Errorf("Failed to get Pokemon Generation: %s", err)
				return nil, err
			}
			entries, err := parquet.ToPokemon(pokemon, generation)
			if err != nil {
				sugar.Errorf("Failed to parse Pokemon response: %s", err)
				return nil, err
			}
			for _, entry := range entries {
				entry.Generation = generation
				sugar.Infof("Writing Pokemon %s", entry.Name)
				err = pokemonWriter.WritePokemon(&entry)
				if err != nil {
					sugar.Errorf("Error writing Pokemon to Parquet: %s", err)
					return nil, err
				}
				err = csvWriter.Write(entry)
				if err != nil {
					sugar.Errorf("Error writing Pokemon to CSV: %s", err)
					return nil, err
				}
			}
		}
		if err := pokemonWriter.Finish(); err != nil {
			return nil, err
		}
		if err := csvWriter.Finish(); err != nil {
			return nil, err
		}
		parquetFileName := fmt.Sprintf("pokemons/%d_%d.parquet", firstId, firstId+resultsCount)
		csvFileName := fmt.Sprintf("pokemons/%d_%d.csv", firstId, firstId+resultsCount)
		sugar.Infof("Sending parquet file of size %d to S3", pokemonWriter.Size())
		err = s3Client.PutFile(pokemonWriter.BufferReader(), bucketName, parquetFileName)
		if err != nil {
			return nil, err
		}
		sugar.Infof("Sending CSV file of size %d to S3", csvWriter.Size())
		err = s3Client.PutFile(csvWriter.BufferReader(), bucketName, csvFileName)
		if err != nil {
			return nil, err
		}
		return &ScraperResult{
			CsvFileName:     csvFileName,
			ParquetFileName: parquetFileName,
		}, nil
	}
	return nil, nil
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
	cfg, err := config.LoadDefaultConfig(context.Background(), config.WithRegion(os.Getenv(region)))
	if err != nil {
		sugar.Fatal("Failed to load SDK config")
	}
	s3Client = s3.NewClient(cfg)
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
