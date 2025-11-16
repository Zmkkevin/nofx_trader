package logger

import (
	"os"
	"testing"
	"time"
)

// TestGetTakerFeeRate tests the getTakerFeeRate function for all supported exchanges
func TestGetTakerFeeRate(t *testing.T) {
	tests := []struct {
		name     string
		exchange string
		wantRate float64
	}{
		{
			name:     "Aster exchange returns 0.035% taker fee",
			exchange: "aster",
			wantRate: 0.00035,
		},
		{
			name:     "Hyperliquid exchange returns 0.045% taker fee",
			exchange: "hyperliquid",
			wantRate: 0.00045,
		},
		{
			name:     "Binance exchange returns 0.050% taker fee",
			exchange: "binance",
			wantRate: 0.0005,
		},
		{
			name:     "Unknown exchange defaults to 0.050% taker fee",
			exchange: "unknown_exchange",
			wantRate: 0.0005,
		},
		{
			name:     "Empty string defaults to 0.050% taker fee",
			exchange: "",
			wantRate: 0.0005,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getTakerFeeRate(tt.exchange)
			if got != tt.wantRate {
				t.Errorf("getTakerFeeRate(%q) = %v, want %v", tt.exchange, got, tt.wantRate)
			}
		})
	}
}

// TestPnLCalculationWithFees tests that P&L calculation correctly includes trading fees
func TestPnLCalculationWithFees(t *testing.T) {
	tests := []struct {
		name         string
		exchange     string
		side         string
		quantity     float64
		openPrice    float64
		closePrice   float64
		wantPnL      float64
		wantPnLRange [2]float64 // [min, max] for floating point tolerance
	}{
		{
			name:       "Long position profit on Aster",
			exchange:   "aster",
			side:       "long",
			quantity:   0.01,
			openPrice:  100000.0,
			closePrice: 101000.0,
			// Price diff: 0.01 * (101000 - 100000) = 10 USDT
			// Open fee: 0.01 * 100000 * 0.00035 = 0.35 USDT
			// Close fee: 0.01 * 101000 * 0.00035 = 0.3535 USDT
			// Total fees: 0.7035 USDT
			// Net PnL: 10 - 0.7035 = 9.2965 USDT
			wantPnLRange: [2]float64{9.296, 9.297},
		},
		{
			name:       "Long position loss on Aster",
			exchange:   "aster",
			side:       "long",
			quantity:   0.002,
			openPrice:  103960.7,
			closePrice: 103425.3,
			// Price diff: 0.002 * (103425.3 - 103960.7) = -1.0708 USDT
			// Open fee: 0.002 * 103960.7 * 0.00035 = 0.0728 USDT
			// Close fee: 0.002 * 103425.3 * 0.00035 = 0.0724 USDT
			// Total fees: 0.1452 USDT
			// Net PnL: -1.0708 - 0.1452 = -1.216 USDT
			wantPnLRange: [2]float64{-1.217, -1.215},
		},
		{
			name:       "Short position profit on Hyperliquid",
			exchange:   "hyperliquid",
			side:       "short",
			quantity:   0.01,
			openPrice:  50000.0,
			closePrice: 49000.0,
			// Price diff: 0.01 * (50000 - 49000) = 10 USDT
			// Open fee: 0.01 * 50000 * 0.00045 = 0.225 USDT
			// Close fee: 0.01 * 49000 * 0.00045 = 0.2205 USDT
			// Total fees: 0.4455 USDT
			// Net PnL: 10 - 0.4455 = 9.5545 USDT
			wantPnLRange: [2]float64{9.554, 9.555},
		},
		{
			name:       "Short position loss on Binance",
			exchange:   "binance",
			side:       "short",
			quantity:   0.1,
			openPrice:  3000.0,
			closePrice: 3100.0,
			// Price diff: 0.1 * (3000 - 3100) = -10 USDT
			// Open fee: 0.1 * 3000 * 0.0005 = 0.15 USDT
			// Close fee: 0.1 * 3100 * 0.0005 = 0.155 USDT
			// Total fees: 0.305 USDT
			// Net PnL: -10 - 0.305 = -10.305 USDT
			wantPnLRange: [2]float64{-10.306, -10.304},
		},
		{
			name:       "Small position on unknown exchange (uses default rate)",
			exchange:   "test_exchange",
			side:       "long",
			quantity:   0.001,
			openPrice:  50000.0,
			closePrice: 50500.0,
			// Price diff: 0.001 * (50500 - 50000) = 0.5 USDT
			// Open fee: 0.001 * 50000 * 0.0005 = 0.025 USDT
			// Close fee: 0.001 * 50500 * 0.0005 = 0.02525 USDT
			// Total fees: 0.05025 USDT
			// Net PnL: 0.5 - 0.05025 = 0.44975 USDT
			wantPnLRange: [2]float64{0.449, 0.451},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Calculate price difference P&L
			var pnl float64
			if tt.side == "long" {
				pnl = tt.quantity * (tt.closePrice - tt.openPrice)
			} else {
				pnl = tt.quantity * (tt.openPrice - tt.closePrice)
			}

			// Deduct trading fees
			feeRate := getTakerFeeRate(tt.exchange)
			openFee := tt.quantity * tt.openPrice * feeRate
			closeFee := tt.quantity * tt.closePrice * feeRate
			totalFees := openFee + closeFee
			pnl -= totalFees

			// Check if PnL is within expected range (for floating point tolerance)
			if pnl < tt.wantPnLRange[0] || pnl > tt.wantPnLRange[1] {
				t.Errorf("P&L calculation = %v, want range [%v, %v]", pnl, tt.wantPnLRange[0], tt.wantPnLRange[1])
				t.Logf("  Exchange: %s, Side: %s", tt.exchange, tt.side)
				t.Logf("  Quantity: %v, Open: %v, Close: %v", tt.quantity, tt.openPrice, tt.closePrice)
				t.Logf("  Fee rate: %v, Total fees: %v", feeRate, totalFees)
			}
		})
	}
}

