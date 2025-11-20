package decision

import (
	"crypto/md5"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"
)

// TestCalculatePromptHashFromTemplate 测试 Prompt Hash 计算逻辑
func TestCalculatePromptHashFromTemplate(t *testing.T) {
	// Setup: 创建临时测试模板
	tempDir := t.TempDir()
	testTemplate := "This is a test template for trading AI"
	testFile := filepath.Join(tempDir, "test.txt")
	if err := os.WriteFile(testFile, []byte(testTemplate), 0644); err != nil {
		t.Fatalf("Failed to create test template: %v", err)
	}

	// 保存原始模板管理器
	originalManager := globalPromptManager
	defer func() { globalPromptManager = originalManager }()

	// 使用测试目录
	globalPromptManager = NewPromptManager()
	if err := globalPromptManager.LoadTemplates(tempDir); err != nil {
		t.Fatalf("Failed to load test templates: %v", err)
	}

	t.Run("相同模板产生相同hash", func(t *testing.T) {
		hash1 := calculatePromptHashFromTemplate("test", "", false)
		hash2 := calculatePromptHashFromTemplate("test", "", false)

		if hash1 != hash2 {
			t.Errorf("相同模板应该产生相同hash: %s != %s", hash1, hash2)
		}

		// 验证hash长度（MD5 = 32字符）
		if len(hash1) != 32 {
			t.Errorf("Hash长度应该是32字符，实际: %d", len(hash1))
		}
	})

	t.Run("hash只基于模板内容", func(t *testing.T) {
		hash := calculatePromptHashFromTemplate("test", "", false)

		// 手动计算预期的hash
		expectedHash := md5.Sum([]byte(testTemplate))
		expectedHashStr := hex.EncodeToString(expectedHash[:])

		if hash != expectedHashStr {
			t.Errorf("Hash应该只基于模板内容:\n期望: %s\n实际: %s", expectedHashStr, hash)
		}
	})

	t.Run("不同customPrompt产生不同hash", func(t *testing.T) {
		hash1 := calculatePromptHashFromTemplate("test", "", false)
		hash2 := calculatePromptHashFromTemplate("test", "custom instruction", false)

		if hash1 == hash2 {
			t.Errorf("不同customPrompt应该产生不同hash")
		}
	})

	t.Run("相同customPrompt产生相同hash", func(t *testing.T) {
		customPrompt := "Always be conservative"
		hash1 := calculatePromptHashFromTemplate("test", customPrompt, false)
		hash2 := calculatePromptHashFromTemplate("test", customPrompt, false)

		if hash1 != hash2 {
			t.Errorf("相同customPrompt应该产生相同hash: %s != %s", hash1, hash2)
		}
	})

	t.Run("overrideBase模式只基于customPrompt", func(t *testing.T) {
		customPrompt := "Override everything"

		// overrideBase=true 时应该忽略模板，只用 customPrompt
		hash := calculatePromptHashFromTemplate("test", customPrompt, true)

		// 手动计算只基于customPrompt的hash
		expectedHash := md5.Sum([]byte(customPrompt))
		expectedHashStr := hex.EncodeToString(expectedHash[:])

		if hash != expectedHashStr {
			t.Errorf("overrideBase模式应该只基于customPrompt:\n期望: %s\n实际: %s", expectedHashStr, hash)
		}
	})

	t.Run("模板不存在时降级到default", func(t *testing.T) {
		// 创建default模板
		defaultTemplate := "Default template content"
		defaultFile := filepath.Join(tempDir, "default.txt")
		if err := os.WriteFile(defaultFile, []byte(defaultTemplate), 0644); err != nil {
			t.Fatalf("Failed to create default template: %v", err)
		}

		// 重新加载模板
		globalPromptManager = NewPromptManager()
		if err := globalPromptManager.LoadTemplates(tempDir); err != nil {
			t.Fatalf("Failed to reload templates: %v", err)
		}

		// 请求不存在的模板
		hash := calculatePromptHashFromTemplate("nonexistent", "", false)

		// 应该使用default模板
		expectedHash := md5.Sum([]byte(defaultTemplate))
		expectedHashStr := hex.EncodeToString(expectedHash[:])

		if hash != expectedHashStr {
			t.Errorf("不存在的模板应该降级到default:\n期望: %s\n实际: %s", expectedHashStr, hash)
		}
	})

	t.Run("default也不存在时使用builtin_fallback", func(t *testing.T) {
		// 使用空目录
		emptyDir := t.TempDir()
		globalPromptManager = NewPromptManager()
		globalPromptManager.LoadTemplates(emptyDir) // 加载空目录

		hash := calculatePromptHashFromTemplate("anything", "", false)

		// 应该使用builtin_fallback_prompt
		expectedHash := md5.Sum([]byte("builtin_fallback_prompt"))
		expectedHashStr := hex.EncodeToString(expectedHash[:])

		if hash != expectedHashStr {
			t.Errorf("所有模板都不存在时应该使用builtin_fallback:\n期望: %s\n实际: %s", expectedHashStr, hash)
		}
	})
}

