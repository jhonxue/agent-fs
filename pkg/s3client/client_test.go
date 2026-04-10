package s3client

import (
	"testing"
	"time"
)

func TestConfig(t *testing.T) {
	cfg := Config{
		Endpoint:         "https://s3.amazonaws.com",
		Region:           "us-east-1",
		Bucket:           "test-bucket",
		AccessKeyID:      "test-access-key",
		SecretAccessKey:  "test-secret-key",
		PathPrefix:       "prefix/",
		CDNHost:          "cdn.example.com",
		PathStyle:        false,
		UseSSL:           true,
		DisableTLSVerify: false,
	}

	if cfg.Endpoint != "https://s3.amazonaws.com" {
		t.Errorf("expected Endpoint 'https://s3.amazonaws.com', got '%s'", cfg.Endpoint)
	}
	if cfg.Region != "us-east-1" {
		t.Errorf("expected Region 'us-east-1', got '%s'", cfg.Region)
	}
	if cfg.Bucket != "test-bucket" {
		t.Errorf("expected Bucket 'test-bucket', got '%s'", cfg.Bucket)
	}
	if cfg.AccessKeyID != "test-access-key" {
		t.Errorf("expected AccessKeyID 'test-access-key', got '%s'", cfg.AccessKeyID)
	}
	if cfg.SecretAccessKey != "test-secret-key" {
		t.Errorf("expected SecretAccessKey 'test-secret-key', got '%s'", cfg.SecretAccessKey)
	}
	if cfg.PathPrefix != "prefix/" {
		t.Errorf("expected PathPrefix 'prefix/', got '%s'", cfg.PathPrefix)
	}
	if cfg.CDNHost != "cdn.example.com" {
		t.Errorf("expected CDNHost 'cdn.example.com', got '%s'", cfg.CDNHost)
	}
	if cfg.PathStyle != false {
		t.Error("expected PathStyle false")
	}
	if cfg.UseSSL != true {
		t.Error("expected UseSSL true")
	}
	if cfg.DisableTLSVerify != false {
		t.Error("expected DisableTLSVerify false")
	}
}

func TestObjectInfo(t *testing.T) {
	obj := ObjectInfo{
		Key:          "path/to/file.txt",
		SizeBytes:    1024,
		LastModified: time.Now(),
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

func TestBuildKey(t *testing.T) {
	tests := []struct {
		name     string
		prefix   string
		key      string
		expected string
	}{
		{
			name:     "empty prefix and key",
			prefix:   "",
			key:      "",
			expected: "",
		},
		{
			name:     "only prefix",
			prefix:   "prefix",
			key:      "",
			expected: "prefix",
		},
		{
			name:     "only key",
			prefix:   "",
			key:      "file.txt",
			expected: "file.txt",
		},
		{
			name:     "both prefix and key",
			prefix:   "prefix",
			key:      "file.txt",
			expected: "prefix/file.txt",
		},
		{
			name:     "prefix and key with slashes",
			prefix:   "prefix/",
			key:      "/path/file.txt",
			expected: "prefix/path/file.txt",
		},
		{
			name:     "multiple slashes in prefix",
			prefix:   "/prefix/",
			key:      "/path/to/file.txt",
			expected: "prefix/path/to/file.txt",
		},
		{
			name:     "key with leading slash only",
			prefix:   "prefix",
			key:      "/file.txt",
			expected: "prefix/file.txt",
		},
		{
			name:     "key with trailing slash",
			prefix:   "prefix",
			key:      "file.txt/",
			expected: "prefix/file.txt",
		},
		{
			name:     "space trimming in prefix",
			prefix:   "  prefix  ",
			key:      "file.txt",
			expected: "prefix/file.txt",
		},
		{
			name:     "space trimming in key",
			prefix:   "prefix",
			key:      "  file.txt  ",
			expected: "prefix/file.txt",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildKey(tt.prefix, tt.key)
			if result != tt.expected {
				t.Errorf("buildKey('%s', '%s') = '%s', expected '%s'", tt.prefix, tt.key, result, tt.expected)
			}
		})
	}
}

func TestEnsureEndpoint(t *testing.T) {
	tests := []struct {
		name     string
		endpoint string
		useSSL   bool
		expected string
	}{
		{
			name:     "empty endpoint with SSL",
			endpoint: "",
			useSSL:   true,
			expected: "https://",
		},
		{
			name:     "empty endpoint without SSL",
			endpoint: "",
			useSSL:   false,
			expected: "http://",
		},
		{
			name:     "http already present",
			endpoint: "http://example.com",
			useSSL:   true,
			expected: "http://example.com",
		},
		{
			name:     "https already present",
			endpoint: "https://example.com",
			useSSL:   false,
			expected: "https://example.com",
		},
		{
			name:     "no scheme with SSL",
			endpoint: "example.com",
			useSSL:   true,
			expected: "https://example.com",
		},
		{
			name:     "no scheme without SSL",
			endpoint: "example.com",
			useSSL:   false,
			expected: "http://example.com",
		},
		{
			name:     "endpoint with port",
			endpoint: "localhost:9000",
			useSSL:   true,
			expected: "https://localhost:9000",
		},
		{
			name:     "endpoint with leading space",
			endpoint: "  example.com  ",
			useSSL:   true,
			expected: "https://example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ensureEndpoint(tt.endpoint, tt.useSSL)
			if result != tt.expected {
				t.Errorf("ensureEndpoint('%s', %v) = '%s', expected '%s'", tt.endpoint, tt.useSSL, result, tt.expected)
			}
		})
	}
}

func TestContentTypeByName(t *testing.T) {
	tests := []struct {
		name     string
		fileName string
		expected string
	}{
		{
			name:     "zip file",
			fileName: "test.zip",
			expected: "application/zip",
		},
		{
			name:     "json file",
			fileName: "data.json",
			expected: "application/json",
		},
		{
			name:     "txt file",
			fileName: "readme.txt",
			expected: "text/plain; charset=utf-8",
		},
		{
			name:     "log file",
			fileName: "app.log",
			expected: "text/plain; charset=utf-8",
		},
		{
			name:     "uppercase extension",
			fileName: "test.ZIP",
			expected: "application/zip",
		},
		{
			name:     "mixed case extension",
			fileName: "test.Json",
			expected: "application/json",
		},
		{
			name:     "unknown extension",
			fileName: "test.bin",
			expected: "application/octet-stream",
		},
		{
			name:     "no extension",
			fileName: " Makefile",
			expected: "application/octet-stream",
		},
		{
			name:     "file with path",
			fileName: "/path/to/file.zip",
			expected: "application/zip",
		},
		{
			name:     "dotfile",
			fileName: ".gitignore",
			expected: "application/octet-stream",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := contentTypeByName(tt.fileName)
			if result != tt.expected {
				t.Errorf("contentTypeByName('%s') = '%s', expected '%s'", tt.fileName, result, tt.expected)
			}
		})
	}
}

// Note: We can't easily test the New() function without mocking AWS SDK
// However, we can verify that the config validation logic is correct by
// testing the helper functions that don't require network access