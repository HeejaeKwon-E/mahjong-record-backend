# Mahjong Record Backend (Rust + Axum + SQLite)

기존 `mahjong-stat-back` Go 서버를 **프론트엔드/API/SQLite 테이블 변경 없이** Rust로 대체하는 구현입니다.

## Rust 관용구 개선판 안내

기존 Go→Rust 1차 이식의 API, 프론트엔드 및 SQLite 파일 형식을 유지하면서 쿼리 타입(`DateQuery`/`DateRangeQuery`), SQLx `FromRow`, `thiserror`, `NaiveDate` 내부 검증, 파싱 실패 전파를 적용했습니다. **자세한 변경 파일·의도적인 오류 동작 차이·검증 방법은 [RUST_REFACTOR_NOTES.md](RUST_REFACTOR_NOTES.md)에 있습니다.**

```bash
# Cargo.lock이 없는 새 체크아웃에서 처음 한 번 실행
cargo generate-lockfile
# Rust 스타일 검사, Clippy, 컴파일, 디버그/릴리스 테스트
bash scripts/verify.sh
```

**참고:** 이 환경에는 Rust 컴파일러가 없어 `cargo check/test/clippy`를 직접 실행하지 못했습니다. 각 검사의 실제 통과 여부는 사용자 환경에서 확인해야 합니다.

## 1. 변경된 파일

| 파일 | 내용 |
|---|---|
| `src/main.rs` | `config.json` 로드, DB 연결, HTTP 서버 시작 |
| `src/api/mod.rs` | Gin의 9개 REST API를 Axum으로 구현, 웹 정적 파일 서빙 |
| `src/domain/mod.rs` | JSON 모델, 타입 지정 쿼리 DTO 및 날짜 범위 |
| `src/service/mod.rs` | 한국 날짜 기준 기록 검증, 사용자명 검증, 통계 계산 |
| `src/repository/mod.rs` | 기존 SQLite 쿼리, FromRow, timestamp 검증, INSERT/DELETE 트랜잭션 |
| `src/schema/mod.rs` | 기존 3테이블 및 2인덱스 (IF NOT EXISTS) |
| `tests/api_compat.rs` | Axum HTTP/API/SQLite 및 타입·타임스탬프 회귀 테스트 |
| `scripts/smoke.sh` | 서버 실행 후 간단한 curl 검사 |
| `scripts/verify.sh` | Cargo fmt/check/clippy/test/release 테스트 묶음 |
| `config.json` | 기존 Go와 동일한 설정 (`mode`, `port`) |

Go 프로젝트의 `go.mod` 및 소스 파일을 Rust 프로젝트로 옮기지 않았습니다. 원래 Go 백엔드는 따로 보관하고, Rust 백엔드는 새 폴더에서 실행하세요.

## 2. 준비 환경

- Rust 안정판 + Cargo (`rustup` 권장)
- macOS / Ubuntu / Windows (Rust 빌드 대상)
- SQLite 파일은 자동 생성됨 (별도 DB 서버 불필요)
- 프론트 빌드에는 Node.js, npm 필요

```bash
cargo --version
rustc --version
```

### 실행

```bash
cd mahjong-stat-back-rust
cargo check
cargo test
cargo run
```

`config.json` 기본값:

```json
{
  "mode": "debug",
  "port": 8080
}
```

`mode=debug`일 때 상세 로그, 그 외 모드에서는 일반 로그를 기록합니다. 로그 수준은 `RUST_LOG=debug` 등으로 덮어쓸 수 있습니다.

**중요:** 실행 디렉터리가 `mahjong-stat-back-rust`여야 하며, 이 디렉터리에서 `config.json`, `mahjong.db`, `web/dist`를 찾습니다.

### 기존 Go DB 데이터 이어받기

먼저 Go 서버를 중지하고 기존 SQLite의 `.db` 파일을 복사하세요.

```bash
# Go 서버 종료 후 실행. 경로는 사용 환경에 맞게 변경하세요.
cp ../mahjong-stat-back/mahjong.db ./mahjong.db
```

`mahjong.db`의 경로, 테이블명(`players`, `rounds`, `round_results`), 인덱스와 컬럼은 변경하지 않습니다. 백업을 안전하게 유지하고 Rust로 테스트를 마친 뒤 전환하세요.

### 프론트 통합 배포

```bash
cd ../mahjong-stat-front
npm install
npm run build

cd ../mahjong-stat-back-rust
mkdir -p web/dist
cp -R ../mahjong-stat-front/dist/. web/dist/
cargo run --release
```

개발 서버로 프론트가 실행 중인 경우에도 `vite.config.ts`의 `/api` 프록시가 그대로 `localhost:8080`으로 전달되므로 **프론트 코드는 수정할 필요가 없습니다.**

브라우저: `http://localhost:8080/`

## 3. API Usage

**기존 9개 엔드포인트/HTTP 상태 코드/JSON 필드명을 유지합니다.**

| Method | Path | 성공 응답 |
|---|---|---|
| GET | `/api/server-date` | `{"today":"YYYY-MM-DD"}` |
| GET | `/api/players` | `[Player,...]`, 데이터 없을 때 `null` |
| POST | `/api/players` | 생성 `Player`, `200` |
| GET | `/api/players/by-date?date=...` | `[Player,...]`, 데이터 없을 때 `null` |
| POST | `/api/rounds` | `{"round_id":1}`, `200` |
| GET | `/api/rounds/by-date?date=...` | `[Round,...]`, 데이터 없을 때 `null` |
| DELETE | `/api/rounds/1` | 삭제된 `Round`, `200` |
| GET | `/api/stats/all?start_date=&end_date=` | `[PlayerTotalStats,...]`, 데이터 없을 때 `[]` |
| GET | `/api/stats/players/1/history?start_date=&end_date=` | `PlayerStatsHistory` |

