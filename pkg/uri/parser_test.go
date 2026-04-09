package uri

import (
	"testing"
)

func TestParse(t *testing.T) {
	tests := []struct {
		name     string
		raw      string
		wantURI  *URI
		wantErr  bool
	}{
		{
			name: "absolute path without scheme",
			raw:  "/home/user/file.txt",
			wantURI: &URI{
				Scheme: "file",
				Path:   "/home/user/file.txt",
			},
			wantErr: false,
		},
		{
			name: "relative path without scheme",
			raw:  "./data/file.json",
			wantURI: &URI{
				Scheme: "file",
				Path:   "./data/file.json",
			},
			wantErr: false,
		},
		{
			name: "file scheme with absolute path",
			raw:  "file:///home/user/file.txt",
			wantURI: &URI{
				Scheme: "file",
				Path:   "/home/user/file.txt",
			},
			wantErr: false,
		},
		{
			name: "file scheme with authority (host dropped, path enforced absolute)",
			raw:  "file://data/file.txt",
			wantURI: &URI{
				Scheme: "file",
				Path:   "/file.txt", // 'data' is treated as host and dropped
			},
			wantErr: false,
		},
		{
			name: "S3 URI",
			raw:  "s3://my-bucket/path/to/object.txt",
			wantURI: &URI{
				Scheme: "s3",
				Bucket: "my-bucket",
				Key:    "path/to/object.txt",
			},
			wantErr: false,
		},
		{
			name: "R2 URI",
			raw:  "r2://my-bucket/backup/data.tar.gz",
			wantURI: &URI{
				Scheme: "r2",
				Bucket: "my-bucket",
				Key:    "backup/data.tar.gz",
			},
			wantErr: false,
		},
		{
			name: "S3 URI without key",
			raw:  "s3://my-bucket",
			wantURI: &URI{
				Scheme: "s3",
				Bucket: "my-bucket",
				Key:    "",
			},
			wantErr: false,
		},
		{
			name: "MinIO URI",
			raw:  "minio://my-bucket/logs/app.log",
			wantURI: &URI{
				Scheme: "minio",
				Bucket: "my-bucket",
				Key:    "logs/app.log",
			},
			wantErr: false,
		},
		{
			name: "GCS URI (not implemented)",
			raw:  "gcs://my-bucket/docs/readme.md",
			wantURI: nil,
			wantErr: true,
		},
		{
			name: "Azure URI (not implemented)",
			raw:  "azure://my-container/blobs/data.bin",
			wantURI: nil,
			wantErr: true,
		},
		{
			name: "Tencent COS URI",
			raw:  "cos://my-bucket/path/to/file.txt",
			wantURI: &URI{
				Scheme: "cos",
				Bucket: "my-bucket",
				Key:    "path/to/file.txt",
			},
			wantErr: false,
		},
		{
			name: "Alibaba OSS URI",
			raw:  "oss://my-bucket/data/logs/app.log",
			wantURI: &URI{
				Scheme: "oss",
				Bucket: "my-bucket",
				Key:    "data/logs/app.log",
			},
			wantErr: false,
		},
		{
			name: "unsupported scheme",
			raw:  "ftp://server/file.txt",
			wantURI: nil,
			wantErr: true,
		},
		{
			name: "empty URI",
			raw:  "",
			wantURI: nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Parse(tt.raw)
			if (err != nil) != tt.wantErr {
				t.Errorf("Parse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}
			if got.Scheme != tt.wantURI.Scheme {
				t.Errorf("Scheme = %v, want %v", got.Scheme, tt.wantURI.Scheme)
			}
			if got.Path != tt.wantURI.Path {
				t.Errorf("Path = %v, want %v", got.Path, tt.wantURI.Path)
			}
			if got.Bucket != tt.wantURI.Bucket {
				t.Errorf("Bucket = %v, want %v", got.Bucket, tt.wantURI.Bucket)
			}
			if got.Key != tt.wantURI.Key {
				t.Errorf("Key = %v, want %v", got.Key, tt.wantURI.Key)
			}
		})
	}
}

func TestURI_IsLocal(t *testing.T) {
	tests := []struct {
		name    string
		u       *URI
		want    bool
	}{
		{"file is local", &URI{Scheme: "file", Path: "/path"}, true},
		{"s3 is cloud", &URI{Scheme: "s3", Bucket: "b", Key: "k"}, false},
		{"r2 is cloud", &URI{Scheme: "r2", Bucket: "b", Key: "k"}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.u.IsLocal(); got != tt.want {
				t.Errorf("IsLocal() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestURI_IsCloud(t *testing.T) {
	tests := []struct {
		name    string
		u       *URI
		want    bool
	}{
		{"file is not cloud", &URI{Scheme: "file", Path: "/path"}, false},
		{"s3 is cloud", &URI{Scheme: "s3", Bucket: "b", Key: "k"}, true},
		{"r2 is cloud", &URI{Scheme: "r2", Bucket: "b", Key: "k"}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.u.IsCloud(); got != tt.want {
				t.Errorf("IsCloud() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestURI_String(t *testing.T) {
	tests := []struct {
		name  string
		u     *URI
		want  string
	}{
		{
			name: "file string",
			u:   &URI{Scheme: "file", Path: "/home/user/file.txt"},
			want: "file:///home/user/file.txt",
		},
		{
			name: "s3 string",
			u:   &URI{Scheme: "s3", Bucket: "my-bucket", Key: "path/to/file.txt"},
			want: "s3://my-bucket/path/to/file.txt",
		},
		{
			name: "r2 string with only bucket",
			u:   &URI{Scheme: "r2", Bucket: "my-bucket", Key: ""},
			want: "r2://my-bucket",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.u.String(); got != tt.want {
				t.Errorf("String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestURI_BaseName(t *testing.T) {
	tests := []struct {
		name  string
		u     *URI
		want  string
	}{
		{
			name: "file base name",
			u:   &URI{Scheme: "file", Path: "/path/to/file.txt"},
			want: "file.txt",
		},
		{
			name: "s3 base name",
			u:   &URI{Scheme: "s3", Bucket: "bucket", Key: "path/to/object.log"},
			want: "object.log",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.u.BaseName(); got != tt.want {
				t.Errorf("BaseName() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestURI_DirName(t *testing.T) {
	tests := []struct {
		name  string
		u     *URI
		want  string
	}{
		{
			name: "file dir name",
			u:   &URI{Scheme: "file", Path: "/path/to/file.txt"},
			want: "/path/to",
		},
		{
			name: "s3 dir name",
			u:   &URI{Scheme: "s3", Bucket: "bucket", Key: "dir1/dir2/file.txt"},
			want: "dir1/dir2",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.u.DirName(); got != tt.want {
				t.Errorf("DirName() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseWithEndpoint(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		wantURI *URI
		wantErr bool
	}{
		// VHost style URLs
		{
			name: "S3 VHost style",
			raw:  "https://bucket.s3.amazonaws.com/key",
			wantURI: &URI{
				Scheme:  "s3",
				Host:    "bucket.s3.amazonaws.com",
				Port:    0,
				Bucket:  "bucket",
				Key:     "key",
				IsVHost: true,
			},
			wantErr: false,
		},
		{
			name: "R2 VHost style",
			raw:  "https://mybucket.r2.cloudflarestorage.com/a/b.txt",
			wantURI: &URI{
				Scheme:  "r2",
				Host:    "mybucket.r2.cloudflarestorage.com",
				Port:    0,
				Bucket:  "mybucket",
				Key:     "a/b.txt",
				IsVHost: true,
			},
			wantErr: false,
		},
		{
			name: "OSS VHost style",
			raw:  "https://mybucket.oss-cn-hangzhou.aliyuncs.com/data.txt",
			wantURI: &URI{
				Scheme:  "oss",
				Host:    "mybucket.oss-cn-hangzhou.aliyuncs.com",
				Port:    0,
				Bucket:  "mybucket",
				Key:     "data.txt",
				IsVHost: true,
			},
			wantErr: false,
		},
		{
			name: "COS VHost style",
			raw:  "https://mybucket.cos.ap-guangzhou.myqcloud.com/file.txt",
			wantURI: &URI{
				Scheme:  "cos",
				Host:    "mybucket.cos.ap-guangzhou.myqcloud.com",
				Port:    0,
				Bucket:  "mybucket",
				Key:     "file.txt",
				IsVHost: true,
			},
			wantErr: false,
		},
		// Path style URLs
		{
			name: "S3 Path style",
			raw:  "https://s3.amazonaws.com/bucket/key",
			wantURI: &URI{
				Scheme:  "s3",
				Host:    "s3.amazonaws.com",
				Port:    0,
				Bucket:  "bucket",
				Key:     "key",
				IsVHost: false,
			},
			wantErr: false,
		},
		{
			name: "COS Path style",
			raw:  "https://cos.ap-guangzhou.myqcloud.com/mybucket/file.txt",
			wantURI: &URI{
				Scheme:  "cos",
				Host:    "cos.ap-guangzhou.myqcloud.com",
				Port:    0,
				Bucket:  "mybucket",
				Key:     "file.txt",
				IsVHost: false,
			},
			wantErr: false,
		},
		// With port
		{
			name: "MinIO with port",
			raw:  "http://localhost:9000/bucket/key",
			wantURI: &URI{
				Scheme:  "minio",
				Host:    "localhost",
				Port:    9000,
				Bucket:  "bucket",
				Key:     "key",
				IsVHost: false,
			},
			wantErr: false,
		},
		{
			name: "IP address with port",
			raw:  "https://192.168.1.100:9000/bucket/key",
			wantURI: &URI{
				Scheme:  "minio",
				Host:    "192.168.1.100",
				Port:    9000,
				Bucket:  "bucket",
				Key:     "key",
				IsVHost: false,
			},
			wantErr: false,
		},
		// Legacy format (should work with original Parse)
		{
			name: "Legacy S3 format",
			raw:  "s3://bucket/key",
			wantURI: &URI{
				Scheme:  "s3",
				Host:    "",
				Port:    0,
				Bucket:  "bucket",
				Key:     "key",
				IsVHost: false,
			},
			wantErr: false,
		},
		// Local path (falls back to Parse)
		{
			name: "Local path fallback",
			raw:  "/tmp/test.txt",
			wantURI: &URI{
				Scheme:  "file",
				Host:    "",
				Port:    0,
				Bucket:  "",
				Key:     "",
				Path:    "/tmp/test.txt",
				IsVHost: false,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseWithEndpoint(tt.raw)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseWithEndpoint() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if got.Scheme != tt.wantURI.Scheme {
					t.Errorf("Scheme = %v, want %v", got.Scheme, tt.wantURI.Scheme)
				}
				if got.Host != tt.wantURI.Host {
					t.Errorf("Host = %v, want %v", got.Host, tt.wantURI.Host)
				}
				if got.Port != tt.wantURI.Port {
					t.Errorf("Port = %v, want %v", got.Port, tt.wantURI.Port)
				}
				if got.Bucket != tt.wantURI.Bucket {
					t.Errorf("Bucket = %v, want %v", got.Bucket, tt.wantURI.Bucket)
				}
				if got.Key != tt.wantURI.Key {
					t.Errorf("Key = %v, want %v", got.Key, tt.wantURI.Key)
				}
				if got.IsVHost != tt.wantURI.IsVHost {
					t.Errorf("IsVHost = %v, want %v", got.IsVHost, tt.wantURI.IsVHost)
				}
			}
		})
	}
}

func TestDetectSchemeFromHost(t *testing.T) {
	tests := []struct {
		name string
		host string
		want string
	}{
		// S3: exact match
		{"AWS S3 exact", "s3.amazonaws.com", "s3"},
		// S3: bucket.variant.s3.amazonaws.com
		{"AWS S3 bucket", "mybucket.s3.amazonaws.com", "s3"},
		// AWS China requires exact .amazonaws.com.cn suffix
		{"AWS China", "s3.cn-north-1.amazonaws.com.cn", "s3"},
		// R2
		{"Cloudflare R2", "account.r2.cloudflarestorage.com", "r2"},
		{"R2 with bucket", "mybucket.r2.cloudflarestorage.com", "r2"},
		// OSS
		{"Aliyun OSS", "bucket.oss-cn-hangzhou.aliyuncs.com", "oss"},
		// COS
		{"Tencent COS", "bucket.cos.ap-guangzhou.myqcloud.com", "cos"},
		// Unknown domains default to minio
		{"Unknown domain", "my-custom-service.com", "minio"},
		{"localhost", "localhost", "minio"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := detectSchemeFromHost(tt.host)
			if got != tt.want {
				t.Errorf("detectSchemeFromHost(%q) = %q, want %q", tt.host, got, tt.want)
			}
		})
	}
}

func TestIsVHostStyle(t *testing.T) {
	tests := []struct {
		name   string
		host   string
		scheme string
		want   bool
	}{
		{"S3 VHost", "bucket.s3.amazonaws.com", "s3", true},
		{"R2 VHost", "bucket.r2.cloudflarestorage.com", "r2", true},
		{"OSS VHost", "bucket.oss-cn-hangzhou.aliyuncs.com", "oss", true},
		{"COS VHost", "bucket.cos.ap-guangzhou.myqcloud.com", "cos", true},
		{"MinIO VHost", "bucket.minio.example.com", "minio", true},
		{"Path style S3", "s3.amazonaws.com", "s3", false},
		{"Path style MinIO with port", "localhost:9000", "minio", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isVHostStyle(tt.host, tt.scheme)
			if got != tt.want {
				t.Errorf("isVHostStyle(%q, %q) = %v, want %v", tt.host, tt.scheme, got, tt.want)
			}
		})
	}
}

func TestExtractBucketFromHost(t *testing.T) {
	tests := []struct {
		name   string
		host   string
		scheme string
		want   string
	}{
		{"S3 VHost", "mybucket.s3.amazonaws.com", "s3", "mybucket"},
		{"R2 VHost", "mybucket.r2.cloudflarestorage.com", "r2", "mybucket"},
		{"OSS VHost", "mybucket.oss-cn-hangzhou.aliyuncs.com", "oss", "mybucket"},
		{"COS VHost", "mybucket.cos.ap-guangzhou.myqcloud.com", "cos", "mybucket"},
		{"MinIO VHost", "mybucket.minio.example.com", "minio", "mybucket"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractBucketFromHost(tt.host, tt.scheme)
			if got != tt.want {
				t.Errorf("extractBucketFromHost(%q, %q) = %q, want %q", tt.host, tt.scheme, got, tt.want)
			}
		})
	}
}

func TestIsIPAddress(t *testing.T) {
	tests := []struct {
		name string
		host string
		want bool
	}{
		{"IPv4", "192.168.1.1", true},
		{"IPv4 localhost", "127.0.0.1", true},
		{"IPv6", "::1", true},
		{"IPv6 full", "2001:db8::1", true},
		{"Domain", "example.com", false},
		{"Subdomain", "bucket.s3.amazonaws.com", false},
		{"Empty", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isIPAddress(tt.host)
			if got != tt.want {
				t.Errorf("isIPAddress(%q) = %v, want %v", tt.host, got, tt.want)
			}
		})
	}
}
