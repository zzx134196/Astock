#!/bin/bash
set -e

cd "$(dirname "$0")/.."

echo "=== Git Push ==="
git add -A
git commit -m "update: $(date '+%Y-%m-%d %H:%M:%S')" || echo "Nothing to commit"
git push origin main || git push origin master || echo "Push failed"

echo "=== Sync to remote server ==="
ssh root@103.210.237.2 "cd /opt/astock && git pull" 2>/dev/null || echo "Remote sync skipped"

echo "=== Done ==="
