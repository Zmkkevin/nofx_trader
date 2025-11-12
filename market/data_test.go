package market

import (
	"fmt"
	"math"
	"testing"
)

// generateTestKlines 生成测试用的 K线数据
func generateTestKlines(count int) []Kline {
	klines := make([]Kline, count)
	for i := 0; i < count; i++ {
		// 生成模拟的价格数据，有一定的波动
		basePrice := 100.0
		variance := float64(i%10) * 0.5
		open := basePrice + variance
		high := open + 1.0
		low := open - 0.5
		close := open + 0.3
		volume := 1000.0 + float64(i*100)

		klines[i] = Kline{
			OpenTime:  int64(i * 180000), // 3分钟间隔
			Open:      open,
			High:      high,
			Low:       low,
			Close:     close,
			Volume:    volume,
			CloseTime: int64((i+1)*180000 - 1),
		}
	}
	return klines
}

// TestCalculateIntradaySeries_VolumeCollection 测试 Volume 数据收集
func TestCalculateIntradaySeries_VolumeCollection(t *testing.T) {
	tests := []struct {
		name           string
		klineCount     int
		expectedVolLen int
	}{
		{
			name:           "正常情况 - 20个K线",
			klineCount:     20,
			expectedVolLen: 10, // 应该收集最近10个
		},
		{
			name:           "刚好10个K线",
			klineCount:     10,
			expectedVolLen: 10,
		},
		{
			name:           "少于10个K线",
			klineCount:     5,
			expectedVolLen: 5, // 应该返回所有5个
		},
		{
			name:           "超过10个K线",
			klineCount:     30,
			expectedVolLen: 10, // 应该只返回最近10个
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			klines := generateTestKlines(tt.klineCount)
			data := calculateIntradaySeries(klines)

			if data == nil {
				t.Fatal("calculateIntradaySeries returned nil")
			}

			if len(data.Volume) != tt.expectedVolLen {
				t.Errorf("Volume length = %d, want %d", len(data.Volume), tt.expectedVolLen)
			}

			// 验证 Volume 数据正确性
			if len(data.Volume) > 0 {
				// 计算期望的起始索引
				start := tt.klineCount - 10
				if start < 0 {
					start = 0
				}

				// 验证第一个 Volume 值
				expectedFirstVolume := klines[start].Volume
				if data.Volume[0] != expectedFirstVolume {
					t.Errorf("First volume = %.2f, want %.2f", data.Volume[0], expectedFirstVolume)
				}

				// 验证最后一个 Volume 值
				expectedLastVolume := klines[tt.klineCount-1].Volume
				lastVolume := data.Volume[len(data.Volume)-1]
				if lastVolume != expectedLastVolume {
					t.Errorf("Last volume = %.2f, want %.2f", lastVolume, expectedLastVolume)
				}
			}
		})
	}
}

// TestCalculateIntradaySeries_VolumeValues 测试 Volume 值的正确性
func TestCalculateIntradaySeries_VolumeValues(t *testing.T) {
	klines := []Kline{
		{Close: 100.0, Volume: 1000.0, High: 101.0, Low: 99.0, Open: 100.0},
		{Close: 101.0, Volume: 1100.0, High: 102.0, Low: 100.0, Open: 101.0},
		{Close: 102.0, Volume: 1200.0, High: 103.0, Low: 101.0, Open: 102.0},
		{Close: 103.0, Volume: 1300.0, High: 104.0, Low: 102.0, Open: 103.0},
		{Close: 104.0, Volume: 1400.0, High: 105.0, Low: 103.0, Open: 104.0},
		{Close: 105.0, Volume: 1500.0, High: 106.0, Low: 104.0, Open: 105.0},
		{Close: 106.0, Volume: 1600.0, High: 107.0, Low: 105.0, Open: 106.0},
		{Close: 107.0, Volume: 1700.0, High: 108.0, Low: 106.0, Open: 107.0},
		{Close: 108.0, Volume: 1800.0, High: 109.0, Low: 107.0, Open: 108.0},
		{Close: 109.0, Volume: 1900.0, High: 110.0, Low: 108.0, Open: 109.0},
	}

	data := calculateIntradaySeries(klines)

	expectedVolumes := []float64{1000.0, 1100.0, 1200.0, 1300.0, 1400.0, 1500.0, 1600.0, 1700.0, 1800.0, 1900.0}

	if len(data.Volume) != len(expectedVolumes) {
		t.Fatalf("Volume length = %d, want %d", len(data.Volume), len(expectedVolumes))
	}

	for i, expected := range expectedVolumes {
		if data.Volume[i] != expected {
			t.Errorf("Volume[%d] = %.2f, want %.2f", i, data.Volume[i], expected)
		}
	}
}

