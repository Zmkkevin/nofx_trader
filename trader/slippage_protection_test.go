package trader

import (
	"math"
	"testing"
)

// TestCalculatePositionRisk_Short 测试空单风险计算
func TestCalculatePositionRisk_Short(t *testing.T) {
	tests := []struct {
		name             string
		entryPrice       float64
		stopLoss         float64
		positionSizeUSD  float64
		totalBalance     float64
		expectedRiskPct  float64 // 预期风险百分比
		tolerance        float64 // 允许误差
	}{
		{
			name:             "空单_正常风险",
			entryPrice:       94237.8,
			stopLoss:         95000.0,
			positionSizeUSD:  463.0,
			totalBalance:     171.67,
			expectedRiskPct:  2.45, // (463 * 0.808% + 0.46) / 171.67 * 100
			tolerance:        0.05,
		},
		{
			name:             "空单_低风险",
			entryPrice:       94000.0,
			stopLoss:         94500.0,
			positionSizeUSD:  400.0,
			totalBalance:     200.0,
			expectedRiskPct:  1.27, // (400 * 0.532% + 0.4) / 200 * 100
			tolerance:        0.05,
		},
		{
			name:             "空单_高风险",
			entryPrice:       94000.0,
			stopLoss:         96000.0,
			positionSizeUSD:  500.0,
			totalBalance:     150.0,
			expectedRiskPct:  7.43, // (500 * 2.128% + 0.5) / 150 * 100 = 7.43%
			tolerance:        0.05,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			risk := calculatePositionRisk(
				tt.entryPrice,
				tt.stopLoss,
				tt.positionSizeUSD,
				tt.totalBalance,
				"short",
			)

			if math.Abs(risk.RiskPercent-tt.expectedRiskPct) > tt.tolerance {
				t.Errorf("风险计算错误: got %.2f%%, want %.2f%% (±%.2f%%)",
					risk.RiskPercent, tt.expectedRiskPct, tt.tolerance)
			}

			// 验证风险金额为正
			if risk.TotalRiskUSD <= 0 {
				t.Errorf("总风险金额应该为正: got %.2f", risk.TotalRiskUSD)
			}

			// 验证手续费计算
			expectedFee := tt.positionSizeUSD * 0.0005 * 2
			if math.Abs(risk.FeeUSD-expectedFee) > 0.01 {
				t.Errorf("手续费计算错误: got %.2f, want %.2f", risk.FeeUSD, expectedFee)
			}
		})
	}
}

// TestCalculatePositionRisk_Long 测试多单风险计算
func TestCalculatePositionRisk_Long(t *testing.T) {
	tests := []struct {
		name             string
		entryPrice       float64
		stopLoss         float64
		positionSizeUSD  float64
		totalBalance     float64
		expectedRiskPct  float64
		tolerance        float64
	}{
		{
			name:             "多单_正常风险",
			entryPrice:       94000.0,
			stopLoss:         93000.0,
			positionSizeUSD:  450.0,
			totalBalance:     200.0,
			expectedRiskPct:  2.61, // (450 * 1.064% + 0.45) / 200 * 100
			tolerance:        0.05,
		},
		{
			name:             "多单_低风险",
			entryPrice:       3000.0,
			stopLoss:         2950.0,
			positionSizeUSD:  300.0,
			totalBalance:     200.0,
			expectedRiskPct:  2.65, // (300 * 1.667% + 0.3) / 200 * 100 = 2.65%
			tolerance:        0.05,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			risk := calculatePositionRisk(
				tt.entryPrice,
				tt.stopLoss,
				tt.positionSizeUSD,
				tt.totalBalance,
				"long",
			)

			if math.Abs(risk.RiskPercent-tt.expectedRiskPct) > tt.tolerance {
				t.Errorf("风险计算错误: got %.2f%%, want %.2f%% (±%.2f%%)",
					risk.RiskPercent, tt.expectedRiskPct, tt.tolerance)
			}
		})
	}
}

