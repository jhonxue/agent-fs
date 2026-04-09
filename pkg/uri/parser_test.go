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
