package trader

import (
	"nofx/decision"
	"nofx/logger"
	"testing"
	"time"
)

// TestVerifyAndUpdateCloseFillPrice_CloseLong tests close long position fill price verification
func TestVerifyAndUpdateCloseFillPrice_CloseLong(t *testing.T) {
	// Spec: 平仓后应该能够获取真实成交价格并更新到 actionRecord
	// 场景：平多仓时，市场价 50000，但真实成交价 49950

	// 创建 mock trader
	mockTrader := &mockTraderWithFills{
		fills: []mockFill{
			{
				symbol:    "BTCUSDT",
				side:      "Sell", // Close Long = Sell
				price:     49950.0, // 真实成交价
				quantity:  0.1,
				timestamp: time.Now().UnixMilli(),
			},
		},
	}

	// 创建 AutoTrader
	at := &AutoTrader{
		trader: mockTrader,
	}

	// 创建 decision 和 actionRecord
	decision := &decision.Decision{
		Symbol: "BTCUSDT",
		Action: "close_long",
	}

	actionRecord := &logger.DecisionAction{
		Symbol:    "BTCUSDT",
		Action:    "close_long",
		Price:     50000.0, // 平仓前的市场价
		Timestamp: time.Now(),
	}

	// 调用矫正函数
	err := at.verifyAndUpdateCloseFillPrice(decision, actionRecord, time.Now().UnixMilli())
	if err != nil {
		t.Fatalf("verifyAndUpdateCloseFillPrice 失败: %v", err)
	}

	// 验证：价格应该被更新为真实成交价
	if actionRecord.Price != 49950.0 {
		t.Errorf("期望价格被更新为 49950.0，实际: %.2f", actionRecord.Price)
	}
}

// TestVerifyAndUpdateCloseFillPrice_CloseShort tests close short position fill price verification
func TestVerifyAndUpdateCloseFillPrice_CloseShort(t *testing.T) {
	// Spec: 平空仓后应该能够获取真实成交价格并更新到 actionRecord
	// 场景：平空仓时，市场价 30000，但真实成交价 30050

	mockTrader := &mockTraderWithFills{
		fills: []mockFill{
			{
				symbol:    "ETHUSDT",
				side:      "Buy", // Close Short = Buy
				price:     30050.0,
				quantity:  1.0,
				timestamp: time.Now().UnixMilli(),
			},
		},
	}

	at := &AutoTrader{trader: mockTrader}

	decision := &decision.Decision{
		Symbol: "ETHUSDT",
		Action: "close_short",
	}

	actionRecord := &logger.DecisionAction{
		Symbol:    "ETHUSDT",
		Action:    "close_short",
		Price:     30000.0,
		Timestamp: time.Now(),
	}

	err := at.verifyAndUpdateCloseFillPrice(decision, actionRecord, time.Now().UnixMilli())
	if err != nil {
		t.Fatalf("verifyAndUpdateCloseFillPrice 失败: %v", err)
	}

	if actionRecord.Price != 30050.0 {
		t.Errorf("期望价格被更新为 30050.0，实际: %.2f", actionRecord.Price)
	}
}

// TestVerifyAndUpdateCloseFillPrice_NoFillRecordFound tests fallback when no fill record found
func TestVerifyAndUpdateCloseFillPrice_NoFillRecordFound(t *testing.T) {
	// Spec: 如果无法获取成交记录，应该保持使用平仓前的市场价格，并记录警告日志
	// 不应该阻断流程或抛出错误

	mockTrader := &mockTraderWithFills{
		fills: []mockFill{}, // 空列表
	}

	at := &AutoTrader{trader: mockTrader}

	decision := &decision.Decision{
		Symbol: "BTCUSDT",
		Action: "close_long",
	}

	actionRecord := &logger.DecisionAction{
		Symbol:    "BTCUSDT",
		Action:    "close_long",
		Price:     50000.0,
		Timestamp: time.Now(),
	}

	err := at.verifyAndUpdateCloseFillPrice(decision, actionRecord, time.Now().UnixMilli())

	// 不应该报错
	if err != nil {
		t.Errorf("未找到成交记录时不应该报错，实际: %v", err)
	}

	// 价格应该保持不变
	if actionRecord.Price != 50000.0 {
		t.Errorf("未找到成交记录时价格应该保持不变，实际: %.2f", actionRecord.Price)
	}
}