// TestCalculateIntradaySeries_ATR14 测试 ATR14 计算
func TestCalculateIntradaySeries_ATR14(t *testing.T) {
	tests := []struct {
		name          string
		klineCount    int
		expectZero    bool
		expectNonZero bool
	}{
		{
			name:          "足够数据 - 20个K线",
			klineCount:    20,
			expectNonZero: true,
		},
		{
			name:          "刚好15个K线（ATR14需要至少15个）",
			klineCount:    15,
			expectNonZero: true,
		},
		{
			name:       "数据不足 - 14个K线",
			klineCount: 14,
			expectZero: true,
		},
		{
			name:       "数据不足 - 10个K线",
			klineCount: 10,
			expectZero: true,
		},
		{
			name:       "数据不足 - 5个K线",
			klineCount: 5,
			expectZero: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			klines := generateTestKlines(tt.klineCount)
			data := calculateIntradaySeries(klines)

			if data == nil {
				t.Fatal("calculateIntradaySeries returned nil")
			}

			if tt.expectZero && data.ATR14 != 0 {
				t.Errorf("ATR14 = %.3f, expected 0 (insufficient data)", data.ATR14)
			}

			if tt.expectNonZero && data.ATR14 <= 0 {
				t.Errorf("ATR14 = %.3f, expected > 0", data.ATR14)
			}
		})
	}
}

// TestCalculateATR 测试 ATR 计算函数
func TestCalculateATR(t *testing.T) {
	tests := []struct {
		name       string
		klines     []Kline
		period     int
		expectZero bool
	}{
		{
			name: "正常计算 - 足够数据",
			klines: []Kline{
				{High: 102.0, Low: 100.0, Close: 101.0},
				{High: 103.0, Low: 101.0, Close: 102.0},
				{High: 104.0, Low: 102.0, Close: 103.0},
				{High: 105.0, Low: 103.0, Close: 104.0},
				{High: 106.0, Low: 104.0, Close: 105.0},
				{High: 107.0, Low: 105.0, Close: 106.0},
				{High: 108.0, Low: 106.0, Close: 107.0},
				{High: 109.0, Low: 107.0, Close: 108.0},
				{High: 110.0, Low: 108.0, Close: 109.0},
				{High: 111.0, Low: 109.0, Close: 110.0},
				{High: 112.0, Low: 110.0, Close: 111.0},
				{High: 113.0, Low: 111.0, Close: 112.0},
				{High: 114.0, Low: 112.0, Close: 113.0},
				{High: 115.0, Low: 113.0, Close: 114.0},
				{High: 116.0, Low: 114.0, Close: 115.0},
			},
			period:     14,
			expectZero: false,
		},
		{
			name: "数据不足 - 等于period",
			klines: []Kline{
				{High: 102.0, Low: 100.0, Close: 101.0},
				{High: 103.0, Low: 101.0, Close: 102.0},
			},
			period:     2,
			expectZero: true,
		},
		{
			name: "数据不足 - 少于period",
			klines: []Kline{
				{High: 102.0, Low: 100.0, Close: 101.0},
			},
			period:     14,
			expectZero: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			atr := calculateATR(tt.klines, tt.period)

			if tt.expectZero {
				if atr != 0 {
					t.Errorf("calculateATR() = %.3f, expected 0 (insufficient data)", atr)
				}
			} else {
				if atr <= 0 {
					t.Errorf("calculateATR() = %.3f, expected > 0", atr)
				}
			}
		})
	}
}

