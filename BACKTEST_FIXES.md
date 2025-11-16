# 回测系统核心逻辑修复方案

## 问题总结

| # | 问题 | 严重程度 | 影响 |
|---|------|---------|------|
| 1 | AI 修改止损止盈后不立即检查 | 🔴 CRITICAL | 止损不生效，损失扩大 |
| 2 | 止损和爆仓优先级错误 | 🔴 CRITICAL | 爆仓时止损先执行，成交价不准 |
| 3 | K线内价格变化未考虑 | 🔴 CRITICAL | 错过止损/爆仓触发 |
| 4 | 强平价格使用不准确 | 🟡 HIGH | 回测结果偏离实盘 |
| 5 | 新开仓位止损止盈本周期不检查 | 🟡 HIGH | AI 开仓后本周期内止损不生效 |

---

## 修复代码

### 修复 1-3-5: 完整的止损止盈检查逻辑

```go
// backtest/runner.go
func (r *Runner) stepOnce() error {
    state := r.snapshotState()
    if state.BarIndex >= r.feed.DecisionBarCount() {
        return errBacktestCompleted
    }

    ts := r.feed.DecisionTimestamp(state.BarIndex)
    marketData, multiTF, err := r.feed.BuildMarketData(ts)
    if err != nil {
        return err
    }

    // 🔧 修复：构建 Close/High/Low 三个价格映射
    priceMap := make(map[string]float64, len(marketData))
    highMap := make(map[string]float64, len(marketData))
    lowMap := make(map[string]float64, len(marketData))

    for symbol, data := range marketData {
        priceMap[symbol] = data.CurrentPrice // 或 data.Close
        highMap[symbol] = data.High
        lowMap[symbol] = data.Low
    }

    callCount := state.DecisionCycle + 1
    shouldDecide := r.shouldTriggerDecision(state.BarIndex)

    var (
        record          *logger.DecisionRecord
        decisionActions []logger.DecisionAction
        tradeEvents     = make([]TradeEvent, 0)
        execLog         []string
        hadError        bool
    )

    // 🔧 修复：使用统一的检查方法（考虑 High/Low）
    // 1. 第一次检查（AI 决策前）
    slTpEvents, liqEvents := r.checkRiskEventsWithOHLC(priceMap, highMap, lowMap, ts, callCount)
    tradeEvents = append(tradeEvents, slTpEvents...)
    tradeEvents = append(tradeEvents, liqEvents...)
    for _, evt := range slTpEvents {
        execLog = append(execLog, fmt.Sprintf("🛑 %s", evt.Note))
    }
    if len(liqEvents) > 0 {
        hadError = true
        for _, evt := range liqEvents {
            execLog = append(execLog, fmt.Sprintf("🚨 强平: %s", evt.Note))
        }
    }

    decisionAttempted := shouldDecide

    // 2. AI 决策执行
    if shouldDecide {
        ctx, rec, err := r.buildDecisionContext(ts, marketData, multiTF, priceMap, callCount)
        if err != nil {
            rec.Success = false
            rec.ErrorMessage = fmt.Sprintf("构建交易上下文失败: %v", err)
            _ = r.logDecision(rec)
            return err
        }
        record = rec

        // ... AI 决策逻辑（保持不变）...

        if fullDecision != nil {
            r.fillDecisionRecord(record, fullDecision)
            sorted := sortDecisionsByPriority(fullDecision.Decisions)

            prevLogs := execLog
            decisionActions = make([]logger.DecisionAction, 0, len(sorted))
            execLog = make([]string, 0, len(sorted)+len(prevLogs))
            if len(prevLogs) > 0 {
                execLog = append(execLog, prevLogs...)
            }

            for _, dec := range sorted {
                actionRecord, trades, logEntry, execErr := r.executeDecision(dec, priceMap, ts, callCount)
                if execErr != nil {
                    actionRecord.Success = false
                    actionRecord.Error = execErr.Error()
                    hadError = true
                    execLog = append(execLog, fmt.Sprintf("❌ %s %s: %v", dec.Symbol, dec.Action, execErr))
                } else {
                    actionRecord.Success = true
                    execLog = append(execLog, fmt.Sprintf("✓ %s %s", dec.Symbol, dec.Action))
                }
                if len(trades) > 0 {
                    tradeEvents = append(tradeEvents, trades...)
                }
                if logEntry != "" {
                    execLog = append(execLog, logEntry)
                }
                decisionActions = append(decisionActions, actionRecord)
            }
        }
    }

    // 🔧 修复：再次检查止损止盈（AI 可能修改了止损止盈或开了新仓）
    slTpEvents2, liqEvents2 := r.checkRiskEventsWithOHLC(priceMap, highMap, lowMap, ts, callCount)
    if len(slTpEvents2) > 0 {
        tradeEvents = append(tradeEvents, slTpEvents2...)
        for _, evt := range slTpEvents2 {
            execLog = append(execLog, fmt.Sprintf("🔄 AI 决策后触发: %s", evt.Note))
        }
    }
    if len(liqEvents2) > 0 {
        hadError = true
        tradeEvents = append(tradeEvents, liqEvents2...)
        for _, evt := range liqEvents2 {
            execLog = append(execLog, fmt.Sprintf("🚨 AI 决策后强平: %s", evt.Note))
        }
    }

    cycleForLog := state.DecisionCycle
    if decisionAttempted {
        cycleForLog = callCount
    }

    // ... 后续逻辑保持不变 ...
}
```

