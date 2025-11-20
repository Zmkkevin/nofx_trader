package backtest

import (
	"testing"
)

func TestStopLossTakeProfitManagement(t *testing.T) {
	t.Run("should store stop loss and take profit on open", func(t *testing.T) {
		acc := NewBacktestAccount(10000, 5, 2)

		pos, _, _, err := acc.Open("BTCUSDT", "long", 0.1, 10, 50000, 49000, 52000, 0)
		if err != nil {
			t.Fatalf("Failed to open position: %v", err)
		}

		if pos.StopLoss != 49000 {
			t.Errorf("Expected stop loss 49000, got %.2f", pos.StopLoss)
		}
		if pos.TakeProfit != 52000 {
			t.Errorf("Expected take profit 52000, got %.2f", pos.TakeProfit)
		}
	})

	t.Run("should update stop loss", func(t *testing.T) {
		acc := NewBacktestAccount(10000, 5, 2)

		_, _, _, err := acc.Open("BTCUSDT", "long", 0.1, 10, 50000, 49000, 52000, 0)
		if err != nil {
			t.Fatalf("Failed to open position: %v", err)
		}

		err = acc.UpdateStopLoss("BTCUSDT", "long", 49500)
		if err != nil {
			t.Errorf("Failed to update stop loss: %v", err)
		}

		positions := acc.Positions()
		if len(positions) != 1 {
			t.Fatalf("Expected 1 position, got %d", len(positions))
		}

		pos := positions[0]
		if pos.StopLoss != 49500 {
			t.Errorf("Expected updated stop loss 49500, got %.2f", pos.StopLoss)
		}
	})

	t.Run("should update take profit", func(t *testing.T) {
		acc := NewBacktestAccount(10000, 5, 2)

		_, _, _, err := acc.Open("ETHUSDT", "short", 1.0, 10, 3000, 3100, 2900, 0)
		if err != nil {
			t.Fatalf("Failed to open position: %v", err)
		}

		err = acc.UpdateTakeProfit("ETHUSDT", "short", 2850)
		if err != nil {
			t.Errorf("Failed to update take profit: %v", err)
		}

		positions := acc.Positions()
		if len(positions) != 1 {
			t.Fatalf("Expected 1 position, got %d", len(positions))
		}

		pos := positions[0]
		if pos.TakeProfit != 2850 {
			t.Errorf("Expected updated take profit 2850, got %.2f", pos.TakeProfit)
		}
	})

	t.Run("should fail to update stop loss for non-existent position", func(t *testing.T) {
		acc := NewBacktestAccount(10000, 5, 2)

		err := acc.UpdateStopLoss("BTCUSDT", "long", 49500)
		if err == nil {
			t.Error("Expected error when updating stop loss for non-existent position, got nil")
		}
	})

	t.Run("should allow opening position without stop loss or take profit", func(t *testing.T) {
		acc := NewBacktestAccount(10000, 5, 2)

		pos, _, _, err := acc.Open("BTCUSDT", "long", 0.1, 10, 50000, 0, 0, 0)
		if err != nil {
			t.Fatalf("Failed to open position: %v", err)
		}

		if pos.StopLoss != 0 {
			t.Errorf("Expected stop loss 0 (not set), got %.2f", pos.StopLoss)
		}
		if pos.TakeProfit != 0 {
			t.Errorf("Expected take profit 0 (not set), got %.2f", pos.TakeProfit)
		}
	})

	t.Run("should update stop loss and take profit on add to position", func(t *testing.T) {
		acc := NewBacktestAccount(10000, 5, 2)

		// 第一次开仓
		_, _, _, err := acc.Open("BTCUSDT", "long", 0.1, 10, 50000, 49000, 52000, 0)
		if err != nil {
			t.Fatalf("Failed to open position: %v", err)
		}

		// 加仓并更新止损止盈
		pos, _, _, err := acc.Open("BTCUSDT", "long", 0.05, 10, 51000, 50000, 53000, 0)
		if err != nil {
			t.Fatalf("Failed to add to position: %v", err)
		}

		if pos.StopLoss != 50000 {
			t.Errorf("Expected updated stop loss 50000, got %.2f", pos.StopLoss)
		}
		if pos.TakeProfit != 53000 {
			t.Errorf("Expected updated take profit 53000, got %.2f", pos.TakeProfit)
		}
	})
}

