package provider

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	smithy "github.com/aws/smithy-go"

	"github.com/geekjourneyx/agent-fs/pkg/s3client"
)

// S3BaseProvider 包含所有 S3-compatible provider 的共用实现
type S3BaseProvider struct {
	scheme    string
	client    *s3client.Client
	bucket    string
	endpoint  string
	accessKey string
	secretKey string
	pathStyle bool
	useSSL    bool
}

// NewS3BaseProvider 创建 S3BaseProvider 实例
func NewS3BaseProvider(scheme string, cfg s3client.Config) (*S3BaseProvider, error) {
	// 验证配置
	if scheme == "s3" {
		if cfg.Bucket == "" {
			return nil, &providerConfigError{
				msg: fmt.Sprintf("%s provider requires %s_BUCKET environment variable",
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

	return &S3BaseProvider{
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

// Scheme returns the URI scheme this provider handles
func (p *S3BaseProvider) Scheme() string {
	return p.scheme
}

// Read reads an object from S3-compatible storage
func (p *S3BaseProvider) Read(ctx context.Context, key string) (io.ReadCloser, error) {
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
func (p *S3BaseProvider) Write(ctx context.Context, key string, data io.Reader) error {
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
func (p *S3BaseProvider) Delete(ctx context.Context, key string) error {
	input := &s3.DeleteObjectInput{
		Bucket: aws.String(p.bucket),
		Key:    aws.String(key),
	}

	_, err := p.client.DeleteObject(ctx, input)
	return err
}

// List lists objects in a bucket with given prefix
func (p *S3BaseProvider) List(ctx context.Context, prefix string) ([]FileInfo, error) {
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
func (p *S3BaseProvider) Stat(ctx context.Context, key string) (*FileInfo, error) {
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
func (p *S3BaseProvider) Exists(ctx context.Context, key string) (bool, error) {
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
func (p *S3BaseProvider) Copy(ctx context.Context, srcKey, dstKey string) error {
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
func (p *S3BaseProvider) GetObject(ctx context.Context, input *s3.GetObjectInput) (*s3.GetObjectOutput, error) {
	return p.client.GetObject(ctx, input)
}

// PutObject is a helper to put object to the underlying s3 client
func (p *S3BaseProvider) PutObject(ctx context.Context, input *s3.PutObjectInput) (*s3.PutObjectOutput, error) {
	return p.client.PutObject(ctx, input)
}

// DeleteObject is a helper to delete object from the underlying s3 client
func (p *S3BaseProvider) DeleteObject(ctx context.Context, input *s3.DeleteObjectInput) (*s3.DeleteObjectOutput, error) {
	return p.client.DeleteObject(ctx, input)
}

// HeadObject is a helper to get object metadata
func (p *S3BaseProvider) HeadObject(ctx context.Context, input *s3.HeadObjectInput) (*s3.HeadObjectOutput, error) {
	return p.client.HeadObject(ctx, input)
}

// CopyObject is a helper to copy an object
func (p *S3BaseProvider) CopyObject(ctx context.Context, input *s3.CopyObjectInput) (*s3.CopyObjectOutput, error) {
	return p.client.CopyObject(ctx, input)
}

// ConfigInfo returns the provider configuration for comparison
func (p *S3BaseProvider) ConfigInfo() ProviderConfigInfo {
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