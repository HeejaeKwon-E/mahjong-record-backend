# Mahjong Record Backend (Go + SQLite)

모바일 마작 기록 웹앱을 위한 **백엔드 서버**입니다.  
프론트엔드는 React + TypeScript + Tailwind CSS 기반, 백엔드는 **Go + Gin + SQLite** 기반으로 설계되었습니다.

이 서버는 다음과 같은 역할을 담당합니다:

- 플레이어 CRUD 관리
- 날짜별 플레이어 조회
- 라운드 기록 저장/조회
- “오늘 날짜만 라운드 저장 허용” 검증
- 프론트엔드 정적 파일(Vite build) 서빙

---

## ✨ 주요 기능

### 🧩 플레이어 관리 API

| 메서드 | 엔드포인트                             | 설명                             |
| ------ | -------------------------------------- | -------------------------------- |
| `GET`  | `/api/players`                         | 전체 플레이어 목록 조회          |
| `POST` | `/api/players`                         | 플레이어 추가 (이름 중복 불가)   |
| `GET`  | `/api/players/by-date?date=YYYY-MM-DD` | 특정 날짜에 등장한 플레이어 목록 |

플레이어 추가 시 프론트와 백엔드 모두 **한글 이름 2자 + 출생연도 2자리** 형식을
검증하며, DB의 `UNIQUE` 제약으로 중복 이름을 차단합니다.

---

### 🀄 라운드 기록 API

| 메서드 | 엔드포인트                            | 설명                               |
| ------ | ------------------------------------- | ---------------------------------- |
| `POST` | `/api/rounds`                         | 라운드 저장 — 단, 오늘 날짜만 허용 |
| `GET`  | `/api/rounds/by-date?date=YYYY-MM-DD` | 특정 날짜의 모든 라운드 조회       |

### 📈 통계 API

| 메서드 | 엔드포인트                                                   | 설명                           |
| ------ | ------------------------------------------------------------ | ------------------------------ |
| `GET`  | `/api/stats/all?start_date=&end_date=`                       | 기간별 전체 플레이어 요약 통계 |
| `GET`  | `/api/stats/players/:playerId/history?start_date=&end_date=` | 플레이어 개인 상세 통계        |

`start_date`, `end_date`는 선택 값이며 형식은 `YYYY-MM-DD`입니다. 두 값을 모두
전달하면 시작일과 종료일을 포함해 조회합니다.

개인 상세 통계 요청 예시:

```bash
curl "http://localhost:8080/api/stats/players/1/history?start_date=2026-01-01&end_date=2026-09-30"
```

응답 예시:

```json
{
  "player": {
    "id": 1,
    "name": "희재93",
    "created_at": "2026-01-01T00:00:00Z"
  },
  "summary": {
    "games": 184,
    "avg_rank": 2.32,
    "first_rate": 0.315,
    "top2_rate": 0.571,
    "fourth_rate": 0.207
  },
  "rank_counts": {
    "first": 58,
    "second": 47,
    "third": 41,
    "fourth": 38
  },
  "daily": [
    {
      "date": "2026-09-01",
      "games": 5,
      "first": 2,
      "second": 1,
      "third": 1,
      "fourth": 1,
      "avg_rank": 2.4
    }
  ],
  "recent_rounds": [
    {
      "round_id": 201,
      "date": "2026-09-01",
      "rank": 1,
      "created_at": "2026-09-01T20:30:00Z"
    }
  ]
}
```

`recent_rounds`는 선택 기간 안에서 최신순 최대 50게임을 반환합니다. 프론트에서는
이를 최근 10/20/50게임으로 전환해 볼 수 있습니다. `daily`는
기존 API 호환성을 위해 날짜 오름차순으로 유지하지만 현재 프론트에서는 사용하지 않습니다.

오류 응답:

| 상태 코드 | 조건                                 | 응답 예시                                              |
| --------- | ------------------------------------ | ------------------------------------------------------ |
| `400`     | 플레이어 ID가 숫자가 아니거나 0 이하 | `{"error":"invalid player id"}`                        |
| `400`     | 날짜 형식이 잘못됨                   | `{"error":"invalid date format; expected YYYY-MM-DD"}` |
| `400`     | 시작일이 종료일보다 늦음             | `{"error":"start_date must not be after end_date"}`    |
| `404`     | 플레이어가 존재하지 않음             | `{"error":"player not found"}`                         |
| `500`     | DB 조회 실패                         | `{"error":"failed to load player stats"}`              |

**라운드 저장 제한 규칙:**

- 백엔드는 `POST /api/rounds` 요청 시 `date == 오늘` 인지 검증
- 플레이어 수는 프론트에서 4명 고정으로 제한하지만, 백엔드에서도 다시 확인 가능
- 기록 가능 날짜는 서버의 **Asia/Seoul** 기준 오늘로 검증
- `created_at`은 SQLite `CURRENT_TIMESTAMP` 형식으로 저장

---

## 🗂 기술 스택

- **언어**: Go 1.24+
- **웹 프레임워크**: Gin
- **DB**: SQLite3 (`modernc.org/sqlite`, pure Go)
- **ORM**: 없음 (직접 SQL 사용)
- **빌드 도구**: Go Modules
- **정적 파일 빌드/서빙**: Vite → `web/dist/` → Gin StaticFile

---

## 📁 프로젝트 구조 예시

```
/mahjong-stat-back
 ├── main.go
 ├── internal/
 │    ├── api/
 │    ├── domain/
 │    ├── repository/
 │    ├── schema/
 │    └── service/
 ├── web/
 │    └── dist/ ← 프론트엔드 빌드 결과
 ├── go.mod
 └── config.json
```

---

## ⚙️ config.json (모드 스위칭 가능)

```json
{
  "mode": "release",
  "port": 8080
}
```

### 모드 설정

- `"debug"` → 개발용 (로그 등 자세하게 출력)
- `"release"` → 배포용 (Gin ReleaseMode)

---

## 🚀 로컬 실행

### 1) 모듈 설치

```bash
go mod tidy
```

### 2) 개발 모드 실행

```bash
go run main.go
```

### 3) 릴리즈 빌드

```bash
go build -o mahjong-server
./mahjong-server
```

---

## 🛠 CORS

프론트엔드를 같은 서버에서 서빙하므로 기본적으로 CORS 이슈가 없습니다.  
외부 도메인 접근 시에는 아래처럼 CORS 등록이 필요할 수 있습니다:

```go
r.Use(cors.Default())
```

---

## 📦 프론트엔드 배포 연동

1. 프론트에서는 다음 실행:

```bash
npm run build
```

2. 나온 `dist/` 폴더 전체를 백엔드의 `web/dist/`로 복사

```
/mahjong-backend/web/dist
```

3. Go 서버에서 정적 파일 제공:

```go
r.Static("/assets", "./web/dist/assets")
r.StaticFile("/", "./web/dist/index.html")
```

---

## 📝 라운드 데이터 저장 구조

- 라운드와 등수는 정규화된 두 테이블에 저장합니다.

| 컬럼                      | 타입       | 설명        |
| ------------------------- | ---------- | ----------- |
| `rounds.id`               | INTEGER PK | 라운드 ID   |
| `rounds.date`             | TEXT       | YYYY-MM-DD  |
| `rounds.created_at`       | DATETIME   | 저장 시각   |
| `round_results.round_id`  | INTEGER FK | 라운드 ID   |
| `round_results.player_id` | INTEGER FK | 플레이어 ID |
| `round_results.rank`      | INTEGER    | 1~4위       |

프론트에서 보내는 JSON:

```json
{
  "date": "2025-11-17",
  "ranking": [3, 1, 4, 2]
}
```

---

## 🔒 라운드 저장 안전 장치

백엔드에서 다음 검증을 수행합니다:

1. **요청 날짜 == 오늘 날짜인지 검사**
2. player_id가 존재하는 플레이어인지 확인
3. 배열 길이가 정확히 4인지 검사

---

## License

MIT 또는 원하는 내용으로 수정 가능합니다.