func TestStopLossTakeProfitAutoTrigger(t *testing.T) {
	t.Run("should trigger stop loss for long position", func(t *testing.T) {
		acc := NewBacktestAccount(10000, 5, 2)

		// 开多仓，止损 49000
		_, _, _, err := acc.Open("BTCUSDT", "long", 0.1, 10, 50000, 49000, 52000, 0)
		if err != nil {
			t.Fatalf("Failed to open position: %v", err)
		}

		// 价格跌至止损价以下
		priceMap := map[string]float64{"BTCUSDT": 48900}

		// 检查触发
		triggers := acc.CheckStopLossTakeProfit(priceMap)

		if len(triggers) != 1 {
			t.Fatalf("Expected 1 trigger, got %d", len(triggers))
		}

		trigger := triggers[0]
		if trigger.TriggerType != "stop_loss" {
			t.Errorf("Expected trigger type 'stop_loss', got '%s'", trigger.TriggerType)
		}
		if trigger.Position.Symbol != "BTCUSDT" {
			t.Errorf("Expected symbol BTCUSDT, got %s", trigger.Position.Symbol)
		}
		if trigger.TriggerPrice != 49000 {
			t.Errorf("Expected trigger price 49000, got %.2f", trigger.TriggerPrice)
		}

		// 执行平仓
		_, _, _, err = acc.Close(trigger.Position.Symbol, trigger.Position.Side, trigger.Position.Quantity, 48900)
		if err != nil {
			t.Errorf("Failed to close position: %v", err)
		}

		// 验证持仓已平
		positions := acc.Positions()
		if len(positions) != 0 {
			t.Errorf("Expected position to be closed, but still exists")
		}
	})

	t.Run("should trigger take profit for long position", func(t *testing.T) {
		acc := NewBacktestAccount(10000, 5, 2)

		// 开多仓，止盈 52000
		_, _, _, err := acc.Open("BTCUSDT", "long", 0.1, 10, 50000, 49000, 52000, 0)
		if err != nil {
			t.Fatalf("Failed to open position: %v", err)
		}

		// 价格涨至止盈价以上
		priceMap := map[string]float64{"BTCUSDT": 52100}

		// 检查触发
		triggers := acc.CheckStopLossTakeProfit(priceMap)

		if len(triggers) != 1 {
			t.Fatalf("Expected 1 trigger, got %d", len(triggers))
		}

		trigger := triggers[0]
		if trigger.TriggerType != "take_profit" {
			t.Errorf("Expected trigger type 'take_profit', got '%s'", trigger.TriggerType)
		}

		// 执行平仓
		_, _, _, err = acc.Close(trigger.Position.Symbol, trigger.Position.Side, trigger.Position.Quantity, 52100)
		if err != nil {
			t.Errorf("Failed to close position: %v", err)
		}

		// 验证持仓已平
		positions := acc.Positions()
		if len(positions) != 0 {
			t.Errorf("Expected position to be closed, but still exists")
		}
	})

	t.Run("should trigger stop loss for short position", func(t *testing.T) {
		acc := NewBacktestAccount(10000, 5, 2)

		// 开空仓，止损 3100
		_, _, _, err := acc.Open("ETHUSDT", "short", 1.0, 10, 3000, 3100, 2900, 0)
		if err != nil {
			t.Fatalf("Failed to open position: %v", err)
		}

		// 价格涨至止损价以上
		priceMap := map[string]float64{"ETHUSDT": 3150}

		// 检查触发
		triggers := acc.CheckStopLossTakeProfit(priceMap)

		if len(triggers) != 1 {
			t.Fatalf("Expected 1 trigger, got %d", len(triggers))
		}

		trigger := triggers[0]
		if trigger.TriggerType != "stop_loss" {
			t.Errorf("Expected trigger type 'stop_loss', got '%s'", trigger.TriggerType)
		}

		// 执行平仓
		_, _, _, err = acc.Close(trigger.Position.Symbol, trigger.Position.Side, trigger.Position.Quantity, 3150)
		if err != nil {
			t.Errorf("Failed to close position: %v", err)
		}

		// 验证持仓已平
		positions := acc.Positions()
		if len(positions) != 0 {
			t.Errorf("Expected position to be closed, but still exists")
		}
	})

	t.Run("should trigger take profit for short position", func(t *testing.T) {
		acc := NewBacktestAccount(10000, 5, 2)

		// 开空仓，止盈 2900
		_, _, _, err := acc.Open("ETHUSDT", "short", 1.0, 10, 3000, 3100, 2900, 0)
		if err != nil {
			t.Fatalf("Failed to open position: %v", err)
		}

		// 价格跌至止盈价以下
		priceMap := map[string]float64{"ETHUSDT": 2850}

		// 检查触发
		triggers := acc.CheckStopLossTakeProfit(priceMap)

		if len(triggers) != 1 {
			t.Fatalf("Expected 1 trigger, got %d", len(triggers))
		}

		trigger := triggers[0]
		if trigger.TriggerType != "take_profit" {
			t.Errorf("Expected trigger type 'take_profit', got '%s'", trigger.TriggerType)
		}

		// 执行平仓
		_, _, _, err = acc.Close(trigger.Position.Symbol, trigger.Position.Side, trigger.Position.Quantity, 2850)
		if err != nil {
			t.Errorf("Failed to close position: %v", err)
		}

		// 验证持仓已平
		positions := acc.Positions()
		if len(positions) != 0 {
			t.Errorf("Expected position to be closed, but still exists")
		}
	})

	t.Run("should not trigger when price is within range", func(t *testing.T) {
		acc := NewBacktestAccount(10000, 5, 2)

		// 开多仓，止损 49000，止盈 52000
		_, _, _, err := acc.Open("BTCUSDT", "long", 0.1, 10, 50000, 49000, 52000, 0)
		if err != nil {
			t.Fatalf("Failed to open position: %v", err)
		}

		// 价格在止损止盈范围内
		priceMap := map[string]float64{"BTCUSDT": 50500}

		// 检查触发
		triggers := acc.CheckStopLossTakeProfit(priceMap)

		if len(triggers) != 0 {
			t.Errorf("Expected no triggers, but got %d", len(triggers))
		}

		// 验证持仓仍然存在
		positions := acc.Positions()
		if len(positions) != 1 {
			t.Errorf("Expected position to still exist, got %d positions", len(positions))
		}
	})

	t.Run("should prioritize stop loss over take profit", func(t *testing.T) {
		acc := NewBacktestAccount(10000, 5, 2)

		// 开多仓
		pos, _, _, _ := acc.Open("BTCUSDT", "long", 0.1, 10, 50000, 49000, 52000, 0)

		// 手动设置一个不可能的状态（止损和止盈同时满足，用于测试优先级）
		// 实际场景中不会发生，但用于测试代码逻辑
		pos.StopLoss = 50500  // 设置止损在当前价上方
		pos.TakeProfit = 49500 // 设置止盈在当前价下方

		// 价格 50000，既满足止损又满足止盈
		priceMap := map[string]float64{"BTCUSDT": 50000}

		triggers := acc.CheckStopLossTakeProfit(priceMap)

		// 应该只触发止损（优先级更高）
		if len(triggers) != 1 {
			t.Fatalf("Expected 1 trigger, got %d", len(triggers))
		}

		if triggers[0].TriggerType != "stop_loss" {
			t.Errorf("Expected stop_loss to be prioritized, got %s", triggers[0].TriggerType)
		}
	})

	t.Run("should handle multiple positions with different triggers", func(t *testing.T) {
		acc := NewBacktestAccount(100000, 5, 2)

		// 开两个仓位
		acc.Open("BTCUSDT", "long", 0.1, 10, 50000, 49000, 52000, 0)
		acc.Open("ETHUSDT", "short", 1.0, 10, 3000, 3100, 2900, 0)

		// BTC 触发止损，ETH 触发止盈
		priceMap := map[string]float64{
			"BTCUSDT": 48900, // 触发 BTC 止损
			"ETHUSDT": 2850,  // 触发 ETH 止盈
		}

		triggers := acc.CheckStopLossTakeProfit(priceMap)

		if len(triggers) != 2 {
			t.Fatalf("Expected 2 triggers, got %d", len(triggers))
		}

		// 验证触发类型
		btcTriggered := false
		ethTriggered := false
		for _, trigger := range triggers {
			if trigger.Position.Symbol == "BTCUSDT" && trigger.TriggerType == "stop_loss" {
				btcTriggered = true
			}
			if trigger.Position.Symbol == "ETHUSDT" && trigger.TriggerType == "take_profit" {
				ethTriggered = true
			}
		}

		if !btcTriggered {
			t.Error("Expected BTCUSDT stop loss to be triggered")
		}
		if !ethTriggered {
			t.Error("Expected ETHUSDT take profit to be triggered")
		}
	})
}