// TestCalculateATR_TrueRange 测试 ATR 的 True Range 计算正确性
func TestCalculateATR_TrueRange(t *testing.T) {
	// 创建一个简单的测试用例，手动计算期望的 ATR
	klines := []Kline{
		{High: 50.0, Low: 48.0, Close: 49.0}, // TR = 2.0
		{High: 51.0, Low: 49.0, Close: 50.0}, // TR = max(2.0, 2.0, 1.0) = 2.0
		{High: 52.0, Low: 50.0, Close: 51.0}, // TR = max(2.0, 2.0, 1.0) = 2.0
		{High: 53.0, Low: 51.0, Close: 52.0}, // TR = 2.0
		{High: 54.0, Low: 52.0, Close: 53.0}, // TR = 2.0
	}

	atr := calculateATR(klines, 3)

	// 期望的计算：
	// TR[1] = max(51-49, |51-49|, |49-49|) = 2.0
	// TR[2] = max(52-50, |52-50|, |50-50|) = 2.0
	// TR[3] = max(53-51, |53-51|, |51-51|) = 2.0
	// 初始 ATR = (2.0 + 2.0 + 2.0) / 3 = 2.0
	// TR[4] = max(54-52, |54-52|, |52-52|) = 2.0
	// 平滑 ATR = (2.0*2 + 2.0) / 3 = 2.0

	expectedATR := 2.0
	tolerance := 0.01 // 允许小的浮点误差

	if math.Abs(atr-expectedATR) > tolerance {
		t.Errorf("calculateATR() = %.3f, want approximately %.3f", atr, expectedATR)
	}
}

// TestCalculateIntradaySeries_ConsistencyWithOtherIndicators 测试 Volume 和其他指标的一致性
func TestCalculateIntradaySeries_ConsistencyWithOtherIndicators(t *testing.T) {
	klines := generateTestKlines(30)
	data := calculateIntradaySeries(klines)

	// 所有数组应该存在
	if data.MidPrices == nil {
		t.Error("MidPrices should not be nil")
	}
	if data.Volume == nil {
		t.Error("Volume should not be nil")
	}

	// MidPrices 和 Volume 应该有相同的长度（都是最近10个）
	if len(data.MidPrices) != len(data.Volume) {
		t.Errorf("MidPrices length (%d) should equal Volume length (%d)",
			len(data.MidPrices), len(data.Volume))
	}

	// 所有 Volume 值应该大于 0
	for i, vol := range data.Volume {
		if vol <= 0 {
			t.Errorf("Volume[%d] = %.2f, should be > 0", i, vol)
		}
	}
}

// TestCalculateIntradaySeries_EmptyKlines 测试空 K线数据
func TestCalculateIntradaySeries_EmptyKlines(t *testing.T) {
	klines := []Kline{}
	data := calculateIntradaySeries(klines)

	if data == nil {
		t.Fatal("calculateIntradaySeries should not return nil for empty klines")
	}

	// 所有切片应该为空
	if len(data.MidPrices) != 0 {
		t.Errorf("MidPrices length = %d, want 0", len(data.MidPrices))
	}
	if len(data.Volume) != 0 {
		t.Errorf("Volume length = %d, want 0", len(data.Volume))
	}

	// ATR14 应该为 0（数据不足）
	if data.ATR14 != 0 {
		t.Errorf("ATR14 = %.3f, want 0", data.ATR14)
	}
}

