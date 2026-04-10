package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestProviderConfig(t *testing.T) {
	cfg := ProviderConfig{
		Type:       "s3",
		Endpoint:   "https://s3.amazonaws.com",
		Bucket:     "test-bucket",
		Region:     "us-east-1",
		AccessKey:  "test-key",
		SecretKey:  "test-secret",
		PathStyle:  false,
		UseSSL:     true,
	}

	if cfg.Type != "s3" {
		t.Errorf("expected Type 's3', got '%s'", cfg.Type)
	}
	if cfg.Endpoint != "https://s3.amazonaws.com" {
		t.Errorf("expected Endpoint 'https://s3.amazonaws.com', got '%s'", cfg.Endpoint)
	}
	if cfg.Bucket != "test-bucket" {
		t.Errorf("expected Bucket 'test-bucket', got '%s'", cfg.Bucket)
	}
	if cfg.Region != "us-east-1" {
		t.Errorf("expected Region 'us-east-1', got '%s'", cfg.Region)
	}
	if cfg.AccessKey != "test-key" {
		t.Errorf("expected AccessKey 'test-key', got '%s'", cfg.AccessKey)
	}
	if cfg.SecretKey != "test-secret" {
		t.Errorf("expected SecretKey 'test-secret', got '%s'", cfg.SecretKey)
	}
	if cfg.PathStyle != false {
		t.Error("expected PathStyle false")
	}
	if cfg.UseSSL != true {
		t.Error("expected UseSSL true")
	}
}