// TestAnalyzePerformance_WithFees tests that AnalyzePerformance correctly calculates P&L with fees
func TestAnalyzePerformance_WithFees(t *testing.T) {
	// Create a temporary test logger
	logger := NewDecisionLogger(t.TempDir())

	// Create test records with open and close actions
	openTime := time.Now().Add(-1 * time.Hour)
	closeTime := time.Now()

	// Test case: Aster long position loss (from user's example)
	record := &DecisionRecord{
		Exchange:    "aster",
		CycleNumber: 1,
		Timestamp:   openTime,
		Success:     true,
		Decisions: []DecisionAction{
			{
				Action:    "open_long",
				Symbol:    "BTCUSDT",
				Quantity:  0.002,
				Leverage:  5,
				Price:     103960.7,
				Timestamp: openTime,
				Success:   true,
			},
		},
	}

	// Log the open position
	err := logger.LogDecision(record)
	if err != nil {
		t.Fatalf("Failed to log open position: %v", err)
	}

	// Create close position record
	closeRecord := &DecisionRecord{
		Exchange:    "aster",
		CycleNumber: 2,
		Timestamp:   closeTime,
		Success:     true,
		Decisions: []DecisionAction{
			{
				Action:    "close_long",
				Symbol:    "BTCUSDT",
				Quantity:  0.002,
				Leverage:  5,
				Price:     103425.3,
				Timestamp: closeTime,
				Success:   true,
			},
		},
	}

	err = logger.LogDecision(closeRecord)
	if err != nil {
		t.Fatalf("Failed to log close position: %v", err)
	}

	// Analyze performance
	analysis, err := logger.AnalyzePerformance(10)
	if err != nil {
		t.Fatalf("AnalyzePerformance failed: %v", err)
	}

	// Verify results
	if analysis.TotalTrades != 1 {
		t.Errorf("Expected 1 trade, got %d", analysis.TotalTrades)
	}

	if len(analysis.RecentTrades) != 1 {
		t.Fatalf("Expected 1 recent trade, got %d", len(analysis.RecentTrades))
	}

	trade := analysis.RecentTrades[0]

	// Expected P&L with fees (Aster 0.035% taker fee)
	// Price diff: 0.002 * (103425.3 - 103960.7) = -1.0708 USDT
	// Open fee: 0.002 * 103960.7 * 0.00035 = 0.0728 USDT
	// Close fee: 0.002 * 103425.3 * 0.00035 = 0.0724 USDT
	// Total fees: 0.1452 USDT
	// Net PnL: -1.0708 - 0.1452 = -1.216 USDT
	expectedPnLMin := -1.217
	expectedPnLMax := -1.215

	if trade.PnL < expectedPnLMin || trade.PnL > expectedPnLMax {
		t.Errorf("Trade P&L = %v, want range [%v, %v]", trade.PnL, expectedPnLMin, expectedPnLMax)
		t.Logf("  Symbol: %s, Side: %s", trade.Symbol, trade.Side)
		t.Logf("  Open: %v, Close: %v, Quantity: %v", trade.OpenPrice, trade.ClosePrice, trade.Quantity)
	}

	// Verify it's counted as a losing trade
	if analysis.LosingTrades != 1 {
		t.Errorf("Expected 1 losing trade, got %d", analysis.LosingTrades)
	}

	if analysis.WinningTrades != 0 {
		t.Errorf("Expected 0 winning trades, got %d", analysis.WinningTrades)
	}
}

// TestAnalyzePerformance_PartialCloseWithFees tests partial close fee accumulation
func TestAnalyzePerformance_PartialCloseWithFees(t *testing.T) {
	logger := NewDecisionLogger(t.TempDir())

	openTime := time.Now().Add(-2 * time.Hour)
	partialCloseTime := time.Now().Add(-1 * time.Hour)
	finalCloseTime := time.Now()

	// Open position
	openRecord := &DecisionRecord{
		Exchange:    "hyperliquid",
		CycleNumber: 1,
		Timestamp:   openTime,
		Success:     true,
		Decisions: []DecisionAction{
			{
				Action:    "open_long",
				Symbol:    "ETHUSDT",
				Quantity:  1.0, // 1 ETH
				Leverage:  10,
				Price:     2000.0,
				Timestamp: openTime,
				Success:   true,
			},
		},
	}
	logger.LogDecision(openRecord)

	// Partial close (50%)
	partialCloseRecord := &DecisionRecord{
		Exchange:    "hyperliquid",
		CycleNumber: 2,
		Timestamp:   partialCloseTime,
		Success:     true,
		Decisions: []DecisionAction{
			{
				Action:    "partial_close",
				Symbol:    "ETHUSDT",
				Quantity:  0.5, // Close 0.5 ETH
				Price:     2100.0,
				Timestamp: partialCloseTime,
				Success:   true,
			},
		},
	}
	logger.LogDecision(partialCloseRecord)

	// Final close (remaining 50%)
	finalCloseRecord := &DecisionRecord{
		Exchange:    "hyperliquid",
		CycleNumber: 3,
		Timestamp:   finalCloseTime,
		Success:     true,
		Decisions: []DecisionAction{
			{
				Action:    "close_long",
				Symbol:    "ETHUSDT",
				Quantity:  0.5, // Close remaining 0.5 ETH
				Price:     2150.0,
				Timestamp: finalCloseTime,
				Success:   true,
			},
		},
	}
	logger.LogDecision(finalCloseRecord)

	// Analyze performance
	analysis, err := logger.AnalyzePerformance(10)
	if err != nil {
		t.Fatalf("AnalyzePerformance failed: %v", err)
	}

	// Should count as 1 complete trade
	if analysis.TotalTrades != 1 {
		t.Errorf("Expected 1 trade, got %d", analysis.TotalTrades)
	}

	if len(analysis.RecentTrades) != 1 {
		t.Fatalf("Expected 1 recent trade, got %d", len(analysis.RecentTrades))
	}

	trade := analysis.RecentTrades[0]

	// Calculate expected P&L (Hyperliquid 0.045% taker fee)
	// Partial close: 0.5 * (2100 - 2000) = 50 USDT
	//   Open fee: 0.5 * 2000 * 0.00045 = 0.45 USDT
	//   Close fee: 0.5 * 2100 * 0.00045 = 0.4725 USDT
	//   Partial PnL: 50 - 0.45 - 0.4725 = 49.0775 USDT
	//
	// Final close: 0.5 * (2150 - 2000) = 75 USDT
	//   Open fee: 0.5 * 2000 * 0.00045 = 0.45 USDT
	//   Close fee: 0.5 * 2150 * 0.00045 = 0.48375 USDT
	//   Final PnL: 75 - 0.45 - 0.48375 = 74.06625 USDT
	//
	// Total PnL: 49.0775 + 74.06625 = 123.14375 USDT
	expectedPnLMin := 123.14
	expectedPnLMax := 123.15

	if trade.PnL < expectedPnLMin || trade.PnL > expectedPnLMax {
		t.Errorf("Trade P&L = %v, want range [%v, %v]", trade.PnL, expectedPnLMin, expectedPnLMax)
		t.Logf("  Symbol: %s, Side: %s", trade.Symbol, trade.Side)
		t.Logf("  Quantity: %v, Open: %v, Close: %v", trade.Quantity, trade.OpenPrice, trade.ClosePrice)
	}

	// Should be a winning trade
	if analysis.WinningTrades != 1 {
		t.Errorf("Expected 1 winning trade, got %d", analysis.WinningTrades)
	}
}

