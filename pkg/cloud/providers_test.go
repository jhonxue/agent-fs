package cloud

import (
	"testing"
)

func TestGetProviders(t *testing.T) {
	providers := GetProviders()

	if len(providers) == 0 {
		t.Fatal("expected providers to be non-empty")
	}

	// Verify expected providers exist
	providerMap := make(map[string]ProviderInfo)
	for _, p := range providers {
		providerMap[p.Name] = p
	}

	tests := []struct {
		name        string
		description string
		endpoint    string
	}{
		{"s3", "AWS S3", "https://s3.amazonaws.com"},
		{"r2", "Cloudflare R2", "https://{account_id}.r2.cloudflarestorage.com"},
		{"minio", "MinIO Self-Hosted Object Storage", "http://localhost:9000"},
		{"alioss", "Alibaba Cloud Object Storage Service (OSS)", "https://oss-cn-hangzhou.aliyuncs.com"},
		{"txcos", "Tencent Cloud Object Storage (COS)", "https://cos.ap-guangzhou.myqcloud.com"},
		{"b2", "Backblaze B2 (S3 Compatible)", "https://s3.us-west-004.backblazeb2.com"},
		{"wasabi", "Wasabi Hot Cloud Storage", "https://s3.wasabisys.com"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, ok := providerMap[tt.name]
			if !ok {
				t.Errorf("provider %s not found", tt.name)
				return
			}
			if p.Description != tt.description {
				t.Errorf("expected description '%s', got '%s'", tt.description, p.Description)
			}
			if p.Endpoint != tt.endpoint {
				t.Errorf("expected endpoint '%s', got '%s'", tt.endpoint, p.Endpoint)
			}
		})
	}
}

func TestProviderInfo(t *testing.T) {
	p := ProviderInfo{
		Name:        "test",
		Description: "Test Provider",
		Endpoint:    "https://test.example.com",
		ConfigNote:  "Test config note",
	}

	if p.Name != "test" {
		t.Errorf("expected Name 'test', got '%s'", p.Name)
	}
	if p.Description != "Test Provider" {
		t.Errorf("expected Description 'Test Provider', got '%s'", p.Description)
	}
	if p.Endpoint != "https://test.example.com" {
		t.Errorf("expected Endpoint 'https://test.example.com', got '%s'", p.Endpoint)
	}
	if p.ConfigNote != "Test config note" {
		t.Errorf("expected ConfigNote 'Test config note', got '%s'", p.ConfigNote)
	}
}

func TestProviderInfoOptionalFields(t *testing.T) {
	// Test that ConfigNote is optional
	p := ProviderInfo{
		Name:        "test",
		Description: "Test Provider",
		Endpoint:    "https://test.example.com",
	}

	if p.ConfigNote != "" {
		t.Errorf("expected empty ConfigNote, got '%s'", p.ConfigNote)
	}
}

func TestProvidersAreUnique(t *testing.T) {
	providers := GetProviders()
	seen := make(map[string]bool)

	for _, p := range providers {
		if seen[p.Name] {
			t.Errorf("duplicate provider name: %s", p.Name)
		}
		seen[p.Name] = true
	}
}

func TestAllProvidersHaveRequiredFields(t *testing.T) {
	providers := GetProviders()

	for _, p := range providers {
		if p.Name == "" {
			t.Error("provider Name should not be empty")
		}
		if p.Description == "" {
			t.Errorf("provider %s has empty Description", p.Name)
		}
		if p.Endpoint == "" {
			t.Errorf("provider %s has empty Endpoint", p.Name)
		}
	}
}