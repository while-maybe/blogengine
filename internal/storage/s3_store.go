package storage

import (
	"blogengine/internal/config"
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

type S3Store struct {
	client *s3.Client
	bucket string
	tracer trace.Tracer
}

func NewS3Store(cfg config.S3Config) (*S3Store, error) {
	client := s3.New(s3.Options{
		Region:       cfg.Region,
		BaseEndpoint: aws.String(cfg.Endpoint),
		Credentials: credentials.NewStaticCredentialsProvider(
			cfg.AccessKey, cfg.SecretKey, "",
		),
		UsePathStyle: true,
	})

	return &S3Store{
		client: client,
		bucket: cfg.Bucket,
		tracer: otel.Tracer("blogengine/storage/s3"),
	}, nil
}

func (s *S3Store) Open(ctx context.Context, key string) (io.ReadCloser, error) {
	if key == "" {
		return nil, fmt.Errorf("key cannot be empty")
	}

	key = strings.TrimSpace(key)

	ctx, span := s.tracer.Start(ctx, "S3.Open", trace.WithAttributes(attribute.String("s3.key", key)))

	obj := &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	}

	objOutput, err := s.client.GetObject(ctx, obj)
	if err != nil {
		span.RecordError(err)
		return nil, err
	}

	return &spanClosingReader{
		ReadCloser: objOutput.Body,
		span:       span,
	}, nil
}

func (s *S3Store) Exists(ctx context.Context, key string) bool {
	if key == "" {
		return false
	}

	key = strings.TrimSpace(key)

	obj := &s3.HeadObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	}

	_, err := s.client.HeadObject(ctx, obj)

	return err == nil
}

func (s *S3Store) Save(ctx context.Context, key string, body io.ReadSeeker) error {

	ctx, span := s.tracer.Start(ctx, "S3.Save", trace.WithAttributes(attribute.String("s3.key", key)))
	defer span.End()

	_, err := s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
		Body:   body,
	})

	if err != nil {
		span.RecordError(err)
		return err
	}

	return nil
}

func (s *S3Store) Delete(ctx context.Context, key string) error {

	_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})

	return err
}

type spanClosingReader struct {
	io.ReadCloser
	span trace.Span
}

func (r *spanClosingReader) Close() error {
	r.span.End()
	return r.ReadCloser.Close()
}
