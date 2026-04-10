package provider

import (
	"context"
	"errors"
	"io"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	smithy "github.com/aws/smithy-go"

	"github.com/geekjourneyx/agent-fs/pkg/s3client"
)

// OSSProvider implements StorageProvider for Alibaba Cloud OSS
type OSSProvider struct {
	scheme    string
	client    *s3client.Client
	bucket    string
	endpoint  string
	accessKey string
	secretKey string
}

// NewOSSProvider creates a new Alibaba Cloud OSS provider
func NewOSSProvider() (StorageProvider, error) {
	// Use environment variables for OSS credentials
	cfg := s3client.Config{
		Endpoint:         os.Getenv("OSS_ENDPOINT"),
		Region:           os.Getenv("OSS_REGION"),
		Bucket:           os.Getenv("OSS_BUCKET"),
		AccessKeyID:      os.Getenv("OSS_ACCESS_KEY_ID"),
		SecretAccessKey:  os.Getenv("OSS_SECRET_ACCESS_KEY"),
		PathStyle:        true, // OSS requires path-style
		UseSSL:           true,
	}

	if cfg.Endpoint == "" || cfg.Bucket == "" {
		return nil, &providerConfigError{"OSS provider requires OSS_ENDPOINT and OSS_BUCKET environment variables"}
	}

	client, err := s3client.New(context.Background(), cfg)
	if err != nil {
		return nil, err
	}
	return &OSSProvider{
		scheme:    "oss",
		client:    client,
		bucket:    cfg.Bucket,
		endpoint:  cfg.Endpoint,
		accessKey: cfg.AccessKeyID,
		secretKey: cfg.SecretAccessKey,
	}, nil
}

// Scheme returns "oss"
func (p *OSSProvider) Scheme() string {
	return p.scheme
}

// Read reads an object from OSS
func (p *OSSProvider) Read(ctx context.Context, key string) (io.ReadCloser, error) {
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

// Write writes data to OSS
func (p *OSSProvider) Write(ctx context.Context, key string, data io.Reader) error {
	input := &s3.PutObjectInput{
		Bucket:      aws.String(p.bucket),
		Key:         aws.String(key),
		Body:        data,
		ContentType: aws.String("application/octet-stream"),
	}

	_, err := p.client.PutObject(ctx, input)
	return err
}

// Delete deletes an object from OSS
func (p *OSSProvider) Delete(ctx context.Context, key string) error {
	input := &s3.DeleteObjectInput{
		Bucket: aws.String(p.bucket),
		Key:    aws.String(key),
	}

	_, err := p.client.DeleteObject(ctx, input)
	return err
}

// List lists objects in an OSS bucket with given prefix
func (p *OSSProvider) List(ctx context.Context, prefix string) ([]FileInfo, error) {
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

// Stat returns metadata about an OSS object
func (p *OSSProvider) Stat(ctx context.Context, key string) (*FileInfo, error) {
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

// Exists checks if an OSS object exists
func (p *OSSProvider) Exists(ctx context.Context, key string) (bool, error) {
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

// Copy copies an object within OSS
func (p *OSSProvider) Copy(ctx context.Context, srcKey, dstKey string) error {
	copySource := p.bucket + "/" + srcKey
	input := &s3.CopyObjectInput{
		Bucket:     aws.String(p.bucket),
		Key:        aws.String(dstKey),
		CopySource: aws.String(copySource),
	}

	_, err := p.client.CopyObject(ctx, input)
	return err
}

// ConfigInfo returns the provider configuration for comparison
func (p *OSSProvider) ConfigInfo() ProviderConfigInfo {
	return ProviderConfigInfo{
		Scheme:    p.scheme,
		Bucket:    p.bucket,
		Endpoint:  p.endpoint,
		AccessKey: p.accessKey,
		SecretKey: p.secretKey,
		PathStyle: true,
		UseSSL:    true,
	}
}

func init() {
	// Register the OSS provider
	Register("oss", NewOSSProvider)
}
