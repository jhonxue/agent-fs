package provider

import (
	"os"

	"github.com/geekjourneyx/agent-fs/pkg/s3client"
)

// COSProvider 通过嵌入 S3BaseProvider 实现 Tencent Cloud COS
type COSProvider struct {
	*S3BaseProvider
}

// NewCOSProvider 创建 COS provider
func NewCOSProvider() (StorageProvider, error) {
	cfg := s3client.Config{
		Endpoint:        os.Getenv("COS_ENDPOINT"),
		Region:          os.Getenv("COS_REGION"),
		Bucket:          os.Getenv("COS_BUCKET"),
		AccessKeyID:     os.Getenv("COS_ACCESS_KEY_ID"),
		SecretAccessKey: os.Getenv("COS_SECRET_ACCESS_KEY"),
		PathStyle:       false, // COS uses virtual hosting style by default
		UseSSL:          true,
	}

	base, err := NewS3BaseProvider("cos", cfg)
	if err != nil {
		return nil, err
	}

	return &COSProvider{S3BaseProvider: base}, nil
}

func init() {
	// Register the COS provider
	Register("cos", NewCOSProvider)
}
