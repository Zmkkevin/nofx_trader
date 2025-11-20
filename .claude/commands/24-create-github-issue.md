---
name: create-github-issue
description: 快速创建 GitHub Issue
---

# 创建 GitHub Issue

## 任务

根据用户提供的信息，创建一个清晰的 GitHub Issue。

## 执行步骤

### 1. 理解需求
如果用户已经提供了标题和描述，直接使用。如果信息不完整，使用 AskUserQuestion 询问：
- Issue 标题
- 详细描述

### 2. 分析上下文（可选）
如果 Issue 涉及现有代码：
- 快速搜索相关代码（Grep）
- 检查当前实现状态
- 在 Issue 中引用关键文件位置（`file.ts:123` 格式）

### 3. 创建 Issue
使用 `gh issue create` 命令（创建到 nofxai/nofx repo）：

```bash
gh issue create --repo nofxai/nofx --title "标题" --body "$(cat <<'EOF'
内容
EOF
)"
```

### 4. 返回结果
告诉用户 Issue URL

## Issue 内容结构

根据情况选择合适的结构：

**Bug 类型：**
```markdown
## 问题描述
[描述问题]

## 复现步骤
1. ...
2. ...

## 期望行为 vs 实际行为
- 期望：...
- 实际：...

## 相关文件
- `path/to/file:123`
```

**功能/增强类型：**
```markdown
## 当前状态
✅/❌ 已实现/未实现

[当前情况说明]

## 建议实现/改进
- 改进点 1
- 改进点 2

## 验收标准
- [ ] 标准 1
- [ ] 标准 2

## 相关文件
- `path/to/file:123`
```

## 原则

1. **简洁明了** - 标题和内容都要一目了然
2. **提供上下文** - 包含必要的代码引用
3. **可操作** - 让开发者能直接着手处理
4. **不过度设计** - 只包含必要信息

## 示例

**用户：** 创建 issue：回测缺少 prompt 信息展示

**助手执行：**
1. 搜索回测相关代码，了解当前实现
2. 创建 Issue，包含：
   - 清晰的标题
   - 问题说明
   - 当前实现分析
   - 建议改进方案
   - 相关文件引用
3. 返回 Issue URL
