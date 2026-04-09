package config

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/spf13/viper"
)

// ProviderConfig Provider 配置
type ProviderConfig struct {
	Type       string `mapstructure:"type"`
	Endpoint   string `mapstructure:"endpoint"`
	Bucket     string `mapstructure:"bucket"`
	Region     string `mapstructure:"region"`
	AccessKey  string `mapstructure:"access_key"`
	SecretKey  string `mapstructure:"secret_key"`
	PathStyle  bool   `mapstructure:"path_style"`
	UseSSL     bool   `mapstructure:"use_ssl"`
}

// Config 存储服务配置
type Config struct {
	mu            sync.RWMutex
	providers     map[string]*ProviderConfig
	filePath      string
	loaded        bool  // 标记配置是否已尝试加载
}

// 全局配置实例
var (
	defaultConfig *Config
	configOnce    sync.Once
)

// Default 返回默认配置实例
func Default() *Config {
	configOnce.Do(func() {
		defaultConfig = &Config{
			providers: make(map[string]*ProviderConfig),
		}
	})
	return defaultConfig
}

// Load 从指定路径加载配置文件
func (c *Config) Load(path string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// 解析配置文件内容
	if err := c.loadConfig(path); err != nil {
		return err
	}

	c.loaded = true
	return nil
}

// loadConfig 实际加载配置文件的内部方法（需要持有锁）
func (c *Config) loadConfig(path string) error {
	v := viper.New()
	v.SetConfigFile(path)
	v.SetConfigType("yaml")

	if err := v.ReadInConfig(); err != nil {
		return fmt.Errorf("failed to read config file %s: %w", path, err)
	}

	// 解析 providers 节点
	if !v.IsSet("providers") {
		return nil // 没有 providers 配置
	}

	providers := v.GetStringMap("providers")
	for name, value := range providers {
		if providerMap, ok := value.(map[string]any); ok {
			// 转换环境变量引用
			typeName := getStringFromMap(providerMap, "type")
			endpoint := resolveEnvVar(getStringFromMap(providerMap, "endpoint"))
			bucket := getStringFromMap(providerMap, "bucket")
			region := getStringFromMap(providerMap, "region")
			accessKey := resolveEnvVar(getStringFromMap(providerMap, "access_key"))
			secretKey := resolveEnvVar(getStringFromMap(providerMap, "secret_key"))
			pathStyle := getBoolFromMap(providerMap, "path_style")
			useSSL := getBoolFromMap(providerMap, "use_ssl")

			c.providers[name] = &ProviderConfig{
				Type:      typeName,
				Endpoint:  endpoint,
				Bucket:    bucket,
				Region:    region,
				AccessKey: accessKey,
				SecretKey: secretKey,
				PathStyle: pathStyle,
				UseSSL:    useSSL,
			}
		}
	}

	c.filePath = path

	// 验证 provider 唯一性
	if err := c.ValidateUniqueProviders(); err != nil {
		return fmt.Errorf("invalid provider configuration: %w", err)
	}

	return nil
}

// GetProvider 根据名称获取 provider 配置
func (c *Config) GetProvider(name string) (*ProviderConfig, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	cfg, ok := c.providers[name]
	return cfg, ok
}

// GetProviderByEndpoint 根据 endpoint 查找 provider 配置
func (c *Config) GetProviderByEndpoint(endpoint string) (*ProviderConfig, string) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// 精确匹配
	for name, cfg := range c.providers {
		if cfg.Endpoint == endpoint {
			return cfg, name
		}
	}

	// 前缀匹配 (适用于 bucket.endpoint 格式)
	for name, cfg := range c.providers {
		if cfg.Endpoint != "" && len(endpoint) > len(cfg.Endpoint) {
			if endpoint[len(endpoint)-len(cfg.Endpoint)-1:] == "."+cfg.Endpoint {
				return cfg, name
			}
		}
	}

	return nil, ""
}

// GetProviderByBucketAndEndpoint 根据 bucket 和 endpoint 查找 provider 配置
func (c *Config) GetProviderByBucketAndEndpoint(bucket, endpoint string) (*ProviderConfig, string) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// 同时匹配 bucket 和 endpoint
	for name, cfg := range c.providers {
		if cfg.Endpoint == endpoint && cfg.Bucket == bucket {
			return cfg, name
		}
	}

	// 只匹配 endpoint
	if bucket == "" {
		return c.getProviderByEndpointUnlocked(endpoint)
	}

	// 只匹配 bucket
	if endpoint == "" {
		for name, cfg := range c.providers {
			if cfg.Bucket == bucket {
				return cfg, name
			}
		}
	}

	return nil, ""
}

func (c *Config) getProviderByEndpointUnlocked(endpoint string) (*ProviderConfig, string) {
	// 精确匹配
	for name, cfg := range c.providers {
		if cfg.Endpoint == endpoint {
			return cfg, name
		}
	}

	// 前缀匹配
	for name, cfg := range c.providers {
		if cfg.Endpoint != "" && len(endpoint) > len(cfg.Endpoint) {
			if endpoint[len(endpoint)-len(cfg.Endpoint)-1:] == "."+cfg.Endpoint {
				return cfg, name
			}
		}
	}

	return nil, ""
}