**Player** 응답: `{"id":1,"name":"희재93","created_at":"2026-10-08T06:00:00Z"}`

**Round** 응답: `{"id":1,"date":"2026-10-08","created_at":"2026-10-08T06:00:00Z","ranking":[1,2,3,4]}`

**PlayerTotalStats** 필드: `player_id`, `name`, `games`, `first`, `second`, `third`, `fourth`, `first_rate`, `top2_rate`, `fourth_rate`, `avg_rank`.

**PlayerStatsHistory** 필드: `player`, `summary`, `rank_counts`, `daily`, `recent_rounds`. 최근 기록은 조회 기간 내 최대 50건·최신순, `daily`는 날짜 오름차순입니다.

### 호출 예시

```bash
# 오늘 날짜 확인
curl -s http://localhost:8080/api/server-date

# 플레이어 등록 (한글 완성형 두 글자 + ASCII 숫자 두 자리)
curl -s -X POST http://localhost:8080/api/players \
  -H 'Content-Type: application/json' \
  -d '{"name":"희재93"}'

# 4명 플레이어가 등록되어 있어야 성공합니다.
# 날짜는 GET /api/server-date 결과를 사용하세요.
curl -s -X POST http://localhost:8080/api/rounds \
  -H 'Content-Type: application/json' \
  -d '{"date":"2026-10-08","ranking":[1,2,3,4]}'

curl -s 'http://localhost:8080/api/rounds/by-date?date=2026-10-08'
curl -s 'http://localhost:8080/api/stats/all?start_date=2026-01-01&end_date=2026-12-31'
curl -s 'http://localhost:8080/api/stats/players/1/history'
curl -s -X DELETE http://localhost:8080/api/rounds/1
```

### 오류 예시

| 상황 | 상태 | 오류 JSON |
|---|---|---|
| 잘못된 JSON 필드 | 400 | `{"error":"invalid payload"}` |
| 잘못된 이름 | 400 | `{"error":"형식: 한글이름 + 2자리 연도"}` |
| 과거 날짜로 라운드 저장 | 400 | `{"error":"라운드는 오늘 날짜에만 저장할 수 있습니다"}` |
| 라운드 사용자 수 != 4 | 400 | `{"error":"ranking must have exactly 4 players"}` |
| 날짜 쿼리 없음 | 400 | `{"error":"date query is required"}` |
| 잘못된 통계 날짜 | 400 | `{"error":"invalid date format; expected YYYY-MM-DD"}` |
| 시작일 > 종료일 | 400 | `{"error":"start_date must not be after end_date"}` |
| 없는 개인 통계 ID | 404 | `{"error":"player not found"}` |
| 없는 라운드 ID 삭제 | 404 | `{"error":"round not found"}` |
| 알 수 없는 `/api/...` 경로 | 404 | `{"error":"not found"}` |
| 통계 조회 중 DB 오류 | 500 | `{"error":"failed to load stats"}` |

**알려진 차이:** UNIQUE/FK 제약 충돌 시, 기존 Go `modernc.org/sqlite`와 Rust SQLx의 DB 오류 문자열이 다를 수 있습니다. 상태 코드(400)와 JSON `error` 필드는 동일하나, DB 드라이버가 만든 메시지 원문까지 바이트 단위로 동일하지는 않습니다.

## 4. 검증 및 회귀 테스트

```bash
cargo fmt --all
cargo fmt --all -- --check
cargo check
cargo test
cargo run --release
# 다른 터미널에서:
bash scripts/smoke.sh
```

중요한 검증 사항:

1. GET 전체 플레이어/날짜별 플레이어/날짜별 라운드의 빈 결과가 `null`인지 확인
2. GET 전체 통계의 빈 결과는 `[]`인지 확인
3. 오늘(Asia/Seoul)만 라운드 저장 가능한지 확인
4. FK 실패 시 라운드/결과가 둘 다 롤백되는지 확인
5. `recent_rounds` 최근 50개 제한과 최신순 정렬 확인
6. 데이터 없는 플레이어도 전체 통계에 0게임으로 표시되는지 확인
7. DELETE 이후 `round_results`도 CASCADE 삭제되는지 확인
8. 브라우저 새로고침으로 `/record`, `/stats` 등 SPA 경로 유지 확인
9. 날짜 경계(한국 시간 자정 전후) 및 `created_at` UTC/프론트 표시 일치 확인

문제 발생 시 `RUST_LOG=debug cargo run`으로 로그 확인. DB 스키마 확인은 `sqlite3 mahjong.db '.schema'`, 파일 확인은 `ls -l mahjong.db`.

**주의:** 이 작업 환경에서는 Cargo/Rust 컴파일러가 없어 직접 `cargo check` 또는 `cargo test`를 수행하지 못했습니다. Rust 실행 환경에서 위 명령으로 최종 빌드와 통합 테스트가 반드시 필요합니다. README의 성공 동작 예시는 확인 대상으로 제시한 것이지 이 환경에서 실행하여 확인된 결과를 의미하지 않습니다.
