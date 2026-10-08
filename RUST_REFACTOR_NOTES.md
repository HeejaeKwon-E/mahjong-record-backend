# Rust 관용구 개선 (API 호환 유지)

## 변경된 파일과 이유

| 파일 | 변경 내용 | 호환성 영향 |
|---|---|---|
| `Cargo.toml` | `thiserror` 오류 derive, SQLx `derive`(FromRow), 최소 Tokio 기능 활성화 | 외부 API 없음. Cargo 의존성 그래프 갱신 필요 |
| `src/domain/mod.rs` | API 쿼리 및 응답용 Serde 구조체, 내부용 `DateRange { Option<NaiveDate> }` 도입 | JSON 필드명·자료형 그대로 |
| `src/api/mod.rs` | `HashMap` 대신 `DateQuery`/`DateRangeQuery`, 동적 JSON 대신 응답 DTO | 기존 API 경로·메서드·HTTP 코드 및 본문 유지 |
| `src/service/mod.rs` | `thiserror` 기반 `ServiceError`, `NaiveDate`를 이용한 날짜 검증 | 정상 요청/기존 검증 에러 동일 |
| `src/repository/mod.rs` | SQLx `FromRow`, `try_get`, 타임스탬프 파싱 실패 전파 | DB 스키마·SQL 정렬·트랜잭션 동일. **손상된 timestamp에 대한 오류 동작 개선** |
| `tests/api_compat.rs` | 빈/누락 쿼리, 범위 오류, 손상된 날짜, 타임존 오프셋 회귀 검증 | 테스트만 추가 |
| `scripts/verify.sh` | fmt/check/clippy/test/release 테스트 자동화 | 개발 편의 추가 |
| `README.md` | 개선 내역, 잠재적 호환성 차이와 검증 명령 | 문서만 수정 |

## 전체 데이터 흐름

`React fetch('/api/...') → Axum Query/Json (DTO) → Service (NaiveDate 유효성 검증) → Repository (SQLx query_as/try_get + 트랜잭션) → SQLite → Repository (timestamp 파싱) → Service 통계 집계 → Serde JSON → React`

기존 SQLite `players`/`rounds`/`round_results`를 변경하지 않으며, 기록 조회 SQL과 순위 계산식 또한 유지합니다.

## 의도적인 오류 처리 변경 (중요)

- **정상 데이터의 날짜 직렬화는 동일**: SQLite 기본 `CURRENT_TIMESTAMP`는 `2026-10-08T06:00:00Z`로, 기존 RFC3339 오프셋 문자열은 오프셋을 보존해 응답합니다.
- **손상된 DB DATETIME 처리만 변경**: 과거 구현은 날짜가 파싱 불가한 경우 가짜 `0001-01-01T00:00:00Z`로 바꿨으나, 이제 SQLx Decode 오류로 취급합니다. 전체 플레이어 조회라면 `500`, 날짜별 라운드 조회는 기존 에러 매핑에 따라 `400`입니다. 이는 정상 데이터에 영향을 주지 않으며 장애를 숨기지 않기 위한 의도적인 변경입니다.
- **쿼리 파라미터**: 일반적인 누락/빈 문자열/잘못된 날짜 형태는 기존과 같은 처리를 합니다. 동일 키가 반복되는 비정상적인 `?date=x&date=y` 요청은 Serde 구조체 파싱 규칙상 기존 `HashMap`과 다르게 거절될 수 있습니다. 프론트엔드에서 이런 요청을 보내지 않는지 확인하십시오.
- **기존 시간 문자열과 DB 경로**: `Round.date`/`created_at`의 API 반환 타입은 여전히 `String`입니다. JSON 및 SQLite의 기존 형식을 그대로 유지하기 위한 결정입니다.

## 검증 명령

```bash
cd mahjong-stat-back-rust
# 처음 실행 시 Cargo.lock을 생성합니다. 프로젝트 Git에 함께 커밋하는 것을 권장합니다.
cargo generate-lockfile
bash scripts/verify.sh
cargo run --release
```

다른 터미널에서:

```bash
curl -i http://127.0.0.1:8080/api/server-date
curl -i http://127.0.0.1:8080/api/players
curl -i 'http://127.0.0.1:8080/api/stats/all?start_date=2026-10-01&end_date=2026-10-31'
curl -i 'http://127.0.0.1:8080/api/stats/all?start_date=2026-10-08&end_date=2026-10-07'
curl -i http://127.0.0.1:8080/api/rounds/by-date
```

- `GET /api/server-date` → `200`, `{"today":"YYYY-MM-DD"}`
- 플레이어 0명 → `GET /api/players` 는 `200 null`
- 통계 대상 0명 → `GET /api/stats/all` 은 `200 []`
- 시작일이 종료일보다 늦음 → `400 {"error":"start_date must not be after end_date"}`
- `date` 누락 → `400 {"error":"date query is required"}`
- 실제 저장/삭제/개인 통계/타임스탬프/외래키 테스트 → `cargo test`에서 검증

오류 발생 시 `RUST_LOG=debug cargo run`으로 서버 로그를 확인하고, 날짜 변환은 `src/repository/mod.rs`, SQL 에러는 `src/service/mod.rs`와 `src/api/mod.rs`를 확인합니다.

## Cargo.lock 참고

현재 제공 환경에는 Cargo가 없어 `Cargo.lock`을 생성하거나 컴파일/Clippy/HTTP 통합 테스트를 직접 실행할 수 없었습니다. 거짓 lockfile을 포함하지 않았으며, `cargo generate-lockfile` 또는 `bash scripts/verify.sh`의 최초 실행으로 생성한 파일을 Git에 추가하십시오.

`python scripts/sql_contract_check.py`는 SQLite 쿼리·스키마 검증만 실시하며 Rust 컴파일 검증은 대체하지 않습니다.