func TestGetStringFromMap(t *testing.T) {
	tests := []struct {
		name     string
		inputMap map[string]any
		key      string
		expected string
	}{
		{
			name:     "string value exists",
			inputMap: map[string]any{"key": "value"},
			key:      "key",
			expected: "value",
		},
		{
			name:     "key does not exist",
			inputMap: map[string]any{"other": "value"},
			key:      "key",
			expected: "",
		},
		{
			name:     "value is not string",
			inputMap: map[string]any{"key": 123},
			key:      "key",
			expected: "",
		},
		{
			name:     "nil map",
			inputMap: nil,
			key:      "key",
			expected: "",
		},
		{
			name:     "empty map",
			inputMap: map[string]any{},
			key:      "key",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getStringFromMap(tt.inputMap, tt.key)
			if result != tt.expected {
				t.Errorf("expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

func TestGetBoolFromMap(t *testing.T) {
	tests := []struct {
		name     string
		inputMap map[string]any
		key      string
		expected bool
	}{
		{
			name:     "bool true",
			inputMap: map[string]any{"key": true},
			key:      "key",
			expected: true,
		},
		{
			name:     "bool false",
			inputMap: map[string]any{"key": false},
			key:      "key",
			expected: false,
		},
		{
			name:     "key does not exist",
			inputMap: map[string]any{"other": true},
			key:      "key",
			expected: false,
		},
		{
			name:     "value is not bool",
			inputMap: map[string]any{"key": "true"},
			key:      "key",
			expected: false,
		},
		{
			name:     "nil map",
			inputMap: nil,
			key:      "key",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getBoolFromMap(tt.inputMap, tt.key)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestResolveEnvVar(t *testing.T) {
	// Set a test environment variable
	os.Setenv("TEST_ENV_VAR", "test_value")
	defer os.Unsetenv("TEST_ENV_VAR")

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "no env var reference",
			input:    "plain_value",
			expected: "plain_value",
		},
		{
			name:     "env var reference",
			input:    "${TEST_ENV_VAR}",
			expected: "test_value",
		},
		{
			name:     "non-existent env var",
			input:    "${NON_EXISTENT_VAR_12345}",
			expected: "",
		},
		{
			name:     "partial env var reference",
			input:    "prefix${TEST_ENV_VAR}suffix",
			expected: "prefix${TEST_ENV_VAR}suffix",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := resolveEnvVar(tt.input)
			if result != tt.expected {
				t.Errorf("expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

func TestDefaultConfigPath(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		// If we can't get home dir, should return default
		result := DefaultConfigPath()
		if result != ".afs.yaml" {
			t.Errorf("expected '.afs.yaml', got '%s'", result)
		}
		return
	}

	result := DefaultConfigPath()
	expected := filepath.Join(home, ".afs.yaml")
	if result != expected {
		t.Errorf("expected '%s', got '%s'", expected, result)
	}
}

func TestValidateProviderConfig(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *ProviderConfig
		wantErr bool
	}{
		{
			name:    "nil config",
			cfg:     nil,
			wantErr: true,
		},
		{
			name: "empty type",
			cfg: &ProviderConfig{
				Type:       "",
				Endpoint:   "https://example.com",
				Bucket:     "test-bucket",
			},
			wantErr: true,
		},
		{
			name: "valid config",
			cfg: &ProviderConfig{
				Type:       "s3",
				Endpoint:   "https://example.com",
				Bucket:     "test-bucket",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateProviderConfig(tt.cfg)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateProviderConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestMatchURLWithConfig(t *testing.T) {
	tests := []struct {
		name        string
		cfg         *ProviderConfig
		urlBucket   string
		urlEndpoint string
		wantErr     bool
	}{
		{
			name:        "nil config",
			cfg:         nil,
			urlBucket:   "bucket",
			urlEndpoint: "endpoint",
			wantErr:    true,
		},
		{
			name: "matching bucket and endpoint",
			cfg: &ProviderConfig{
				Bucket:   "my-bucket",
				Endpoint: "s3.amazonaws.com",
			},
			urlBucket:   "my-bucket",
			urlEndpoint: "s3.amazonaws.com",
			wantErr:    false,
		},
		{
			name: "mismatched bucket",
			cfg: &ProviderConfig{
				Bucket:   "my-bucket",
				Endpoint: "s3.amazonaws.com",
			},
			urlBucket:   "other-bucket",
			urlEndpoint: "s3.amazonaws.com",
			wantErr:    true,
		},
		{
			name: "mismatched endpoint",
			cfg: &ProviderConfig{
				Bucket:   "my-bucket",
				Endpoint: "s3.amazonaws.com",
			},
			urlBucket:   "my-bucket",
			urlEndpoint: "other.endpoint.com",
			wantErr:    true,
		},
		{
			name: "empty url fields",
			cfg: &ProviderConfig{
				Bucket:   "my-bucket",
				Endpoint: "s3.amazonaws.com",
			},
			urlBucket:   "",
			urlEndpoint: "",
			wantErr:    false,
		},
		{
			name: "empty config, empty url",
			cfg: &ProviderConfig{
				Bucket:   "",
				Endpoint: "",
			},
			urlBucket:   "",
			urlEndpoint: "",
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := MatchURLWithConfig(tt.cfg, tt.urlBucket, tt.urlEndpoint)
			if (err != nil) != tt.wantErr {
				t.Errorf("MatchURLWithConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestConfigDefault(t *testing.T) {
	// Default should return a valid config that can be used
	cfg1 := Default()
	cfg2 := Default()

	// Both calls should return valid (non-nil) configs
	if cfg1 == nil {
		t.Error("Default() should not return nil")
	}
	if cfg2 == nil {
		t.Error("Default() should not return nil")
	}

	// Both calls should work correctly - behavior test, not identity test
	// Test that config is functional (can get/set providers)
	if cfg1.HasProviders() {
		t.Error("Default config should not have providers initially")
	}
	if cfg2.HasProviders() {
		t.Error("Default config should not have providers initially")
	}

	// Adding to one should be reflected in the other (shared singleton)
	cfg1.providers["test_provider"] = &ProviderConfig{Type: "s3"}
	if !cfg2.HasProviders() {
		t.Error("After adding provider to one config, both should see it (shared state)")
	}
}

func TestConfigNewConfig(t *testing.T) {
	cfg := &Config{
		providers: make(map[string]*ProviderConfig),
	}

	// Test empty providers
	if cfg.HasProviders() {
		t.Error("new config should not have providers")
	}

	// Test GetProvider
	cfg.providers["test"] = &ProviderConfig{Type: "s3"}
	if !cfg.HasProviders() {
		t.Error("config should have providers after adding one")
	}

	p, ok := cfg.GetProvider("test")
	if !ok {
		t.Error("expected to find 'test' provider")
	}
	if p.Type != "s3" {
		t.Errorf("expected type 's3', got '%s'", p.Type)
	}

	// Test GetProvider not found
	_, ok = cfg.GetProvider("nonexistent")
	if ok {
		t.Error("should not find nonexistent provider")
	}
}

func TestConfigListProviders(t *testing.T) {
	cfg := &Config{
		providers: map[string]*ProviderConfig{
			"provider1": {Type: "s3"},
			"provider2": {Type: "minio"},
		},
	}

	providers := cfg.ListProviders()
	if len(providers) != 2 {
		t.Errorf("expected 2 providers, got %d", len(providers))
	}

	if providers["provider1"] == nil {
		t.Error("expected provider1 to exist")
	}
	if providers["provider2"] == nil {
		t.Error("expected provider2 to exist")
	}
}

func TestResolveEnvVarEdgeCases(t *testing.T) {
	// Note: "${}" has no variable name, so resolveEnvVar returns empty string
	// because Getenv("") returns empty string
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "single char - just $",
			input:    "$",
			expected: "$",
		},
		{
			name:     "two chars - ${",
			input:    "${",
			expected: "${",
		},
		{
			name:     "incomplete env var ${VAR",
			input:    "${VAR",
			expected: "${VAR",
		},
		{
			name:     "empty env var ${} - returns empty string",
			input:    "${}",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := resolveEnvVar(tt.input)
			if result != tt.expected {
				t.Errorf("expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}