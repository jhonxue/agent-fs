package cloud

import (
	"context"
	"errors"
	"testing"
)

// mockProvider implements Provider interface for testing
type mockProvider struct {
	uploadFunc   func(ctx context.Context, req UploadRequest) (UploadResult, error)
	downloadFunc func(ctx context.Context, req DownloadRequest) (DownloadResult, error)
	listFunc     func(ctx context.Context, req ListRequest) (ListResult, error)
	urlFunc      func(ctx context.Context, req URLRequest) (URLResult, error)
	shouldErr    bool
	errToReturn  error
}

func (m *mockProvider) Upload(ctx context.Context, req UploadRequest) (UploadResult, error) {
	if m.uploadFunc != nil {
		return m.uploadFunc(ctx, req)
	}
	if m.shouldErr {
		return UploadResult{}, m.errToReturn
	}
	return UploadResult{Provider: "mock", RemoteKey: req.RemoteKey}, nil
}

func (m *mockProvider) Download(ctx context.Context, req DownloadRequest) (DownloadResult, error) {
	if m.downloadFunc != nil {
		return m.downloadFunc(ctx, req)
	}
	if m.shouldErr {
		return DownloadResult{}, m.errToReturn
	}
	return DownloadResult{Provider: "mock", RemoteKey: req.RemoteKey}, nil
}

func (m *mockProvider) List(ctx context.Context, req ListRequest) (ListResult, error) {
	if m.listFunc != nil {
		return m.listFunc(ctx, req)
	}
	if m.shouldErr {
		return ListResult{}, m.errToReturn
	}
	return ListResult{Provider: "mock", Count: 0}, nil
}

func (m *mockProvider) URL(ctx context.Context, req URLRequest) (URLResult, error) {
	if m.urlFunc != nil {
		return m.urlFunc(ctx, req)
	}
	if m.shouldErr {
		return URLResult{}, m.errToReturn
	}
	return URLResult{Provider: "mock", RemoteKey: req.RemoteKey, URL: "https://example.com"}, nil
}

func TestUploadRequest(t *testing.T) {
	req := UploadRequest{
		LocalPath: "/local/path/file.txt",
		RemoteKey: "remote/path/file.txt",
	}

	if req.LocalPath != "/local/path/file.txt" {
		t.Errorf("expected LocalPath '/local/path/file.txt', got '%s'", req.LocalPath)
	}
	if req.RemoteKey != "remote/path/file.txt" {
		t.Errorf("expected RemoteKey 'remote/path/file.txt', got '%s'", req.RemoteKey)
	}
}

func TestUploadResult(t *testing.T) {
	result := UploadResult{
		Provider:   "s3",
		LocalPath:  "/local/path/file.txt",
		RemoteKey:  "remote/path/file.txt",
		RemoteURL:  "https://bucket.s3.amazonaws.com/remote/path/file.txt",
		SizeBytes:  1024,
		TimeTaken:  100,
		Compressed: false,
	}

	if result.Provider != "s3" {
		t.Errorf("expected Provider 's3', got '%s'", result.Provider)
	}
	if result.SizeBytes != 1024 {
		t.Errorf("expected SizeBytes 1024, got %d", result.SizeBytes)
	}
	if result.Compressed != false {
		t.Error("expected Compressed false")
	}
}

func TestDownloadRequest(t *testing.T) {
	req := DownloadRequest{
		RemoteKey: "remote/path/file.txt",
		LocalPath: "/local/path/file.txt",
	}

	if req.RemoteKey != "remote/path/file.txt" {
		t.Errorf("expected RemoteKey 'remote/path/file.txt', got '%s'", req.RemoteKey)
	}
	if req.LocalPath != "/local/path/file.txt" {
		t.Errorf("expected LocalPath '/local/path/file.txt', got '%s'", req.LocalPath)
	}
}

func TestDownloadResult(t *testing.T) {
	result := DownloadResult{
		Provider:       "s3",
		RemoteKey:      "remote/path/file.txt",
		LocalPath:      "/local/path/file.txt",
		SizeBytes:      2048,
		TimeTaken:      200,
		Decompressed:   true,
		ExtractedFiles: 5,
		ExtractedBytes: 4096,
	}

	if result.Provider != "s3" {
		t.Errorf("expected Provider 's3', got '%s'", result.Provider)
	}
	if result.ExtractedFiles != 5 {
		t.Errorf("expected ExtractedFiles 5, got %d", result.ExtractedFiles)
	}
	if result.ExtractedBytes != 4096 {
		t.Errorf("expected ExtractedBytes 4096, got %d", result.ExtractedBytes)
	}
}

