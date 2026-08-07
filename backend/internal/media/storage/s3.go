package storage

import (
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/url"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/smithy-go"
	mediaapp "github.com/ilhamnugraha8944/tauco/backend/internal/media/application"
)

type S3Config struct {
	Endpoint, Region, Bucket, Prefix string
	AccessKeyID, SecretAccessKey     string
}

type s3Client interface {
	PutObject(context.Context, *s3.PutObjectInput, ...func(*s3.Options)) (*s3.PutObjectOutput, error)
	GetObject(context.Context, *s3.GetObjectInput, ...func(*s3.Options)) (*s3.GetObjectOutput, error)
	HeadObject(context.Context, *s3.HeadObjectInput, ...func(*s3.Options)) (*s3.HeadObjectOutput, error)
	DeleteObject(context.Context, *s3.DeleteObjectInput, ...func(*s3.Options)) (*s3.DeleteObjectOutput, error)
	HeadBucket(context.Context, *s3.HeadBucketInput, ...func(*s3.Options)) (*s3.HeadBucketOutput, error)
}

type s3Presigner interface {
	PresignPutObject(context.Context, *s3.PutObjectInput, ...func(*s3.PresignOptions)) (*v4.PresignedHTTPRequest, error)
}

type S3 struct {
	client    s3Client
	presigner s3Presigner
	bucket    string
	prefix    string
}

func NewS3(client s3Client, presigner s3Presigner, bucket, prefix string) (*S3, error) {
	if client == nil || presigner == nil || strings.TrimSpace(bucket) == "" {
		return nil, errors.New("S3 storage requires a client, presigner, and bucket")
	}
	prefix = strings.Trim(strings.TrimSpace(prefix), "/")
	if strings.Contains(prefix, "..") || strings.Contains(prefix, "\\") {
		return nil, errors.New("S3 prefix must be a safe object-key prefix")
	}
	return &S3{client: client, presigner: presigner, bucket: bucket, prefix: prefix}, nil
}

func NewS3FromConfig(config S3Config) (*S3, error) {
	endpoint, err := url.Parse(strings.TrimSpace(config.Endpoint))
	if err != nil || endpoint.Scheme == "" || endpoint.Host == "" || endpoint.User != nil || endpoint.RawQuery != "" || endpoint.Fragment != "" {
		return nil, errors.New("S3 endpoint must be an absolute URL without credentials, query, or fragment")
	}
	if strings.TrimSpace(config.Region) == "" || strings.TrimSpace(config.Bucket) == "" ||
		strings.TrimSpace(config.AccessKeyID) == "" || strings.TrimSpace(config.SecretAccessKey) == "" {
		return nil, errors.New("S3 region, bucket, access key, and secret key are required")
	}
	credentials := aws.NewCredentialsCache(staticCredentials{
		accessKeyID: config.AccessKeyID, secretAccessKey: config.SecretAccessKey,
	})
	client := s3.NewFromConfig(aws.Config{Region: config.Region, Credentials: credentials}, func(options *s3.Options) {
		options.BaseEndpoint = aws.String(endpoint.String())
		options.UsePathStyle = true
	})
	return NewS3(client, s3.NewPresignClient(client), config.Bucket, config.Prefix)
}

type staticCredentials struct{ accessKeyID, secretAccessKey string }

func (provider staticCredentials) Retrieve(context.Context) (aws.Credentials, error) {
	return aws.Credentials{
		AccessKeyID: provider.accessKeyID, SecretAccessKey: provider.secretAccessKey,
		Source: "tauco-media-s3",
	}, nil
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
	return store.GetBounded(ctx, key, mediaapp.MaxStoredObjectBytes)
}

func (store *S3) GetBounded(ctx context.Context, key string, maximum int64) ([]byte, error) {
	if maximum < 1 {
		return nil, errors.New("maximum object size must be positive")
	}
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
	data, readErr := io.ReadAll(io.LimitReader(output.Body, maximum+1))
	closeErr := output.Body.Close()
	if readErr != nil {
		return nil, fmt.Errorf("read S3 object: %w", readErr)
	}
	if closeErr != nil {
		return nil, fmt.Errorf("close S3 object: %w", closeErr)
	}
	if int64(len(data)) > maximum {
		return nil, mediaapp.ErrObjectTooLarge
	}
	return data, nil
}

func (store *S3) PresignPut(ctx context.Context, key, mediaType, sha256 string, bytes int64, lifetime time.Duration) (mediaapp.PresignedUpload, error) {
	if bytes < 1 || bytes > mediaapp.MaxUploadBytes || lifetime <= 0 || lifetime > 10*time.Minute {
		return mediaapp.PresignedUpload{}, errors.New("invalid presigned upload bounds")
	}
	if decoded, err := hex.DecodeString(sha256); err != nil || len(decoded) != 32 || sha256 != strings.ToLower(sha256) {
		return mediaapp.PresignedUpload{}, errors.New("invalid upload SHA-256")
	}
	key, err := store.key(key)
	if err != nil {
		return mediaapp.PresignedUpload{}, err
	}
	expiresAt := time.Now().UTC().Add(lifetime)
	request, err := store.presigner.PresignPutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(store.bucket), Key: aws.String(key),
		ContentLength: aws.Int64(bytes), ContentType: aws.String(mediaType),
		CacheControl: aws.String("private, no-store"),
		Metadata:     map[string]string{"sha256": sha256},
	}, func(options *s3.PresignOptions) { options.Expires = lifetime })
	if err != nil {
		return mediaapp.PresignedUpload{}, fmt.Errorf("presign S3 upload: %w", err)
	}
	headers := make(map[string]string)
	for name, values := range request.SignedHeader {
		if !strings.EqualFold(name, "host") && !strings.EqualFold(name, "content-length") && len(values) > 0 {
			headers[name] = strings.Join(values, ",")
		}
	}
	return mediaapp.PresignedUpload{URL: request.URL, Headers: headers, ExpiresAt: expiresAt}, nil
}

func (store *S3) Head(ctx context.Context, key string) (mediaapp.ObjectInfo, error) {
	key, err := store.key(key)
	if err != nil {
		return mediaapp.ObjectInfo{}, err
	}
	output, err := store.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(store.bucket), Key: aws.String(key),
	})
	if err != nil {
		var apiError smithy.APIError
		if errors.As(err, &apiError) && (apiError.ErrorCode() == "NotFound" || apiError.ErrorCode() == "NoSuchKey") {
			return mediaapp.ObjectInfo{}, mediaapp.ErrObjectNotFound
		}
		return mediaapp.ObjectInfo{}, fmt.Errorf("head S3 object: %w", err)
	}
	return mediaapp.ObjectInfo{
		Bytes: aws.ToInt64(output.ContentLength), MIMEType: aws.ToString(output.ContentType),
		SHA256: output.Metadata["sha256"],
	}, nil
}

func (store *S3) Delete(ctx context.Context, key string) error {
	key, err := store.key(key)
	if err != nil {
		return err
	}
	if _, err := store.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(store.bucket), Key: aws.String(key),
	}); err != nil {
		return fmt.Errorf("delete S3 object: %w", err)
	}
	return nil
}

func (store *S3) Health(ctx context.Context) error {
	if _, err := store.client.HeadBucket(ctx, &s3.HeadBucketInput{Bucket: aws.String(store.bucket)}); err != nil {
		return fmt.Errorf("check S3 bucket: %w", err)
	}
	return nil
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

var _ mediaapp.QuarantineStore = (*S3)(nil)
