package config

import (
	"testing"
)

// TestCreateTrader_WithValidForeignKeys 测试创建交易员 - 有效的外键
func TestCreateTrader_WithValidForeignKeys(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	userID := "test-user-trader-1"

	// 创建测试用户
	user := &User{
		ID:           userID,
		Email:        userID + "@test.com",
		PasswordHash: "hash",
		OTPSecret:    "",
		OTPVerified:  false,
	}
	if err := db.CreateUser(user); err != nil {
		t.Fatalf("创建用户失败: %v", err)
	}

	// 先创建该用户的 exchange 配置
	err := db.UpdateExchange(
		userID,
		"binance",
		true,
		"test-api-key",
		"test-secret",
		false,
		"",
		"",
		"",
		"",
	)
	if err != nil {
		t.Fatalf("创建exchange配置失败: %v", err)
	}

	// 创建交易员（使用默认的 exchange_id 和 ai_model_id）
	trader := &TraderRecord{
		ID:                  "test_trader_1",
		UserID:              userID,
		Name:                "Test Trader",
		AIModelID:           "deepseek", // 应该存在于 ai_models 表
		ExchangeID:          "binance",  // 应该存在于 exchanges 表
		InitialBalance:      1000.0,
		ScanIntervalMinutes: 3,
		IsRunning:           false,
		BTCETHLeverage:      5,
		AltcoinLeverage:     5,
		TradingSymbols:      "BTCUSDT,ETHUSDT",
		UseCoinPool:         false,
		UseOITop:            false,
		CustomPrompt:        "",
		OverrideBasePrompt:  false,
		SystemPromptTemplate: "default",
		IsCrossMargin:       true,
	}

	// 这应该成功
	err = db.CreateTrader(trader)
	if err != nil {
		t.Fatalf("创建交易员失败: %v", err)
	}

	// 验证交易员已创建
	traders, err := db.GetTraders(userID)
	if err != nil {
		t.Fatalf("获取交易员失败: %v", err)
	}

	if len(traders) != 1 {
		t.Fatalf("期望1个交易员，实际%d个", len(traders))
	}

	if traders[0].Name != "Test Trader" {
		t.Errorf("交易员名称不匹配，期望 Test Trader，实际 %s", traders[0].Name)
	}
}

// TestCreateTrader_WithInvalidExchangeID 测试创建交易员 - 无效的 exchange_id
// 这应该失败，因为 foreign key 约束
func TestCreateTrader_WithInvalidExchangeID(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	userID := "test-user-trader-2"

	// 创建测试用户
	user := &User{
		ID:           userID,
		Email:        userID + "@test.com",
		PasswordHash: "hash",
		OTPSecret:    "",
		OTPVerified:  false,
	}
	if err := db.CreateUser(user); err != nil {
		t.Fatalf("创建用户失败: %v", err)
	}

	// 尝试用不存在的 exchange_id 创建交易员
	trader := &TraderRecord{
		ID:                  "test_trader_2",
		UserID:              userID,
		Name:                "Test Trader 2",
		AIModelID:           "deepseek",
		ExchangeID:          "non_existent_exchange", // ❌ 不存在
		InitialBalance:      1000.0,
		ScanIntervalMinutes: 3,
		IsRunning:           false,
		BTCETHLeverage:      5,
		AltcoinLeverage:     5,
		SystemPromptTemplate: "default",
		IsCrossMargin:       true,
	}

	// 这应该失败（如果 foreign keys 启用）
	err := db.CreateTrader(trader)
	if err == nil {
		t.Error("期望因 foreign key 约束失败，但实际成功了")
	} else {
		t.Logf("✓ 符合预期的错误: %v", err)
	}
}

// TestCreateTrader_WithInvalidAIModelID 测试创建交易员 - 无效的 ai_model_id
func TestCreateTrader_WithInvalidAIModelID(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	userID := "test-user-trader-3"

	// 创建测试用户
	user := &User{
		ID:           userID,
		Email:        userID + "@test.com",
		PasswordHash: "hash",
		OTPSecret:    "",
		OTPVerified:  false,
	}
	if err := db.CreateUser(user); err != nil {
		t.Fatalf("创建用户失败: %v", err)
	}

	// 尝试用不存在的 ai_model_id 创建交易员
	trader := &TraderRecord{
		ID:                  "test_trader_3",
		UserID:              userID,
		Name:                "Test Trader 3",
		AIModelID:           "non_existent_model", // ❌ 不存在
		ExchangeID:          "binance",
		InitialBalance:      1000.0,
		ScanIntervalMinutes: 3,
		IsRunning:           false,
		BTCETHLeverage:      5,
		AltcoinLeverage:     5,
		SystemPromptTemplate: "default",
		IsCrossMargin:       true,
	}

	// 这应该失败（如果 foreign keys 启用）
	err := db.CreateTrader(trader)
	if err == nil {
		t.Error("期望因 foreign key 约束失败，但实际成功了")
	} else {
		t.Logf("✓ 符合预期的错误: %v", err)
	}
}

// TestCreateTrader_WithInvalidUserID 测试创建交易员 - 无效的 user_id
func TestCreateTrader_WithInvalidUserID(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// 尝试用不存在的 user_id 创建交易员
	trader := &TraderRecord{
		ID:                  "test_trader_4",
		UserID:              "non_existent_user", // ❌ 不存在
		Name:                "Test Trader 4",
		AIModelID:           "deepseek",
		ExchangeID:          "binance",
		InitialBalance:      1000.0,
		ScanIntervalMinutes: 3,
		IsRunning:           false,
		BTCETHLeverage:      5,
		AltcoinLeverage:     5,
		SystemPromptTemplate: "default",
		IsCrossMargin:       true,
	}

	// 这应该失败（如果 foreign keys 启用）
	err := db.CreateTrader(trader)
	if err == nil {
		t.Error("期望因 foreign key 约束失败，但实际成功了")
	} else {
		t.Logf("✓ 符合预期的错误: %v", err)
	}
}
