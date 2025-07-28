package main

import (
	"os"

	"github.com/aws/aws-lambda-go/lambda"
	"go.uber.org/zap"
)

var sugar *zap.SugaredLogger

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

func handleScraping(request Schedule) {
	sugar.Infof("Starting Scrapping Handler, limit: %d offset: %d",
		request.Limit,
		request.Offset)
}

func syncLogger() {
	_ = sugar.Sync()
}

func main() {
	logger, _ := zap.NewDevelopment(zap.AddStacktrace(zap.FatalLevel))
	sugar = logger.Sugar()
	defer syncLogger()
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
