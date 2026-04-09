package s3client

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type Config struct {
	Endpoint         string
	Region           string
	Bucket           string
	AccessKeyID      string
	SecretAccessKey  string
	PathPrefix       string
	CDNHost          string
	PathStyle        bool
	UseSSL           bool
	DisableTLSVerify bool
}

type Client struct {
	cfg    Config
	client *s3.Client
}

func New(ctx context.Context, cfg Config) (*Client, error) {
	region := strings.TrimSpace(cfg.Region)
	if region == `` {
		region = `auto`
	}

	creds := credentials.NewStaticCredentialsProvider(
		strings.TrimSpace(cfg.AccessKeyID),
		strings.TrimSpace(cfg.SecretAccessKey),
		``,
	)

	awsCfgOpts := []func(*config.LoadOptions) error{
		config.WithRegion(region),
		config.WithCredentialsProvider(creds),
	}

	endpoint := strings.TrimSpace(cfg.Endpoint)
	if endpoint != `` {
		endpoint = ensureEndpoint(endpoint, cfg.UseSSL)
		//nolint:staticcheck // SA1019 - deprecated but needed for S3-compatible services
		customEndpoint := aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
			if service == s3.ServiceID {
				return aws.Endpoint{
					URL:               endpoint,
					HostnameImmutable: true,
					Source:            aws.EndpointSourceCustom,
				}, nil
			}
			return aws.Endpoint{}, &aws.EndpointNotFoundError{}
		})
		//nolint:staticcheck // SA1019 - deprecated but needed for S3-compatible services
		awsCfgOpts = append(awsCfgOpts, config.WithEndpointResolverWithOptions(customEndpoint))
		cfg.Endpoint = endpoint
	}

	awsCfg, err := config.LoadDefaultConfig(ctx, awsCfgOpts...)
	if err != nil {
		return nil, err
	}

	client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		o.UsePathStyle = cfg.PathStyle
	})

	return &Client{
		cfg:    cfg,
		client: client,
	}, nil
}

func (c *Client) UploadFile(ctx context.Context, localPath, remoteKey string) (int64, string, string, error) {
	file, err := os.Open(localPath)
	if err != nil {
		return 0, ``, ``, err
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		return 0, ``, ``, err
	}

	key := buildKey(c.cfg.PathPrefix, remoteKey)
	input := &s3.PutObjectInput{
		Bucket:      aws.String(c.cfg.Bucket),
		Key:         aws.String(key),
		Body:        file,
		ContentType: aws.String(contentTypeByName(localPath)),
	}
	if _, err := c.client.PutObject(ctx, input); err != nil {
		return 0, ``, ``, err
	}

	return stat.Size(), key, c.PublicURL(key), nil
}

func (c *Client) DownloadFile(ctx context.Context, remoteKey, localPath string) (int64, string, error) {
	key := buildKey(c.cfg.PathPrefix, remoteKey)
	output, err := c.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(c.cfg.Bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return 0, ``, err
	}
	defer output.Body.Close()

	if err := os.MkdirAll(filepath.Dir(localPath), 0o755); err != nil {
		return 0, ``, err
	}

	target, err := os.Create(localPath)
	if err != nil {
		return 0, ``, err
	}
	defer target.Close()

	n, err := io.Copy(target, output.Body)
	if err != nil {
		return 0, ``, err
	}
	return n, key, nil
}

func (c *Client) PublicURL(key string) string {
	if host := strings.TrimSpace(c.cfg.CDNHost); host != `` {
		return strings.TrimRight(host, `/`) + `/` + strings.TrimLeft(key, `/`)
	}

	if strings.TrimSpace(c.cfg.Endpoint) == `` {
		region := strings.TrimSpace(c.cfg.Region)
		if region == `` {
			region = `us-east-1`
		}
		return fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", c.cfg.Bucket, region, key)
	}

	endpoint := ensureEndpoint(c.cfg.Endpoint, c.cfg.UseSSL)
	u, err := url.Parse(endpoint)
	if err != nil {
		return endpoint + `/` + key
	}

	scheme := u.Scheme
	host := u.Host
	if c.cfg.PathStyle {
		return fmt.Sprintf("%s://%s/%s/%s", scheme, host, c.cfg.Bucket, key)
	}
	return fmt.Sprintf("%s://%s.%s/%s", scheme, c.cfg.Bucket, host, key)
}

// PresignedURL generates a presigned URL for temporary access to an object.
// expiration is the duration the URL is valid for (e.g., 15*time.Minute).
func (c *Client) PresignedURL(ctx context.Context, key string, expiration time.Duration) (string, error) {
	key = buildKey(c.cfg.PathPrefix, key)

	// Create a presigned client using the v4 signer
	presignClient := s3.NewPresignClient(c.client)

	presignResult, err := presignClient.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(c.cfg.Bucket),
		Key:    aws.String(key),
	}, s3.WithPresignExpires(expiration))
	if err != nil {
		return "", fmt.Errorf("failed to presign URL: %w", err)
	}

	return presignResult.URL, nil
}

// Signer returns the v4 signer for this client (used for presigned URLs)
func (c *Client) Signer() *v4.Signer {
	return v4.NewSigner()
}

type ObjectInfo struct {
	Key          string
	SizeBytes    int64
	LastModified time.Time
	ETag         string
}

