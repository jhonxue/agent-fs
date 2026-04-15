package provider

import (
	"os"

	"github.com/jhonxue/agent-fs/pkg/s3client"
)

// OSSProvider 通过嵌入 S3BaseProvider 实现 Alibaba Cloud OSS
type OSSProvider struct {
	*S3BaseProvider
}

// NewOSSProvider 创建 OSS provider
func NewOSSProvider() (StorageProvider, error) {
	cfg := s3client.Config{
		Endpoint:        os.Getenv("OSS_ENDPOINT"),
		Region:          os.Getenv("OSS_REGION"),
		Bucket:          os.Getenv("OSS_BUCKET"),
		AccessKeyID:     os.Getenv("OSS_ACCESS_KEY_ID"),
		SecretAccessKey: os.Getenv("OSS_SECRET_ACCESS_KEY"),
		PathStyle:       true, // OSS requires path-style
		UseSSL:          true,
	}

	base, err := NewS3BaseProvider("oss", cfg)
	if err != nil {
		return nil, err
	}

	return &OSSProvider{S3BaseProvider: base}, nil
}

func init() {
	// Register the OSS provider
	Register("oss", NewOSSProvider)
}
