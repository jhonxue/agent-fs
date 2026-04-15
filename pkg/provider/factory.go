package provider

import (
	"context"
	"fmt"

	"github.com/jhonxue/agent-fs/pkg/config"
	"github.com/jhonxue/agent-fs/pkg/s3client"
)

// ConfiguredProviderFactory creates a provider from config
type ConfiguredProviderFactory func(*config.ProviderConfig, string) (StorageProvider, error)

var configuredFactories = make(map[string]ConfiguredProviderFactory)

// RegisterConfiguredProvider registers a provider factory that accepts config
func RegisterConfiguredProvider(providerType string, factory ConfiguredProviderFactory) {
	configuredFactories[providerType] = factory
}

// CreateProviderFromConfig creates a provider using configuration
func CreateProviderFromConfig(ctx context.Context, providerCfg *config.ProviderConfig, scheme string) (StorageProvider, error) {
	if providerCfg == nil {
		return nil, fmt.Errorf("provider config is nil")
	}

	factory, ok := configuredFactories[providerCfg.Type]
	if !ok {
		return nil, fmt.Errorf("unsupported provider type: %s", providerCfg.Type)
	}

	return factory(providerCfg, scheme)
}

// CreateS3ProviderFromConfig creates an S3-compatible provider from config
func CreateS3ProviderFromConfig(providerCfg *config.ProviderConfig, scheme string) (StorageProvider, error) {
	clientCfg := s3client.Config{
		Endpoint:        providerCfg.Endpoint,
		Region:          providerCfg.Region,
		Bucket:          providerCfg.Bucket,
		AccessKeyID:     providerCfg.AccessKey,
		SecretAccessKey: providerCfg.SecretKey,
		PathStyle:       providerCfg.PathStyle,
		UseSSL:          providerCfg.UseSSL,
	}

	client, err := s3client.New(context.Background(), clientCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create S3 client: %w", err)
	}

	return &S3CompatibleProvider{
		scheme:    scheme,
		client:    client,
		bucket:    providerCfg.Bucket,
		endpoint:  providerCfg.Endpoint,
		accessKey: providerCfg.AccessKey,
		secretKey: providerCfg.SecretKey,
		pathStyle: providerCfg.PathStyle,
		useSSL:    providerCfg.UseSSL,
	}, nil
}

func init() {
	// Register S3-compatible provider factory for different types
	RegisterConfiguredProvider("s3", CreateS3ProviderFromConfig)
	RegisterConfiguredProvider("minio", CreateS3ProviderFromConfig)
	RegisterConfiguredProvider("r2", CreateS3ProviderFromConfig)
}
