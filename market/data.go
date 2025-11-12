package market

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math"
	"strconv"
	"strings"
	"sync"
	"time"
)

// FundingRateCache 资金费率缓存结构
// Binance Funding Rate 每 8 小时才更新一次，使用 1 小时缓存可显著减少 API 调用
type FundingRateCache struct {
	Rate      float64
	UpdatedAt time.Time
}

var (
	fundingRateMap sync.Map // map[string]*FundingRateCache
	frCacheTTL     = 1 * time.Hour
)

// Get 获取指定代币的市场数据
func Get(symbol string) (*Data, error) {
	var klines3m, klines15m, klines1h, klines4h []Kline
	var err error
	// 标准化symbol
	symbol = Normalize(symbol)
	// 获取3分钟K线数据 (最近10个)
	klines3m, err = WSMonitorCli.GetCurrentKlines(symbol, "3m") // 多获取一些用于计算
	if err != nil {
		return nil, fmt.Errorf("获取3分钟K线失败: %v", err)
	}

	// 获取15分钟K线数据
	klines15m, err = WSMonitorCli.GetCurrentKlines(symbol, "15m")
	if err != nil {
		return nil, fmt.Errorf("获取15分钟K线失败: %v", err)
	}

	// 获取1小时K线数据
	klines1h, err = WSMonitorCli.GetCurrentKlines(symbol, "1h")
	if err != nil {
		return nil, fmt.Errorf("获取1小时K线失败: %v", err)
	}

	// 获取4小时K线数据 (最近10个)
	klines4h, err = WSMonitorCli.GetCurrentKlines(symbol, "4h") // 多获取用于计算指标
	if err != nil {
		return nil, fmt.Errorf("获取4小时K线失败: %v", err)
	}

	// 检查数据是否为空
	if len(klines3m) == 0 {
		return nil, fmt.Errorf("3分钟K线数据为空")
	}
	if len(klines15m) == 0 {
		return nil, fmt.Errorf("15分钟K线数据为空")
	}
	if len(klines1h) == 0 {
		return nil, fmt.Errorf("1小时K线数据为空")
	}
	if len(klines4h) == 0 {
		return nil, fmt.Errorf("4小时K线数据为空")
	}

	// 计算当前指标 (基于3分钟最新数据)
	currentPrice := klines3m[len(klines3m)-1].Close
	currentEMA20 := calculateEMA(klines3m, 20)
	currentMACD := calculateMACD(klines3m) // 现在返回完整的MACDData结构体
	currentRSI7 := calculateRSI(klines3m, 7)

	// 计算价格变化百分比
	// 15分钟价格变化 = 1个15分钟K线前的价格
	priceChange15m := 0.0
	if len(klines15m) >= 2 {
		price15mAgo := klines15m[len(klines15m)-2].Close
		if price15mAgo > 0 {
			priceChange15m = ((currentPrice - price15mAgo) / price15mAgo) * 100
		}
	}

	// 1小时价格变化 = 1个1小时K线前的价格
	priceChange1h := 0.0
	if len(klines1h) >= 2 {
		price1hAgo := klines1h[len(klines1h)-2].Close
		if price1hAgo > 0 {
			priceChange1h = ((currentPrice - price1hAgo) / price1hAgo) * 100
		}
	}

	// 4小时价格变化 = 1个4小时K线前的价格
	priceChange4h := 0.0
	if len(klines4h) >= 2 {
		price4hAgo := klines4h[len(klines4h)-2].Close
		if price4hAgo > 0 {
			priceChange4h = ((currentPrice - price4hAgo) / price4hAgo) * 100
		}
	}

	// 获取OI数据
	oiData, err := getOpenInterestData(symbol)
	if err != nil {
		// OI失败不影响整体,使用默认值
		oiData = &OIData{Latest: 0, Average: 0}
	}

	// 获取Funding Rate
	fundingRate, _ := getFundingRate(symbol)

	// 计算日内系列数据
	intradayData := calculateIntradaySeries(klines3m)

	// 计算中期数据(15分钟和1小时)
	midTermData15m := calculateMidTermData(klines15m, "15m")
	midTermData1h := calculateMidTermData(klines1h, "1h")

	// 计算长期数据
	longerTermData := calculateLongerTermData(klines4h)

	// 计算斐波那契OTE区域
	fibonacciOTE := calculateFibonacciOTE(klines4h, currentPrice)

	return &Data{
		Symbol:            symbol,
		CurrentPrice:      currentPrice,
		PriceChange15m:    priceChange15m,
		PriceChange1h:     priceChange1h,
		PriceChange4h:     priceChange4h,
		CurrentEMA20:      currentEMA20,
		CurrentMACD:       currentMACD,
		CurrentRSI7:       currentRSI7,
		OpenInterest:      oiData,
		FundingRate:       fundingRate,
		FibonacciOTE:      fibonacciOTE,
		IntradaySeries:    intradayData,
		MidTermContext15m: midTermData15m,
		MidTermContext1h:  midTermData1h,
		LongerTermContext: longerTermData,
	}, nil
}

