package cloud

import (
	"context"
	"time"

	"github.com/jhonxue/agent-fs/pkg/apperr"
	"github.com/jhonxue/agent-fs/pkg/s3client"
)

type S3CompatibleProvider struct {
	name   string
	client *s3client.Client
}

func NewS3CompatibleProvider(name string, cfg s3client.Config) (*S3CompatibleProvider, error) {
	client, err := s3client.New(context.Background(), cfg)
	if err != nil {
		return nil, apperr.Wrap(`cloud_provider_init`, apperr.CodeProvider, `failed to initialize s3-compatible provider`, err)
	}
	return &S3CompatibleProvider{
		name:   name,
		client: client,
	}, nil
}

func (p *S3CompatibleProvider) Upload(ctx context.Context, req UploadRequest) (UploadResult, error) {
	start := time.Now()
	size, key, url, err := p.client.UploadFile(ctx, req.LocalPath, req.RemoteKey)
	if err != nil {
		return UploadResult{}, apperr.Wrap(`cloud_upload`, apperr.CodeUpload, `upload failed`, err)
	}
	return UploadResult{
		Provider:  p.name,
		LocalPath: req.LocalPath,
		RemoteKey: key,
		RemoteURL: url,
		SizeBytes: size,
		TimeTaken: time.Since(start).Milliseconds(),
	}, nil
}

func (p *S3CompatibleProvider) Download(ctx context.Context, req DownloadRequest) (DownloadResult, error) {
	start := time.Now()
	size, key, err := p.client.DownloadFile(ctx, req.RemoteKey, req.LocalPath)
	if err != nil {
		return DownloadResult{}, apperr.Wrap(`cloud_download`, apperr.CodeDownload, `download failed`, err)
	}
	return DownloadResult{
		Provider:  p.name,
		RemoteKey: key,
		LocalPath: req.LocalPath,
		SizeBytes: size,
		TimeTaken: time.Since(start).Milliseconds(),
	}, nil
}

func (p *S3CompatibleProvider) List(ctx context.Context, req ListRequest) (ListResult, error) {
	s3Objects, isTruncated, err := p.client.ListObjects(ctx, req.Prefix, req.Limit)
	if err != nil {
		return ListResult{}, apperr.Wrap(`cloud_list`, apperr.CodeInternal, `list failed`, err)
	}

	objects := make([]ObjectInfo, len(s3Objects))
	var totalBytes int64
	for i, obj := range s3Objects {
		objects[i] = ObjectInfo{
			Key:          obj.Key,
			SizeBytes:    obj.SizeBytes,
			LastModified: obj.LastModified.Format(time.RFC3339),
			ETag:         obj.ETag,
		}
		totalBytes += obj.SizeBytes
	}

	return ListResult{
		Provider:    p.name,
		Objects:     objects,
		Count:       len(objects),
		TotalBytes:  totalBytes,
		Prefix:      req.Prefix,
		IsTruncated: isTruncated,
	}, nil
}

func (p *S3CompatibleProvider) URL(ctx context.Context, req URLRequest) (URLResult, error) {
	if req.PublicOnly {
		publicURL := p.client.PublicURL(req.RemoteKey)
		return URLResult{
			Provider:    p.name,
			RemoteKey:   req.RemoteKey,
			URL:         publicURL,
			IsPresigned: false,
		}, nil
	}

	// Default expiration is 15 minutes if not specified
	expiration := 15 * 60
	if req.Expiration > 0 {
		expiration = int(req.Expiration)
	}

	presignedURL, err := p.client.PresignedURL(ctx, req.RemoteKey, time.Duration(expiration)*time.Second)
	if err != nil {
		return URLResult{}, apperr.Wrap(`cloud_url`, apperr.CodeInternal, `failed to generate presigned URL`, err)
	}

	// Calculate expiration time
	expiresAt := time.Now().Add(time.Duration(expiration) * time.Second)

	return URLResult{
		Provider:    p.name,
		RemoteKey:   req.RemoteKey,
		URL:         presignedURL,
		ExpiresIn:   int64(expiration),
		ExpiresAt:   expiresAt.Format(time.RFC3339),
		IsPresigned: true,
	}, nil
}
