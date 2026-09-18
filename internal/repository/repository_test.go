package repository

import (
	"database/sql"
	"mahjong-stat-back/internal/schema"
	"testing"

	_ "modernc.org/sqlite"
)

// newTestRepository는 테스트별로 독립된 인메모리 DB를 준비합니다.
// :memory: SQLite는 연결마다 다른 DB가 되므로 연결을 반드시 하나로 제한합니다.
func newTestRepository(t *testing.T) (*Repository, *sql.DB) {
	t.Helper()

	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	db.SetMaxOpenConns(1)

	if err := schema.InitSchema(db); err != nil {
		db.Close()
		t.Fatalf("initialize test schema: %v", err)
	}

	return NewRepository(db), db
}

func insertTestPlayer(t *testing.T, db *sql.DB, name string) int64 {
	t.Helper()

	result, err := db.Exec(`INSERT INTO players (name) VALUES (?)`, name)
	if err != nil {
		t.Fatalf("insert player %s: %v", name, err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("read player id: %v", err)
	}
	return id
}

func insertTestRound(
	t *testing.T,
	db *sql.DB,
	date string,
	createdAt string,
	playerIDs []int64,
) int64 {
	t.Helper()

	result, err := db.Exec(
		`INSERT INTO rounds (date, created_at) VALUES (?, ?)`,
		date,
		createdAt,
	)
	if err != nil {
		t.Fatalf("insert round: %v", err)
	}
	roundID, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("read round id: %v", err)
	}

	for index, playerID := range playerIDs {
		if _, err := db.Exec(
			`INSERT INTO round_results (round_id, player_id, rank) VALUES (?, ?, ?)`,
			roundID,
			playerID,
			index+1,
		); err != nil {
			t.Fatalf("insert round result: %v", err)
		}
	}

	return roundID
}

// GetRoundsByDate가 단일 JOIN 결과를 기존 API 구조와 같은 Round 배열로 묶는지 확인합니다.
func TestGetRoundsByDatePreservesRoundAndRankOrder(t *testing.T) {
	repo, db := newTestRepository(t)
	defer db.Close()

	playerIDs := []int64{
		insertTestPlayer(t, db, "희재93"),
		insertTestPlayer(t, db, "도윤95"),
		insertTestPlayer(t, db, "지호99"),
		insertTestPlayer(t, db, "인섭02"),
	}
	olderRoundID := insertTestRound(
		t,
		db,
		"2026-09-18",
		"2026-09-18 18:00:00",
		playerIDs,
	)
	newerRanking := []int64{playerIDs[2], playerIDs[0], playerIDs[3], playerIDs[1]}
	newerRoundID := insertTestRound(
		t,
		db,
		"2026-09-18",
		"2026-09-18 19:00:00",
		newerRanking,
	)

	rounds, err := repo.GetRoundsByDate("2026-09-18")
	if err != nil {
		t.Fatalf("GetRoundsByDate returned error: %v", err)
	}
	if len(rounds) != 2 {
		t.Fatalf("want 2 rounds, got %d", len(rounds))
	}
	if rounds[0].ID != int(newerRoundID) || rounds[1].ID != int(olderRoundID) {
		t.Fatalf("unexpected round order: %+v", rounds)
	}
	if len(rounds[0].Ranking) != 4 {
		t.Fatalf("want 4 ranking entries, got %d", len(rounds[0].Ranking))
	}
	for index, playerID := range newerRanking {
		if rounds[0].Ranking[index] != int(playerID) {
			t.Fatalf(
				"rank %d: want player %d, got %d",
				index+1,
				playerID,
				rounds[0].Ranking[index],
			)
		}
	}
}

// 기간 필터가 LEFT JOIN 의미를 깨뜨리지 않고 0게임 플레이어도 반환하는지 확인합니다.
func TestGetAllPlayerTotalStatsKeepsPlayersWithoutGamesInRange(t *testing.T) {
	repo, db := newTestRepository(t)
	defer db.Close()

	activePlayerID := insertTestPlayer(t, db, "희재93")
	inactivePlayerID := insertTestPlayer(t, db, "도윤95")

	insertTestRound(
		t,
		db,
		"2026-09-18",
		"2026-09-18 18:00:00",
		[]int64{activePlayerID},
	)
	insertTestRound(
		t,
		db,
		"2026-08-01",
		"2026-08-01 18:00:00",
		[]int64{inactivePlayerID},
	)

	stats, err := repo.GetAllPlayerTotalStats("2026-09-01", "2026-09-30")
	if err != nil {
		t.Fatalf("GetAllPlayerTotalStats returned error: %v", err)
	}
	if len(stats) != 2 {
		t.Fatalf("want both players, got %d rows", len(stats))
	}

	gamesByPlayer := make(map[int64]int64, len(stats))
	for _, row := range stats {
		gamesByPlayer[row.PlayerID] = row.Games
	}
	if gamesByPlayer[activePlayerID] != 1 {
		t.Fatalf("active player: want 1 game, got %d", gamesByPlayer[activePlayerID])
	}
	if gamesByPlayer[inactivePlayerID] != 0 {
		t.Fatalf("inactive player: want 0 games, got %d", gamesByPlayer[inactivePlayerID])
	}
}