// TestPromptHashStability 验证核心需求：相同模板不受外部因素影响
func TestPromptHashStability(t *testing.T) {
	// Setup
	tempDir := t.TempDir()
	templateContent := "Trade with risk management"
	if err := os.WriteFile(filepath.Join(tempDir, "stable.txt"), []byte(templateContent), 0644); err != nil {
		t.Fatalf("Failed to create template: %v", err)
	}

	originalManager := globalPromptManager
	defer func() { globalPromptManager = originalManager }()

	globalPromptManager = NewPromptManager()
	globalPromptManager.LoadTemplates(tempDir)

	// 模拟多次调用（在不同"余额"场景下）
	// 这是核心需求：即使外部参数（如accountEquity）不同，hash也应该相同
	hash1 := calculatePromptHashFromTemplate("stable", "", false)
	hash2 := calculatePromptHashFromTemplate("stable", "", false)
	hash3 := calculatePromptHashFromTemplate("stable", "", false)

	if hash1 != hash2 || hash2 != hash3 {
		t.Errorf("Hash应该稳定，不受调用时机影响:\nhash1: %s\nhash2: %s\nhash3: %s", hash1, hash2, hash3)
	}

	// 验证hash确实只基于模板内容
	expectedHash := md5.Sum([]byte(templateContent))
	expectedHashStr := hex.EncodeToString(expectedHash[:])

	if hash1 != expectedHashStr {
		t.Errorf("Hash应该只基于模板内容:\n期望: %s\n实际: %s", expectedHashStr, hash1)
	}

	t.Logf("✅ Hash稳定性验证通过: %s", hash1)
}

// TestPromptHashWithCustomPrompt 测试模板+customPrompt组合的hash
func TestPromptHashWithCustomPrompt(t *testing.T) {
	tempDir := t.TempDir()
	baseTemplate := "Base trading strategy"
	if err := os.WriteFile(filepath.Join(tempDir, "base.txt"), []byte(baseTemplate), 0644); err != nil {
		t.Fatalf("Failed to create template: %v", err)
	}

	originalManager := globalPromptManager
	defer func() { globalPromptManager = originalManager }()

	globalPromptManager = NewPromptManager()
	globalPromptManager.LoadTemplates(tempDir)

	customPrompt := "Focus on BTC only"

	// 计算hash
	hash := calculatePromptHashFromTemplate("base", customPrompt, false)

	// 手动计算预期值
	expectedContent := baseTemplate + "\n\n# CUSTOM\n" + customPrompt
	expectedHash := md5.Sum([]byte(expectedContent))
	expectedHashStr := hex.EncodeToString(expectedHash[:])

	if hash != expectedHashStr {
		t.Errorf("模板+customPrompt的hash不正确:\n期望: %s\n实际: %s", expectedHashStr, hash)
	}

	// 验证稳定性
	hash2 := calculatePromptHashFromTemplate("base", customPrompt, false)
	if hash != hash2 {
		t.Errorf("相同模板+customPrompt应该产生相同hash")
	}
}