// calculateEMA 计算EMA
func calculateEMA(klines []Kline, period int) float64 {
	if len(klines) < period {
		return 0
	}

	// 计算SMA作为初始EMA
	sum := 0.0
	for i := 0; i < period; i++ {
		sum += klines[i].Close
	}
	ema := sum / float64(period)

	// 计算EMA
	multiplier := 2.0 / float64(period+1)
	for i := period; i < len(klines); i++ {
		ema = (klines[i].Close-ema)*multiplier + ema
	}

	return ema
}

// calculateMACD 计算完整的MACD数据（MACD线、信号线、柱状图）
func calculateMACD(klines []Kline) MACDData {
	if len(klines) < 26 {
		return MACDData{0, 0, 0}
	}

	// 计算12期和26期EMA
	ema12 := calculateEMA(klines, 12)
	ema26 := calculateEMA(klines, 26)

	// MACD = EMA12 - EMA26
	macd := ema12 - ema26

	// 计算信号线（MACD的9期EMA）
	// 首先计算历史MACD值
	historicalMacds := make([]float64, 0, 26)
	for i := 25; i < len(klines); i++ {
		histEma12 := calculateEMA(klines[:i+1], 12)
		histEma26 := calculateEMA(klines[:i+1], 26)
		historicalMacds = append(historicalMacds, histEma12-histEma26)
	}

	// 计算信号线
	signal := 0.0
	if len(historicalMacds) >= 9 {
		// 将历史MACD值转换为临时的K线结构以使用calculateEMA
		tempKlines := make([]Kline, len(historicalMacds))
		for i, val := range historicalMacds {
			tempKlines[i] = Kline{Close: val}
		}
		signal = calculateEMA(tempKlines, 9)
	}

	// 柱状图 = MACD - 信号线
	histogram := macd - signal

	return MACDData{
		MACD:      macd,
		Signal:    signal,
		Histogram: histogram,
	}
}

// calculateRSI 计算RSI
func calculateRSI(klines []Kline, period int) float64 {
	if len(klines) <= period {
		return 0
	}

	gains := 0.0
	losses := 0.0

	// 计算初始平均涨跌幅
	for i := 1; i <= period; i++ {
		change := klines[i].Close - klines[i-1].Close
		if change > 0 {
			gains += change
		} else {
			losses += -change
		}
	}

	avgGain := gains / float64(period)
	avgLoss := losses / float64(period)

	// 使用Wilder平滑方法计算后续RSI
	for i := period + 1; i < len(klines); i++ {
		change := klines[i].Close - klines[i-1].Close
		if change > 0 {
			avgGain = (avgGain*float64(period-1) + change) / float64(period)
			avgLoss = (avgLoss * float64(period-1)) / float64(period)
		} else {
			avgGain = (avgGain * float64(period-1)) / float64(period)
			avgLoss = (avgLoss*float64(period-1) + (-change)) / float64(period)
		}
	}

	if avgLoss == 0 {
		return 100
	}

	rs := avgGain / avgLoss
	rsi := 100 - (100 / (1 + rs))

	return rsi
}

