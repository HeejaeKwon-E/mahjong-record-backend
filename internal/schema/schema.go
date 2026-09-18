// internal/mahjong/schema.go
package schema

import "database/sql"

// InitSchema 함수는 필요한 테이블을 생성합니다.
//
// 매개변수:
//   - db: SQLite 데이터베이스 핸들
//
// 반환값:
//   - error: 에러 발생 시 반환, 없으면 nil
func InitSchema(db *sql.DB) error {
	schema := `
PRAGMA foreign_keys = ON;

CREATE TABLE IF NOT EXISTS players (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  name TEXT NOT NULL UNIQUE,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- 🔹 라운드: 날짜별 라운드 기록 (멤버는 round_results에서 관리)
CREATE TABLE IF NOT EXISTS rounds (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  date TEXT NOT NULL,           -- YYYY-MM-DD
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- 🔹 라운드 결과: 각 라운드에 참여한 플레이어와 등수
CREATE TABLE IF NOT EXISTS round_results (
  round_id INTEGER NOT NULL,
  player_id INTEGER NOT NULL,
  rank INTEGER NOT NULL,      -- 1~4
  PRIMARY KEY (round_id, player_id),
  FOREIGN KEY (round_id) REFERENCES rounds(id) ON DELETE CASCADE,
  FOREIGN KEY (player_id) REFERENCES players(id) ON DELETE CASCADE
);

-- 개인 통계는 player_id로 모든 참여 라운드를 조회하므로 역방향 인덱스가 필요합니다.
-- 기존 기본키(round_id, player_id)만으로는 데이터가 늘어날수록 전체 스캔이 발생합니다.
CREATE INDEX IF NOT EXISTS idx_round_results_player_id
  ON round_results(player_id, round_id);

-- 기간 필터와 날짜/기록 시각 정렬에 사용하는 인덱스입니다.
CREATE INDEX IF NOT EXISTS idx_rounds_date_created_at
  ON rounds(date, created_at, id);
`
	_, err := db.Exec(schema)
	return err
}