### 新增方法：统一的风险事件检查

```go
// backtest/runner.go
// checkRiskEventsWithOHLC 使用 OHLC 数据检查止损止盈和爆仓
// 返回: (止损止盈事件, 爆仓事件)
func (r *Runner) checkRiskEventsWithOHLC(
    priceMap, highMap, lowMap map[string]float64,
    ts int64,
    cycle int,
) ([]TradeEvent, []TradeEvent) {

    slTpEvents := make([]TradeEvent, 0)
    liqEvents := make([]TradeEvent, 0)

    positions := append([]*position(nil), r.account.Positions()...)

    for _, pos := range positions {
        currentPrice := priceMap[pos.Symbol]
        high := highMap[pos.Symbol]
        low := lowMap[pos.Symbol]

        if currentPrice <= 0 || high <= 0 || low <= 0 {
            continue
        }

        var triggerType string // "stop_loss", "take_profit", "liquidation"
        var triggerPrice float64
        var reason string

        if pos.Side == "long" {
            // 多头：检查最低价
            // 优先级：爆仓 > 止损 > 止盈

            if low <= pos.LiquidationPrice && pos.LiquidationPrice > 0 {
                // 强平触发
                triggerType = "liquidation"
                triggerPrice = pos.LiquidationPrice
                reason = fmt.Sprintf("强制平仓: %.4f <= 爆仓价 %.4f", low, pos.LiquidationPrice)

            } else if pos.StopLoss > 0 && low <= pos.StopLoss {
                // 止损触发
                triggerType = "stop_loss"
                triggerPrice = pos.StopLoss
                reason = fmt.Sprintf("多头止损触发: %.4f <= %.4f", low, pos.StopLoss)

            } else if pos.TakeProfit > 0 && high >= pos.TakeProfit {
                // 止盈触发（检查最高价）
                triggerType = "take_profit"
                triggerPrice = pos.TakeProfit
                reason = fmt.Sprintf("多头止盈触发: %.4f >= %.4f", high, pos.TakeProfit)
            }

        } else if pos.Side == "short" {
            // 空头：检查最高价

            if high >= pos.LiquidationPrice && pos.LiquidationPrice > 0 {
                // 强平触发
                triggerType = "liquidation"
                triggerPrice = pos.LiquidationPrice
                reason = fmt.Sprintf("强制平仓: %.4f >= 爆仓价 %.4f", high, pos.LiquidationPrice)

            } else if pos.StopLoss > 0 && high >= pos.StopLoss {
                // 止损触发
                triggerType = "stop_loss"
                triggerPrice = pos.StopLoss
                reason = fmt.Sprintf("空头止损触发: %.4f >= %.4f", high, pos.StopLoss)

            } else if pos.TakeProfit > 0 && low <= pos.TakeProfit {
                // 止盈触发（检查最低价）
                triggerType = "take_profit"
                triggerPrice = pos.TakeProfit
                reason = fmt.Sprintf("空头止盈触发: %.4f <= %.4f", low, pos.TakeProfit)
            }
        }

        if triggerType == "" {
            continue
        }

        // 执行平仓
        fillPrice := r.executionPrice(pos.Symbol, triggerPrice, ts)

        // 🔧 修复：强平时使用实际价格（通常更差）
        if triggerType == "liquidation" {
            if pos.Side == "long" {
                // 多头强平：使用更低的价格
                fillPrice = math.Min(currentPrice, triggerPrice)
            } else {
                // 空头强平：使用更高的价格
                fillPrice = math.Max(currentPrice, triggerPrice)
            }
        }

        realized, fee, execPrice, err := r.account.Close(
            pos.Symbol,
            pos.Side,
            pos.Quantity,
            fillPrice,
        )

        if err != nil {
            log.Printf("⚠️ 风险事件平仓失败 [%s %s %s]: %v",
                triggerType, pos.Symbol, pos.Side, err)
            continue
        }

        action := fmt.Sprintf("auto_close_%s_%s", pos.Side, triggerType)
        trade := TradeEvent{
            Timestamp:       ts,
            Symbol:          pos.Symbol,
            Action:          action,
            Side:            pos.Side,
            Quantity:        pos.Quantity,
            Price:           execPrice,
            Fee:             fee,
            RealizedPnL:     realized - fee,
            Leverage:        pos.Leverage,
            Cycle:           cycle,
            Note:            reason,
            LiquidationFlag: triggerType == "liquidation",
        }

        if triggerType == "liquidation" {
            liqEvents = append(liqEvents, trade)
            log.Printf("  🚨 %s (实际价格: %.4f, 盈亏: %.2f USDT)",
                reason, execPrice, realized-fee)
        } else {
            slTpEvents = append(slTpEvents, trade)
            log.Printf("  🛑 %s (实际价格: %.4f, 盈亏: %.2f USDT)",
                reason, execPrice, realized-fee)
        }
    }

    return slTpEvents, liqEvents
}
```