// calculateFibonacciOTE 计算斐波那契OTE区域
func calculateFibonacciOTE(klines []Kline, currentPrice float64) *FibonacciOTE {
	if len(klines) < 20 {
		return nil
	}

	// 找出近期的高低点
	high := klines[0].High
	low := klines[0].Low
	for _, kline := range klines {
		if kline.High > high {
			high = kline.High
		}
		if kline.Low < low {
			low = kline.Low
		}
	}

	// 计算波动范围
	rangeSize := high - low

	// 计算回撤位
	retrace382 := high - rangeSize*0.382
	retrace500 := high - rangeSize*0.500
	retrace618 := high - rangeSize*0.618

	// 计算扩展位
	extension1272 := high + rangeSize*0.272
	extension1618 := high + rangeSize*0.618
	extension2000 := high + rangeSize*1.0

	// 检查价格位置
	isAboveRetrace618 := currentPrice > retrace618

	// 检查是否接近127.2%扩展位（±1%）
	threshold := extension1272 * 0.01
	isNearExtension1272 := math.Abs(currentPrice-extension1272) <= threshold

	return &FibonacciOTE{
		Retracement382:      retrace382,
		Retracement500:      retrace500,
		Retracement618:      retrace618,
		Extension1272:       extension1272,
		Extension1618:       extension1618,
		Extension2000:       extension2000,
		IsAboveRetrace618:   isAboveRetrace618,
		IsNearExtension1272: isNearExtension1272,
	}
}

// calculateATR 计算ATR
func calculateATR(klines []Kline, period int) float64 {
	if len(klines) <= period {
		return 0
	}

	trs := make([]float64, len(klines))
	for i := 1; i < len(klines); i++ {
		high := klines[i].High
		low := klines[i].Low
		prevClose := klines[i-1].Close

		tr1 := high - low
		tr2 := math.Abs(high - prevClose)
		tr3 := math.Abs(low - prevClose)

		trs[i] = math.Max(tr1, math.Max(tr2, tr3))
	}

	// 计算初始ATR
	sum := 0.0
	for i := 1; i <= period; i++ {
		sum += trs[i]
	}
	atr := sum / float64(period)

	// Wilder平滑
	for i := period + 1; i < len(klines); i++ {
		atr = (atr*float64(period-1) + trs[i]) / float64(period)
	}

	return atr
}

// calculateBuySellPressureRatio 计算买卖压力比
// 买卖压力比 = 主动买入成交量 / 总成交量
func calculateBuySellPressureRatio(klines []Kline, period int) float64 {
	if len(klines) < period {
		return 0
	}

	// 获取最近period根K线
	start := len(klines) - period
	recentKlines := klines[start:]

	// 计算主动买入成交量和总成交量
	totalTakerBuyVolume := 0.0
	totalVolume := 0.0

	for _, kline := range recentKlines {
		totalTakerBuyVolume += kline.TakerBuyBaseVolume
		totalVolume += kline.Volume
	}

	// 避免除零错误
	if totalVolume == 0 {
		return 0
	}

	return totalTakerBuyVolume / totalVolume
}

// calculateIntradaySeries 计算日内系列数据
func calculateIntradaySeries(klines []Kline) *IntradayData {
	data := &IntradayData{
		MidPrices:    make([]float64, 0, 10),
		EMA20Values:  make([]float64, 0, 10),
		MACDValues:   make([]float64, 0, 10),
		SignalValues: make([]float64, 0, 10),
		HistoValues:  make([]float64, 0, 10),
		RSI7Values:   make([]float64, 0, 10),
		RSI14Values:  make([]float64, 0, 10),
		FibRetrace382: make([]float64, 0, 10),
		FibRetrace500: make([]float64, 0, 10),
		FibRetrace618: make([]float64, 0, 10),
		Volume:      make([]float64, 0, 10),
		BuySellRatios: make([]float64, 0, 10),
	}

	// 获取最近10个数据点
	start := len(klines) - 10
	if start < 0 {
		start = 0
	}

	for i := start; i < len(klines); i++ {
		data.MidPrices = append(data.MidPrices, klines[i].Close)
		data.Volume = append(data.Volume, klines[i].Volume)

		// 计算每个点的EMA20
		if i >= 19 {
			ema20 := calculateEMA(klines[:i+1], 20)
			data.EMA20Values = append(data.EMA20Values, ema20)
		}

		// 计算每个点的完整MACD数据
		if i >= 25 {
			macdData := calculateMACD(klines[:i+1])
			data.MACDValues = append(data.MACDValues, macdData.MACD)
			data.SignalValues = append(data.SignalValues, macdData.Signal)
			data.HistoValues = append(data.HistoValues, macdData.Histogram)
		}

		// 计算每个点的RSI
		if i >= 7 {
			rsi7 := calculateRSI(klines[:i+1], 7)
			data.RSI7Values = append(data.RSI7Values, rsi7)
		}
		if i >= 14 {
			rsi14 := calculateRSI(klines[:i+1], 14)
			data.RSI14Values = append(data.RSI14Values, rsi14)
		}
		
		// 计算每个点的买卖压力比(使用最近5根K线)
		if i >= 5 {
			buySellRatio := calculateBuySellPressureRatio(klines[:i+1], 5)
			data.BuySellRatios = append(data.BuySellRatios, buySellRatio)
		} else {
			data.BuySellRatios = append(data.BuySellRatios, 0)
		}
	}

	// 计算3m ATR14
	data.ATR14 = calculateATR(klines, 14)

	return data
}

