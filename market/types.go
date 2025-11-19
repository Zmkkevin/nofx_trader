package market

import (
	"fmt"
	"strings"
	"time"
)

// MACDData MACD数据结构
type MACDData struct {
	MACD      float64 // MACD线（快线）
	Signal    float64 // 信号线（慢线）
	Histogram float64 // 柱状图
}

// FibonacciOTE 斐波那契OTE区域数据结构
type FibonacciOTE struct {
	Retracement382      float64 // 38.2% 回撤位
	Retracement500      float64 // 50% 回撤位
	Retracement618      float64 // 61.8% 回撤位
	Extension1272       float64 // 127.2% 扩展位
	Extension1618       float64 // 161.8% 扩展位
	Extension2000       float64 // 200% 扩展位
	IsAboveRetrace618   bool    // 价格是否在61.8%回撤位以上
	IsNearExtension1272 bool    // 价格是否接近127.2%扩展位（±1%）
}

// Data 市场数据结构
type Data struct {
	Symbol            string
	CurrentPrice      float64
	PriceChange15m    float64 // 15分钟价格变化百分比
	PriceChange1h     float64 // 1小时价格变化百分比
	PriceChange4h     float64 // 4小时价格变化百分比
	CurrentEMA20      float64
	CurrentMACD       MACDData // 完整MACD数据
	CurrentRSI7       float64
	OpenInterest      *OIData
	FundingRate       float64
	FibonacciOTE      *FibonacciOTE // 斐波那契OTE区域
	IntradaySeries    *IntradayData
	MidTermContext15m *MidTermData // 15分钟时间框架数据
	MidTermContext1h  *MidTermData // 1小时时间框架数据
	LongerTermContext *LongerTermData
}

// OIData Open Interest数据
type OIData struct {
	Latest  float64
	Average float64
}

// IntradayData 日内数据(3分钟间隔)
type IntradayData struct {
	MidPrices     []float64
	EMA20Values   []float64
	MACDValues    []float64 // MACD线值
	SignalValues  []float64 // 信号线值
	HistoValues   []float64 // 柱状图值
	RSI7Values    []float64
	RSI14Values   []float64
	FibRetrace382 []float64 // 38.2%回撤位序列
	FibRetrace500 []float64 // 50%回撤位序列
	FibRetrace618 []float64 // 61.8%回撤位序列
	Volume        []float64
	BuySellRatios []float64 // 买卖压力比
	ATR14         float64
}

// MidTermData 中期数据(15分钟和1小时时间框架)
type MidTermData struct {
	Timeframe        string // 时间框架标识("15m_1h")
	EMA20            float64
	ATR14            float64
	CurrentVolume    float64
	AverageVolume    float64
	BuySellRatio     float64   // 买卖压力比
	MACDValues       []float64 // MACD线值
	SignalValues     []float64 // 信号线值
	HistoValues      []float64 // 柱状图值
	RSI7Values       []float64 // RSI7值
	RSI14Values      []float64 // RSI14值
	FibRetrace382    float64   // 38.2%回撤位（最新值）
	FibRetrace500    float64   // 50%回撤位（最新值）
	FibRetrace618    float64   // 61.8%回撤位（最新值）
	FibExtension1272 float64   // 127.2%扩展位（最新值）
	FibExtension1618 float64   // 161.8%扩展位（最新值）
	FibExtension2000 float64   // 200%扩展位（最新值）
}

