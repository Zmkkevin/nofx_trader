package decision

import (
	"nofx/market"
	"strings"
	"testing"
)

// TestBuildUserPrompt_NoDuplicateSymbols 测试 prompt 中不应出现重复的币种名称
// Issue: https://github.com/nofxai/nofx/issues/37
func TestBuildUserPrompt_NoDuplicateSymbols(t *testing.T) {
	// 创建测试上下文：持有 BTCUSDT，候选币种也包含 BTCUSDT
	ctx := &Context{
		CurrentTime:    "2024-01-01 12:00:00",
		RuntimeMinutes: 60,
		CallCount:      10,
		Account: AccountInfo{
			TotalEquity:      1000.0,
			AvailableBalance: 500.0,
			UnrealizedPnL:    50.0,
			TotalPnL:         100.0,
			TotalPnLPct:      10.0,
			MarginUsed:       500.0,
			MarginUsedPct:    50.0,
			PositionCount:    1,
		},
		Positions: []PositionInfo{
			{
				Symbol:           "BTCUSDT",
				Side:             "long",
				EntryPrice:       67000.0,
				MarkPrice:        68000.0,
				Quantity:         0.01,
				Leverage:         5,
				UnrealizedPnL:    10.0,
				UnrealizedPnLPct: 1.5,
				PeakPnLPct:       2.0,
				LiquidationPrice: 60000.0,
				MarginUsed:       134.0,
				UpdateTime:       1234567890000,
				StopLoss:         66000.0,
				TakeProfit:       70000.0,
			},
		},
		CandidateCoins: []CandidateCoin{
			{Symbol: "BTCUSDT", Sources: []string{"default"}},
			{Symbol: "ETHUSDT", Sources: []string{"default"}},
		},
		MarketDataMap: map[string]*market.Data{
			"BTCUSDT": {
				Symbol:         "BTCUSDT",
				CurrentPrice:   68000.0,
				PriceChange1h:  0.5,
				PriceChange4h:  1.2,
				CurrentEMA20:   67500.0,
				CurrentMACD:    0.0123,
				CurrentRSI7:    65.4,
				FundingRate:    0.0001,
				OpenInterest:   &market.OIData{Latest: 1200000000, Average: 1100000000},
			},
			"ETHUSDT": {
				Symbol:         "ETHUSDT",
				CurrentPrice:   3500.0,
				PriceChange1h:  0.3,
				PriceChange4h:  0.8,
				CurrentEMA20:   3480.0,
				CurrentMACD:    0.0045,
				CurrentRSI7:    58.2,
				FundingRate:    0.00008,
				OpenInterest:   &market.OIData{Latest: 800000000, Average: 750000000},
			},
		},
	}

	// 生成 prompt
	prompt := buildUserPrompt(ctx)

	// 测试 1: 统计 BTCUSDT 在 prompt 中出现的次数
	btcCount := strings.Count(prompt, "BTCUSDT")

	// 期望：BTCUSDT 应该只出现 1 次（在持仓部分）
	// 之前的bug：会出现 3 次（持仓header + market data section + candidate section）
	if btcCount > 1 {
		t.Errorf("BTCUSDT 在 prompt 中出现了 %d 次，预期最多 1 次\n完整 Prompt:\n%s", btcCount, prompt)
	}

	// 测试 2: 验证持仓的市场数据中不包含 Symbol 名称
	// 查找持仓部分
	positionSectionStart := strings.Index(prompt, "## 当前持仓")
	candidateSectionStart := strings.Index(prompt, "## 候选币种")

	if positionSectionStart == -1 || candidateSectionStart == -1 {
		t.Fatalf("未找到持仓或候选币种部分")
	}

	positionSection := prompt[positionSectionStart:candidateSectionStart]

	// 在持仓部分，BTCUSDT 应该只在 header 出现，不应该在 market data 中重复
	// 检查 "latest BTCUSDT open interest" 这种模式不应存在
	if strings.Contains(positionSection, "latest BTCUSDT") {
		t.Errorf("持仓的市场数据中不应包含币种名称，应使用通用描述\n持仓部分:\n%s", positionSection)
	}

	// 测试 3: 验证已持仓的币种不应出现在候选币种列表中
	candidateSection := prompt[candidateSectionStart:]

	// 在候选币种部分，不应该再次出现 "### N. BTCUSDT"
	btcInCandidates := strings.Contains(candidateSection, "### ") && strings.Contains(candidateSection, " BTCUSDT")

	if btcInCandidates {
		t.Errorf("已持仓的 BTCUSDT 不应出现在候选币种列表中\n候选币种部分:\n%s", candidateSection)
	}

	// 测试 4: 验证 ETHUSDT（未持仓）应该在候选币种中出现
	if !strings.Contains(candidateSection, "ETHUSDT") {
		t.Errorf("未持仓的 ETHUSDT 应该出现在候选币种列表中")
	}
}