// calculateMidTermData 计算中期数据(15分钟和1小时时间框架)
func calculateMidTermData(klines []Kline, timeframe string) *MidTermData {
	data := &MidTermData{
		Timeframe:        timeframe,
		MACDValues:       make([]float64, 0, 10),
		SignalValues:     make([]float64, 0, 10),
		HistoValues:      make([]float64, 0, 10),
		RSI7Values:       make([]float64, 0, 10),
		RSI14Values:      make([]float64, 0, 10),
	}

	// 计算EMA
	data.EMA20 = calculateEMA(klines, 20)

	// 计算ATR
	data.ATR14 = calculateATR(klines, 14)

	// 计算成交量
	if len(klines) > 0 {
		data.CurrentVolume = klines[len(klines)-1].Volume
		// 计算平均成交量
		sum := 0.0
		for _, k := range klines {
			sum += k.Volume
		}
		data.AverageVolume = sum / float64(len(klines))
	}

	// 计算买卖压力比(使用最近20根K线)
	data.BuySellRatio = calculateBuySellPressureRatio(klines, 20)

	// 计算MACD、RSI7和RSI14序列
	start := len(klines) - 10
	if start < 0 {
		start = 0
	}

	for i := start; i < len(klines); i++ {
		if i >= 25 {
			macdData := calculateMACD(klines[:i+1])
			data.MACDValues = append(data.MACDValues, macdData.MACD)
			data.SignalValues = append(data.SignalValues, macdData.Signal)
			data.HistoValues = append(data.HistoValues, macdData.Histogram)
		}
		if i >= 7 {
			rsi7 := calculateRSI(klines[:i+1], 7)
			data.RSI7Values = append(data.RSI7Values, rsi7)
		}
		if i >= 14 {
			rsi14 := calculateRSI(klines[:i+1], 14)
			data.RSI14Values = append(data.RSI14Values, rsi14)
		}
	}

	// 计算斐波那契回撤位和扩展位（只计算最新值）
	if len(klines) >= 20 {
		// 找出最近20根K线的高低点
		high := klines[len(klines)-20].High
		low := klines[len(klines)-20].Low
		for j := len(klines)-20; j < len(klines); j++ {
			if klines[j].High > high {
				high = klines[j].High
			}
			if klines[j].Low < low {
				low = klines[j].Low
			}
		}
		rangeSize := high - low
		// 回撤位
		data.FibRetrace382 = high - rangeSize*0.382
		data.FibRetrace500 = high - rangeSize*0.500
		data.FibRetrace618 = high - rangeSize*0.618
		// 扩展位
		data.FibExtension1272 = high + rangeSize*0.272
		data.FibExtension1618 = high + rangeSize*0.618
		data.FibExtension2000 = high + rangeSize*1.0
	}

	return data
}

// isDataTooSimilar 检查15分钟和1小时数据是否过于相似
func isDataTooSimilar(data15m, data1h *MidTermData) bool {
	// 检查EMA20差异
	emaDiff := math.Abs(data15m.EMA20 - data1h.EMA20)
	emaAvg := (data15m.EMA20 + data1h.EMA20) / 2
	emaSimilarity := emaDiff / emaAvg
	
	// 检查ATR14差异
	atrDiff := math.Abs(data15m.ATR14 - data1h.ATR14)
	atrAvg := (data15m.ATR14 + data1h.ATR14) / 2
	atrSimilarity := atrDiff / atrAvg
	
	// 检查斐波那契回撤位差异
	fibSimilarity := 0.0
	if data15m.FibRetrace382 > 0 && data1h.FibRetrace382 > 0 {
		fibDiff := math.Abs(data15m.FibRetrace382 - data1h.FibRetrace382)
		fibAvg := (data15m.FibRetrace382 + data1h.FibRetrace382) / 2
		fibSimilarity = fibDiff / fibAvg
	}
	
	// 如果所有指标的相似度都小于5%，则认为数据过于相似
	return emaSimilarity < 0.05 && atrSimilarity < 0.05 && fibSimilarity < 0.05
}

