package storage

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/smithy-go"
)

type s3Client interface {
	PutObject(context.Context, *s3.PutObjectInput, ...func(*s3.Options)) (*s3.PutObjectOutput, error)
	GetObject(context.Context, *s3.GetObjectInput, ...func(*s3.Options)) (*s3.GetObjectOutput, error)
}

type S3 struct {
	client s3Client
	bucket string
	prefix string
}

func NewS3(client s3Client, bucket, prefix string) (*S3, error) {
	if client == nil || strings.TrimSpace(bucket) == "" {
		return nil, errors.New("S3 storage requires a client and bucket")
	}
	prefix = strings.Trim(strings.TrimSpace(prefix), "/")
	return &S3{client: client, bucket: bucket, prefix: prefix}, nil
}

func (store *S3) PutIfAbsent(ctx context.Context, key, mediaType string, data []byte) error {
	key, err := store.key(key)
	if err != nil {
		return err
	}
	_, err = store.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(store.bucket), Key: aws.String(key),
		Body: bytes.NewReader(data), ContentType: aws.String(mediaType),
		CacheControl: aws.String("private, max-age=31536000, immutable"),
		IfNoneMatch:  aws.String("*"),
	})
	if err == nil {
		return nil
	}
	var apiError smithy.APIError
	if !errors.As(err, &apiError) || apiError.ErrorCode() != "PreconditionFailed" {
		return fmt.Errorf("put S3 object: %w", err)
	}
	existing, readErr := store.Get(ctx, key)
	if readErr != nil {
		return readErr
	}
	if !bytes.Equal(existing, data) {
		return ErrObjectConflict
	}
	return nil
}

func (store *S3) Get(ctx context.Context, key string) ([]byte, error) {
	key, err := store.key(key)
	if err != nil {
		return nil, err
	}
	output, err := store.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(store.bucket), Key: aws.String(key),
	})
	if err != nil {
		return nil, fmt.Errorf("get S3 object: %w", err)
	}
	data, err := io.ReadAll(io.LimitReader(output.Body, 64<<20))
	if err != nil {
		_ = output.Body.Close()
		return nil, fmt.Errorf("read S3 object: %w", err)
	}
	if err := output.Body.Close(); err != nil {
		return nil, fmt.Errorf("close S3 object: %w", err)
	}
	return data, nil
}

func (store *S3) key(key string) (string, error) {
	key = strings.TrimSpace(key)
	if key == "" || strings.HasPrefix(key, "/") || strings.Contains(key, "..") || strings.Contains(key, "\\") {
		return "", errors.New("invalid S3 object key")
	}
	if store.prefix != "" && !strings.HasPrefix(key, store.prefix+"/") {
		return store.prefix + "/" + key, nil
	}
	return key, nil
}
