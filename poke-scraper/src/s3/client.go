package s3

import (
	"context"
	"io"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type Client struct {
	client *s3.Client
}

func NewClient(cfg aws.Config) *Client {
	return &Client{
		client: s3.NewFromConfig(cfg),
	}
}

func (c *Client) PutFile(reader io.Reader, bucket, key string) error {
	_, err := c.client.PutObject(context.Background(), &s3.PutObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
		Body:   reader,
	})
	return err
}