// TestCalculateMaxStopLoss_Short 测试空单最大止损计算
func TestCalculateMaxStopLoss_Short(t *testing.T) {
	tests := []struct {
		name            string
		entryPrice      float64
		positionSizeUSD float64
		totalBalance    float64
		maxRiskPercent  float64
		expectedStopLoss float64 // 预期止损价格（大约）
		tolerance       float64  // 允许误差
	}{
		{
			name:            "空单_2%风险限制",
			entryPrice:      94237.8,
			positionSizeUSD: 463.0,
			totalBalance:    171.67,
			maxRiskPercent:  2.0,
			expectedStopLoss: 94842.0, // 根据 2% 风险计算的止损价
			tolerance:       100.0,
		},
		{
			name:            "空单_1%风险限制",
			entryPrice:      94000.0,
			positionSizeUSD: 400.0,
			totalBalance:    200.0,
			maxRiskPercent:  1.0,
			expectedStopLoss: 94376.0,
			tolerance:       100.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stopLoss := calculateMaxStopLoss(
				tt.entryPrice,
				tt.positionSizeUSD,
				tt.totalBalance,
				tt.maxRiskPercent,
				"short",
			)

			if stopLoss <= 0 {
				t.Errorf("止损价格应该为正: got %.2f", stopLoss)
			}

			// 验证止损价格在合理范围内
			if math.Abs(stopLoss-tt.expectedStopLoss) > tt.tolerance {
				t.Errorf("止损价格计算错误: got %.2f, want %.2f (±%.2f)",
					stopLoss, tt.expectedStopLoss, tt.tolerance)
			}

			// 验证使用这个止损价格的风险确实 <= maxRiskPercent
			risk := calculatePositionRisk(
				tt.entryPrice,
				stopLoss,
				tt.positionSizeUSD,
				tt.totalBalance,
				"short",
			)

			if risk.RiskPercent > tt.maxRiskPercent+0.1 { // 允许 0.1% 误差
				t.Errorf("计算的止损价格导致风险超限: got %.2f%%, want <= %.2f%%",
					risk.RiskPercent, tt.maxRiskPercent)
			}
		})
	}
}

// TestCalculateMaxStopLoss_Long 测试多单最大止损计算
func TestCalculateMaxStopLoss_Long(t *testing.T) {
	tests := []struct {
		name            string
		entryPrice      float64
		positionSizeUSD float64
		totalBalance    float64
		maxRiskPercent  float64
		expectedStopLoss float64
		tolerance       float64
	}{
		{
			name:            "多单_2%风险限制",
			entryPrice:      94000.0,
			positionSizeUSD: 450.0,
			totalBalance:    200.0,
			maxRiskPercent:  2.0,
			expectedStopLoss: 93258.0, // 根据 2% 风险计算
			tolerance:       100.0,
		},
		{
			name:            "多单_1%风险限制",
			entryPrice:      3000.0,
			positionSizeUSD: 300.0,
			totalBalance:    200.0,
			maxRiskPercent:  1.0,
			expectedStopLoss: 2983.0,
			tolerance:       50.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stopLoss := calculateMaxStopLoss(
				tt.entryPrice,
				tt.positionSizeUSD,
				tt.totalBalance,
				tt.maxRiskPercent,
				"long",
			)

			if stopLoss <= 0 {
				t.Errorf("止损价格应该为正: got %.2f", stopLoss)
			}

			if math.Abs(stopLoss-tt.expectedStopLoss) > tt.tolerance {
				t.Errorf("止损价格计算错误: got %.2f, want %.2f (±%.2f)",
					stopLoss, tt.expectedStopLoss, tt.tolerance)
			}

			// 验证风险 <= maxRiskPercent
			risk := calculatePositionRisk(
				tt.entryPrice,
				stopLoss,
				tt.positionSizeUSD,
				tt.totalBalance,
				"long",
			)

			if risk.RiskPercent > tt.maxRiskPercent+0.1 {
				t.Errorf("计算的止损价格导致风险超限: got %.2f%%, want <= %.2f%%",
					risk.RiskPercent, tt.maxRiskPercent)
			}
		})
	}
}

