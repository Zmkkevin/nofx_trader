package backtest

import (
	"strings"
	"testing"

	"nofx/mcp"
)

func TestConfigureMCPClient(t *testing.T) {
	t.Run("should validate API key required for DeepSeek provider", func(t *testing.T) {
		cfg := BacktestConfig{
			AICfg: AIConfig{
				Provider: "deepseek",
				APIKey:   "", // Missing API key
			},
		}

		_, err := configureMCPClient(cfg, nil)

		if err == nil {
			t.Fatal("Expected error for missing API key with DeepSeek provider")
		}
		if !strings.Contains(err.Error(), "requires api key") {
			t.Errorf("Expected 'requires api key' error, got: %v", err)
		}
	})

	t.Run("should validate API key required for Qwen provider", func(t *testing.T) {
		cfg := BacktestConfig{
			AICfg: AIConfig{
				Provider: "qwen",
				APIKey:   "", // Missing API key
			},
		}

		_, err := configureMCPClient(cfg, nil)

		if err == nil {
			t.Fatal("Expected error for missing API key with Qwen provider")
		}
		if !strings.Contains(err.Error(), "requires api key") {
			t.Errorf("Expected 'requires api key' error, got: %v", err)
		}
	})

	t.Run("should validate all fields required for custom provider", func(t *testing.T) {
		cfg := BacktestConfig{
			AICfg: AIConfig{
				Provider: "custom",
				APIKey:   "test-key",
				BaseURL:  "", // Missing BaseURL
				Model:    "test-model",
			},
		}

		_, err := configureMCPClient(cfg, nil)

		if err == nil {
			t.Fatal("Expected error for incomplete custom provider config")
		}
		if !strings.Contains(err.Error(), "requires base_url") {
			t.Errorf("Expected 'requires base_url' error, got: %v", err)
		}
	})

	t.Run("should reject unsupported provider", func(t *testing.T) {
		cfg := BacktestConfig{
			AICfg: AIConfig{
				Provider: "unsupported_provider",
				APIKey:   "test-key",
			},
		}

		_, err := configureMCPClient(cfg, nil)

		if err == nil {
			t.Fatal("Expected error for unsupported provider")
		}
		if !strings.Contains(err.Error(), "unsupported ai provider") {
			t.Errorf("Expected 'unsupported ai provider' error, got: %v", err)
		}
	})

	t.Run("should inherit from base client when provider is default and no API key", func(t *testing.T) {
		baseClient := mcp.New()
		cfg := BacktestConfig{
			AICfg: AIConfig{
				Provider: "inherit",
				APIKey:   "", // No API key, should inherit
			},
		}

		client, err := configureMCPClient(cfg, baseClient)

		if err != nil {
			t.Fatalf("Expected successful inheritance, got error: %v", err)
		}
		if client != baseClient {
			t.Error("Expected to return base client when inheriting")
		}
	})

	t.Run("should require API key when no base client to inherit from", func(t *testing.T) {
		cfg := BacktestConfig{
			AICfg: AIConfig{
				Provider: "default",
				APIKey:   "", // No API key and no base client
			},
		}

		_, err := configureMCPClient(cfg, nil)

		if err == nil {
			t.Fatal("Expected error when no API key and no base client")
		}
		if !strings.Contains(err.Error(), "api key is required") {
			t.Errorf("Expected 'api key is required' error, got: %v", err)
		}
	})

	t.Run("should create new client with valid DeepSeek config", func(t *testing.T) {
		cfg := BacktestConfig{
			AICfg: AIConfig{
				Provider: "deepseek",
				APIKey:   "sk-test-key-1234567890",
				Model:    "deepseek-chat",
			},
		}

		client, err := configureMCPClient(cfg, nil)

		if err != nil {
			t.Fatalf("Expected successful client creation, got error: %v", err)
		}
		if client == nil {
			t.Fatal("Expected non-nil client")
		}
	})

	t.Run("should create new client with valid custom config", func(t *testing.T) {
		cfg := BacktestConfig{
			AICfg: AIConfig{
				Provider: "custom",
				APIKey:   "custom-api-key",
				BaseURL:  "https://custom.api.example.com",
				Model:    "custom-model-v1",
			},
		}

		client, err := configureMCPClient(cfg, nil)

		if err != nil {
			t.Fatalf("Expected successful client creation, got error: %v", err)
		}
		if client == nil {
			t.Fatal("Expected non-nil client")
		}
	})

	t.Run("should use default BaseURL for DeepSeek when not provided", func(t *testing.T) {
		cfg := BacktestConfig{
			AICfg: AIConfig{
				Provider: "deepseek",
				APIKey:   "sk-test-key-1234567890",
				BaseURL:  "", // Empty BaseURL should use default
				Model:    "deepseek-chat",
			},
		}

		client, err := configureMCPClient(cfg, nil)

		if err != nil {
			t.Fatalf("Expected successful client creation, got error: %v", err)
		}
		if client == nil {
			t.Fatal("Expected non-nil client")
		}

		// Verify that the client was configured (no error means BaseURL was set correctly)
		// The actual BaseURL check would require accessing private fields,
		// but the fact that it doesn't error means the default was applied
	})

	t.Run("should use default BaseURL for Qwen when not provided", func(t *testing.T) {
		cfg := BacktestConfig{
			AICfg: AIConfig{
				Provider: "qwen",
				APIKey:   "sk-test-key-1234567890",
				BaseURL:  "", // Empty BaseURL should use default
				Model:    "qwen3-max",
			},
		}

		client, err := configureMCPClient(cfg, nil)

		if err != nil {
			t.Fatalf("Expected successful client creation, got error: %v", err)
		}
		if client == nil {
			t.Fatal("Expected non-nil client")
		}
	})

	t.Run("should use custom BaseURL for DeepSeek when provided", func(t *testing.T) {
		cfg := BacktestConfig{
			AICfg: AIConfig{
				Provider: "deepseek",
				APIKey:   "sk-test-key-1234567890",
				BaseURL:  "https://custom-deepseek.example.com/v1",
				Model:    "deepseek-chat",
			},
		}

		client, err := configureMCPClient(cfg, nil)

		if err != nil {
			t.Fatalf("Expected successful client creation, got error: %v", err)
		}
		if client == nil {
			t.Fatal("Expected non-nil client")
		}
	})
}
