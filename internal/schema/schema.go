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
`
	_, err := db.Exec(schema)
	return err
}
