#!/bin/bash
# 每日自动运行脚本
# 建议通过crontab设定:
#   收盘后运行(15:30): 30 15 * * 1-5 /path/to/daily_run.sh close
#   竞价后运行(09:26): 26 9 * * 1-5 /path/to/daily_run.sh bid

set -e
cd "$(dirname "$0")/.."

MODE=${1:-close}
LOG_DIR="data/logs"
mkdir -p "$LOG_DIR"
TODAY=$(date +%Y%m%d)
LOG_FILE="$LOG_DIR/${TODAY}_${MODE}.log"

log() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] $1" | tee -a "$LOG_FILE"
}

case "$MODE" in
  close)
    log "=== 收盘后全流程开始 ==="

    log "1. 更新股票列表..."
    go run cmd/collector/main.go -task=stocks >> "$LOG_FILE" 2>&1

    log "2. 增量采集日K线..."
    go run cmd/collector/main.go -task=daily >> "$LOG_FILE" 2>&1

    log "3. 计算涨停数据..."
    go run cmd/collector/main.go -task=zt >> "$LOG_FILE" 2>&1

    log "4. 采集扩展数据(龙虎榜/资金流向/异动)..."
    go run cmd/collector/main.go -task=extend >> "$LOG_FILE" 2>&1

    log "5. 采集人气排行..."
    go run cmd/collector/main.go -task=hot_rank >> "$LOG_FILE" 2>&1

    log "6. 计算技术指标..."
    go run cmd/analyzer/main.go -task=indicators >> "$LOG_FILE" 2>&1

    log "7. 分析涨停特征+情绪..."
    go run cmd/analyzer/main.go -task=all >> "$LOG_FILE" 2>&1

    log "8. 计算溢价..."
    go run cmd/analyzer/main.go -task=premium >> "$LOG_FILE" 2>&1

    log "9. 计算每日情绪明细..."
    go run cmd/analyzer/main.go -task=sentiment_detail >> "$LOG_FILE" 2>&1

    log "10. 收盘选股..."
    go run cmd/selector/main.go -mode=close >> "$LOG_FILE" 2>&1

    log "=== 收盘后全流程完成 ==="
    ;;

  bid)
    log "=== 竞价选股流程开始 ==="

    log "1. 采集竞价数据..."
    go run cmd/collector/main.go -task=bid >> "$LOG_FILE" 2>&1

    log "2. 竞价选股..."
    go run cmd/selector/main.go -mode=bid >> "$LOG_FILE" 2>&1

    log "=== 竞价选股完成 ==="
    ;;

  backtest)
    log "=== 回测流程 ==="
    go run cmd/analyzer/main.go -task=backtest >> "$LOG_FILE" 2>&1
    log "=== 回测完成 ==="
    ;;

  *)
    echo "用法: $0 {close|bid|backtest}"
    exit 1
    ;;
esac