// TestCalculateIntradaySeries_VolumePrecision 测试 Volume 精度保持
func TestCalculateIntradaySeries_VolumePrecision(t *testing.T) {
	klines := []Kline{
		{Close: 100.0, Volume: 1234.5678, High: 101.0, Low: 99.0},
		{Close: 101.0, Volume: 9876.5432, High: 102.0, Low: 100.0},
		{Close: 102.0, Volume: 5555.1111, High: 103.0, Low: 101.0},
	}

	data := calculateIntradaySeries(klines)

	expectedVolumes := []float64{1234.5678, 9876.5432, 5555.1111}

	for i, expected := range expectedVolumes {
		if data.Volume[i] != expected {
			t.Errorf("Volume[%d] = %.4f, want %.4f (precision not preserved)",
				i, data.Volume[i], expected)
		}
	}
}

// TestCalculateBuySellPressureRatio 测试买卖压力比计算
func TestCalculateBuySellPressureRatio(t *testing.T) {
	// 创建测试用的K线数据
	klines := []Kline{
		{Close: 100.0, Volume: 1000.0, TakerBuyBaseVolume: 400.0, TakerBuyQuoteVolume: 40000.0},
		{Close: 101.0, Volume: 1500.0, TakerBuyBaseVolume: 900.0, TakerBuyQuoteVolume: 90900.0},
		{Close: 99.0, Volume: 800.0, TakerBuyBaseVolume: 300.0, TakerBuyQuoteVolume: 29700.0},
		{Close: 102.0, Volume: 1200.0, TakerBuyBaseVolume: 500.0, TakerBuyQuoteVolume: 51000.0},
		{Close: 103.0, Volume: 2000.0, TakerBuyBaseVolume: 1200.0, TakerBuyQuoteVolume: 123600.0},
	}

	// 测试正常情况
	ratio := calculateBuySellPressureRatio(klines, 5)
	// 计算期望值：
	// Taker买入成交量总和 = 40000.0 + 90900.0 + 29700.0 + 51000.0 + 123600.0 = 335200.0
	// 总成交量 = 1000.0 + 1500.0 + 800.0 + 1200.0 + 2000.0 = 6500.0
	// Taker卖出成交量总和 = 总成交量 * 平均价格 - Taker买入成交量总和
	// 平均价格 = (100.0 + 101.0 + 99.0 + 102.0 + 103.0) / 5 = 101.0
	// Taker卖出成交量总和 = 6500.0 * 101.0 - 335200.0 = 656500.0 - 335200.0 = 321300.0
	// 比值 = 335200.0 / (335200.0 + 321300.0) = 335200.0 / 656500.0 ≈ 0.5106
	expected := 335200.0 / 656500.0
	if math.Abs(ratio-expected) > 0.0001 {
		t.Errorf("Expected ratio %.4f, got %.4f", expected, ratio)
	}

	// 测试K线数量不足的情况
	ratio = calculateBuySellPressureRatio(klines[:3], 5)
	if ratio != 0.0 {
		t.Errorf("Expected 0.0 when klines count is less than period, got %.4f", ratio)
	}

	// 测试空K线情况
	ratio = calculateBuySellPressureRatio([]Kline{}, 5)
	if ratio != 0.0 {
		t.Errorf("Expected 0.0 when klines is empty, got %.4f", ratio)
	}

	// 测试零总成交量情况
	zeroVolumeKlines := []Kline{
		{Close: 100.0, Volume: 0.0, TakerBuyBaseVolume: 0.0, TakerBuyQuoteVolume: 0.0},
		{Close: 101.0, Volume: 0.0, TakerBuyBaseVolume: 0.0, TakerBuyQuoteVolume: 0.0},
	}
	ratio = calculateBuySellPressureRatio(zeroVolumeKlines, 2)
	if ratio != 0.0 {
		t.Errorf("Expected 0.0 when total volume is zero, got %.4f", ratio)
	}
}

