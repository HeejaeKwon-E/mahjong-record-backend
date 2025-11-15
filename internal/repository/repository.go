package repository

import (
	"database/sql"
	"mahjong-stat-back/internal/domain"
	"time"
)

// Repository는 SQLite 기반 데이터 접근 레이어입니다.
type Repository struct {
	db *sql.DB
}

// NewRepository는 Repository 인스턴스를 생성합니다.
//
// 매개변수:
//   - db: 데이터베이스 커넥션
//
// 반환값:
//   - *Repository: 레포지토리 인스턴스
func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

// 🔹 SQLite 날짜 문자열 파싱 헬퍼
func parseSQLiteTime(s string) time.Time {
	// SQLite DEFAULT CURRENT_TIMESTAMP → "2006-01-02 15:04:05"
	layouts := []string{
		time.RFC3339Nano,
		"2006-01-02 15:04:05",
	}
	for _, layout := range layouts {
		if t, err := time.Parse(layout, s); err == nil {
			return t
		}
	}
	return time.Time{}
}

// GetAllPlayers는 모든 플레이어를 조회합니다.
//
// 반환값:
//   - []Player: 플레이어 목록
//   - error: 에러 정보
func (r *Repository) GetAllPlayers() ([]domain.Player, error) {
	rows, err := r.db.Query(`SELECT id, name, created_at FROM players ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var players []domain.Player
	for rows.Next() {
		var p domain.Player
		var createdAtStr string
		if err := rows.Scan(&p.ID, &p.Name, &createdAtStr); err != nil {
			return nil, err
		}
		p.CreatedAt = parseSQLiteTime(createdAtStr)
		players = append(players, p)
	}
	return players, rows.Err()
}

// CreatePlayer는 새로운 플레이어를 생성합니다.
func (r *Repository) CreatePlayer(name string) (domain.Player, error) {
	res, err := r.db.Exec(`INSERT INTO players (name) VALUES (?)`, name)
	if err != nil {
		return domain.Player{}, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return domain.Player{}, err
	}

	row := r.db.QueryRow(`SELECT id, name, created_at FROM players WHERE id = ?`, id)
	var p domain.Player
	var createdAtStr string
	if err := row.Scan(&p.ID, &p.Name, &createdAtStr); err != nil {
		return domain.Player{}, err
	}
	p.CreatedAt = parseSQLiteTime(createdAtStr)

	return p, nil
}

// CreateRound는 라운드와 해당 결과를 함께 생성합니다.
//
// 매개변수:
//   - date: 라운드 날짜 (YYYY-MM-DD)
//   - ranking: 플레이어 ID 배열 (1위부터 순서대로)
//
// 반환값:
//   - int64: 생성된 라운드 ID
//   - error: 에러 정보
func (r *Repository) CreateRound(date string, ranking []int) (int64, error) {
	tx, err := r.db.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	res, err := tx.Exec(`INSERT INTO rounds (date) VALUES (?)`, date)
	if err != nil {
		return 0, err
	}
	roundID, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}

	stmt, err := tx.Prepare(
		`INSERT INTO round_results (round_id, player_id, rank) VALUES (?, ?, ?)`,
	)
	if err != nil {
		return 0, err
	}
	defer stmt.Close()

	for i, pid := range ranking {
		rank := i + 1
		if _, err := stmt.Exec(roundID, pid, rank); err != nil {
			return 0, err
		}
	}

	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return roundID, nil
}

// GetRoundsByDate는 특정 날짜의 라운드 목록을 조회합니다.
//
// 매개변수:
//   - date: 조회할 날짜 (YYYY-MM-DD)
//
// 반환값:
//   - []Round: 해당 날짜의 라운드 목록 (각 라운드는 Ranking 포함)
//   - error: 에러 정보
func (r *Repository) GetRoundsByDate(date string) ([]domain.Round, error) {
	rows, err := r.db.Query(
		`SELECT id, date, created_at 
         FROM rounds 
         WHERE date = ? 
         ORDER BY created_at DESC, id DESC`,
		date,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rounds []domain.Round

	for rows.Next() {
		var (
			id           int
			dateStr      string
			createdAtStr string
		)
		if err := rows.Scan(&id, &dateStr, &createdAtStr); err != nil {
			return nil, err
		}

		// 라운드별 등수 가져오기
		resRows, err := r.db.Query(
			`SELECT player_id, rank 
             FROM round_results 
             WHERE round_id = ? 
             ORDER BY rank ASC`,
			id,
		)
		if err != nil {
			return nil, err
		}

		var ranking []int
		for resRows.Next() {
			var pid, rank int
			if err := resRows.Scan(&pid, &rank); err != nil {
				resRows.Close()
				return nil, err
			}
			ranking = append(ranking, pid)
		}
		resRows.Close()

		rounds = append(rounds, domain.Round{
			ID:        id,
			Date:      dateStr,
			CreatedAt: parseSQLiteTime(createdAtStr),
			Ranking:   ranking,
		})
	}

	return rounds, rows.Err()
}

// GetPlayersByDate는 특정 날짜에 한 번이라도 라운드에 등장한 플레이어들을 반환합니다.
//
// SELECT DISTINCT p.*
// FROM players p
// JOIN round_results rr ON p.id = rr.player_id
// JOIN rounds r ON rr.round_id = r.id
// WHERE r.date = ?
func (r *Repository) GetPlayersByDate(date string) ([]domain.Player, error) {
	rows, err := r.db.Query(
		`SELECT DISTINCT p.id, p.name, p.created_at
         FROM players p
         JOIN round_results rr ON p.id = rr.player_id
         JOIN rounds r ON rr.round_id = r.id
         WHERE r.date = ?
         ORDER BY p.id`,
		date,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var players []domain.Player

	for rows.Next() {
		var p domain.Player
		var createdAtStr string
		if err := rows.Scan(&p.ID, &p.Name, &createdAtStr); err != nil {
			return nil, err
		}
		p.CreatedAt = parseSQLiteTime(createdAtStr)
		players = append(players, p)
	}

	return players, rows.Err()
}
