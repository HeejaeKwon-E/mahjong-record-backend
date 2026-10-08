#!/usr/bin/env bash
# Rust 백엔드 품질 검증. 개발 환경에 Rust + Cargo가 설치되어 있어야 합니다.
# 기존 운영 SQLite를 변경하지 않도록 `cargo test`는 메모리 DB 기반 테스트를 사용합니다.
set -euo pipefail

# 어느 경로에서 실행해도 백엔드 루트로 이동합니다.
cd "$(dirname "$0")/.."

# Cargo.lock은 서버/애플리케이션에서 커밋하는 것이 좋습니다.
# 로컬 첫 실행에서는 lockfile이 없을 수 있으므로 이를 먼저 생성합니다.
if [[ ! -f Cargo.lock ]]; then
    echo '[INFO] Cargo.lock이 없어 로컬 의존성 lockfile을 생성합니다.'
    cargo generate-lockfile
fi

echo '[1/5] rustfmt 자동 정리 및 확인'
cargo fmt --all
cargo fmt --all -- --check

echo '[2/5] 컴파일 검사 (Cargo.lock 고정)'
cargo check --locked

echo '[3/5] Clippy 정적 분석'
cargo clippy --locked --all-targets -- -D warnings

echo '[4/5] 디버그 테스트'
cargo test --locked

echo '[5/5] 릴리스 테스트'
cargo test --locked --release

echo '[OK] 모든 Cargo 검사가 통과했습니다. Cargo.lock을 Git에 포함하세요.'