// TestCalculateIntradaySeries_BuySellRatios 测试日内序列买卖压力比计算
func TestCalculateIntradaySeries_BuySellRatios(t *testing.T) {
	// 创建测试用的K线数据
	klines := []Kline{
		{Close: 100.0, Volume: 1000.0, TakerBuyBaseVolume: 400.0, TakerBuyQuoteVolume: 40000.0},
		{Close: 101.0, Volume: 1500.0, TakerBuyBaseVolume: 900.0, TakerBuyQuoteVolume: 90900.0},
		{Close: 99.0, Volume: 800.0, TakerBuyBaseVolume: 300.0, TakerBuyQuoteVolume: 29700.0},
		{Close: 102.0, Volume: 1200.0, TakerBuyBaseVolume: 500.0, TakerBuyQuoteVolume: 51000.0},
		{Close: 103.0, Volume: 2000.0, TakerBuyBaseVolume: 1200.0, TakerBuyQuoteVolume: 123600.0},
		{Close: 104.0, Volume: 1800.0, TakerBuyBaseVolume: 1000.0, TakerBuyQuoteVolume: 104000.0},
		{Close: 105.0, Volume: 1600.0, TakerBuyBaseVolume: 800.0, TakerBuyQuoteVolume: 84000.0},
		{Close: 106.0, Volume: 1400.0, TakerBuyBaseVolume: 700.0, TakerBuyQuoteVolume: 74200.0},
		{Close: 107.0, Volume: 1300.0, TakerBuyBaseVolume: 600.0, TakerBuyQuoteVolume: 65100.0},
		{Close: 108.0, Volume: 1100.0, TakerBuyBaseVolume: 500.0, TakerBuyQuoteVolume: 54000.0},
	}

	data := calculateIntradaySeries(klines)

	// 检查买卖压力比数组是否存在
	if data.BuySellRatios == nil {
		t.Error("BuySellRatios should not be nil")
	}

	// 检查买卖压力比数组长度是否正确（应该与Volume数组长度相同）
	if len(data.BuySellRatios) != len(data.Volume) {
		t.Errorf("BuySellRatios length (%d) should equal Volume length (%d)",
			len(data.BuySellRatios), len(data.Volume))
	}

	// 检查所有买卖压力比值是否在合理范围内（0到1之间）
	for i, ratio := range data.BuySellRatios {
		if ratio < 0 || ratio > 1 {
			t.Errorf("BuySellRatios[%d] = %.4f, should be between 0 and 1", i, ratio)
		}
	}
}

