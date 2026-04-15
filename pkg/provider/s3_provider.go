package provider

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	smithy "github.com/aws/smithy-go"

	"github.com/jhonxue/agent-fs/pkg/s3client"
)

// S3CompatibleProvider implements StorageProvider for S3-compatible cloud storage
type S3CompatibleProvider struct {
	scheme    string
	client    *s3client.Client
	bucket    string
	endpoint  string
	accessKey string
	secretKey string
	pathStyle bool
	useSSL    bool
}

// NewS3CompatibleProvider creates a new S3-compatible storage provider
// The scheme determines which environment variables to use for configuration:
// - s3://   -> S3_* environment variables
// - r2://   -> R2_* environment variables  
// - minio:// -> MINIO_* environment variables
func NewS3CompatibleProvider(scheme string) (StorageProvider, error) {
	cfg := s3client.Config{
		Endpoint:   getEnvVar(scheme, "ENDPOINT"),
		Region:     getEnvVar(scheme, "REGION"),
		Bucket:     getEnvVar(scheme, "BUCKET"),
		AccessKeyID:     getEnvVar(scheme, "ACCESS_KEY_ID"),
		SecretAccessKey:  getEnvVar(scheme, "SECRET_ACCESS_KEY"),
		PathStyle:  scheme != "s3", // s3 uses virtual hosting style by default
		UseSSL:     true,
	}

	// Validate required config based on scheme
	if scheme == "s3" {
		if cfg.Bucket == "" {
			return nil, &providerConfigError{
				msg: fmt.Sprintf("%s provider requires %s_BUCKET environment variables",
					strings.ToUpper(scheme), strings.ToUpper(scheme)),
			}
		}
	} else {
		if cfg.Endpoint == "" || cfg.Bucket == "" {
			return nil, &providerConfigError{
				msg: fmt.Sprintf("%s provider requires %s_ENDPOINT and %s_BUCKET environment variables",
					strings.ToUpper(scheme), strings.ToUpper(scheme), strings.ToUpper(scheme)),
			}
		}
	}

	client, err := s3client.New(context.Background(), cfg)
	if err != nil {
		return nil, err
	}

	return &S3CompatibleProvider{
		scheme:    scheme,
		client:    client,
		bucket:    cfg.Bucket,
		endpoint:  cfg.Endpoint,
		accessKey: cfg.AccessKeyID,
		secretKey: cfg.SecretAccessKey,
		pathStyle: cfg.PathStyle,
		useSSL:    cfg.UseSSL,
	}, nil
}

func NewS3Provider() (StorageProvider, error) {
	return NewS3CompatibleProvider("s3")
}

func NewR2Provider() (StorageProvider, error) {
	return NewS3CompatibleProvider("r2")
}

func NewMinioProvider() (StorageProvider, error) {
	return NewS3CompatibleProvider("minio")
}

// getEnvVar returns the environment variable value for a given scheme and suffix
// e.g., for scheme "s3" and suffix "ENDPOINT", returns os.Getenv("S3_ENDPOINT")
func getEnvVar(scheme, suffix string) string {
	key := strings.ToUpper(scheme) + "_" + suffix
	return os.Getenv(key)
}

// Scheme returns the URI scheme this provider handles
func (p *S3CompatibleProvider) Scheme() string {
	return p.scheme
}

// Read reads an object from S3-compatible storage
func (p *S3CompatibleProvider) Read(ctx context.Context, key string) (io.ReadCloser, error) {
	input := &s3.GetObjectInput{
		Bucket: aws.String(p.bucket),
		Key:    aws.String(key),
	}

	output, err := p.client.GetObject(ctx, input)
	if err != nil {
		return nil, err
	}

	return output.Body, nil
}

// Write writes data to S3-compatible storage using streaming
func (p *S3CompatibleProvider) Write(ctx context.Context, key string, data io.Reader) error {
	input := &s3.PutObjectInput{
		Bucket:      aws.String(p.bucket),
		Key:         aws.String(key),
		Body:        data,
		ContentType: aws.String("application/octet-stream"),
	}

	_, err := p.client.PutObject(ctx, input)
	return err
}

// Delete deletes an object from S3-compatible storage
func (p *S3CompatibleProvider) Delete(ctx context.Context, key string) error {
	input := &s3.DeleteObjectInput{
		Bucket: aws.String(p.bucket),
		Key:    aws.String(key),
	}

	_, err := p.client.DeleteObject(ctx, input)
	return err
}