// TestVerifyAndUpdateCloseFillPrice_MultiplePartialFills tests handling of multiple partial fills
func TestVerifyAndUpdateCloseFillPrice_MultiplePartialFills(t *testing.T) {
	// Spec: 如果一次平仓产生了多个部分成交，应该计算加权平均成交价格
	// 例如：
	// - Fill 1: 0.5 BTC @ $91,800
	// - Fill 2: 0.3 BTC @ $91,820
	// 加权平均: (0.5*91800 + 0.3*91820) / (0.5 + 0.3) = $91,807.5

	mockTrader := &mockTraderWithFills{
		fills: []mockFill{
			{symbol: "BTCUSDT", side: "Sell", price: 91800.0, quantity: 0.5, timestamp: time.Now().UnixMilli()},
			{symbol: "BTCUSDT", side: "Sell", price: 91820.0, quantity: 0.3, timestamp: time.Now().UnixMilli()},
		},
	}

	at := &AutoTrader{trader: mockTrader}

	decision := &decision.Decision{
		Symbol: "BTCUSDT",
		Action: "close_long",
	}

	actionRecord := &logger.DecisionAction{
		Symbol:    "BTCUSDT",
		Action:    "close_long",
		Price:     91000.0,
		Timestamp: time.Now(),
	}

	err := at.verifyAndUpdateCloseFillPrice(decision, actionRecord, time.Now().UnixMilli())
	if err != nil {
		t.Fatalf("verifyAndUpdateCloseFillPrice 失败: %v", err)
	}

	// 计算期望的加权平均价格
	expectedPrice := (0.5*91800.0 + 0.3*91820.0) / (0.5 + 0.3)

	// 允许 0.01 的误差
	if actionRecord.Price < expectedPrice-0.01 || actionRecord.Price > expectedPrice+0.01 {
		t.Errorf("期望加权平均价格 %.2f，实际: %.2f", expectedPrice, actionRecord.Price)
	}
}


// Mock trader for testing
type mockTraderWithFills struct {
	fills []mockFill
}

type mockFill struct {
	symbol    string
	side      string  // "Buy" or "Sell"
	price     float64
	quantity  float64
	timestamp int64 // 毫秒时间戳
}

func (m *mockTraderWithFills) GetBalance() (map[string]interface{}, error) {
	return map[string]interface{}{
		"totalWalletBalance":    1000.0,
		"totalUnrealizedProfit": 0.0,
		"availableBalance":      1000.0,
	}, nil
}

func (m *mockTraderWithFills) GetPositions() ([]map[string]interface{}, error) {
	return []map[string]interface{}{}, nil
}

func (m *mockTraderWithFills) OpenLong(symbol string, quantity float64, leverage int) (map[string]interface{}, error) {
	return map[string]interface{}{"orderId": int64(1)}, nil
}

func (m *mockTraderWithFills) OpenShort(symbol string, quantity float64, leverage int) (map[string]interface{}, error) {
	return map[string]interface{}{"orderId": int64(1)}, nil
}

func (m *mockTraderWithFills) CloseLong(symbol string, quantity float64) (map[string]interface{}, error) {
	// 模拟平仓成功，返回订单ID和时间戳
	return map[string]interface{}{
		"orderId":   int64(123),
		"timestamp": time.Now().UnixMilli(),
	}, nil
}

func (m *mockTraderWithFills) CloseShort(symbol string, quantity float64) (map[string]interface{}, error) {
	return map[string]interface{}{
		"orderId":   int64(124),
		"timestamp": time.Now().UnixMilli(),
	}, nil
}

func (m *mockTraderWithFills) SetLeverage(symbol string, leverage int) error {
	return nil
}

func (m *mockTraderWithFills) SetMarginMode(symbol string, isCrossMargin bool) error {
	return nil
}

func (m *mockTraderWithFills) GetMarketPrice(symbol string) (float64, error) {
	return 50000.0, nil
}

func (m *mockTraderWithFills) SetStopLoss(symbol string, positionSide string, quantity, stopPrice float64) error {
	return nil
}

func (m *mockTraderWithFills) SetTakeProfit(symbol string, positionSide string, quantity, takeProfitPrice float64) error {
	return nil
}

func (m *mockTraderWithFills) CancelStopLossOrders(symbol string) error {
	return nil
}

func (m *mockTraderWithFills) CancelTakeProfitOrders(symbol string) error {
	return nil
}

func (m *mockTraderWithFills) CancelAllOrders(symbol string) error {
	return nil
}

func (m *mockTraderWithFills) CancelStopOrders(symbol string) error {
	return nil
}

func (m *mockTraderWithFills) FormatQuantity(symbol string, quantity float64) (string, error) {
	return "", nil
}

// 新增：获取成交记录的方法
func (m *mockTraderWithFills) GetRecentFills(symbol string, startTime int64, endTime int64) ([]map[string]interface{}, error) {
	// 模拟返回成交记录
	var result []map[string]interface{}

	for _, fill := range m.fills {
		if fill.symbol == symbol && fill.timestamp >= startTime && fill.timestamp <= endTime {
			result = append(result, map[string]interface{}{
				"symbol":    fill.symbol,
				"side":      fill.side,
				"price":     fill.price,
				"quantity":  fill.quantity,
				"timestamp": fill.timestamp,
			})
		}
	}

	return result, nil
}