// TestCalculateMidTermData 测试中期数据计算功能
func TestCalculateMidTermData(t *testing.T) {
	// 创建测试用的K线数据
	klines := []Kline{
		{Open: 100.0, High: 102.0, Low: 99.0, Close: 101.0, Volume: 1000.0},
		{Open: 101.0, High: 103.0, Low: 100.0, Close: 102.0, Volume: 1100.0},
		{Open: 102.0, High: 104.0, Low: 101.0, Close: 103.0, Volume: 1200.0},
		{Open: 103.0, High: 105.0, Low: 102.0, Close: 104.0, Volume: 1300.0},
		{Open: 104.0, High: 106.0, Low: 103.0, Close: 105.0, Volume: 1400.0},
		{Open: 105.0, High: 107.0, Low: 104.0, Close: 106.0, Volume: 1500.0},
		{Open: 106.0, High: 108.0, Low: 105.0, Close: 107.0, Volume: 1600.0},
		{Open: 107.0, High: 109.0, Low: 106.0, Close: 108.0, Volume: 1700.0},
		{Open: 108.0, High: 110.0, Low: 107.0, Close: 109.0, Volume: 1800.0},
		{Open: 109.0, High: 111.0, Low: 108.0, Close: 110.0, Volume: 1900.0},
		{Open: 110.0, High: 112.0, Low: 109.0, Close: 111.0, Volume: 2000.0},
		{Open: 111.0, High: 113.0, Low: 110.0, Close: 112.0, Volume: 2100.0},
		{Open: 112.0, High: 114.0, Low: 111.0, Close: 113.0, Volume: 2200.0},
		{Open: 113.0, High: 115.0, Low: 112.0, Close: 114.0, Volume: 2300.0},
		{Open: 114.0, High: 116.0, Low: 113.0, Close: 115.0, Volume: 2400.0},
		{Open: 115.0, High: 117.0, Low: 114.0, Close: 116.0, Volume: 2500.0},
		{Open: 116.0, High: 118.0, Low: 115.0, Close: 117.0, Volume: 2600.0},
		{Open: 117.0, High: 119.0, Low: 116.0, Close: 118.0, Volume: 2700.0},
		{Open: 118.0, High: 120.0, Low: 117.0, Close: 119.0, Volume: 2800.0},
		{Open: 119.0, High: 121.0, Low: 118.0, Close: 120.0, Volume: 2900.0},
	}

	// 测试15分钟时间框架
	data15m := calculateMidTermData(klines, "15m")

	// 验证返回的数据不为nil
	if data15m == nil {
		t.Fatal("calculateMidTermData() returned nil data for 15m timeframe")
	}

	// 验证基本字段
	if data15m.Timeframe != "15m" {
		t.Errorf("Timeframe = %s, want 15m", data15m.Timeframe)
	}

	// 验证EMA20存在且合理
	if data15m.EMA20 <= 0 {
		t.Error("EMA20 should be greater than 0")
	}

	// 验证ATR14存在且合理
	if data15m.ATR14 <= 0 {
		t.Error("ATR14 should be greater than 0")
	}

	// 验证成交量数据
	if data15m.CurrentVolume <= 0 {
		t.Error("CurrentVolume should be greater than 0")
	}

	if data15m.AverageVolume <= 0 {
		t.Error("AverageVolume should be greater than 0")
	}

	// 验证买卖压力比
	if data15m.BuySellRatio < 0 || data15m.BuySellRatio > 1 {
		t.Errorf("BuySellRatio = %.4f, should be between 0 and 1", data15m.BuySellRatio)
	}

	// 验证MACD数据数组存在
	if data15m.MACDValues == nil {
		t.Error("MACDValues should not be nil")
	}

	if data15m.SignalValues == nil {
		t.Error("SignalValues should not be nil")
	}

	if data15m.HistoValues == nil {
		t.Error("HistoValues should not be nil")
	}

	// 验证RSI数据数组存在
	if data15m.RSI7Values == nil {
		t.Error("RSI7Values should not be nil")
	}

	if data15m.RSI14Values == nil {
		t.Error("RSI14Values should not be nil")
	}

	// 验证斐波那契数据数组存在
	if data15m.FibRetrace382 == nil {
		t.Error("FibRetrace382 should not be nil")
	}

	if data15m.FibRetrace500 == nil {
		t.Error("FibRetrace500 should not be nil")
	}

	if data15m.FibRetrace618 == nil {
		t.Error("FibRetrace618 should not be nil")
	}

	if data15m.FibExtension1272 == nil {
		t.Error("FibExtension1272 should not be nil")
	}

	if data15m.FibExtension1618 == nil {
		t.Error("FibExtension1618 should not be nil")
	}

	if data15m.FibExtension2000 == nil {
		t.Error("FibExtension2000 should not be nil")
	}

	// 测试1小时时间框架
	data1h := calculateMidTermData(klines, "1h")

	// 验证返回的数据不为nil
	if data1h == nil {
		t.Fatal("calculateMidTermData() returned nil data for 1h timeframe")
	}

	// 验证基本字段
	if data1h.Timeframe != "1h" {
		t.Errorf("Timeframe = %s, want 1h", data1h.Timeframe)
	}
}