// TestCalculateMaxStopLoss_EdgeCases 测试边界情况
func TestCalculateMaxStopLoss_EdgeCases(t *testing.T) {
	tests := []struct {
		name            string
		entryPrice      float64
		positionSizeUSD float64
		totalBalance    float64
		maxRiskPercent  float64
		side            string
		expectedZero    bool // 是否期望返回 0
	}{
		{
			name:            "余额不足_无法满足风险要求",
			entryPrice:      94000.0,
			positionSizeUSD: 463.0,
			totalBalance:    20.0, // 太少！20 < 463*0.05 = 23.15
			maxRiskPercent:  2.0,
			side:            "short",
			expectedZero:    true,
		},
		{
			name:            "临界情况_刚好不够",
			entryPrice:      94000.0,
			positionSizeUSD: 400.0,
			totalBalance:    20.0, // 20 = 400*0.05，临界值
			maxRiskPercent:  2.0,
			side:            "long",
			expectedZero:    true,
		},
		{
			name:            "手续费占比太高",
			entryPrice:      3000.0,
			positionSizeUSD: 100.0,
			totalBalance:    5.0, // 5*0.02 = 0.1, 手续费 100*0.001 = 0.1
			maxRiskPercent:  2.0,
			side:            "long",
			expectedZero:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stopLoss := calculateMaxStopLoss(
				tt.entryPrice,
				tt.positionSizeUSD,
				tt.totalBalance,
				tt.maxRiskPercent,
				tt.side,
			)

			if tt.expectedZero {
				if stopLoss != 0 {
					t.Errorf("Expected stopLoss = 0 (insufficient balance), got %.2f", stopLoss)
				}
				t.Logf("✓ 正确返回 0，表示无法满足风险要求")
			} else {
				if stopLoss <= 0 {
					t.Errorf("Expected stopLoss > 0, got %.2f", stopLoss)
				}
			}

			// 额外验证：计算说明
			feeUSD := tt.positionSizeUSD * 0.0005 * 2
			maxRiskUSD := (tt.totalBalance * tt.maxRiskPercent / 100) - feeUSD
			t.Logf("余额: %.2f, 仓位: %.2f, 手续费: %.2f, maxRiskUSD: %.2f",
				tt.totalBalance, tt.positionSizeUSD, feeUSD, maxRiskUSD)
		})
	}
}

// TestRealCaseFromUser 测试用户实际遇到的案例
func TestRealCaseFromUser(t *testing.T) {
	// 用户的实际交易数据
	entryPrice := 94237.8
	stopLoss := 95000.0
	positionSizeUSD := 463.0
	totalBalance := 171.67

	t.Run("验证实际风险超过2%", func(t *testing.T) {
		risk := calculatePositionRisk(
			entryPrice,
			stopLoss,
			positionSizeUSD,
			totalBalance,
			"short",
		)

		t.Logf("实际风险: %.2f%%", risk.RiskPercent)
		t.Logf("止损金额: $%.2f", risk.StopLossUSD)
		t.Logf("手续费: $%.2f", risk.FeeUSD)
		t.Logf("总风险: $%.2f", risk.TotalRiskUSD)

		if risk.RiskPercent <= 2.0 {
			t.Errorf("预期风险应该超过 2%%，但得到 %.2f%%", risk.RiskPercent)
		}
	})

	t.Run("计算应该使用的止损价格", func(t *testing.T) {
		maxStopLoss := calculateMaxStopLoss(
			entryPrice,
			positionSizeUSD,
			totalBalance,
			2.0,
			"short",
		)

		t.Logf("推荐止损价格: %.2f (原止损: %.2f)", maxStopLoss, stopLoss)

		// 验证新止损价格的风险确实 <= 2%
		risk := calculatePositionRisk(
			entryPrice,
			maxStopLoss,
			positionSizeUSD,
			totalBalance,
			"short",
		)

		if risk.RiskPercent > 2.1 {
			t.Errorf("调整后的风险仍超过 2%%: got %.2f%%", risk.RiskPercent)
		}

		t.Logf("调整后的风险: %.2f%%", risk.RiskPercent)
	})
}
