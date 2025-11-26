# Mahjong Record Backend (Go + SQLite)

모바일 마작 기록 웹앱을 위한 **백엔드 서버**입니다.  
프론트엔드는 React + TypeScript + MUI 기반, 백엔드는 **Go + Gin + SQLite** 기반으로 설계되었습니다.

이 서버는 다음과 같은 역할을 담당합니다:

- 플레이어 CRUD 관리
- 날짜별 플레이어 조회
- 라운드 기록 저장/조회
- “오늘 날짜만 라운드 저장 허용” 검증
- 프론트엔드 정적 파일(Vite build) 서빙

---

## ✨ 주요 기능

### 🧩 플레이어 관리 API

| 메서드 | 엔드포인트 | 설명 |
|--------|------------|------|
| `GET` | `/api/players` | 전체 플레이어 목록 조회 |
| `POST` | `/api/players` | 플레이어 추가 (중복 이름 가능) |
| `GET` | `/api/players/by-date?date=YYYY-MM-DD` | 특정 날짜에 등장한 플레이어 목록 |

플레이어 추가 시 **프론트에서 한글 이름 + 2자리 출생년도 형식으로 제한**하고  
백엔드에서는 단순 문자열로 저장합니다.

---

### 🀄 라운드 기록 API

| 메서드 | 엔드포인트 | 설명 |
|--------|------------|------|
| `POST` | `/api/rounds` | 라운드 저장 — 단, 오늘 날짜만 허용 |
| `GET` | `/api/rounds/by-date?date=YYYY-MM-DD` | 특정 날짜의 모든 라운드 조회 |

**라운드 저장 제한 규칙:**

- 백엔드는 `POST /api/rounds` 요청 시 `date == 오늘` 인지 검증
- 플레이어 수는 프론트에서 4명 고정으로 제한하지만, 백엔드에서도 다시 확인 가능
- 저장 시 UTC가 아닌 **로컬 시간(Asia/Seoul)** 기준 timestamp 저장

---

## 🗂 기술 스택

- **언어**: Go 1.23+
- **웹 프레임워크**: Gin
- **DB**: SQLite3 (CGO 필요)
- **ORM**: 없음 (직접 SQL 사용)
- **빌드 도구**: Go Modules
- **정적 파일 빌드/서빙**: Vite → `dist/` → Gin StaticFile

---

## 📁 프로젝트 구조 예시

```
/mahjong-backend
 ├── main.go
 ├── api/
 │    ├── players.go
 │    ├── rounds.go
 │    └── register.go
 ├── repository/
 │    ├── player_repo.go
 │    ├── round_repo.go
 ├── service/
 │    ├── player_service.go
 │    ├── round_service.go
 ├── model/
 │    ├── player.go
 │    ├── round.go
 ├── db/
 │    ├── sqlite.go
 ├── dist/ ← 프론트엔드 빌드 결과
 └── config.json
```

---

## ⚙️ config.json (모드 스위칭 가능)

```json
{
  "mode": "release",
  "port": 8080,
  "database": "mahjong.db"
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

⚠️ SQLite는 **CGO 필요**하므로 반드시 CGO를 켜고 빌드해야 합니다.

```bash
CGO_ENABLED=1 go build -o mahjong-server
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

2. 나온 `dist/` 폴더 전체를 백엔드 프로젝트 루트로 복사

```
/mahjong-backend/dist
```

3. Go 서버에서 정적 파일 제공:

```go
r.Static("/assets", "./dist/assets")
r.StaticFile("/", "./dist/index.html")
```

---

## 📝 라운드 데이터 저장 구조

- `rounds` 테이블 예시:

| 컬럼 | 타입 | 설명 |
|------|------|------|
| id | INTEGER PK | 라운드 ID |
| date | TEXT | YYYY-MM-DD |
| created_at | TEXT | 저장 시각 |
| p1 | INTEGER | 1위 player_id |
| p2 | INTEGER | 2위 player_id |
| p3 | INTEGER | 3위 player_id |
| p4 | INTEGER | 4위 player_id |

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