// TestCalculateLongerTermData_BuySellRatio 测试长期数据买卖压力比计算
func TestCalculateLongerTermData_BuySellRatio(t *testing.T) {
	// 创建测试用的K线数据
	klines := []Kline{
		{Close: 100.0, Volume: 1000.0, TakerBuyBaseVolume: 400.0, TakerBuyQuoteVolume: 40000.0},
		{Close: 101.0, Volume: 1500.0, TakerBuyBaseVolume: 900.0, TakerBuyQuoteVolume: 90900.0},
		{Close: 99.0, Volume: 800.0, TakerBuyBaseVolume: 300.0, TakerBuyQuoteVolume: 29700.0},
		{Close: 102.0, Volume: 1200.0, TakerBuyBaseVolume: 500.0, TakerBuyQuoteVolume: 51000.0},
		{Close: 103.0, Volume: 2000.0, TakerBuyBaseVolume: 1200.0, TakerBuyQuoteVolume: 123600.0},
		{Close: 104.0, Volume: 1800.0, TakerBuyBaseVolume: 1000.0, TakerBuyQuoteVolume: 104000.0},
		{Close: 105.0, Volume: 1600.0, TakerBuyBaseVolume: 800.0, TakerBuyQuoteVolume: 84000.0},
		{Close: 106.0, Volume: 1400.0, TakerBuyBaseVolume: 700.0, TakerBuyQuoteVolume: 74200.0},
		{Close: 107.0, Volume: 1300.0, TakerBuyBaseVolume: 600.0, TakerBuyQuoteVolume: 65100.0},
		{Close: 108.0, Volume: 1100.0, TakerBuyBaseVolume: 500.0, TakerBuyQuoteVolume: 54000.0},
		{Close: 109.0, Volume: 1200.0, TakerBuyBaseVolume: 600.0, TakerBuyQuoteVolume: 65400.0},
		{Close: 110.0, Volume: 1300.0, TakerBuyBaseVolume: 700.0, TakerBuyQuoteVolume: 77000.0},
		{Close: 111.0, Volume: 1400.0, TakerBuyBaseVolume: 800.0, TakerBuyQuoteVolume: 88800.0},
		{Close: 112.0, Volume: 1500.0, TakerBuyBaseVolume: 900.0, TakerBuyQuoteVolume: 100800.0},
		{Close: 113.0, Volume: 1600.0, TakerBuyBaseVolume: 1000.0, TakerBuyQuoteVolume: 113600.0},
		{Close: 114.0, Volume: 1700.0, TakerBuyBaseVolume: 1100.0, TakerBuyQuoteVolume: 127400.0},
		{Close: 115.0, Volume: 1800.0, TakerBuyBaseVolume: 1200.0, TakerBuyQuoteVolume: 141600.0},
		{Close: 116.0, Volume: 1900.0, TakerBuyBaseVolume: 1300.0, TakerBuyQuoteVolume: 156400.0},
		{Close: 117.0, Volume: 2000.0, TakerBuyBaseVolume: 1400.0, TakerBuyQuoteVolume: 171800.0},
		{Close: 118.0, Volume: 2100.0, TakerBuyBaseVolume: 1500.0, TakerBuyQuoteVolume: 187800.0},
	}

	data := calculateLongerTermData(klines)

	// 检查买卖压力比值是否存在
	if data.BuySellRatio <= 0 {
		t.Error("BuySellRatio should be greater than 0")
	}

	// 检查买卖压力比值是否在合理范围内（0到1之间）
	if data.BuySellRatio < 0 || data.BuySellRatio > 1 {
		t.Errorf("BuySellRatio = %.4f, should be between 0 and 1", data.BuySellRatio)
	}
}

