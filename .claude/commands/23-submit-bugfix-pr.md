---
name: submit-bugfix-pr
description: 提交 bugfix 到新分支并创建 PR（带 @claude 评审）
---

# 提交 Bugfix 并创建 PR

## 任务
将当前的 bugfix 提交到新分支，并创建 PR 到 dev 分支，自动触发 Claude Code Review。

## 使用场景
- 当前在 local/feature 分支，有其他未合并的改动
- 需要将特定的 bugfix 单独提交到 dev
- 希望通过 cherry-pick 提取干净的 commit

## 工作流程

### 1. 准备阶段
- **检查当前状态**：
  - 运行 `git status` 查看当前改动
  - 确认是否有未提交的修复代码
  - 确认当前分支名称
- **从上下文提取信息**：
  - Issue number（从之前的 /21 命令上下文或 git commit 中提取）
  - Commit message（从最近的 commit 或修复内容自动生成）
  - Bugfix 分支名称（根据 Issue 自动生成：bugfix/issue-{number}-{简短描述}）
- **只在必要时询问用户**：
  - 如果无法从上下文提取 Issue number，才询问
  - PR description 的补充说明（可选）

### 2. 提交修复代码（如果需要）
```bash
# 如果有未提交的改动，先提交
git add <files>
git commit -m "fix: <message>"
```

### 3. 保存当前 commit hash
```bash
# 记录当前 commit
COMMIT_HASH=$(git rev-parse HEAD)
echo "Bugfix commit: $COMMIT_HASH"
```

### 4. 创建 bugfix 分支（基于 fork 的 dev）
```bash
# 重要：从 origin/dev 创建分支（你的 fork）
# 因为 PR 是到 fork 的 dev，所以要基于 origin/dev

# 1. 更新 origin/dev
git fetch origin

# 2. 从 origin/dev 创建 bugfix 分支
git checkout -b bugfix/issue-{number}-{description} origin/dev
```

### 5. Cherry-pick 修复 commit
```bash
# 应用修复 commit
git cherry-pick $COMMIT_HASH

# 如果有冲突，提示用户解决
# git cherry-pick --continue 或 --abort
```

### 6. 推送到远程（你的 fork）
```bash
# 推送新分支到你的 fork (origin = xqliu/nofx)
git push -u origin bugfix/issue-{number}-{description}
```

### 7. 创建 PR 到你的 fork（带 Claude 评审触发）
```bash
# 重要：创建 PR 到你的 fork (xqliu/nofx)，不是 upstream
# 这样可以在自己的仓库中先测试和审查

gh pr create \
  --repo xqliu/nofx \
  --base dev \
  --head bugfix/issue-{number}-{description} \
  --title "fix: {title}" \
  --body "$(cat <<'EOF'
## 问题描述
Fixes #{issue-number}

{problem-description}

## 修复方案
{solution-description}

## 测试验证
- [ ] 单元测试通过
- [ ] 手动验证修复效果
- [ ] 不影响其他功能

## 相关 Issue
Closes #{issue-number}

---
@claude please review this PR
EOF
)"
```

### 8. 返回原分支
```bash
# 切回 local 分支
git checkout local
```

## 输出要求

### 执行报告
提供以下内容：

1. **执行摘要**
   - ✓ Commit 已创建/已存在
   - ✓ Bugfix 分支已创建: `bugfix/issue-X-xxx`
   - ✓ Cherry-pick 成功
   - ✓ 推送到远程成功
   - ✓ PR 已创建: {PR_URL}

2. **PR 信息**
   - PR URL
   - PR 标题
   - 关联的 Issue
   - Claude 评审状态（已触发 @claude）

3. **下一步操作**
   - 等待 Claude 自动评审
   - 查看 CI/CD 状态
   - 合并前的检查清单

## 错误处理

1. **Cherry-pick 冲突**：
   - 提示用户手动解决冲突
   - 提供解决冲突的命令提示
   - 询问是否继续或中止

2. **推送失败**：
   - 检查分支是否已存在
   - 检查网络连接
   - 检查 Git 权限

3. **PR 创建失败**：
   - 检查 gh CLI 是否已认证
   - 检查分支是否已推送
   - 手动创建 PR 的备选方案

## 关键原则

1. **安全第一**：始终先 stash 或提交当前改动
2. **清晰沟通**：每步都有明确的输出信息
3. **可回滚**：出错时能回到原状态
4. **自动化**：尽量减少手动操作

## 参数说明

调用方式：
```
/23-submit-bugfix-pr
```

**自动从上下文提取**：
- Issue number（从对话历史中的 Issue URL 或 commit message）
- Commit message（从最近的 commit 或根据修复内容自动生成）
- Bugfix 分支名称（自动生成格式：`bugfix/issue-{number}-{描述}`）
- PR 标题（根据 Issue 标题和修复内容自动生成）
- PR 描述（根据修复分析自动生成，包括问题描述、修复方案、测试验证）

**仅在无法自动提取时才询问用户**，实现零交互或最小交互。

## 注意事项

1. **智能上下文感知**：
   - 如果从 `/21-fix-bug-from-issue` 调用，自动识别 Issue number 和修复内容
   - 自动分析 git log 找到最近的 bugfix commit
   - 根据修复报告自动生成完整的 PR 描述

2. **确保 dev 分支是最新的**：避免合并冲突

3. **检查 commit 内容**：确保只包含 bugfix，不包含其他改动

4. **@claude 触发**：确保 PR description 包含 `@claude please review`

## 典型工作流

```
用户: /21-fix-bug-from-issue https://github.com/xqliu/nofx/issues/8
AI: [分析并修复 bug，生成修复报告]
用户: /23-submit-bugfix-pr
AI: [自动提取 Issue #8，创建 bugfix/issue-8-pnl-percentage，提交 PR]
```

完全自动化，无需用户输入任何信息！