// Format 格式化中期数据为字符串
func (m *MidTermData) Format() string {
	if m == nil {
		return ""
	}

	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("Timeframe: %s\n", m.Timeframe))
	sb.WriteString(fmt.Sprintf("20-Period EMA: %.3f\n", m.EMA20))
	sb.WriteString(fmt.Sprintf("14-Period ATR: %.3f\n", m.ATR14))
	sb.WriteString(fmt.Sprintf("Current Volume: %.3f vs. Average Volume: %.3f\n", m.CurrentVolume, m.AverageVolume))
	sb.WriteString(fmt.Sprintf("Buy/Sell Pressure Ratio (20-period): %.3f\n\n", m.BuySellRatio))

	if len(m.MACDValues) > 0 {
		sb.WriteString(fmt.Sprintf("MACD indicators: %s\n\n", formatFloatSlice(m.MACDValues)))
	}

	if len(m.SignalValues) > 0 {
		sb.WriteString(fmt.Sprintf("Signal indicators: %s\n\n", formatFloatSlice(m.SignalValues)))
	}

	if len(m.HistoValues) > 0 {
		sb.WriteString(fmt.Sprintf("Histogram indicators: %s\n\n", formatFloatSlice(m.HistoValues)))
	}

	if len(m.RSI7Values) > 0 {
		sb.WriteString(fmt.Sprintf("RSI indicators (7-Period): %s\n\n", formatFloatSlice(m.RSI7Values)))
	}

	if len(m.RSI14Values) > 0 {
		sb.WriteString(fmt.Sprintf("RSI indicators (14-Period): %s\n\n", formatFloatSlice(m.RSI14Values)))
	}

	// 添加斐波那契回撤和扩展数据
	if m.FibRetrace382 > 0 {
		sb.WriteString(fmt.Sprintf("Fibonacci 38.2%% Retracement: %s\n", formatPriceWithDynamicPrecision(m.FibRetrace382)))
		sb.WriteString(fmt.Sprintf("Fibonacci 50.0%% Retracement: %s\n", formatPriceWithDynamicPrecision(m.FibRetrace500)))
		sb.WriteString(fmt.Sprintf("Fibonacci 61.8%% Retracement: %s\n", formatPriceWithDynamicPrecision(m.FibRetrace618)))
		sb.WriteString(fmt.Sprintf("Fibonacci 127.2%% Extension: %s\n", formatPriceWithDynamicPrecision(m.FibExtension1272)))
		sb.WriteString(fmt.Sprintf("Fibonacci 161.8%% Extension: %s\n", formatPriceWithDynamicPrecision(m.FibExtension1618)))
		sb.WriteString(fmt.Sprintf("Fibonacci 200.0%% Extension: %s\n\n", formatPriceWithDynamicPrecision(m.FibExtension2000)))
	}

	return sb.String()
}

// LongerTermData 长期数据(4小时时间框架)
type LongerTermData struct {
	EMA20            float64
	EMA50            float64
	ATR3             float64
	ATR14            float64
	CurrentVolume    float64
	AverageVolume    float64
	BuySellRatio     float64   // 买卖压力比
	MACDValues       []float64 // MACD线值
	SignalValues     []float64 // 信号线值
	HistoValues      []float64 // 柱状图值
	RSI14Values      []float64
	FibRetrace382    float64 // 38.2%回撤位（最新值）
	FibRetrace500    float64 // 50%回撤位（最新值）
	FibRetrace618    float64 // 61.8%回撤位（最新值）
	FibExtension1272 float64 // 127.2%扩展位（最新值）
	FibExtension1618 float64 // 161.8%扩展位（最新值）
	FibExtension2000 float64 // 200%扩展位（最新值）
}

// Format 格式化长期数据为字符串
func (l *LongerTermData) Format() string {
	if l == nil {
		return ""
	}

	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("20-Period EMA: %.3f vs. 50-Period EMA: %.3f\n", l.EMA20, l.EMA50))
	sb.WriteString(fmt.Sprintf("3-Period ATR: %.3f vs. 14-Period ATR: %.3f\n", l.ATR3, l.ATR14))
	sb.WriteString(fmt.Sprintf("Current Volume: %.3f vs. Average Volume: %.3f\n", l.CurrentVolume, l.AverageVolume))
	sb.WriteString(fmt.Sprintf("Buy/Sell Pressure Ratio (20-period): %.3f\n\n", l.BuySellRatio))

	if len(l.MACDValues) > 0 {
		sb.WriteString(fmt.Sprintf("MACD indicators: %s\n\n", formatFloatSlice(l.MACDValues)))
	}

	if len(l.SignalValues) > 0 {
		sb.WriteString(fmt.Sprintf("Signal indicators: %s\n\n", formatFloatSlice(l.SignalValues)))
	}

	if len(l.HistoValues) > 0 {
		sb.WriteString(fmt.Sprintf("Histogram indicators: %s\n\n", formatFloatSlice(l.HistoValues)))
	}

	if len(l.RSI14Values) > 0 {
		sb.WriteString(fmt.Sprintf("RSI indicators (14-Period): %s\n\n", formatFloatSlice(l.RSI14Values)))
	}

	// 添加斐波那契回撤和扩展数据
	if l.FibRetrace382 > 0 {
		sb.WriteString(fmt.Sprintf("Fibonacci 38.2%% Retracement: %s\n", formatPriceWithDynamicPrecision(l.FibRetrace382)))
		sb.WriteString(fmt.Sprintf("Fibonacci 50.0%% Retracement: %s\n", formatPriceWithDynamicPrecision(l.FibRetrace500)))
		sb.WriteString(fmt.Sprintf("Fibonacci 61.8%% Retracement: %s\n", formatPriceWithDynamicPrecision(l.FibRetrace618)))
		sb.WriteString(fmt.Sprintf("Fibonacci 127.2%% Extension: %s\n", formatPriceWithDynamicPrecision(l.FibExtension1272)))
		sb.WriteString(fmt.Sprintf("Fibonacci 161.8%% Extension: %s\n", formatPriceWithDynamicPrecision(l.FibExtension1618)))
		sb.WriteString(fmt.Sprintf("Fibonacci 200.0%% Extension: %s\n\n", formatPriceWithDynamicPrecision(l.FibExtension2000)))
	}

	return sb.String()
}