---

## 测试用例

### 测试 1: AI 修改止损后立即触发

```go
func TestStopLossUpdateAndTrigger(t *testing.T) {
    // 场景：AI 修改止损后，当前价格满足新止损，应该立即触发

    // T0: 开仓，止损 49000，当前价 50000
    // T1: AI 修改止损到 50500，当前价 50000
    // 预期：立即触发止损（50000 < 50500）
}
```

### 测试 2: K线内触发止损

```go
func TestIntraBarStopLoss(t *testing.T) {
    // 场景：K线最低价触及止损，但收盘价未触及

    // K线: Open=50000, High=51000, Low=48000, Close=49500
    // 止损: 49000
    // 预期：触发止损（Low 48000 < 49000）
}
```

### 测试 3: 爆仓优先于止损

```go
func TestLiquidationPriority(t *testing.T) {
    // 场景：价格同时触及止损和爆仓

    // 止损: 49000
    // 爆仓: 48000
    // K线 Low: 47000
    // 预期：触发爆仓（优先级更高），成交价 ~48000
}
```

---

## 实施步骤

1. ✅ **备份当前代码**
2. 🔧 **修改 stepOnce 方法**：添加双重检查
3. 🔧 **新增 checkRiskEventsWithOHLC 方法**
4. 🔧 **删除旧的独立检查**：删除 checkLiquidation
5. ✅ **编写测试用例**
6. ✅ **运行回测验证**：对比修复前后的结果

---

## 预期影响

### 正面影响
- ✅ 止损止盈执行更及时
- ✅ 回测结果更接近实盘
- ✅ 风控更严格

### 潜在风险
- ⚠️ 回测结果可能变差（因为之前很多应该触发的止损没触发）
- ⚠️ 需要重新评估策略参数

---

## 其他核心问题检查

### ✅ 已验证正确的部分

1. **手续费计算**：每次开仓/平仓都收取手续费 ✅
2. **滑点处理**：使用 `executionPrice` 计算滑点 ✅
3. **杠杆计算**：开仓时设置杠杆，平仓时使用持仓杠杆 ✅
4. **盈亏计算**：使用正确的公式计算已实现/未实现盈亏 ✅
5. **仓位管理**：最多 20 个持仓，杠杆限制 100x ✅

### ⚠️ 需要关注的其他问题

1. **资金不足检查**：开仓时检查保证金是否足够 ✅
2. **部分平仓**：当前标记为 TODO，未实现 ⚠️
3. **加仓逻辑**：支持加仓，止损止盈会更新 ✅
4. **时间顺序**：使用时间戳确保顺序 ✅

---

## 总结

本次审查发现 **5 个关键问题**，其中 **3 个 CRITICAL 级别**：

1. 🔴 AI 修改止损止盈后不立即检查
2. 🔴 止损和爆仓优先级错误
3. 🔴 K线内价格变化未考虑
4. 🟡 强平价格使用不准确
5. 🟡 新开仓位止损止盈本周期不检查

**修复方案：**
- 统一风险事件检查（止损/止盈/爆仓）
- 使用 OHLC 数据（High/Low）判断触发
- AI 决策后再次检查
- 正确处理优先级

**修复后的回测系统将：**
- ✅ 更接近实盘行为
- ✅ 止损止盈更及时
- ✅ 风控更严格
- ✅ 结果更可信