// ListObjects lists objects in the bucket with optional prefix and limit.
func (c *Client) ListObjects(ctx context.Context, prefix string, limit int) ([]ObjectInfo, bool, error) {
	prefix = buildKey(c.cfg.PathPrefix, prefix)

	input := &s3.ListObjectsV2Input{
		Bucket: aws.String(c.cfg.Bucket),
		Prefix: aws.String(prefix),
	}

	if limit > 0 {
		// MaxKeys is an int32, cap at the maximum safe value to prevent overflow
		if limit > 1<<31-1 {
			limit = 1<<31 - 1
		}
		//nolint:gosec // G115 - we cap limit above to prevent overflow
		input.MaxKeys = aws.Int32(int32(limit))
	}

	output, err := c.client.ListObjectsV2(ctx, input)
	if err != nil {
		return nil, false, err
	}

	objects := make([]ObjectInfo, 0, len(output.Contents))
	for _, obj := range output.Contents {
		objects = append(objects, ObjectInfo{
			Key:          aws.ToString(obj.Key),
			SizeBytes:    aws.ToInt64(obj.Size),
			LastModified: *obj.LastModified,
			ETag:         aws.ToString(obj.ETag),
		})
	}

	isTruncated := output.IsTruncated != nil && *output.IsTruncated

	return objects, isTruncated, nil
}

func buildKey(prefix, key string) string {
	k := strings.Trim(strings.TrimSpace(key), `/`)
	p := strings.Trim(strings.TrimSpace(prefix), `/`)
	if p == `` {
		return k
	}
	if k == `` {
		return p
	}
	return path.Join(p, k)
}

// GetObject retrieves an object from S3 bucket
func (c *Client) GetObject(ctx context.Context, input *s3.GetObjectInput) (*s3.GetObjectOutput, error) {
	bucket, err := c.getBucket(input.Bucket)
	if err != nil {
		return nil, err
	}
	input.Bucket = aws.String(bucket)
	if input.Key != nil {
		input.Key = aws.String(buildKey(c.cfg.PathPrefix, *input.Key))
	}
	return c.client.GetObject(ctx, input)
}

// PutObject uploads an object to S3 bucket
func (c *Client) PutObject(ctx context.Context, input *s3.PutObjectInput) (*s3.PutObjectOutput, error) {
	bucket, err := c.getBucket(input.Bucket)
	if err != nil {
		return nil, err
	}
	input.Bucket = aws.String(bucket)
	if input.Key != nil {
		input.Key = aws.String(buildKey(c.cfg.PathPrefix, *input.Key))
	}
	return c.client.PutObject(ctx, input)
}

// DeleteObject deletes an object from S3 bucket
func (c *Client) DeleteObject(ctx context.Context, input *s3.DeleteObjectInput) (*s3.DeleteObjectOutput, error) {
	bucket, err := c.getBucket(input.Bucket)
	if err != nil {
		return nil, err
	}
	input.Bucket = aws.String(bucket)
	if input.Key != nil {
		input.Key = aws.String(buildKey(c.cfg.PathPrefix, *input.Key))
	}
	return c.client.DeleteObject(ctx, input)
}

// HeadObject retrieves metadata about an object without fetching the object itself
func (c *Client) HeadObject(ctx context.Context, input *s3.HeadObjectInput) (*s3.HeadObjectOutput, error) {
	bucket, err := c.getBucket(input.Bucket)
	if err != nil {
		return nil, err
	}
	input.Bucket = aws.String(bucket)
	if input.Key != nil {
		input.Key = aws.String(buildKey(c.cfg.PathPrefix, *input.Key))
	}
	return c.client.HeadObject(ctx, input)
}

// CopyObject copies an object within S3 bucket
func (c *Client) CopyObject(ctx context.Context, input *s3.CopyObjectInput) (*s3.CopyObjectOutput, error) {
	bucket, err := c.getBucket(input.Bucket)
	if err != nil {
		return nil, err
	}
	input.Bucket = aws.String(bucket)
	if input.Key != nil {
		input.Key = aws.String(buildKey(c.cfg.PathPrefix, *input.Key))
	}
	if input.CopySource != nil {
		// Build source key with path prefix
		// CopySource format: bucket/key or bucket/path/to/key
		source := *input.CopySource
		// Split on first "/" to separate bucket from key
		idx := strings.Index(source, "/")
		if idx > 0 {
			bucket := source[:idx]
			key := source[idx+1:]
			source = bucket + "/" + buildKey(c.cfg.PathPrefix, key)
		}
		input.CopySource = aws.String(source)
	}
	return c.client.CopyObject(ctx, input)
}

// getBucket returns the bucket name from the config or an error if not configured
func (c *Client) getBucket(bucket *string) (string, error) {
	if bucket != nil && *bucket != "" {
		return *bucket, nil
	}
	if c.cfg.Bucket != "" {
		return c.cfg.Bucket, nil
	}
	return "", fmt.Errorf("bucket not configured: please provide bucket name in input or client config")
}

func ensureEndpoint(endpoint string, useSSL bool) string {
	ep := strings.TrimSpace(endpoint)
	if strings.HasPrefix(ep, `http://`) || strings.HasPrefix(ep, `https://`) {
		return ep
	}
	if useSSL {
		return `https://` + ep
	}
	return `http://` + ep
}

func contentTypeByName(fileName string) string {
	name := strings.ToLower(fileName)
	switch {
	case strings.HasSuffix(name, `.zip`):
		return `application/zip`
	case strings.HasSuffix(name, `.json`):
		return `application/json`
	case strings.HasSuffix(name, `.txt`), strings.HasSuffix(name, `.log`):
		return `text/plain; charset=utf-8`
	default:
		return `application/octet-stream`
	}
}