func TestListRequest(t *testing.T) {
	req := ListRequest{
		Prefix: "prefix/",
		Limit:  100,
	}

	if req.Prefix != "prefix/" {
		t.Errorf("expected Prefix 'prefix/', got '%s'", req.Prefix)
	}
	if req.Limit != 100 {
		t.Errorf("expected Limit 100, got %d", req.Limit)
	}
}

func TestObjectInfo(t *testing.T) {
	obj := ObjectInfo{
		Key:          "path/to/file.txt",
		SizeBytes:    1024,
		LastModified: "2024-01-01T00:00:00Z",
		ETag:         "abc123",
	}

	if obj.Key != "path/to/file.txt" {
		t.Errorf("expected Key 'path/to/file.txt', got '%s'", obj.Key)
	}
	if obj.SizeBytes != 1024 {
		t.Errorf("expected SizeBytes 1024, got %d", obj.SizeBytes)
	}
	if obj.ETag != "abc123" {
		t.Errorf("expected ETag 'abc123', got '%s'", obj.ETag)
	}
}

func TestListResult(t *testing.T) {
	objects := []ObjectInfo{
		{Key: "file1.txt", SizeBytes: 100},
		{Key: "file2.txt", SizeBytes: 200},
	}

	result := ListResult{
		Provider:    "s3",
		Prefix:      "prefix/",
		Objects:     objects,
		Count:       2,
		TotalBytes:  300,
		IsTruncated: false,
	}

	if result.Provider != "s3" {
		t.Errorf("expected Provider 's3', got '%s'", result.Provider)
	}
	if result.Count != 2 {
		t.Errorf("expected Count 2, got %d", result.Count)
	}
	if result.TotalBytes != 300 {
		t.Errorf("expected TotalBytes 300, got %d", result.TotalBytes)
	}
	if len(result.Objects) != 2 {
		t.Errorf("expected 2 objects, got %d", len(result.Objects))
	}
}

func TestURLRequest(t *testing.T) {
	req := URLRequest{
		RemoteKey:  "path/to/file.txt",
		Expiration: 3600,
		PublicOnly: false,
	}

	if req.RemoteKey != "path/to/file.txt" {
		t.Errorf("expected RemoteKey 'path/to/file.txt', got '%s'", req.RemoteKey)
	}
	if req.Expiration != 3600 {
		t.Errorf("expected Expiration 3600, got %d", req.Expiration)
	}
	if req.PublicOnly != false {
		t.Error("expected PublicOnly false")
	}
}

func TestURLResult(t *testing.T) {
	result := URLResult{
		Provider:    "s3",
		RemoteKey:   "path/to/file.txt",
		URL:         "https://bucket.s3.amazonaws.com/path/to/file.txt",
		ExpiresIn:   3600,
		ExpiresAt:   "2024-01-01T01:00:00Z",
		IsPresigned: true,
	}

	if result.Provider != "s3" {
		t.Errorf("expected Provider 's3', got '%s'", result.Provider)
	}
	if result.URL != "https://bucket.s3.amazonaws.com/path/to/file.txt" {
		t.Errorf("unexpected URL: %s", result.URL)
	}
	if !result.IsPresigned {
		t.Error("expected IsPresigned true")
	}
}

func TestNewDispatcher(t *testing.T) {
	providers := map[string]Provider{
		"s3":   &mockProvider{},
		"r2":   &mockProvider{},
		"Minio": &mockProvider{},
	}

	dispatcher := NewDispatcher(providers)
	ctx := context.Background()

	// Test that provider names work correctly via public methods
	// Test "s3" provider works
	_, err := dispatcher.Upload(ctx, "s3", UploadRequest{LocalPath: "/test", RemoteKey: "test"})
	if err != nil {
		t.Errorf("expected 's3' provider to work, got error: %v", err)
	}

	// Test "r2" provider works
	_, err = dispatcher.Download(ctx, "r2", DownloadRequest{RemoteKey: "test", LocalPath: "/test"})
	if err != nil {
		t.Errorf("expected 'r2' provider to work, got error: %v", err)
	}

	// Test "minio" (normalized from "Minio") works
	_, err = dispatcher.List(ctx, "minio", ListRequest{Prefix: "test"})
	if err != nil {
		t.Errorf("expected 'minio' provider to work, got error: %v", err)
	}
}