// TestGetWith15mAnd1hTimeframes 测试Get方法使用15分钟和1小时时间框架数据
func TestGetWith15mAnd1hTimeframes(t *testing.T) {
	// 创建模拟的WSMonitor实例
	mockWSMonitor := &MockWSMonitor{
		klines3m:  generateTestKlinesWithTimeframe(20, "3m"),
		klines15m: generateTestKlinesWithTimeframe(10, "15m"),
		klines1h:  generateTestKlinesWithTimeframe(10, "1h"),
		klines4h:  generateTestKlinesWithTimeframe(5, "4h"),
	}

	// 保存原始的WSMonitorCli
	originalWSMonitorCli := WSMonitorCli
	WSMonitorCli = mockWSMonitor
	defer func() {
		WSMonitorCli = originalWSMonitorCli // 恢复原始值
	}()

	// 测试获取数据
	data, err := Get("BTCUSDT")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	// 验证返回的数据不为nil
	if data == nil {
		t.Fatal("Get() returned nil data")
	}

	// 验证所有价格变化字段都存在且合理
	if math.IsNaN(data.PriceChange15m) {
		t.Error("PriceChange15m should not be NaN")
	}

	if math.IsNaN(data.PriceChange1h) {
		t.Error("PriceChange1h should not be NaN")
	}

	if math.IsNaN(data.PriceChange4h) {
		t.Error("PriceChange4h should not be NaN")
	}

	// 验证价格变化值在合理范围内（-100到100之间）
	if data.PriceChange15m < -100 || data.PriceChange15m > 100 {
		t.Errorf("PriceChange15m = %.2f, should be between -100 and 100", data.PriceChange15m)
	}

	if data.PriceChange1h < -100 || data.PriceChange1h > 100 {
		t.Errorf("PriceChange1h = %.2f, should be between -100 and 100", data.PriceChange1h)
	}

	if data.PriceChange4h < -100 || data.PriceChange4h > 100 {
		t.Errorf("PriceChange4h = %.2f, should be between -100 and 100", data.PriceChange4h)
	}
}

// MockWSMonitor 模拟WSMonitor实现
type MockWSMonitor struct {
	klines3m  []Kline
	klines15m []Kline
	klines1h  []Kline
	klines4h  []Kline
}

// GetCurrentKlines 模拟获取K线数据的方法
func (m *MockWSMonitor) GetCurrentKlines(symbol string, timeframe string) ([]Kline, error) {
	switch timeframe {
	case "3m":
		return m.klines3m, nil
	case "15m":
		return m.klines15m, nil
	case "1h":
		return m.klines1h, nil
	case "4h":
		return m.klines4h, nil
	default:
		return nil, fmt.Errorf("unsupported timeframe: %s", timeframe)
	}
}

// generateTestKlinesWithTimeframe 生成指定时间框架的测试K线数据
func generateTestKlinesWithTimeframe(count int, timeframe string) []Kline {
	klines := make([]Kline, count)

	// 根据时间框架设置时间间隔（毫秒）
	interval := getTimeframeInterval(timeframe)

	for i := 0; i < count; i++ {
		// 生成模拟的价格数据，有一定的波动
		basePrice := 100.0
		variance := float64(i%10) * 0.5
		open := basePrice + variance
		high := open + 1.0
		low := open - 0.5
		close := open + 0.3
		volume := 1000.0 + float64(i*100)

		klines[i] = Kline{
			OpenTime:  int64(i * interval),
			Open:      open,
			High:      high,
			Low:       low,
			Close:     close,
			Volume:    volume,
			CloseTime: int64((i+1)*interval - 1),
		}
	}
	return klines
}

// getTimeframeInterval 获取时间框架对应的时间间隔（毫秒）
func getTimeframeInterval(timeframe string) int {
	switch timeframe {
	case "1m":
		return 60000
	case "3m":
		return 180000
	case "5m":
		return 300000
	case "15m":
		return 900000
	case "30m":
		return 1800000
	case "1h":
		return 3600000
	case "2h":
		return 7200000
	case "4h":
		return 14400000
	case "6h":
		return 21600000
	case "8h":
		return 28800000
	case "12h":
		return 43200000
	case "1d":
		return 86400000
	case "3d":
		return 259200000
	case "1w":
		return 604800000
	default:
		return 180000 // 默认3分钟
	}
}
