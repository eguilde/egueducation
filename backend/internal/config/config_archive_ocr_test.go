package config

import "testing"

func TestValidateArchiveOCRFailsClosedForPartialAzureConfiguration(t *testing.T) {
	cfg := Config{AzureDocumentIntelligenceEndpoint: "https://example.cognitiveservices.azure.com"}
	if err := cfg.ValidateArchiveOCR(); err == nil {
		t.Fatal("expected incomplete Azure configuration to fail")
	}
}

func TestValidateArchiveOCRAcceptsCompleteAzureConfiguration(t *testing.T) {
	cfg := Config{
		AzureDocumentIntelligenceEndpoint: "https://example.cognitiveservices.azure.com",
		AzureDocumentIntelligenceKey:      "test-key", AzureDocumentIntelligenceModel: "prebuilt-layout",
		AzureDocumentIntelligenceAPIVersion: "2024-11-30",
	}
	if err := cfg.ValidateArchiveOCR(); err != nil {
		t.Fatalf("ValidateArchiveOCR() error = %v", err)
	}
	if !cfg.AzureDocumentIntelligenceEnabled() {
		t.Fatal("AzureDocumentIntelligenceEnabled() = false")
	}
}
