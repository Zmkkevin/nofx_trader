#!/bin/bash

# ═══════════════════════════════════════════════════════════════
# NOFX - Log Cleanup Script
# 用途：清理旧日志文件，建议通过cron定期执行
# ═══════════════════════════════════════════════════════════════

set -e

# 配置
LOG_DIR="logs"
DECISION_LOG_DIR="decision_logs"
MAX_AGE_DAYS=7  # 保留天数
DRY_RUN=false

# 颜色定义
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

# 打印函数
print_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# 显示帮助
show_help() {
    cat <<EOF
用法: $0 [选项]

清理超过指定天数的日志文件

选项:
    -d, --days DAYS     保留最近N天的日志 (默认: 7)
    -n, --dry-run       仅显示将要删除的文件，不实际删除
    -h, --help          显示此帮助信息

示例:
    $0                  # 删除7天前的日志
    $0 -d 30            # 删除30天前的日志
    $0 --dry-run        # 查看将要删除哪些文件

添加到crontab (每天凌晨2点执行):
    0 2 * * * /path/to/scripts/cleanup_logs.sh -d 7
EOF
}

# 解析参数
while [[ $# -gt 0 ]]; do
    case $1 in
        -d|--days)
            MAX_AGE_DAYS="$2"
            shift 2
            ;;
        -n|--dry-run)
            DRY_RUN=true
            shift
            ;;
        -h|--help)
            show_help
            exit 0
            ;;
        *)
            print_error "未知选项: $1"
            show_help
            exit 1
            ;;
    esac
done

# 验证参数
if ! [[ "$MAX_AGE_DAYS" =~ ^[0-9]+$ ]]; then
    print_error "无效的天数: $MAX_AGE_DAYS"
    exit 1
fi

# 开始清理
print_info "🗑️  开始清理日志..."
print_info "保留天数: $MAX_AGE_DAYS 天"
if [ "$DRY_RUN" = true ]; then
    print_warning "DRY-RUN 模式：不会实际删除文件"
fi

total_deleted=0
total_size=0

# 清理主日志目录
if [ -d "$LOG_DIR" ]; then
    print_info "检查 $LOG_DIR/ ..."

    # 查找并删除旧日志
    while IFS= read -r -d '' file; do
        size=$(stat -f "%z" "$file" 2>/dev/null || stat -c "%s" "$file" 2>/dev/null)
        total_size=$((total_size + size))

        if [ "$DRY_RUN" = true ]; then
            print_info "  [DRY-RUN] 将删除: $file ($(numfmt --to=iec-i --suffix=B $size 2>/dev/null || echo "${size}B"))"
        else
            rm -f "$file"
            print_info "  ✓ 已删除: $file"
        fi
        total_deleted=$((total_deleted + 1))
    done < <(find "$LOG_DIR" -type f -name "*.log" -mtime +"$MAX_AGE_DAYS" -print0 2>/dev/null)
fi

# 清理决策日志目录（按trader_id分组）
if [ -d "$DECISION_LOG_DIR" ]; then
    print_info "检查 $DECISION_LOG_DIR/ ..."

    # 清理每个trader的旧日志
    for trader_dir in "$DECISION_LOG_DIR"/*; do
        if [ -d "$trader_dir" ]; then
            while IFS= read -r -d '' file; do
                size=$(stat -f "%z" "$file" 2>/dev/null || stat -c "%s" "$file" 2>/dev/null)
                total_size=$((total_size + size))

                if [ "$DRY_RUN" = true ]; then
                    print_info "  [DRY-RUN] 将删除: $file ($(numfmt --to=iec-i --suffix=B $size 2>/dev/null || echo "${size}B"))"
                else
                    rm -f "$file"
                    print_info "  ✓ 已删除: $file"
                fi
                total_deleted=$((total_deleted + 1))
            done < <(find "$trader_dir" -type f -name "*.json" -mtime +"$MAX_AGE_DAYS" -print0 2>/dev/null)
        fi
    done
fi

# 显示统计
echo ""
if [ "$total_deleted" -eq 0 ]; then
    print_info "没有找到需要清理的日志文件"
else
    formatted_size=$(numfmt --to=iec-i --suffix=B $total_size 2>/dev/null || echo "${total_size}B")
    if [ "$DRY_RUN" = true ]; then
        print_warning "DRY-RUN: 共找到 $total_deleted 个文件，总大小 $formatted_size"
    else
        print_info "✅ 清理完成！"
        print_info "  • 删除文件数: $total_deleted"
        print_info "  • 释放空间: $formatted_size"
    fi
fi

print_info "✅ 日志清理完成！"