func TestDispatcherUpload(t *testing.T) {
	ctx := context.Background()
	providers := map[string]Provider{
		"s3": &mockProvider{},
	}
	dispatcher := NewDispatcher(providers)

	req := UploadRequest{
		LocalPath: "/local/file.txt",
		RemoteKey: "remote/file.txt",
	}

	result, err := dispatcher.Upload(ctx, "s3", req)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if result.Provider != "mock" {
		t.Errorf("expected provider 'mock', got '%s'", result.Provider)
	}
	if result.RemoteKey != "remote/file.txt" {
		t.Errorf("expected RemoteKey 'remote/file.txt', got '%s'", result.RemoteKey)
	}
}

func TestDispatcherUploadError(t *testing.T) {
	ctx := context.Background()
	expectedErr := errors.New("upload failed")
	providers := map[string]Provider{
		"s3": &mockProvider{
			shouldErr:   true,
			errToReturn: expectedErr,
		},
	}
	dispatcher := NewDispatcher(providers)

	req := UploadRequest{
		LocalPath: "/local/file.txt",
		RemoteKey: "remote/file.txt",
	}

	_, err := dispatcher.Upload(ctx, "s3", req)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestDispatcherUploadUnknownProvider(t *testing.T) {
	ctx := context.Background()
	providers := map[string]Provider{
		"s3": &mockProvider{},
	}
	dispatcher := NewDispatcher(providers)

	req := UploadRequest{
		LocalPath: "/local/file.txt",
		RemoteKey: "remote/file.txt",
	}

	_, err := dispatcher.Upload(ctx, "unknown", req)
	if err == nil {
		t.Fatal("expected error for unknown provider, got nil")
	}
}

func TestDispatcherDownload(t *testing.T) {
	ctx := context.Background()
	providers := map[string]Provider{
		"s3": &mockProvider{},
	}
	dispatcher := NewDispatcher(providers)

	req := DownloadRequest{
		RemoteKey: "remote/file.txt",
		LocalPath: "/local/file.txt",
	}

	result, err := dispatcher.Download(ctx, "s3", req)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if result.Provider != "mock" {
		t.Errorf("expected provider 'mock', got '%s'", result.Provider)
	}
}

func TestDispatcherList(t *testing.T) {
	ctx := context.Background()
	providers := map[string]Provider{
		"s3": &mockProvider{},
	}
	dispatcher := NewDispatcher(providers)

	req := ListRequest{
		Prefix: "prefix/",
		Limit:  100,
	}

	result, err := dispatcher.List(ctx, "s3", req)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if result.Provider != "mock" {
		t.Errorf("expected provider 'mock', got '%s'", result.Provider)
	}
}

func TestDispatcherURL(t *testing.T) {
	ctx := context.Background()
	providers := map[string]Provider{
		"s3": &mockProvider{},
	}
	dispatcher := NewDispatcher(providers)

	req := URLRequest{
		RemoteKey:  "remote/file.txt",
		Expiration: 3600,
	}

	result, err := dispatcher.URL(ctx, "s3", req)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if result.Provider != "mock" {
		t.Errorf("expected provider 'mock', got '%s'", result.Provider)
	}
	if result.URL != "https://example.com" {
		t.Errorf("expected URL 'https://example.com', got '%s'", result.URL)
	}
}

func TestDispatcherProviderNormalization(t *testing.T) {
	// Test that provider names are case-insensitive and trimmed
	ctx := context.Background()

	// Test with leading/trailing spaces in provider name
	providers := map[string]Provider{
		" S3 ": &mockProvider{},
	}
	dispatcher := NewDispatcher(providers)

	// Should work with normalized name "s3"
	result, err := dispatcher.List(ctx, "s3", ListRequest{Prefix: "test"})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if result.Provider != "mock" {
		t.Errorf("expected provider 'mock', got '%s'", result.Provider)
	}

	// Test with uppercase
	_, err = dispatcher.Upload(ctx, "S3", UploadRequest{LocalPath: "/test", RemoteKey: "test"})
	if err != nil {
		t.Errorf("expected uppercase 'S3' to work, got error: %v", err)
	}
}