// ListProviders 列出所有已配置的 provider
func (c *Config) ListProviders() map[string]*ProviderConfig {
	c.mu.RLock()
	defer c.mu.RUnlock()

	result := make(map[string]*ProviderConfig)
	for name, cfg := range c.providers {
		result[name] = cfg
	}
	return result
}

// HasProviders 检查是否配置了任何 provider
func (c *Config) HasProviders() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.providers) > 0
}

// getStringFromMap 安全获取字符串
func getStringFromMap(m map[string]any, key string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// getBoolFromMap 安全获取布尔值
func getBoolFromMap(m map[string]any, key string) bool {
	if v, ok := m[key]; ok {
		if b, ok := v.(bool); ok {
			return b
		}
	}
	return false
}

// resolveEnvVar 解析环境变量引用 (${VAR_NAME})
func resolveEnvVar(value string) string {
	if value == "" {
		return value
	}

	// 支持 ${VAR_NAME} 格式
	if len(value) >= 3 && value[0] == '$' && value[1] == '{' && value[len(value)-1] == '}' {
		varName := value[2 : len(value)-1]
		return os.Getenv(varName)
	}

	return value
}

// DefaultConfigPath 返回默认配置文件路径
func DefaultConfigPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ".afs.yaml"
	}
	return filepath.Join(home, ".afs.yaml")
}

// LoadFromDefault 从默认路径加载配置（仅加载一次）
func LoadFromDefault() error {
	cfg := Default()

	// 检查是否已尝试加载（快速路径）
	cfg.mu.RLock()
	if cfg.loaded {
		cfg.mu.RUnlock()
		return nil
	}
	cfg.mu.RUnlock()

	// 获取配置文件路径
	path := os.Getenv("AFS_CONFIG")
	if path == "" {
		path = DefaultConfigPath()
	}

	// 如果默认文件不存在，不报错，直接标记已尝试
	if _, err := os.Stat(path); os.IsNotExist(err) {
		cfg.mu.Lock()
		cfg.loaded = true
		cfg.mu.Unlock()
		return nil
	}

	// 加载配置文件（Load 方法会设置 loaded 标记）
	return cfg.Load(path)
}

// ValidateProviderConfig 验证 provider 配置
func ValidateProviderConfig(cfg *ProviderConfig) error {
	if cfg == nil {
		return fmt.Errorf("provider config is nil")
	}

	if cfg.Type == "" {
		return fmt.Errorf("provider type is required")
	}

	return nil
}

// ValidateUniqueProviders 验证所有 provider 配置的唯一性
// 确保没有两个 provider 具有相同的 bucket + endpoint + access_key + secret_key 组合
func (c *Config) ValidateUniqueProviders() error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// 用于检测重复: bucket+endpoint+ak+sk -> [alias1, alias2, ...]
	dupMap := make(map[string][]string)

	for name, cfg := range c.providers {
		// 构建唯一性键: bucket:endpoint:access_key:secret_key
		// 注意：空字符串也参与比较，这样不同的配置可以共享相同的 bucket 但使用不同的 AK/SK
		key := fmt.Sprintf("%s:%s:%s:%s", cfg.Bucket, cfg.Endpoint, cfg.AccessKey, cfg.SecretKey)
		dupMap[key] = append(dupMap[key], name)
	}

	// 检查是否有重复
	var duplicates []string
	for key, aliases := range dupMap {
		if len(aliases) > 1 {
			duplicates = append(duplicates, fmt.Sprintf("bucket+endpoint+ak+sk=%s, aliases=%v", key, aliases))
		}
	}

	if len(duplicates) > 0 {
		return fmt.Errorf("duplicate provider configurations found: %v", duplicates)
	}

	return nil
}

// GetProviderByFullConfig 根据完整的 bucket+endpoint+ak+sk 查找 provider
// 返回精确匹配的配置（需要所有四个字段完全匹配）
func (c *Config) GetProviderByFullConfig(bucket, endpoint, accessKey, secretKey string) (*ProviderConfig, string) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// 精确匹配 bucket + endpoint + access_key + secret_key
	for name, cfg := range c.providers {
		if cfg.Bucket == bucket &&
			cfg.Endpoint == endpoint &&
			cfg.AccessKey == accessKey &&
			cfg.SecretKey == secretKey {
			return cfg, name
		}
	}

	return nil, ""
}

// MatchURLWithConfig 验证 URL 信息是否与配置匹配
func MatchURLWithConfig(cfg *ProviderConfig, urlBucket, urlEndpoint string) error {
	if cfg == nil {
		return fmt.Errorf("no matching provider config found")
	}

	// URL 包含 bucket，验证与配置的 bucket 是否一致
	if urlBucket != "" && cfg.Bucket != "" && urlBucket != cfg.Bucket {
		return fmt.Errorf("URL bucket (%s) does not match configured bucket (%s)", urlBucket, cfg.Bucket)
	}

	// URL 包含 endpoint，验证与配置的 endpoint 是否一致
	if urlEndpoint != "" && cfg.Endpoint != "" && urlEndpoint != cfg.Endpoint {
		return fmt.Errorf("URL endpoint (%s) does not match configured endpoint (%s)", urlEndpoint, cfg.Endpoint)
	}

	return nil
}