// TestFeeImpactOnPerformanceMetrics verifies that fees affect performance metrics correctly
func TestFeeImpactOnPerformanceMetrics(t *testing.T) {
	logger := NewDecisionLogger(t.TempDir())

	// Create two trades: one winning, one losing (after fees)
	baseTime := time.Now().Add(-2 * time.Hour)

	// Trade 1: Slight profit before fees, loss after fees
	// Open: 100, Close: 100.5, Quantity: 10 (Binance 0.05% fee)
	// Price diff: 10 * (100.5 - 100) = 5 USDT
	// Fees: 10*100*0.0005 + 10*100.5*0.0005 = 0.5 + 0.5025 = 1.0025 USDT
	// Net: 5 - 1.0025 = 3.9975 USDT (actually still profit, let me recalculate)
	// Let's use a closer price to demonstrate the fee impact

	records := []*DecisionRecord{
		// Trade 1 - open
		{
			Exchange:    "binance",
			CycleNumber: 1,
			Timestamp:   baseTime,
			Success:     true,
			Decisions: []DecisionAction{
				{
					Action:    "open_long",
					Symbol:    "BTCUSDT",
					Quantity:  0.01,
					Leverage:  5,
					Price:     50000.0,
					Timestamp: baseTime,
					Success:   true,
				},
			},
		},
		// Trade 1 - close (small profit after fees)
		{
			Exchange:    "binance",
			CycleNumber: 2,
			Timestamp:   baseTime.Add(30 * time.Minute),
			Success:     true,
			Decisions: []DecisionAction{
				{
					Action:    "close_long",
					Symbol:    "BTCUSDT",
					Price:     51000.0,
					Timestamp: baseTime.Add(30 * time.Minute),
					Success:   true,
				},
			},
		},
		// Trade 2 - open
		{
			Exchange:    "binance",
			CycleNumber: 3,
			Timestamp:   baseTime.Add(1 * time.Hour),
			Success:     true,
			Decisions: []DecisionAction{
				{
					Action:    "open_short",
					Symbol:    "ETHUSDT",
					Quantity:  0.5,
					Leverage:  5,
					Price:     3000.0,
					Timestamp: baseTime.Add(1 * time.Hour),
					Success:   true,
				},
			},
		},
		// Trade 2 - close (loss)
		{
			Exchange:    "binance",
			CycleNumber: 4,
			Timestamp:   baseTime.Add(90 * time.Minute),
			Success:     true,
			Decisions: []DecisionAction{
				{
					Action:    "close_short",
					Symbol:    "ETHUSDT",
					Price:     3100.0,
					Timestamp: baseTime.Add(90 * time.Minute),
					Success:   true,
				},
			},
		},
	}

	// Log all records
	for _, record := range records {
		if err := logger.LogDecision(record); err != nil {
			t.Fatalf("Failed to log decision: %v", err)
		}
	}

	// Analyze
	analysis, err := logger.AnalyzePerformance(10)
	if err != nil {
		t.Fatalf("AnalyzePerformance failed: %v", err)
	}

	// Should have 2 trades
	if analysis.TotalTrades != 2 {
		t.Errorf("Expected 2 trades, got %d", analysis.TotalTrades)
	}

	// Verify that win rate is calculated correctly
	if analysis.TotalTrades > 0 {
		expectedWinRate := (float64(analysis.WinningTrades) / float64(analysis.TotalTrades)) * 100
		if analysis.WinRate != expectedWinRate {
			t.Errorf("Win rate = %v, expected %v", analysis.WinRate, expectedWinRate)
		}
	}

	// All trades should have non-zero P&L (including fees)
	for i, trade := range analysis.RecentTrades {
		if trade.PnL == 0 {
			t.Errorf("Trade %d has zero P&L, fees may not be applied", i)
		}
	}
}

// TestTradesCache_AddAndGet 测试基本的添加和读取功能
func TestTradesCache_AddAndGet(t *testing.T) {
	logger := NewDecisionLogger("/tmp/test_cache")

	// 添加 3 笔交易
	trade1 := TradeOutcome{
		Symbol:     "BTCUSDT",
		Side:       "long",
		OpenPrice:  50000,
		ClosePrice: 51000,
		PnL:        100,
		OpenTime:   time.Now().Add(-2 * time.Hour),
		CloseTime:  time.Now().Add(-1 * time.Hour),
	}
	trade2 := TradeOutcome{
		Symbol:     "ETHUSDT",
		Side:       "short",
		OpenPrice:  3000,
		ClosePrice: 2900,
		PnL:        50,
		OpenTime:   time.Now().Add(-1 * time.Hour),
		CloseTime:  time.Now().Add(-30 * time.Minute),
	}
	trade3 := TradeOutcome{
		Symbol:     "BNBUSDT",
		Side:       "long",
		OpenPrice:  400,
		ClosePrice: 410,
		PnL:        10,
		OpenTime:   time.Now().Add(-30 * time.Minute),
		CloseTime:  time.Now(),
	}

	logger.AddTradeToCache(trade1)
	logger.AddTradeToCache(trade2)
	logger.AddTradeToCache(trade3)

	// 测试读取所有交易
	trades := logger.GetRecentTrades(10)
	if len(trades) != 3 {
		t.Errorf("Expected 3 trades, got %d", len(trades))
	}

	// 测试限制数量
	trades = logger.GetRecentTrades(2)
	if len(trades) != 2 {
		t.Errorf("Expected 2 trades, got %d", len(trades))
	}

	// 测试最新的在前（trade3 应该是第一个）
	if trades[0].Symbol != "BNBUSDT" {
		t.Errorf("Expected first trade to be BNBUSDT, got %s", trades[0].Symbol)
	}
}