// Binance API 响应结构
type ExchangeInfo struct {
	Symbols []SymbolInfo `json:"symbols"`
}

type SymbolInfo struct {
	Symbol            string `json:"symbol"`
	Status            string `json:"status"`
	BaseAsset         string `json:"baseAsset"`
	QuoteAsset        string `json:"quoteAsset"`
	ContractType      string `json:"contractType"`
	PricePrecision    int    `json:"pricePrecision"`
	QuantityPrecision int    `json:"quantityPrecision"`
}

type Kline struct {
	OpenTime            int64   `json:"openTime"`
	Open                float64 `json:"open"`
	High                float64 `json:"high"`
	Low                 float64 `json:"low"`
	Close               float64 `json:"close"`
	Volume              float64 `json:"volume"`
	CloseTime           int64   `json:"closeTime"`
	QuoteVolume         float64 `json:"quoteVolume"`
	Trades              int     `json:"trades"`
	TakerBuyBaseVolume  float64 `json:"takerBuyBaseVolume"`
	TakerBuyQuoteVolume float64 `json:"takerBuyQuoteVolume"`
}

type KlineResponse []interface{}

type PriceTicker struct {
	Symbol string `json:"symbol"`
	Price  string `json:"price"`
}

type Ticker24hr struct {
	Symbol             string `json:"symbol"`
	PriceChange        string `json:"priceChange"`
	PriceChangePercent string `json:"priceChangePercent"`
	Volume             string `json:"volume"`
	QuoteVolume        string `json:"quoteVolume"`
}

// 特征数据结构
type SymbolFeatures struct {
	Symbol           string    `json:"symbol"`
	Timestamp        time.Time `json:"timestamp"`
	Price            float64   `json:"price"`
	PriceChange15Min float64   `json:"price_change_15min"`
	PriceChange1H    float64   `json:"price_change_1h"`
	PriceChange4H    float64   `json:"price_change_4h"`
	Volume           float64   `json:"volume"`
	VolumeRatio5     float64   `json:"volume_ratio_5"`
	VolumeRatio20    float64   `json:"volume_ratio_20"`
	VolumeTrend      float64   `json:"volume_trend"`
	RSI14            float64   `json:"rsi_14"`
	SMA5             float64   `json:"sma_5"`
	SMA10            float64   `json:"sma_10"`
	SMA20            float64   `json:"sma_20"`
	HighLowRatio     float64   `json:"high_low_ratio"`
	Volatility20     float64   `json:"volatility_20"`
	PositionInRange  float64   `json:"position_in_range"`
}

// 警报数据结构
type Alert struct {
	Type      string    `json:"type"`
	Symbol    string    `json:"symbol"`
	Value     float64   `json:"value"`
	Threshold float64   `json:"threshold"`
	Message   string    `json:"message"`
	Timestamp time.Time `json:"timestamp"`
}

type Config struct {
	AlertThresholds AlertThresholds `json:"alert_thresholds"`
	UpdateInterval  int             `json:"update_interval"` // seconds
	CleanupConfig   CleanupConfig   `json:"cleanup_config"`
}

type AlertThresholds struct {
	VolumeSpike      float64 `json:"volume_spike"`
	PriceChange15Min float64 `json:"price_change_15min"`
	VolumeTrend      float64 `json:"volume_trend"`
	RSIOverbought    float64 `json:"rsi_overbought"`
	RSIOversold      float64 `json:"rsi_oversold"`
}
type CleanupConfig struct {
	InactiveTimeout   time.Duration `json:"inactive_timeout"`    // 不活跃超时时间
	MinScoreThreshold float64       `json:"min_score_threshold"` // 最低评分阈值
	NoAlertTimeout    time.Duration `json:"no_alert_timeout"`    // 无警报超时时间
	CheckInterval     time.Duration `json:"check_interval"`      // 检查间隔
}

var config = Config{
	AlertThresholds: AlertThresholds{
		VolumeSpike:      3.0,
		PriceChange15Min: 0.05,
		VolumeTrend:      2.0,
		RSIOverbought:    70,
		RSIOversold:      30,
	},
	CleanupConfig: CleanupConfig{
		InactiveTimeout:   30 * time.Minute,
		MinScoreThreshold: 15.0,
		NoAlertTimeout:    20 * time.Minute,
		CheckInterval:     5 * time.Minute,
	},
	UpdateInterval: 60, // 1 minute
}
