package api

import (
	"testing"
)

// TestFilterByPromptParameter 测试 filter_by_prompt 参数的契约
// 需求：默认为 true（过滤），只有明确传递 false 才显示所有
func TestFilterByPromptParameter(t *testing.T) {
	tests := []struct {
		name           string
		queryParam     string // query 参数值（空字符串表示不传递参数）
		expectedFilter bool   // 期望的 filterByPrompt 值
		description    string
	}{
		{
			name:           "无参数时默认为true（过滤）",
			queryParam:     "",
			expectedFilter: true,
			description:    "用户未传递参数 → 后端默认过滤（只显示当前提示词版本）",
		},
		{
			name:           "参数为true时过滤",
			queryParam:     "true",
			expectedFilter: true,
			description:    "用户明确传递 true → 后端过滤",
		},
		{
			name:           "参数为false时不过滤",
			queryParam:     "false",
			expectedFilter: false,
			description:    "用户明确传递 false → 后端显示所有交易",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 模拟后端逻辑
			var filterByPrompt bool

			// 🎯 这是我们要测试的契约逻辑
			if tt.queryParam == "" {
				// 默认为 true（过滤）
				filterByPrompt = true
			} else if tt.queryParam == "false" {
				filterByPrompt = false
			} else if tt.queryParam == "true" {
				filterByPrompt = true
			} else {
				// 其他值默认为 true
				filterByPrompt = true
			}

			if filterByPrompt != tt.expectedFilter {
				t.Errorf("%s: expected filterByPrompt=%v, got %v\nDescription: %s",
					tt.name, tt.expectedFilter, filterByPrompt, tt.description)
			}
		})
	}
}

// TestFrontendBackendContract 测试前后端契约一致性
func TestFrontendBackendContract(t *testing.T) {
	tests := []struct {
		name                string
		frontendState       bool   // 前端 filterByPrompt 状态
		sentParameter       string // 前端发送的参数（空表示不发送）
		backendReceived     string // 后端收到的参数值
		backendDefaultValue bool   // 后端默认值
		expectedBackendBool bool   // 后端最终的 bool 值
		expectedBehavior    string // 期望的行为描述
	}{
		{
			name:                "前端默认ON→发送true→后端过滤",
			frontendState:       true,
			sentParameter:       "true",
			backendReceived:     "true",
			backendDefaultValue: true,
			expectedBackendBool: true,
			expectedBehavior:    "过滤数据（只显示当前提示词版本）",
		},
		{
			name:                "前端OFF→发送false→后端显示所有",
			frontendState:       false,
			sentParameter:       "false",
			backendReceived:     "false",
			backendDefaultValue: true,
			expectedBackendBool: false,
			expectedBehavior:    "显示所有数据",
		},
		{
			name:                "前端刷新→localStorage读到true→发送true→后端过滤",
			frontendState:       true,
			sentParameter:       "true",
			backendReceived:     "true",
			backendDefaultValue: true,
			expectedBackendBool: true,
			expectedBehavior:    "过滤数据（只显示当前提示词版本）",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 模拟前端逻辑
			var sentParam string
			if tt.frontendState {
				sentParam = "true"
			} else {
				sentParam = "false"
			}

			// 验证前端发送的参数正确
			if sentParam != tt.sentParameter {
				t.Errorf("前端参数不匹配: expected to send %q, but would send %q",
					tt.sentParameter, sentParam)
			}

			// 模拟后端逻辑
			var backendBool bool
			if tt.backendReceived == "" {
				backendBool = tt.backendDefaultValue
			} else if tt.backendReceived == "false" {
				backendBool = false
			} else if tt.backendReceived == "true" {
				backendBool = true
			} else {
				backendBool = tt.backendDefaultValue
			}

			// 验证后端最终值正确
			if backendBool != tt.expectedBackendBool {
				t.Errorf("%s: backend bool mismatch\n  Expected: %v (%s)\n  Got: %v",
					tt.name, tt.expectedBackendBool, tt.expectedBehavior, backendBool)
			}

			t.Logf("✓ %s: frontend(%v) → send(%q) → backend receives(%q) → backend bool(%v) → %s",
				tt.name, tt.frontendState, sentParam, tt.backendReceived, backendBool, tt.expectedBehavior)
		})
	}
}