// TestBuildUserPrompt_MarketDataStillPresent 测试去重后市场数据仍然存在
func TestBuildUserPrompt_MarketDataStillPresent(t *testing.T) {
	ctx := &Context{
		CurrentTime:    "2024-01-01 12:00:00",
		RuntimeMinutes: 60,
		CallCount:      10,
		Account: AccountInfo{
			TotalEquity:      1000.0,
			AvailableBalance: 500.0,
			MarginUsed:       500.0,
			MarginUsedPct:    50.0,
			PositionCount:    1,
		},
		Positions: []PositionInfo{
			{
				Symbol:     "BTCUSDT",
				Side:       "long",
				EntryPrice: 67000.0,
				MarkPrice:  68000.0,
				Quantity:   0.01,
				Leverage:   5,
			},
		},
		CandidateCoins: []CandidateCoin{},
		MarketDataMap: map[string]*market.Data{
			"BTCUSDT": {
				Symbol:         "BTCUSDT",
				CurrentPrice:   68000.0,
				CurrentEMA20:   67500.0,
				CurrentMACD:    0.0123,
				CurrentRSI7:    65.4,
				FundingRate:    0.0001,
				OpenInterest:   &market.OIData{Latest: 1200000000, Average: 1100000000},
			},
		},
	}

	prompt := buildUserPrompt(ctx)

	// 验证关键市场数据仍然存在
	requiredData := []string{
		"current_price",
		"current_ema20",
		"current_macd",
		"current_rsi",
		"Open Interest",
		"Funding Rate",
	}

	for _, data := range requiredData {
		if !strings.Contains(prompt, data) {
			t.Errorf("Prompt 中缺少必要的市场数据: %s\n完整 Prompt:\n%s", data, prompt)
		}
	}
}

// TestBuildUserPrompt_MultiplePositions 测试多个持仓时的去重
func TestBuildUserPrompt_MultiplePositions(t *testing.T) {
	ctx := &Context{
		CurrentTime:    "2024-01-01 12:00:00",
		RuntimeMinutes: 60,
		CallCount:      10,
		Account: AccountInfo{
			TotalEquity:      1000.0,
			AvailableBalance: 300.0,
			MarginUsed:       700.0,
			MarginUsedPct:    70.0,
			PositionCount:    3,
		},
		Positions: []PositionInfo{
			{Symbol: "BTCUSDT", Side: "long", EntryPrice: 67000.0, MarkPrice: 68000.0, Quantity: 0.01, Leverage: 5},
			{Symbol: "ETHUSDT", Side: "short", EntryPrice: 3500.0, MarkPrice: 3450.0, Quantity: 0.1, Leverage: 5},
			{Symbol: "SOLUSDT", Side: "long", EntryPrice: 100.0, MarkPrice: 105.0, Quantity: 1.0, Leverage: 5},
		},
		CandidateCoins: []CandidateCoin{
			{Symbol: "BTCUSDT", Sources: []string{"default"}},
			{Symbol: "ETHUSDT", Sources: []string{"default"}},
			{Symbol: "SOLUSDT", Sources: []string{"default"}},
			{Symbol: "BNBUSDT", Sources: []string{"default"}},
		},
		MarketDataMap: map[string]*market.Data{
			"BTCUSDT": {Symbol: "BTCUSDT", CurrentPrice: 68000.0, CurrentEMA20: 67500.0, CurrentMACD: 0.0123, CurrentRSI7: 65.4},
			"ETHUSDT": {Symbol: "ETHUSDT", CurrentPrice: 3450.0, CurrentEMA20: 3480.0, CurrentMACD: 0.0045, CurrentRSI7: 58.2},
			"SOLUSDT": {Symbol: "SOLUSDT", CurrentPrice: 105.0, CurrentEMA20: 103.0, CurrentMACD: 0.0012, CurrentRSI7: 62.1},
			"BNBUSDT": {Symbol: "BNBUSDT", CurrentPrice: 450.0, CurrentEMA20: 445.0, CurrentMACD: 0.0008, CurrentRSI7: 55.3},
		},
	}

	prompt := buildUserPrompt(ctx)

	// 测试：每个持仓币种应该只出现 1 次
	for _, pos := range ctx.Positions {
		count := strings.Count(prompt, pos.Symbol)
		if count > 1 {
			t.Errorf("%s 在 prompt 中出现了 %d 次，预期最多 1 次", pos.Symbol, count)
		}
	}

	// 测试：未持仓的 BNBUSDT 应该在候选币种中出现
	if !strings.Contains(prompt, "BNBUSDT") {
		t.Errorf("未持仓的 BNBUSDT 应该出现在候选币种列表中")
	}

	// 测试：候选币种数量应该只有 1 个（BNBUSDT），其他3个已持仓
	candidateSectionStart := strings.Index(prompt, "## 候选币种")
	if candidateSectionStart == -1 {
		t.Fatalf("未找到候选币种部分")
	}

	// 检查 "### 1. " 这种模式的数量（候选币种的编号）
	candidateSection := prompt[candidateSectionStart:]
	candidateCount := strings.Count(candidateSection, "### ")

	if candidateCount != 1 {
		t.Errorf("候选币种应该只有 1 个（BNBUSDT），实际有 %d 个\n候选币种部分:\n%s", candidateCount, candidateSection)
	}
}