// calculateLongerTermData 计算长期数据
func calculateLongerTermData(klines []Kline) *LongerTermData {
	data := &LongerTermData{
		MACDValues:   make([]float64, 0, 10),
		SignalValues: make([]float64, 0, 10),
		HistoValues:  make([]float64, 0, 10),
		RSI14Values:  make([]float64, 0, 10),
	}

	// 计算EMA
	data.EMA20 = calculateEMA(klines, 20)
	data.EMA50 = calculateEMA(klines, 50)

	// 计算ATR
	data.ATR3 = calculateATR(klines, 3)
	data.ATR14 = calculateATR(klines, 14)

	// 计算成交量
	if len(klines) > 0 {
		data.CurrentVolume = klines[len(klines)-1].Volume
		// 计算平均成交量
		sum := 0.0
		for _, k := range klines {
			sum += k.Volume
		}
		data.AverageVolume = sum / float64(len(klines))
	}

	// 计算买卖压力比(使用最近20根K线)
	data.BuySellRatio = calculateBuySellPressureRatio(klines, 20)

	// 计算MACD和RSI序列
	start := len(klines) - 10
	if start < 0 {
		start = 0
	}

	for i := start; i < len(klines); i++ {
		if i >= 25 {
			macdData := calculateMACD(klines[:i+1])
			data.MACDValues = append(data.MACDValues, macdData.MACD)
			data.SignalValues = append(data.SignalValues, macdData.Signal)
			data.HistoValues = append(data.HistoValues, macdData.Histogram)
		}
		if i >= 14 {
			rsi14 := calculateRSI(klines[:i+1], 14)
			data.RSI14Values = append(data.RSI14Values, rsi14)
		}
	}

	// 计算斐波那契回撤位和扩展位（只计算最新值）
	if len(klines) >= 20 {
		// 找出最近20根K线的高低点
		high := klines[len(klines)-20].High
		low := klines[len(klines)-20].Low
		for j := len(klines)-20; j < len(klines); j++ {
			if klines[j].High > high {
				high = klines[j].High
			}
			if klines[j].Low < low {
				low = klines[j].Low
			}
		}
		rangeSize := high - low
		// 回撤位
		data.FibRetrace382 = high - rangeSize*0.382
		data.FibRetrace500 = high - rangeSize*0.500
		data.FibRetrace618 = high - rangeSize*0.618
		// 扩展位
		data.FibExtension1272 = high + rangeSize*0.272
		data.FibExtension1618 = high + rangeSize*0.618
		data.FibExtension2000 = high + rangeSize*1.0
	}

	return data
}

// getOpenInterestData 获取OI数据
func getOpenInterestData(symbol string) (*OIData, error) {
	url := fmt.Sprintf("https://fapi.binance.com/fapi/v1/openInterest?symbol=%s", symbol)

	apiClient := NewAPIClient()
	resp, err := apiClient.client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result struct {
		OpenInterest string `json:"openInterest"`
		Symbol       string `json:"symbol"`
		Time         int64  `json:"time"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	oi, _ := strconv.ParseFloat(result.OpenInterest, 64)

	return &OIData{
		Latest:  oi,
		Average: oi * 0.999, // 近似平均值
	}, nil
}

// getFundingRate 获取资金费率（优化：使用 1 小时缓存）
func getFundingRate(symbol string) (float64, error) {
	// 检查缓存（有效期 1 小时）
	// Funding Rate 每 8 小时才更新，1 小时缓存非常合理
	if cached, ok := fundingRateMap.Load(symbol); ok {
		cache := cached.(*FundingRateCache)
		if time.Since(cache.UpdatedAt) < frCacheTTL {
			// 缓存命中，直接返回
			return cache.Rate, nil
		}
	}

	// 缓存过期或不存在，调用 API
	url := fmt.Sprintf("https://fapi.binance.com/fapi/v1/premiumIndex?symbol=%s", symbol)

	apiClient := NewAPIClient()
	resp, err := apiClient.client.Get(url)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return 0, err
	}

	var result struct {
		Symbol          string `json:"symbol"`
		MarkPrice       string `json:"markPrice"`
		IndexPrice      string `json:"indexPrice"`
		LastFundingRate string `json:"lastFundingRate"`
		NextFundingTime int64  `json:"nextFundingTime"`
		InterestRate    string `json:"interestRate"`
		Time            int64  `json:"time"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return 0, err
	}

	rate, _ := strconv.ParseFloat(result.LastFundingRate, 64)

	// 更新缓存
	fundingRateMap.Store(symbol, &FundingRateCache{
		Rate:      rate,
		UpdatedAt: time.Now(),
	})

	return rate, nil
}