// List lists objects in a bucket with given prefix
func (p *S3CompatibleProvider) List(ctx context.Context, prefix string) ([]FileInfo, error) {
	objects, _, err := p.client.ListObjects(ctx, prefix, 1000)
	if err != nil {
		return nil, err
	}

	result := make([]FileInfo, 0, len(objects))
	for _, obj := range objects {
		result = append(result, FileInfo{
			Name:         obj.Key,
			Size:         obj.SizeBytes,
			LastModified: obj.LastModified,
			ETag:         obj.ETag,
			Path:         obj.Key,
		})
	}

	return result, nil
}

// Stat returns metadata about an object
func (p *S3CompatibleProvider) Stat(ctx context.Context, key string) (*FileInfo, error) {
	input := &s3.HeadObjectInput{
		Bucket: aws.String(p.bucket),
		Key:    aws.String(key),
	}

	output, err := p.client.HeadObject(ctx, input)
	if err != nil {
		return nil, err
	}

	var lastModified time.Time
	if output.LastModified != nil {
		lastModified = *output.LastModified
	}

	var size int64
	if output.ContentLength != nil {
		size = *output.ContentLength
	}

	var etag string
	if output.ETag != nil {
		etag = strings.Trim(*output.ETag, "\"")
	}

	return &FileInfo{
		Name:         key,
		Size:         size,
		LastModified: lastModified,
		ETag:         etag,
		Path:         key,
	}, nil
}

// Exists checks if an object exists, properly handling NotFound vs other errors
func (p *S3CompatibleProvider) Exists(ctx context.Context, key string) (bool, error) {
	_, err := p.Stat(ctx, key)
	if err != nil {
		var apiErr smithy.APIError
		if errors.As(err, &apiErr) {
			code := apiErr.ErrorCode()
			if code == "NotFound" || code == "NoSuchKey" {
				return false, nil
			}
		}
		return false, err
	}
	return true, nil
}

// Copy copies an object within the same bucket
func (p *S3CompatibleProvider) Copy(ctx context.Context, srcKey, dstKey string) error {
	copySource := p.bucket + "/" + srcKey
	input := &s3.CopyObjectInput{
		Bucket:     aws.String(p.bucket),
		Key:        aws.String(dstKey),
		CopySource: aws.String(copySource),
	}

	_, err := p.client.CopyObject(ctx, input)
	return err
}

// GetObject is a helper to get object from the underlying s3 client
func (p *S3CompatibleProvider) GetObject(ctx context.Context, input *s3.GetObjectInput) (*s3.GetObjectOutput, error) {
	return p.client.GetObject(ctx, input)
}

// PutObject is a helper to put object to the underlying s3 client
func (p *S3CompatibleProvider) PutObject(ctx context.Context, input *s3.PutObjectInput) (*s3.PutObjectOutput, error) {
	return p.client.PutObject(ctx, input)
}

// DeleteObject is a helper to delete object from the underlying s3 client
func (p *S3CompatibleProvider) DeleteObject(ctx context.Context, input *s3.DeleteObjectInput) (*s3.DeleteObjectOutput, error) {
	return p.client.DeleteObject(ctx, input)
}

// HeadObject is a helper to get object metadata
func (p *S3CompatibleProvider) HeadObject(ctx context.Context, input *s3.HeadObjectInput) (*s3.HeadObjectOutput, error) {
	return p.client.HeadObject(ctx, input)
}

// CopyObject is a helper to copy an object
func (p *S3CompatibleProvider) CopyObject(ctx context.Context, input *s3.CopyObjectInput) (*s3.CopyObjectOutput, error) {
	return p.client.CopyObject(ctx, input)
}

// ConfigInfo returns the provider configuration for comparison
func (p *S3CompatibleProvider) ConfigInfo() ProviderConfigInfo {
	return ProviderConfigInfo{
		Scheme:    p.scheme,
		Bucket:    p.bucket,
		Endpoint:  p.endpoint,
		AccessKey: p.accessKey,
		SecretKey: p.secretKey,
		PathStyle: p.pathStyle,
		UseSSL:    p.useSSL,
	}
}

func init() {
	// Register S3-compatible providers
	// These require environment variables to be set for credentials:
	// S3_ENDPOINT, S3_REGION, S3_BUCKET, S3_ACCESS_KEY_ID, S3_SECRET_ACCESS_KEY
	// R2_ENDPOINT, R2_REGION, R2_BUCKET, R2_ACCESS_KEY_ID, R2_SECRET_ACCESS_KEY
	// MINIO_ENDPOINT, MINIO_REGION, MINIO_BUCKET, MINIO_ACCESS_KEY_ID, MINIO_SECRET_ACCESS_KEY
	Register("s3", NewS3Provider)
	Register("r2", NewR2Provider)
	Register("minio", NewMinioProvider)
}
