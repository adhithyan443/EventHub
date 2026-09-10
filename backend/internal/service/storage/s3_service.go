package storage

import (
	"context"
	"fmt"
	"io"
	"log/slog"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"

	appConfig "github.com/adhithyan443/EventHub/backend/config"
)

type S3Service struct {
	client *s3.Client
	bucket string
	logger *slog.Logger
}

func NewS3Service(
	cfg appConfig.Config,
	logger *slog.Logger,
) (*S3Service, error) {
	awsCfg, err := config.LoadDefaultConfig(
		context.Background(),
		config.WithRegion(cfg.AWSRegion),
	)
	if err != nil {
		logger.Error(
			"s3_config_initialization_failed",
			"error", err,
		)

		return nil, fmt.Errorf("failed to load AWS configuration: %w", err)
	}

	client := s3.NewFromConfig(awsCfg)

	logger.Info(
		"s3_service_initialized",
		"region", cfg.AWSRegion,
		"bucket", cfg.AWSS3Bucket,
	)

	return &S3Service{
		client: client,
		bucket: cfg.AWSS3Bucket,
		logger: logger,
	}, nil
}

func (s *S3Service) Upload(
	ctx context.Context,
	key string,
	file io.Reader,
	contentType string,
) error {
	_, err := s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(s.bucket),
		Key:         aws.String(key),
		Body:        file,
		ContentType: aws.String(contentType),
	})
	if err != nil {
		s.logger.Error(
			"s3_object_upload_failed",
			"bucket", s.bucket,
			"key", key,
			"content_type", contentType,
			"error", err,
		)

		return fmt.Errorf("failed to upload object to S3: %w", err)
	}

	s.logger.Info(
		"s3_object_uploaded",
		"bucket", s.bucket,
		"key", key,
		"content_type", contentType,
	)

	return nil
}