// Format 格式化市场数据为字符串
func Format(data *Data) string {
	var sb strings.Builder

	// 使用动态精度格式化价格
	priceStr := formatPriceWithDynamicPrecision(data.CurrentPrice)
	sb.WriteString(fmt.Sprintf("current_price = %s, 15m_change = %+.2f%%, 1h_change = %+.2f%%, current_ema20 = %.3f, current_macd = %.3f, signal = %.3f, histogram = %.3f, current_rsi (7 period) = %.3f\n\n",
		priceStr, data.PriceChange15m, data.PriceChange1h, data.CurrentEMA20, data.CurrentMACD.MACD, data.CurrentMACD.Signal, data.CurrentMACD.Histogram, data.CurrentRSI7))
	
	// 添加斐波那契OTE区域数据
	if data.FibonacciOTE != nil {
		sb.WriteString("Fibonacci OTE Levels:\n\n")
		sb.WriteString(fmt.Sprintf("38.2%% Retracement: %s\n", formatPriceWithDynamicPrecision(data.FibonacciOTE.Retracement382)))
		sb.WriteString(fmt.Sprintf("50.0%% Retracement: %s\n", formatPriceWithDynamicPrecision(data.FibonacciOTE.Retracement500)))
		sb.WriteString(fmt.Sprintf("61.8%% Retracement: %s\n", formatPriceWithDynamicPrecision(data.FibonacciOTE.Retracement618)))
		sb.WriteString(fmt.Sprintf("127.2%% Extension: %s\n", formatPriceWithDynamicPrecision(data.FibonacciOTE.Extension1272)))
		sb.WriteString(fmt.Sprintf("161.8%% Extension: %s\n", formatPriceWithDynamicPrecision(data.FibonacciOTE.Extension1618)))
		sb.WriteString(fmt.Sprintf("200.0%% Extension: %s\n\n", formatPriceWithDynamicPrecision(data.FibonacciOTE.Extension2000)))
		
		sb.WriteString(fmt.Sprintf("Price Position:\n"))
		sb.WriteString(fmt.Sprintf("Is above 61.8%% retracement: %v\n", data.FibonacciOTE.IsAboveRetrace618))
		sb.WriteString(fmt.Sprintf("Is near 127.2%% extension: %v\n\n", data.FibonacciOTE.IsNearExtension1272))
	}

	sb.WriteString(fmt.Sprintf("In addition, here is the latest %s open interest and funding rate for perps:\n\n",
		data.Symbol))

	if data.OpenInterest != nil {
		// 使用动态精度格式化 OI 数据
		oiLatestStr := formatPriceWithDynamicPrecision(data.OpenInterest.Latest)
		oiAverageStr := formatPriceWithDynamicPrecision(data.OpenInterest.Average)
		sb.WriteString(fmt.Sprintf("Open Interest: Latest: %s Average: %s\n\n",
			oiLatestStr, oiAverageStr))
	}

	sb.WriteString(fmt.Sprintf("Funding Rate: %.2e\n\n", data.FundingRate))

	if data.IntradaySeries != nil {
		sb.WriteString("Intraday series (3‑minute intervals, oldest → latest):\n\n")

		if len(data.IntradaySeries.MidPrices) > 0 {
			sb.WriteString(fmt.Sprintf("Mid prices: %s\n\n", formatFloatSlice(data.IntradaySeries.MidPrices)))
		}

		if len(data.IntradaySeries.EMA20Values) > 0 {
			sb.WriteString(fmt.Sprintf("EMA indicators (20‑period): %s\n\n", formatFloatSlice(data.IntradaySeries.EMA20Values)))
		}

		if len(data.IntradaySeries.MACDValues) > 0 {
			sb.WriteString(fmt.Sprintf("MACD indicators: %s\n\n", formatFloatSlice(data.IntradaySeries.MACDValues)))
		}

		if len(data.IntradaySeries.SignalValues) > 0 {
			sb.WriteString(fmt.Sprintf("Signal indicators: %s\n\n", formatFloatSlice(data.IntradaySeries.SignalValues)))
		}

		if len(data.IntradaySeries.HistoValues) > 0 {
			sb.WriteString(fmt.Sprintf("Histogram indicators: %s\n\n", formatFloatSlice(data.IntradaySeries.HistoValues)))
		}

		if len(data.IntradaySeries.RSI7Values) > 0 {
			sb.WriteString(fmt.Sprintf("RSI indicators (7‑Period): %s\n\n", formatFloatSlice(data.IntradaySeries.RSI7Values)))
		}

		if len(data.IntradaySeries.RSI14Values) > 0 {
			sb.WriteString(fmt.Sprintf("RSI indicators (14‑Period): %s\n\n", formatFloatSlice(data.IntradaySeries.RSI14Values)))
		}

		if len(data.IntradaySeries.Volume) > 0 {
			sb.WriteString(fmt.Sprintf("Volume: %s\n\n", formatFloatSlice(data.IntradaySeries.Volume)))
		}

		// 添加买卖压力比输出
		if len(data.IntradaySeries.BuySellRatios) > 0 {
			sb.WriteString(fmt.Sprintf("Buy/Sell Pressure Ratios (5-period): %s\n\n", formatFloatSlice(data.IntradaySeries.BuySellRatios)))
		}

		sb.WriteString(fmt.Sprintf("3m ATR (14‑period): %.3f\n\n", data.IntradaySeries.ATR14))
	}

	if data.MidTermContext15m != nil {
		sb.WriteString("Mid-term context (15-minute timeframe):\n\n")
		sb.WriteString(data.MidTermContext15m.Format())
	}

	if data.MidTermContext1h != nil {
		sb.WriteString("Mid-term context (1-hour timeframe):\n\n")
		sb.WriteString(data.MidTermContext1h.Format())
	}

	if data.LongerTermContext != nil {
		sb.WriteString("Longer‑term context (4‑hour timeframe):\n\n")
		sb.WriteString(data.LongerTermContext.Format())
	}

	return sb.String()
}