// TestTradesCache_SizeLimit 测试缓存大小限制
func TestTradesCache_SizeLimit(t *testing.T) {
	logger := NewDecisionLogger("/tmp/test_cache_limit")

	// 缓存限制是 100 条，添加 120 条测试
	maxSize := 100
	for i := 0; i < maxSize+20; i++ {
		trade := TradeOutcome{
			Symbol:     "BTCUSDT",
			Side:       "long",
			OpenPrice:  50000,
			ClosePrice: 51000,
			PnL:        float64(i),
			OpenTime:   time.Now().Add(-time.Duration(i) * time.Minute),
			CloseTime:  time.Now(),
		}
		logger.AddTradeToCache(trade)
	}

	// 缓存应该只保留最新的 100 条
	trades := logger.GetRecentTrades(maxSize + 50)
	if len(trades) != maxSize {
		t.Errorf("Expected cache size to be limited to %d, got %d", maxSize, len(trades))
	}

	// 最新的交易（PnL = 119）应该在第一个
	if trades[0].PnL != float64(maxSize+19) {
		t.Errorf("Expected first trade PnL to be %d, got %f", maxSize+19, trades[0].PnL)
	}

	// 最旧的交易（PnL = 20）应该在最后
	if trades[len(trades)-1].PnL != 20 {
		t.Errorf("Expected last trade PnL to be 20, got %f", trades[len(trades)-1].PnL)
	}
}

// TestTradesCache_OrderNewestFirst 测试交易顺序（最新的在前）
func TestTradesCache_OrderNewestFirst(t *testing.T) {
	logger := NewDecisionLogger("/tmp/test_cache_order")

	baseTime := time.Now()

	// 按时间顺序添加交易
	for i := 0; i < 5; i++ {
		trade := TradeOutcome{
			Symbol:     "BTCUSDT",
			Side:       "long",
			OpenPrice:  50000,
			ClosePrice: 51000,
			PnL:        float64(i),
			OpenTime:   baseTime.Add(time.Duration(i) * time.Hour),
			CloseTime:  baseTime.Add(time.Duration(i+1) * time.Hour),
		}
		logger.AddTradeToCache(trade)
	}

	trades := logger.GetRecentTrades(5)

	// 验证顺序：最新的在前
	for i := 0; i < len(trades); i++ {
		expectedPnL := float64(4 - i) // 4, 3, 2, 1, 0
		if trades[i].PnL != expectedPnL {
			t.Errorf("Trade at index %d: expected PnL %f, got %f", i, expectedPnL, trades[i].PnL)
		}
	}
}

// TestTradesCache_ConcurrentAccess 测试并发安全
func TestTradesCache_ConcurrentAccess(t *testing.T) {
	logger := NewDecisionLogger("/tmp/test_cache_concurrent")

	// 并发写入
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(id int) {
			for j := 0; j < 100; j++ {
				trade := TradeOutcome{
					Symbol:     "BTCUSDT",
					Side:       "long",
					OpenPrice:  50000,
					ClosePrice: 51000,
					PnL:        float64(id*100 + j),
					OpenTime:   time.Now(),
					CloseTime:  time.Now(),
				}
				logger.AddTradeToCache(trade)
			}
			done <- true
		}(i)
	}

	// 并发读取
	for i := 0; i < 5; i++ {
		go func() {
			for j := 0; j < 100; j++ {
				logger.GetRecentTrades(10)
			}
			done <- true
		}()
	}

	// 等待所有 goroutine 完成
	for i := 0; i < 15; i++ {
		<-done
	}

	// 验证最终缓存有数据且没有 panic
	trades := logger.GetRecentTrades(100)
	if len(trades) == 0 {
		t.Error("Expected trades in cache after concurrent access")
	}
}

// TestTradesCache_NoDuplicatesOnReAnalyze 测试重复分析不会导致缓存重复
func TestTradesCache_NoDuplicatesOnReAnalyze(t *testing.T) {
	// 创建临时日志目录
	logDir := "/tmp/test_no_duplicates"
	os.RemoveAll(logDir)
	os.MkdirAll(logDir, 0700)
	defer os.RemoveAll(logDir)

	logger := NewDecisionLogger(logDir)

	// 模拟决策记录：开仓 -> 平仓
	baseTime := time.Now()
	records := []*DecisionRecord{
		// 开仓
		{
			Exchange:    "binance",
			CycleNumber: 1,
			Timestamp:   baseTime,
			Success:     true,
			Decisions: []DecisionAction{
				{
					Action:    "open_long",
					Symbol:    "BTCUSDT",
					Quantity:  1.0,
					Leverage:  5,
					Price:     50000.0,
					Timestamp: baseTime,
					Success:   true,
				},
			},
		},
		// 平仓
		{
			Exchange:    "binance",
			CycleNumber: 2,
			Timestamp:   baseTime.Add(30 * time.Minute),
			Success:     true,
			Decisions: []DecisionAction{
				{
					Action:    "close_long",
					Symbol:    "BTCUSDT",
					Price:     51000.0,
					Timestamp: baseTime.Add(30 * time.Minute),
					Success:   true,
				},
			},
		},
	}

	// 保存决策记录到文件
	for _, record := range records {
		if err := logger.LogDecision(record); err != nil {
			t.Fatalf("Failed to log decision: %v", err)
		}
	}

	// 第一次分析
	_, err := logger.AnalyzePerformance(10)
	if err != nil {
		t.Fatalf("First AnalyzePerformance failed: %v", err)
	}

	// 获取缓存
	trades1 := logger.GetRecentTrades(10)
	if len(trades1) != 1 {
		t.Errorf("Expected 1 trade after first analysis, got %d", len(trades1))
	}

	// 第二次分析（模拟重新启动或定期刷新）
	_, err = logger.AnalyzePerformance(10)
	if err != nil {
		t.Fatalf("Second AnalyzePerformance failed: %v", err)
	}

	// 再次获取缓存 - 应该还是 1 条，不应该重复
	trades2 := logger.GetRecentTrades(10)
	if len(trades2) != 1 {
		t.Errorf("Expected 1 trade after second analysis (no duplicates), got %d", len(trades2))
	}

	// 验证缓存内容一致
	if trades1[0].Symbol != trades2[0].Symbol ||
		trades1[0].OpenPrice != trades2[0].OpenPrice ||
		trades1[0].ClosePrice != trades2[0].ClosePrice {
		t.Error("Cached trade data changed between analyses")
	}
}

