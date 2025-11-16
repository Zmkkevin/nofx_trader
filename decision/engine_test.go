package decision

import (
	"strings"
	"testing"
)

// TestBuildPromptSnapshot 测试 prompt 快照生成功能
func TestBuildPromptSnapshot(t *testing.T) {
	t.Run("基础模板应该包含硬约束部分", func(t *testing.T) {
		snapshot := BuildPromptSnapshot(
			10000.0, // accountEquity
			5,       // btcEthLeverage
			5,       // altcoinLeverage
			"",      // customPrompt
			false,   // overrideBase
			"default",
		)

		// 验证包含关键部分
		requiredParts := []string{
			"硬约束",
			"风险回报比",
			"最多持仓",
			"杠杆限制",
		}

		for _, part := range requiredParts {
			if !strings.Contains(snapshot, part) {
				t.Errorf("快照应该包含 %q，但没有找到", part)
			}
		}

		// 验证杠杆配置正确
		if !strings.Contains(snapshot, "最大5x杠杆") {
			t.Error("应该包含正确的杠杆配置")
		}
	})

	t.Run("自定义 prompt 应该被追加到基础 prompt", func(t *testing.T) {
		customContent := "我的自定义交易策略：只做多不做空"

		snapshot := BuildPromptSnapshot(
			10000.0,
			5,
			5,
			customContent,
			false, // 不覆盖
			"default",
		)

		// 应该同时包含基础部分和自定义部分
		if !strings.Contains(snapshot, "硬约束") {
			t.Error("应该包含基础 prompt 的硬约束部分")
		}

		if !strings.Contains(snapshot, customContent) {
			t.Errorf("应该包含自定义内容 %q", customContent)
		}

		if !strings.Contains(snapshot, "个性化交易策略") {
			t.Error("应该包含自定义部分的标题")
		}
	})

	t.Run("覆盖模式应该只返回自定义 prompt", func(t *testing.T) {
		customContent := "完全自定义的交易策略"

		snapshot := BuildPromptSnapshot(
			10000.0,
			5,
			5,
			customContent,
			true, // 覆盖基础 prompt
			"default",
		)

		// 应该只包含自定义内容
		if snapshot != customContent {
			t.Errorf("覆盖模式应该返回 %q, 但得到 %q", customContent, snapshot)
		}

		// 不应该包含基础 prompt 的内容
		if strings.Contains(snapshot, "硬约束") {
			t.Error("覆盖模式不应该包含基础 prompt 内容")
		}
	})

	t.Run("空自定义 prompt 应该只返回基础 prompt", func(t *testing.T) {
		snapshot := BuildPromptSnapshot(
			10000.0,
			5,
			5,
			"", // 空自定义
			false,
			"default",
		)

		// 应该包含基础内容
		if !strings.Contains(snapshot, "硬约束") {
			t.Error("应该包含基础 prompt")
		}

		// 不应该包含自定义标记
		if strings.Contains(snapshot, "个性化交易策略") {
			t.Error("空自定义 prompt 不应该有个性化标记")
		}
	})

	t.Run("不同杠杆配置应该正确显示", func(t *testing.T) {
		snapshot := BuildPromptSnapshot(
			20000.0, // 更高的净值
			10,      // BTC/ETH 10x
			3,       // 山寨币 3x
			"",
			false,
			"default",
		)

		// 验证杠杆配置
		if !strings.Contains(snapshot, "最大3x杠杆") {
			t.Error("应该显示山寨币 3x 杠杆")
		}

		if !strings.Contains(snapshot, "最大10x杠杆") {
			t.Error("应该显示 BTC/ETH 10x 杠杆")
		}

		// 验证仓位上限（基于净值计算）
		// 山寨: 20000 * 1.5 = 30000
		// BTC/ETH: 20000 * 10 = 200000
		if !strings.Contains(snapshot, "30000 U") {
			t.Error("应该显示正确的山寨币仓位上限")
		}

		if !strings.Contains(snapshot, "200000 U") {
			t.Error("应该显示正确的 BTC/ETH 仓位上限")
		}
	})

	t.Run("模板不存在应该降级到 default", func(t *testing.T) {
		// 使用不存在的模板名
		snapshot := BuildPromptSnapshot(
			10000.0,
			5,
			5,
			"",
			false,
			"non_existent_template",
		)

		// 应该降级到 default 或内置简化版本
		// 至少应该有基本的提示词结构
		if len(snapshot) < 100 {
			t.Error("即使模板不存在，也应该返回有效的 prompt")
		}
	})

	t.Run("快照应该包含输出格式说明", func(t *testing.T) {
		snapshot := BuildPromptSnapshot(
			10000.0,
			5,
			5,
			"",
			false,
			"default",
		)

		// 验证包含输出格式相关内容
		formatParts := []string{
			"输出格式",
			"<reasoning>",
			"<decision>",
		}

		for _, part := range formatParts {
			if !strings.Contains(snapshot, part) {
				t.Errorf("应该包含输出格式说明 %q", part)
			}
		}
	})
}

// TestBuildPromptSnapshotConsistency 测试快照的一致性
// 相同参数应该生成相同的快照
func TestBuildPromptSnapshotConsistency(t *testing.T) {
	params := struct {
		equity         float64
		btcEthLeverage int
		altLeverage    int
		custom         string
		override       bool
		template       string
	}{
		equity:         15000.0,
		btcEthLeverage: 8,
		altLeverage:    4,
		custom:         "我的策略",
		override:       false,
		template:       "default",
	}

	// 生成两次快照
	snapshot1 := BuildPromptSnapshot(
		params.equity,
		params.btcEthLeverage,
		params.altLeverage,
		params.custom,
		params.override,
		params.template,
	)

	snapshot2 := BuildPromptSnapshot(
		params.equity,
		params.btcEthLeverage,
		params.altLeverage,
		params.custom,
		params.override,
		params.template,
	)

	// 应该完全一致
	if snapshot1 != snapshot2 {
		t.Error("相同参数生成的快照应该一致")
	}
}