// formatPriceWithDynamicPrecision 根据价格区间动态选择精度
// 这样可以完美支持从超低价 meme coin (< 0.0001) 到 BTC/ETH 的所有币种
func formatPriceWithDynamicPrecision(price float64) string {
	switch {
	case price < 0.0001:
		// 超低价 meme coin: 1000SATS, 1000WHY, DOGS
		// 0.00002070 → "0.00002070" (8位小数)
		return fmt.Sprintf("%.8f", price)
	case price < 0.001:
		// 低价 meme coin: NEIRO, HMSTR, HOT, NOT
		// 0.00015060 → "0.000151" (6位小数)
		return fmt.Sprintf("%.6f", price)
	case price < 0.01:
		// 中低价币: PEPE, SHIB, MEME
		// 0.00556800 → "0.005568" (6位小数)
		return fmt.Sprintf("%.6f", price)
	case price < 1.0:
		// 低价币: ASTER, DOGE, ADA, TRX
		// 0.9954 → "0.9954" (4位小数)
		return fmt.Sprintf("%.4f", price)
	case price < 100:
		// 中价币: SOL, AVAX, LINK, MATIC
		// 23.4567 → "23.4567" (4位小数)
		return fmt.Sprintf("%.4f", price)
	default:
		// 高价币: BTC, ETH (节省 Token)
		// 45678.9123 → "45678.91" (2位小数)
		return fmt.Sprintf("%.2f", price)
	}
}

// formatFloatSlice 格式化float64切片为字符串（使用动态精度）
func formatFloatSlice(values []float64) string {
	strValues := make([]string, len(values))
	for i, v := range values {
		strValues[i] = formatPriceWithDynamicPrecision(v)
	}
	return "[" + strings.Join(strValues, ", ") + "]"
}

// Normalize 标准化symbol,确保是USDT交易对
func Normalize(symbol string) string {
	symbol = strings.ToUpper(symbol)
	if strings.HasSuffix(symbol, "USDT") {
		return symbol
	}
	return symbol + "USDT"
}

// parseFloat 解析float值
func parseFloat(v interface{}) (float64, error) {
	switch val := v.(type) {
	case string:
		return strconv.ParseFloat(val, 64)
	case float64:
		return val, nil
	case int:
		return float64(val), nil
	case int64:
		return float64(val), nil
	default:
		return 0, fmt.Errorf("unsupported type: %T", v)
	}
}