// TestLogDecision_AutoUpdateCache 测试 LogDecision 主动更新缓存
// 核心：不调用 AnalyzePerformance，缓存应自动填充
func TestLogDecision_AutoUpdateCache(t *testing.T) {
	logDir := "/tmp/test_auto_update_cache"
	os.RemoveAll(logDir)
	defer os.RemoveAll(logDir)

	logger := NewDecisionLogger(logDir)

	// 模拟一笔完整交易：开仓 -> 平仓
	openTime := time.Now().Add(-10 * time.Minute)
	closeTime := time.Now()

	// 1. 开仓 (open_long)
	openRecord := &DecisionRecord{
		Timestamp:   openTime,
		CycleNumber: 1,
		Exchange:    "hyperliquid",
		Success:     true,
		Decisions: []DecisionAction{
			{
				Action:    "open_long",
				Symbol:    "ETHUSDT",
				Price:     2000.0,
				Quantity:  1.0,
				Leverage:  5,
				Timestamp: openTime,
				Success:   true,
			},
		},
		Positions: []PositionSnapshot{
			{
				Symbol:      "ETHUSDT",
				Side:        "long",
				PositionAmt: 1.0,
				EntryPrice:  2000.0,
				MarkPrice:   2000.0,
			},
		},
	}

	err := logger.LogDecision(openRecord)
	if err != nil {
		t.Fatalf("Failed to log open decision: %v", err)
	}

	// 2. 平仓 (close_long)
	closeRecord := &DecisionRecord{
		Timestamp:   closeTime,
		CycleNumber: 2,
		Exchange:    "hyperliquid",
		Success:     true,
		Decisions: []DecisionAction{
			{
				Action:    "close_long",
				Symbol:    "ETHUSDT",
				Price:     2100.0,
				Quantity:  1.0,
				Timestamp: closeTime,
				Success:   true,
			},
		},
		Positions: []PositionSnapshot{}, // 平仓后没有持仓
	}

	err = logger.LogDecision(closeRecord)
	if err != nil {
		t.Fatalf("Failed to log close decision: %v", err)
	}

	// 3. 关键测试：不调用 AnalyzePerformance，直接检查缓存
	trades := logger.GetRecentTrades(10)

	// 期望：缓存里应该有 1 笔交易
	if len(trades) != 1 {
		t.Errorf("Expected 1 trade in cache (auto-updated), got %d", len(trades))
		return
	}

	// 验证交易数据正确
	trade := trades[0]
	if trade.Symbol != "ETHUSDT" {
		t.Errorf("Expected symbol ETHUSDT, got %s", trade.Symbol)
	}
	if trade.Side != "long" {
		t.Errorf("Expected side long, got %s", trade.Side)
	}
	if trade.OpenPrice != 2000.0 {
		t.Errorf("Expected open price 2000.0, got %f", trade.OpenPrice)
	}
	if trade.ClosePrice != 2100.0 {
		t.Errorf("Expected close price 2100.0, got %f", trade.ClosePrice)
	}
	if trade.PnL <= 0 {
		t.Errorf("Expected positive profit, got %f", trade.PnL)
	}
}

// TestLogDecision_AutoUpdateStats 测试统计信息实时维护
func TestLogDecision_AutoUpdateStats(t *testing.T) {
	logDir := "/tmp/test_auto_update_stats"
	os.RemoveAll(logDir)
	defer os.RemoveAll(logDir)

	logger := NewDecisionLogger(logDir)

	// 模拟两笔交易：一笔盈利，一笔亏损
	baseTime := time.Now().Add(-1 * time.Hour)

	// 交易 1：盈利 (ETHUSDT long)
	logger.LogDecision(&DecisionRecord{
		Timestamp:   baseTime,
		CycleNumber: 1,
		Exchange:    "hyperliquid",
		Success:     true,
		Decisions: []DecisionAction{
			{Action: "open_long", Symbol: "ETHUSDT", Price: 2000.0, Quantity: 1.0, Timestamp: baseTime, Success: true},
		},
		Positions: []PositionSnapshot{{Symbol: "ETHUSDT", Side: "long", PositionAmt: 1.0, EntryPrice: 2000.0}},
	})

	logger.LogDecision(&DecisionRecord{
		Timestamp:   baseTime.Add(10 * time.Minute),
		CycleNumber: 2,
		Exchange:    "hyperliquid",
		Success:     true,
		Decisions: []DecisionAction{
			{Action: "close_long", Symbol: "ETHUSDT", Price: 2100.0, Quantity: 1.0, Timestamp: baseTime.Add(10 * time.Minute), Success: true},
		},
		Positions: []PositionSnapshot{},
	})

	// 交易 2：亏损 (BTCUSDT short)
	logger.LogDecision(&DecisionRecord{
		Timestamp:   baseTime.Add(20 * time.Minute),
		CycleNumber: 3,
		Exchange:    "hyperliquid",
		Success:     true,
		Decisions: []DecisionAction{
			{Action: "open_short", Symbol: "BTCUSDT", Price: 50000.0, Quantity: 0.1, Timestamp: baseTime.Add(20 * time.Minute), Success: true},
		},
		Positions: []PositionSnapshot{{Symbol: "BTCUSDT", Side: "short", PositionAmt: 0.1, EntryPrice: 50000.0}},
	})

	logger.LogDecision(&DecisionRecord{
		Timestamp:   baseTime.Add(30 * time.Minute),
		CycleNumber: 4,
		Exchange:    "hyperliquid",
		Success:     true,
		Decisions: []DecisionAction{
			{Action: "close_short", Symbol: "BTCUSDT", Price: 51000.0, Quantity: 0.1, Timestamp: baseTime.Add(30 * time.Minute), Success: true},
		},
		Positions: []PositionSnapshot{},
	})

	// 关键测试：从缓存读取交易（不调用 AnalyzePerformance）
	trades := logger.GetRecentTrades(10)

	// 验证缓存有 2 笔交易
	if len(trades) != 2 {
		t.Errorf("Expected 2 trades in cache (auto-updated), got %d", len(trades))
		return
	}

	// 验证交易顺序（最新的在前）
	if trades[0].Symbol != "BTCUSDT" {
		t.Errorf("Expected first trade to be BTCUSDT (newest), got %s", trades[0].Symbol)
	}
	if trades[1].Symbol != "ETHUSDT" {
		t.Errorf("Expected second trade to be ETHUSDT (oldest), got %s", trades[1].Symbol)
	}

	// 验证盈亏计算正确
	ethTrade := trades[1] // ETHUSDT long 盈利
	if ethTrade.PnL <= 0 {
		t.Errorf("Expected ETHUSDT trade to be profitable, got PnL: %f", ethTrade.PnL)
	}

	btcTrade := trades[0] // BTCUSDT short 亏损
	if btcTrade.PnL >= 0 {
		t.Errorf("Expected BTCUSDT trade to be loss, got PnL: %f", btcTrade.PnL)
	}
}

