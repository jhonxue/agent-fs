package provider

import (
	"context"
	"io"
	"strings"
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

func TestProviderConfigInfoSafeString(t *testing.T) {
	cfg := ProviderConfigInfo{
		Scheme:    "s3",
		Bucket:    "test-bucket",
		Endpoint:  "s3.amazonaws.com",
		AccessKey: "secret-access-key", // should not appear in output
		SecretKey: "super-secret-key",  // should not appear in output
		PathStyle: true,
		UseSSL:    true,
	}

	result := cfg.SafeString()

	// Verify non-sensitive fields are present
	if !strings.Contains(result, "scheme=s3") {
		t.Error("SafeString should contain scheme")
	}
	if !strings.Contains(result, "bucket=test-bucket") {
		t.Error("SafeString should contain bucket")
	}
	if !strings.Contains(result, "endpoint=s3.amazonaws.com") {
		t.Error("SafeString should contain endpoint")
	}

	// Verify sensitive fields are NOT present
	if strings.Contains(result, "secret-access-key") {
		t.Error("SafeString should NOT contain AccessKey")
	}
	if strings.Contains(result, "super-secret-key") {
		t.Error("SafeString should NOT contain SecretKey")
	}
	if strings.Contains(result, "AccessKey") {
		t.Error("SafeString should NOT contain AccessKey field name")
	}
	if strings.Contains(result, "SecretKey") {
		t.Error("SafeString should NOT contain SecretKey field name")
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

// TestOSSProviderScheme tests that OSSProvider would return "oss" scheme
func TestOSSProviderScheme(t *testing.T) {
	// Verify OSSProvider type can be created (it embeds S3BaseProvider)
	// The actual provider creation requires environment variables,
	// so we test the type structure and interface compliance
	var _ StorageProvider = (*OSSProvider)(nil)

	// Verify OSSProvider has S3BaseProvider embedded
	provider := &OSSProvider{}
	if provider.S3BaseProvider != nil {
		t.Error("New OSSProvider should have nil S3BaseProvider before initialization")
	}
}

// TestCOSProviderScheme tests that COSProvider would return "cos" scheme
func TestCOSProviderScheme(t *testing.T) {
	// Verify COSProvider type can be created (it embeds S3BaseProvider)
	// The actual provider creation requires environment variables,
	// so we test the type structure and interface compliance
	var _ StorageProvider = (*COSProvider)(nil)

	// Verify COSProvider has S3BaseProvider embedded
	provider := &COSProvider{}
	if provider.S3BaseProvider != nil {
		t.Error("New COSProvider should have nil S3BaseProvider before initialization")
	}
}

// TestS3BaseProviderImplementsInterface verifies S3BaseProvider implements StorageProvider
func TestS3BaseProviderImplementsInterface(t *testing.T) {
	// Compile-time check that S3BaseProvider implements StorageProvider
	var _ StorageProvider = (*S3BaseProvider)(nil)
}

// TestOSSProviderMissingEnv tests OSS provider creation with missing environment variables
func TestOSSProviderMissingEnv(t *testing.T) {
	// This should fail because OSS environment variables are not set
	_, err := NewOSSProvider()
	if err == nil {
		t.Error("NewOSSProvider() should fail without OSS environment variables")
	}
}

// TestCOSProviderMissingEnv tests COS provider creation with missing environment variables
func TestCOSProviderMissingEnv(t *testing.T) {
	// This should fail because COS environment variables are not set
	_, err := NewCOSProvider()
	if err == nil {
		t.Error("NewCOSProvider() should fail without COS environment variables")
	}
}

// TestOSSProviderRegistration tests that OSS provider is registered
func TestOSSProviderRegistration(t *testing.T) {
	// Check that OSS provider is registered by checking supported schemes
	schemes := SupportedSchemes()
	found := false
	for _, s := range schemes {
		if s == "oss" {
			found = true
			break
		}
	}
	if !found {
		t.Error("OSS provider should be registered in supported schemes")
	}
}

// TestCOSProviderRegistration tests that COS provider is registered
func TestCOSProviderRegistration(t *testing.T) {
	// Check that COS provider is registered by checking supported schemes
	schemes := SupportedSchemes()
	found := false
	for _, s := range schemes {
		if s == "cos" {
			found = true
			break
		}
	}
	if !found {
		t.Error("COS provider should be registered in supported schemes")
	}
}

// TestOSSProviderErrorMessages tests that OSS provider returns correct error messages
func TestOSSProviderErrorMessages(t *testing.T) {
	_, err := NewOSSProvider()
	if err == nil {
		t.Error("NewOSSProvider() should return error without environment variables")
	}

	errMsg := err.Error()
	if !containsAny(errMsg, []string{"OSS_ENDPOINT", "OSS_BUCKET"}) {
		t.Errorf("Error message should mention OSS environment variables, got: %s", errMsg)
	}
}

// TestCOSProviderErrorMessages tests that COS provider returns correct error messages
func TestCOSProviderErrorMessages(t *testing.T) {
	_, err := NewCOSProvider()
	if err == nil {
		t.Error("NewCOSProvider() should return error without environment variables")
	}

	errMsg := err.Error()
	if !containsAny(errMsg, []string{"COS_ENDPOINT", "COS_BUCKET"}) {
		t.Errorf("Error message should mention COS environment variables, got: %s", errMsg)
	}
}

// containsAny checks if s contains any of the substrings
func containsAny(s string, substrs []string) bool {
	for _, substr := range substrs {
		if strings.Contains(s, substr) {
			return true
		}
	}
	return false
}
