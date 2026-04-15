package cloud

import (
	"context"
	"strings"

	"github.com/jhonxue/agent-fs/pkg/apperr"
)

type UploadRequest struct {
	LocalPath string
	RemoteKey string
}

type UploadResult struct {
	Provider   string `json:"provider"`
	LocalPath  string `json:"local_path"`
	RemoteKey  string `json:"remote_key"`
	RemoteURL  string `json:"remote_url"`
	SizeBytes  int64  `json:"size_bytes"`
	TimeTaken  int64  `json:"time_taken_ms"`
	Compressed bool   `json:"compressed"`
}

type DownloadRequest struct {
	RemoteKey string
	LocalPath string
}

type DownloadResult struct {
	Provider       string `json:"provider"`
	RemoteKey      string `json:"remote_key"`
	LocalPath      string `json:"local_path"`
	SizeBytes      int64  `json:"size_bytes"`
	TimeTaken      int64  `json:"time_taken_ms"`
	Decompressed   bool   `json:"decompressed"`
	ExtractedFiles int    `json:"extracted_files,omitempty"`
	ExtractedBytes int64  `json:"extracted_bytes,omitempty"`
}

type ListRequest struct {
	Prefix string
	Limit  int
}

type ObjectInfo struct {
	Key          string `json:"key"`
	SizeBytes    int64  `json:"size_bytes"`
	LastModified string `json:"last_modified"`
	ETag         string `json:"etag,omitempty"`
}

type ListResult struct {
	Provider    string       `json:"provider"`
	Objects     []ObjectInfo `json:"objects"`
	Count       int          `json:"count"`
	TotalBytes  int64        `json:"total_bytes"`
	Prefix      string       `json:"prefix,omitempty"`
	IsTruncated bool         `json:"is_truncated"`
}

type URLRequest struct {
	RemoteKey  string
	Expiration int64 // Expiration duration in seconds
	PublicOnly bool  // If true, return public URL instead of presigned URL
}

type URLResult struct {
	Provider    string `json:"provider"`
	RemoteKey   string `json:"remote_key"`
	URL         string `json:"url"`
	ExpiresIn   int64  `json:"expires_in_seconds,omitempty"`
	ExpiresAt   string `json:"expires_at,omitempty"`
	IsPresigned bool   `json:"is_presigned"`
}

type Provider interface {
	Upload(ctx context.Context, req UploadRequest) (UploadResult, error)
	Download(ctx context.Context, req DownloadRequest) (DownloadResult, error)
	List(ctx context.Context, req ListRequest) (ListResult, error)
	URL(ctx context.Context, req URLRequest) (URLResult, error)
}

type Dispatcher struct {
	providers map[string]Provider
}

func NewDispatcher(providers map[string]Provider) *Dispatcher {
	normalized := make(map[string]Provider, len(providers))
	for name, provider := range providers {
		normalized[strings.ToLower(strings.TrimSpace(name))] = provider
	}
	return &Dispatcher{providers: normalized}
}

func (d *Dispatcher) Upload(ctx context.Context, providerName string, req UploadRequest) (UploadResult, error) {
	provider, err := d.getProvider(providerName, `cloud_upload`)
	if err != nil {
		return UploadResult{}, err
	}
	return provider.Upload(ctx, req)
}

func (d *Dispatcher) Download(ctx context.Context, providerName string, req DownloadRequest) (DownloadResult, error) {
	provider, err := d.getProvider(providerName, `cloud_download`)
	if err != nil {
		return DownloadResult{}, err
	}
	return provider.Download(ctx, req)
}

func (d *Dispatcher) List(ctx context.Context, providerName string, req ListRequest) (ListResult, error) {
	provider, err := d.getProvider(providerName, `cloud_list`)
	if err != nil {
		return ListResult{}, err
	}
	return provider.List(ctx, req)
}

func (d *Dispatcher) URL(ctx context.Context, providerName string, req URLRequest) (URLResult, error) {
	provider, err := d.getProvider(providerName, `cloud_url`)
	if err != nil {
		return URLResult{}, err
	}
	return provider.URL(ctx, req)
}

func (d *Dispatcher) getProvider(providerName, action string) (Provider, error) {
	key := strings.ToLower(strings.TrimSpace(providerName))
	provider, ok := d.providers[key]
	if !ok {
		return nil, apperr.New(action, apperr.CodeProvider, `unsupported provider: `+providerName)
	}
	return provider, nil
}
