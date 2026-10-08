#!/usr/bin/env bash
# 안전한 읽기 전용 API 스모크 테스트. DB를 수정하지 않습니다.
set -euo pipefail
BASE_URL="${BASE_URL:-http://127.0.0.1:8080}"

check() {
  local path="$1"
  echo "==> GET $path"
  curl --fail-with-body --silent --show-error "${BASE_URL}${path}"
  printf '\n\n'
}

check '/api/server-date'
check '/api/players'
check '/api/stats/all'
check '/api/rounds/by-date?date=2026-10-08'
echo 'Read-only smoke requests completed. Run cargo test for write/rollback scenarios.'