// TestGetPerformanceWithCache 测试缓存懒加载逻辑
func TestGetPerformanceWithCache(t *testing.T) {
	// 创建临时测试目录
	tmpDir, err := os.MkdirTemp("", "test_performance_cache_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	logger := NewDecisionLogger(tmpDir)

	// 模拟一些历史交易数据
	for i := 1; i <= 5; i++ {
		record := &DecisionRecord{
			Timestamp: time.Now().Add(-time.Duration(i) * time.Hour),
			Success:   true,
			Decisions: []DecisionAction{
				{
					Action:    "open_long",
					Symbol:    "BTCUSDT",
					Quantity:  0.1,
					Price:     50000.0,
					Leverage:  10,
					Timestamp: time.Now().Add(-time.Duration(i) * time.Hour),
					Success:   true,
				},
				{
					Action:    "close_long",
					Symbol:    "BTCUSDT",
					Quantity:  0.1,
					Price:     51000.0,
					Timestamp: time.Now().Add(-time.Duration(i) * time.Hour).Add(30 * time.Minute),
					Success:   true,
				},
			},
		}
		if err := logger.LogDecision(record); err != nil {
			t.Fatalf("Failed to log decision: %v", err)
		}
	}

	// 测试 1: 首次调用应该触发大窗口扫描
	performance1, err := logger.GetPerformanceWithCache(20)
	if err != nil {
		t.Fatalf("GetPerformanceWithCache failed: %v", err)
	}

	if performance1 == nil {
		t.Fatal("Expected performance analysis, got nil")
	}

	if performance1.TotalTrades == 0 {
		t.Error("Expected total_trades > 0")
	}

	if len(performance1.RecentTrades) == 0 {
		t.Error("Expected recent_trades to be populated")
	}

	// 测试 2: 第二次调用应该使用缓存（不重新扫描）
	performance2, err := logger.GetPerformanceWithCache(10)
	if err != nil {
		t.Fatalf("Second GetPerformanceWithCache failed: %v", err)
	}

	if performance2 == nil {
		t.Fatal("Expected performance analysis, got nil")
	}

	// 验证返回的交易数量限制正确
	if len(performance2.RecentTrades) > 10 {
		t.Errorf("Expected at most 10 trades, got %d", len(performance2.RecentTrades))
	}

	// 测试 3: 统计信息应该一致（因为使用的是同一批数据）
	if performance1.TotalTrades != performance2.TotalTrades {
		t.Errorf("Expected same total_trades, got %d vs %d",
			performance1.TotalTrades, performance2.TotalTrades)
	}
}

// TestPerformanceDataConsistency 测试统计信息和交易列表的数据一致性
// 🎯 目标: 确保 TotalTrades 等统计信息与 RecentTrades 列表基于相同的数据源
func TestPerformanceDataConsistency(t *testing.T) {
	// Setup
	tmpDir := t.TempDir()
	logger := NewDecisionLogger(tmpDir)

	// 模拟通过主动维护填充缓存
	// 创建 10 笔交易: 6 笔盈利, 4 笔亏损
	trades := []struct {
		symbol     string
		side       string
		openPrice  float64
		closePrice float64
		quantity   float64
		leverage   int
	}{
		{"BTC", "long", 50000, 51000, 0.1, 10},  // +100 USDT (盈利)
		{"ETH", "long", 3000, 3100, 1.0, 10},    // +100 USDT (盈利)
		{"BTC", "short", 51000, 50500, 0.1, 10}, // +50 USDT (盈利)
		{"ETH", "short", 3100, 3150, 1.0, 10},   // -50 USDT (亏损)
		{"BTC", "long", 50500, 51500, 0.1, 10},  // +100 USDT (盈利)
		{"SOL", "long", 100, 95, 5.0, 10},       // -25 USDT (亏损)
		{"BTC", "short", 51500, 51000, 0.1, 10}, // +50 USDT (盈利)
		{"ETH", "long", 3150, 3100, 1.0, 10},    // -50 USDT (亏损)
		{"SOL", "short", 95, 90, 5.0, 10},       // +25 USDT (盈利)
		{"BTC", "long", 51000, 50800, 0.1, 10},  // -20 USDT (亏损)
	}

	baseTime := time.Now().Add(-1 * time.Hour)
	initialBalance := 10000.0
	currentBalance := initialBalance

	for i, trade := range trades {
		// 记录开仓
		openAction := "open_" + trade.side
		openRecord := &DecisionRecord{
			Timestamp: baseTime.Add(time.Duration(i*10) * time.Minute),
			Success:   true,
			Exchange:  "binance",
			Decisions: []DecisionAction{
				{
					Action:    openAction,
					Symbol:    trade.symbol,
					Price:     trade.openPrice,
					Quantity:  trade.quantity,
					Leverage:  trade.leverage,
					Timestamp: baseTime.Add(time.Duration(i*10) * time.Minute),
					Success:   true,
				},
			},
			AccountState: AccountSnapshot{
				TotalBalance: currentBalance,
			},
		}
		if err := logger.LogDecision(openRecord); err != nil {
			t.Fatalf("Failed to log open decision: %v", err)
		}

		// 计算盈亏
		var pnl float64
		if trade.side == "long" {
			pnl = (trade.closePrice - trade.openPrice) * trade.quantity
		} else {
			pnl = (trade.openPrice - trade.closePrice) * trade.quantity
		}
		// 扣除手续费
		feeRate := 0.0005 // Binance taker fee
		openFee := trade.openPrice * trade.quantity * feeRate
		closeFee := trade.closePrice * trade.quantity * feeRate
		pnl -= (openFee + closeFee)

		currentBalance += pnl

		// 记录平仓
		closeAction := "close_" + trade.side
		closeRecord := &DecisionRecord{
			Timestamp: baseTime.Add(time.Duration(i*10+5) * time.Minute),
			Success:   true,
			Exchange:  "binance",
			Decisions: []DecisionAction{
				{
					Action:    closeAction,
					Symbol:    trade.symbol,
					Price:     trade.closePrice,
					Quantity:  trade.quantity,
					Timestamp: baseTime.Add(time.Duration(i*10+5) * time.Minute),
					Success:   true,
				},
			},
			AccountState: AccountSnapshot{
				TotalBalance: currentBalance,
			},
		}
		if err := logger.LogDecision(closeRecord); err != nil {
			t.Fatalf("Failed to log close decision: %v", err)
		}
	}

	// 等待缓存更新
	time.Sleep(10 * time.Millisecond)

	// 🔬 测试: 获取性能分析 (请求所有交易)
	performance, err := logger.GetPerformanceWithCache(100)
	if err != nil {
		t.Fatalf("GetPerformanceWithCache failed: %v", err)
	}

	// ✅ 断言1: TotalTrades 应该等于 RecentTrades 的长度
	if performance.TotalTrades != len(performance.RecentTrades) {
		t.Errorf("❌ Data inconsistency: TotalTrades=%d but RecentTrades has %d items",
			performance.TotalTrades, len(performance.RecentTrades))
	}

	// ✅ 断言2: TotalTrades 应该等于实际交易数量
	expectedTrades := len(trades)
	if performance.TotalTrades != expectedTrades {
		t.Errorf("❌ Expected %d trades, but TotalTrades=%d",
			expectedTrades, performance.TotalTrades)
	}

	// ✅ 断言3: WinningTrades + LosingTrades 应该等于 TotalTrades
	if performance.WinningTrades+performance.LosingTrades != performance.TotalTrades {
		t.Errorf("❌ WinningTrades(%d) + LosingTrades(%d) != TotalTrades(%d)",
			performance.WinningTrades, performance.LosingTrades, performance.TotalTrades)
	}

	// ✅ 断言4: 验证盈利/亏损交易数量正确
	expectedWinning := 6
	expectedLosing := 4
	if performance.WinningTrades != expectedWinning {
		t.Errorf("❌ Expected %d winning trades, got %d",
			expectedWinning, performance.WinningTrades)
	}
	if performance.LosingTrades != expectedLosing {
		t.Errorf("❌ Expected %d losing trades, got %d",
			expectedLosing, performance.LosingTrades)
	}

	// ✅ 断言5: 胜率应该正确 (60%)
	expectedWinRate := 60.0
	if performance.WinRate != expectedWinRate {
		t.Errorf("❌ Expected win rate %.1f%%, got %.1f%%",
			expectedWinRate, performance.WinRate)
	}

	t.Logf("✅ Performance data consistency verified:")
	t.Logf("   TotalTrades: %d", performance.TotalTrades)
	t.Logf("   RecentTrades length: %d", len(performance.RecentTrades))
	t.Logf("   WinningTrades: %d, LosingTrades: %d", performance.WinningTrades, performance.LosingTrades)
	t.Logf("   WinRate: %.1f%%", performance.WinRate)
}

// TestEquityCacheMaintenance 测试 equity 历史缓存的正确维护
func TestEquityCacheMaintenance(t *testing.T) {
	tmpDir := t.TempDir()
	logger := NewDecisionLogger(tmpDir)

	baseTime := time.Now()

	// 记录5个决策，每个都有不同的账户余额
	equities := []float64{10000.0, 10100.0, 10050.0, 10200.0, 10150.0}

	for i, equity := range equities {
		record := &DecisionRecord{
			Timestamp:   baseTime.Add(time.Duration(i) * time.Minute),
			CycleNumber: i + 1,
			Success:     true,
			Exchange:    "binance",
			Decisions:   []DecisionAction{}, // hold 没有 decisions
			AccountState: AccountSnapshot{
				TotalBalance: equity,
			},
		}

		err := logger.LogDecision(record)
		if err != nil {
			t.Fatalf("Failed to log decision %d: %v", i+1, err)
		}
	}

	// 验证 equity 缓存（转换为具体类型以访问内部字段）
	concreteLogger := logger.(*DecisionLogger)
	concreteLogger.cacheMutex.RLock()
	cache := concreteLogger.equityCache
	concreteLogger.cacheMutex.RUnlock()

	// 1. 验证缓存条数正确
	if len(cache) != len(equities) {
		t.Errorf("Expected %d equity points, got %d", len(equities), len(cache))
	}

	// 2. 验证顺序：应该是倒序（最新的在前）
	expectedOrder := []float64{10150.0, 10200.0, 10050.0, 10100.0, 10000.0}
	for i, expected := range expectedOrder {
		if i < len(cache) {
			if cache[i].Equity != expected {
				t.Errorf("Equity point %d: expected %.2f, got %.2f", i, expected, cache[i].Equity)
			}
		}
	}

	// 3. 验证时间戳也是倒序
	for i := 0; i < len(cache)-1; i++ {
		if cache[i].Timestamp.Before(cache[i+1].Timestamp) {
			t.Errorf("Equity cache not in reverse chronological order at index %d", i)
		}
	}

	t.Logf("✅ Equity cache maintenance verified:")
	t.Logf("   Cache size: %d", len(cache))
	t.Logf("   Order: newest first (reverse chronological)")
	t.Logf("   Equity values: %v", expectedOrder)
}

// TestEquityCacheMaxSize 测试 equity 缓存的大小限制
func TestEquityCacheMaxSize(t *testing.T) {
	tmpDir := t.TempDir()
	logger := NewDecisionLogger(tmpDir)

	baseTime := time.Now()
	maxSize := 200 // 默认最大缓存大小

	// 记录超过最大缓存数量的决策
	for i := 0; i < maxSize+50; i++ {
		record := &DecisionRecord{
			Timestamp:   baseTime.Add(time.Duration(i) * time.Minute),
			CycleNumber: i + 1,
			Success:     true,
			Exchange:    "binance",
			Decisions:   []DecisionAction{},
			AccountState: AccountSnapshot{
				TotalBalance: 10000.0 + float64(i),
			},
		}

		err := logger.LogDecision(record)
		if err != nil {
			t.Fatalf("Failed to log decision %d: %v", i+1, err)
		}
	}

	// 验证缓存大小不超过限制（转换为具体类型）
	concreteLogger := logger.(*DecisionLogger)
	concreteLogger.cacheMutex.RLock()
	cacheSize := len(concreteLogger.equityCache)
	concreteLogger.cacheMutex.RUnlock()

	if cacheSize > maxSize {
		t.Errorf("Equity cache exceeded max size: got %d, max %d", cacheSize, maxSize)
	}

	if cacheSize != maxSize {
		t.Errorf("Equity cache size incorrect: expected %d, got %d", maxSize, cacheSize)
	}

	// 验证保留的是最新的数据
	concreteLogger.cacheMutex.RLock()
	newestEquity := concreteLogger.equityCache[0].Equity
	oldestEquity := concreteLogger.equityCache[len(concreteLogger.equityCache)-1].Equity
	concreteLogger.cacheMutex.RUnlock()

	expectedNewest := 10000.0 + float64(maxSize+49) // 最后一个记录
	expectedOldest := 10000.0 + float64(50)         // 第51个记录（因为保留最新200个）

	if newestEquity != expectedNewest {
		t.Errorf("Newest equity incorrect: expected %.2f, got %.2f", expectedNewest, newestEquity)
	}

	if oldestEquity != expectedOldest {
		t.Errorf("Oldest equity incorrect: expected %.2f, got %.2f", expectedOldest, oldestEquity)
	}

	t.Logf("✅ Equity cache max size verified:")
	t.Logf("   Cache size: %d (max: %d)", cacheSize, maxSize)
	t.Logf("   Newest equity: %.2f", newestEquity)
	t.Logf("   Oldest equity: %.2f", oldestEquity)
}

// TestSharpeRatioCalculation 测试从 equity 缓存计算 SharpeRatio
func TestSharpeRatioCalculation(t *testing.T) {
	tmpDir := t.TempDir()
	logger := NewDecisionLogger(tmpDir)
	concreteLogger := logger.(*DecisionLogger)

	baseTime := time.Now()

	// 测试用例1: 稳定增长的equity序列
	// 10000 -> 10100 (+1.0%) -> 10200 (+0.99%) -> 10300 (+0.98%)
	stableGrowth := []float64{10000.0, 10100.0, 10200.0, 10300.0}

	for i, equity := range stableGrowth {
		record := &DecisionRecord{
			Timestamp:   baseTime.Add(time.Duration(i) * time.Minute),
			CycleNumber: i + 1,
			Success:     true,
			Exchange:    "binance",
			Decisions:   []DecisionAction{},
			AccountState: AccountSnapshot{
				TotalBalance: equity,
			},
		}

		err := logger.LogDecision(record)
		if err != nil {
			t.Fatalf("Failed to log decision %d: %v", i+1, err)
		}
	}

	// 计算 SharpeRatio
	sharpeRatio := concreteLogger.calculateSharpeRatioFromEquity()

	// 验证 SharpeRatio 不为0（因为有正收益）
	if sharpeRatio == 0 {
		t.Errorf("Expected non-zero Sharpe ratio for stable growth, got 0")
	}

	// 对于稳定增长的序列，SharpeRatio 应该是正数
	if sharpeRatio < 0 {
		t.Errorf("Expected positive Sharpe ratio for stable growth, got %.4f", sharpeRatio)
	}

	t.Logf("✅ Stable growth Sharpe ratio: %.4f", sharpeRatio)

	// 测试用例2: 波动的equity序列
	tmpDir2 := t.TempDir()
	logger2 := NewDecisionLogger(tmpDir2)
	concreteLogger2 := logger2.(*DecisionLogger)

	volatileEquities := []float64{10000.0, 10100.0, 9900.0, 10200.0, 9800.0, 10300.0}

	for i, equity := range volatileEquities {
		record := &DecisionRecord{
			Timestamp:   baseTime.Add(time.Duration(i) * time.Minute),
			CycleNumber: i + 1,
			Success:     true,
			Exchange:    "binance",
			Decisions:   []DecisionAction{},
			AccountState: AccountSnapshot{
				TotalBalance: equity,
			},
		}

		err := logger2.LogDecision(record)
		if err != nil {
			t.Fatalf("Failed to log decision %d: %v", i+1, err)
		}
	}

	sharpeRatio2 := concreteLogger2.calculateSharpeRatioFromEquity()

	// 波动序列的 SharpeRatio 应该比稳定增长的小（因为标准差更大）
	if sharpeRatio2 >= sharpeRatio {
		t.Logf("⚠ Warning: Volatile series Sharpe (%.4f) >= Stable growth Sharpe (%.4f)",
			sharpeRatio2, sharpeRatio)
	}

	t.Logf("✅ Volatile series Sharpe ratio: %.4f", sharpeRatio2)

	// 测试用例3: 只有一个equity点（应该返回0）
	tmpDir3 := t.TempDir()
	logger3 := NewDecisionLogger(tmpDir3)
	concreteLogger3 := logger3.(*DecisionLogger)

	singleRecord := &DecisionRecord{
		Timestamp:   baseTime,
		CycleNumber: 1,
		Success:     true,
		Exchange:    "binance",
		Decisions:   []DecisionAction{},
		AccountState: AccountSnapshot{
			TotalBalance: 10000.0,
		},
	}

	err := logger3.LogDecision(singleRecord)
	if err != nil {
		t.Fatalf("Failed to log single decision: %v", err)
	}

	sharpeRatio3 := concreteLogger3.calculateSharpeRatioFromEquity()

	if sharpeRatio3 != 0 {
		t.Errorf("Expected Sharpe ratio = 0 for single equity point, got %.4f", sharpeRatio3)
	}

	t.Logf("✅ Single equity point Sharpe ratio: %.4f (expected 0)", sharpeRatio3)

	// 测试用例4: 空缓存（应该返回0）
	tmpDir4 := t.TempDir()
	logger4 := NewDecisionLogger(tmpDir4)
	concreteLogger4 := logger4.(*DecisionLogger)

	sharpeRatio4 := concreteLogger4.calculateSharpeRatioFromEquity()

	if sharpeRatio4 != 0 {
		t.Errorf("Expected Sharpe ratio = 0 for empty cache, got %.4f", sharpeRatio4)
	}

	t.Logf("✅ Empty cache Sharpe ratio: %.4f (expected 0)", sharpeRatio4)
}
