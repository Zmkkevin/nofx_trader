package backtest

import (
	"testing"
)

// TestIntraBarStopLossTrigger 测试 K线内价格触发止损
// BUG 3: 当前代码只检查收盘价，忽略了 K线的 Low
func TestIntraBarStopLossTrigger(t *testing.T) {
	t.Run("long position should trigger stop loss at intrabar low price", func(t *testing.T) {
		// 场景：K线最低价触及止损，但收盘价未触及
		// 预期：应该触发止损

		acc := NewBacktestAccount(10000, 5, 2)

		// 开多仓，止损 49000
		_, _, _, err := acc.Open("BTCUSDT", "long", 0.1, 10, 50000, 49000, 52000, 0)
		if err != nil {
			t.Fatalf("Failed to open position: %v", err)
		}

		// K线数据：
		// Open:  50000
		// High:  51000
		// Low:   48000  ← 触及止损 49000
		// Close: 49500  ← 收盘价未触及止损

		// 当前代码只检查 Close，所以不会触发
		// 修复后应该检查 Low，触发止损

		// 构建价格映射（模拟当前代码行为）
		priceMapClose := map[string]float64{"BTCUSDT": 49500}

		// 当前的检查方法（只用 Close）
		triggers := acc.CheckStopLossTakeProfit(priceMapClose)

		// ❌ 当前行为：不会触发（因为 49500 > 49000）
		if len(triggers) != 0 {
			t.Errorf("Current behavior: should not trigger with close price only")
		}

		// 验证持仓还在
		positions := acc.Positions()
		if len(positions) != 1 {
			t.Fatalf("Position should still exist with current implementation")
		}
	})

	t.Run("long position stop loss should use low price when available", func(t *testing.T) {
		// 这个测试描述修复后的行为
		// 将在修复代码后启用
		t.Skip("Will implement after adding OHLC support")

		acc := NewBacktestAccount(10000, 5, 2)

		// 开多仓，止损 49000
		_, _, _, err := acc.Open("BTCUSDT", "long", 0.1, 10, 50000, 49000, 52000, 0)
		if err != nil {
			t.Fatalf("Failed to open position: %v", err)
		}

		// K线数据
		_ = map[string]float64{
			"BTCUSDT": 49500, // Close
		}
		_ = map[string]float64{
			"BTCUSDT": 51000, // High
		}
		_ = map[string]float64{
			"BTCUSDT": 48000, // Low ← 触及止损
		}

		// 期望：使用新的检查方法（支持 OHLC）
		// triggers := CheckStopLossTakeProfitWithOHLC(priceMap, highMap, lowMap)

		// ✅ 期望行为：应该触发（因为 Low 48000 < 49000）
		// if len(triggers) != 1 {
		// 	t.Errorf("Expected 1 trigger with low price, got %d", len(triggers))
		// }

		// if triggers[0].TriggerType != "stop_loss" {
		// 	t.Errorf("Expected stop_loss trigger")
		// }
	})

	t.Run("short position should trigger stop loss at intrabar high price", func(t *testing.T) {
		t.Skip("Will implement after adding OHLC support")

		acc := NewBacktestAccount(10000, 5, 2)

		// 开空仓，止损 3100
		_, _, _, err := acc.Open("ETHUSDT", "short", 1.0, 10, 3000, 3100, 2900, 0)
		if err != nil {
			t.Fatalf("Failed to open position: %v", err)
		}

		// K线数据：
		// Open:  3000
		// High:  3200  ← 触及止损 3100
		// Low:   2950
		// Close: 3050  ← 收盘价未触及止损

		_ = map[string]float64{"ETHUSDT": 3050}
		_ = map[string]float64{"ETHUSDT": 3200} // ← 触及止损
		_ = map[string]float64{"ETHUSDT": 2950}

		// 期望：应该触发止损（因为 High 3200 > 3100）
	})

	t.Run("take profit should use high price for long position", func(t *testing.T) {
		t.Skip("Will implement after adding OHLC support")

		acc := NewBacktestAccount(10000, 5, 2)

		// 开多仓，止盈 52000
		_, _, _, err := acc.Open("BTCUSDT", "long", 0.1, 10, 50000, 49000, 52000, 0)
		if err != nil {
			t.Fatalf("Failed to open position: %v", err)
		}

		// K线数据：
		// High:  52500  ← 触及止盈 52000
		// Close: 51500  ← 收盘价未触及止盈

		_ = map[string]float64{"BTCUSDT": 51500}
		_ = map[string]float64{"BTCUSDT": 52500} // ← 触及止盈
		_ = map[string]float64{"BTCUSDT": 50000}

		// 期望：应该触发止盈（因为 High 52500 > 52000）
	})

	t.Run("liquidation should use low price for long position", func(t *testing.T) {
		t.Skip("Will implement after adding OHLC support")

		acc := NewBacktestAccount(10000, 5, 2)

		// 开多仓，高杠杆，接近爆仓价
		pos, _, _, err := acc.Open("BTCUSDT", "long", 0.1, 20, 50000, 0, 0, 0)
		if err != nil {
			t.Fatalf("Failed to open position: %v", err)
		}

		liqPrice := pos.LiquidationPrice // 约 47500（20x杠杆）

		// K线数据：
		// Low:   47000  ← 触及爆仓价
		// Close: 48000  ← 收盘价未触及爆仓价

		_ = map[string]float64{"BTCUSDT": 48000}
		_ = map[string]float64{"BTCUSDT": 50000}
		_ = map[string]float64{"BTCUSDT": 47000} // ← 触及爆仓价

		// 期望：应该触发爆仓（因为 Low 47000 < liqPrice）
		_ = liqPrice
	})
}

