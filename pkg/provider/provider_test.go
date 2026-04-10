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
