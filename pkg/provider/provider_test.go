package provider

import (
	"context"
	"io"
	"testing"
	"time"
)

// mockProvider is a test implementation of StorageProvider
type mockProvider struct {
	scheme   string
	readErr  bool
	writeErr bool
}

func (m *mockProvider) Scheme() string {
	return m.scheme
}

func (m *mockProvider) Read(ctx context.Context, path string) (io.ReadCloser, error) {
	if m.readErr {
		return nil, ErrMockRead
	}
	return &mockReadCloser{data: []byte("test content")}, nil
}

func (m *mockProvider) Write(ctx context.Context, path string, data io.Reader) error {
	if m.writeErr {
		return ErrMockWrite
	}
	return nil
}

func (m *mockProvider) Delete(ctx context.Context, path string) error {
	return nil
}

func (m *mockProvider) List(ctx context.Context, path string) ([]FileInfo, error) {
	return []FileInfo{
		{Name: "file1.txt", Size: 100},
	}, nil
}

func (m *mockProvider) Stat(ctx context.Context, path string) (*FileInfo, error) {
	return &FileInfo{
		Name:         "test.txt",
		Size:         100,
		IsDir:        false,
		LastModified: time.Now(),
		Path:         path,
	}, nil
}

func (m *mockProvider) Exists(ctx context.Context, path string) (bool, error) {
	return true, nil
}

func (m *mockProvider) Copy(ctx context.Context, srcPath, dstPath string) error {
	return nil
}

func (m *mockProvider) ConfigInfo() ProviderConfigInfo {
	return ProviderConfigInfo{
		Scheme: m.scheme,
	}
}

// mockReadCloser implements a ReadCloser for testing
type mockReadCloser struct {
	data   []byte
	pos    int
}

func (m *mockReadCloser) Read(p []byte) (n int, err error) {
	if m.pos >= len(m.data) {
		return 0, nil
	}
	n = copy(p, m.data[m.pos:])
	m.pos += n
	return n, nil
}

func (m *mockReadCloser) Close() error {
	return nil
}

// Custom errors for testing
var (
	ErrMockRead = &testError{"mock read error"}
	ErrMockWrite = &testError{"mock write error"}
)

type testError struct {
	msg string
}

func (e *testError) Error() string {
	return e.msg
}

func TestProviderConfigInfoEquals(t *testing.T) {
	tests := []struct {
		name     string
		a        ProviderConfigInfo
		b        ProviderConfigInfo
		expected bool
	}{
		{
			name:     "identical configs",
			a:        ProviderConfigInfo{Scheme: "s3", Bucket: "test", Endpoint: "s3.amazonaws.com", AccessKey: "key1", SecretKey: "secret1", PathStyle: true, UseSSL: true},
			b:        ProviderConfigInfo{Scheme: "s3", Bucket: "test", Endpoint: "s3.amazonaws.com", AccessKey: "key1", SecretKey: "secret1", PathStyle: true, UseSSL: true},
			expected: true,
		},
		{
			name:     "empty configs",
			a:        ProviderConfigInfo{},
			b:        ProviderConfigInfo{},
			expected: true,
		},
		{
			name:     "different scheme",
			a:        ProviderConfigInfo{Scheme: "s3"},
			b:        ProviderConfigInfo{Scheme: "oss"},
			expected: false,
		},
		{
			name:     "different bucket",
			a:        ProviderConfigInfo{Scheme: "s3", Bucket: "test1"},
			b:        ProviderConfigInfo{Scheme: "s3", Bucket: "test2"},
			expected: false,
		},
		{
			name:     "different endpoint",
			a:        ProviderConfigInfo{Endpoint: "s3.amazonaws.com"},
			b:        ProviderConfigInfo{Endpoint: "oss.aliyuncs.com"},
			expected: false,
		},
		{
			name:     "different access key",
			a:        ProviderConfigInfo{AccessKey: "key1"},
			b:        ProviderConfigInfo{AccessKey: "key2"},
			expected: false,
		},
		{
			name:     "different secret key",
			a:        ProviderConfigInfo{SecretKey: "secret1"},
			b:        ProviderConfigInfo{SecretKey: "secret2"},
			expected: false,
		},
		{
			name:     "different path style",
			a:        ProviderConfigInfo{PathStyle: true},
			b:        ProviderConfigInfo{PathStyle: false},
			expected: false,
		},
		{
			name:     "different use ssl",
			a:        ProviderConfigInfo{UseSSL: true},
			b:        ProviderConfigInfo{UseSSL: false},
			expected: false,
		},
		{
			name:     "file provider configs equal",
			a:        ProviderConfigInfo{Scheme: "file"},
			b:        ProviderConfigInfo{Scheme: "file"},
			expected: true,
		},
		{
			name:     "cephfs provider configs equal",
			a:        ProviderConfigInfo{Scheme: "cephfs"},
			b:        ProviderConfigInfo{Scheme: "cephfs"},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.a.Equals(tt.b)
			if result != tt.expected {
				t.Errorf("Equals() = %v, want %v", result, tt.expected)
			}
			// Test symmetry
			resultReverse := tt.b.Equals(tt.a)
			if resultReverse != result {
				t.Errorf("Equals() is not symmetric: a.Equals(b)=%v, b.Equals(a)=%v", result, resultReverse)
			}
		})
	}
}

func TestFileInfo(t *testing.T) {
	info := FileInfo{
		Name:         "test.txt",
		Size:         1024,
		IsDir:        false,
		LastModified: time.Now(),
		ETag:         "abc123",
		Path:         "/path/to/test.txt",
	}

	if info.Name != "test.txt" {
		t.Errorf("Name = %v, want test.txt", info.Name)
	}
	if info.Size != 1024 {
		t.Errorf("Size = %v, want 1024", info.Size)
	}
	if info.IsDir {
		t.Error("IsDir should be false")
	}
	if info.ETag != "abc123" {
		t.Errorf("ETag = %v, want abc123", info.ETag)
	}
}

func TestReadCloser(t *testing.T) {
	data := []byte("test data")
	rc := &ReadCloser{
		Reader: &mockReadCloser{data: data},
		Closer: nil,
	}

	// Test Read
	buf := make([]byte, 10)
	n, err := rc.Read(buf)
	if err != nil {
		t.Errorf("Read() error = %v", err)
	}
	if n == 0 {
		t.Error("Read() expected to read data")
	}

	// Test Close with nil Closer
	err = rc.Close()
	if err != nil {
		t.Errorf("Close() error = %v", err)
	}
}