// TestStopLossVsLiquidationPriority 测试止损和爆仓的优先级
// BUG 2: 止损和爆仓的优先级问题
func TestStopLossVsLiquidationPriority(t *testing.T) {
	t.Run("liquidation should have higher priority than stop loss", func(t *testing.T) {
		t.Skip("Will implement after fixing priority logic")

		acc := NewBacktestAccount(10000, 5, 2)

		// 开多仓
		pos, _, _, _ := acc.Open("BTCUSDT", "long", 0.1, 20, 50000, 49000, 52000, 0)
		liqPrice := pos.LiquidationPrice // 约 47500

		// 止损价：49000
		// 爆仓价：47500
		// 当前价：47000（同时触及止损和爆仓）

		_ = map[string]float64{"BTCUSDT": 47000}
		_ = map[string]float64{"BTCUSDT": 47000}
		_ = map[string]float64{"BTCUSDT": 47000}

		// 期望：触发爆仓（优先级高）
		// 成交价应该接近爆仓价 47500，而不是当前价 47000
		_ = liqPrice
	})
}

// TestAIDecisionThenStopLoss 测试 AI 修改止损后的立即检查
// BUG 1: AI 修改止损止盈后不立即检查
func TestAIDecisionThenStopLoss(t *testing.T) {
	t.Run("stop loss should trigger immediately after AI updates it", func(t *testing.T) {
		t.Skip("Will implement after adding double-check logic")

		acc := NewBacktestAccount(10000, 5, 2)

		// T0: 开仓，止损 49000，当前价 50000
		acc.Open("BTCUSDT", "long", 0.1, 10, 50000, 49000, 52000, 0)

		// T1: 当前价仍然是 50000
		priceMap := map[string]float64{"BTCUSDT": 50000}

		// 第一次检查：不触发（50000 > 49000）
		triggers1 := acc.CheckStopLossTakeProfit(priceMap)
		if len(triggers1) != 0 {
			t.Errorf("Should not trigger before AI update")
		}

		// AI 决策：修改止损到 50500
		err := acc.UpdateStopLoss("BTCUSDT", "long", 50500)
		if err != nil {
			t.Fatalf("Failed to update stop loss: %v", err)
		}

		// 第二次检查：应该触发（50000 < 50500）
		triggers2 := acc.CheckStopLossTakeProfit(priceMap)

		// ✅ 期望：立即触发
		if len(triggers2) != 1 {
			t.Errorf("Expected stop loss to trigger after AI update, got %d triggers", len(triggers2))
		}

		if len(triggers2) > 0 && triggers2[0].TriggerType != "stop_loss" {
			t.Errorf("Expected stop_loss trigger type")
		}
	})

	t.Run("newly opened position should check stop loss in same cycle", func(t *testing.T) {
		t.Skip("Will implement after adding double-check logic")

		acc := NewBacktestAccount(10000, 5, 2)

		// 当前价：48000（已经低于将要设置的止损价）
		priceMap := map[string]float64{"BTCUSDT": 48000}

		// AI 决策：开仓，止损 49000
		acc.Open("BTCUSDT", "long", 0.1, 10, 50000, 49000, 52000, 0)

		// 期望：立即检查并触发止损（48000 < 49000）
		triggers := acc.CheckStopLossTakeProfit(priceMap)

		if len(triggers) != 1 {
			t.Errorf("Expected stop loss to trigger for newly opened position, got %d triggers", len(triggers))
		}
	})
}
