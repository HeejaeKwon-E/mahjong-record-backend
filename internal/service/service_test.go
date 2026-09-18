package service

import (
	"database/sql"
	"errors"
	"mahjong-stat-back/internal/repository"
	"mahjong-stat-back/internal/schema"
	"testing"

	_ "modernc.org/sqlite"
)

// newHistoryTestService는 테스트마다 독립된 인메모리 SQLite를 준비합니다.
// 연결을 하나로 제한해야 :memory: 데이터베이스가 쿼리 사이에 분리되지 않습니다.
func newHistoryTestService(t *testing.T) (*Service, *sql.DB) {
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

	repo := repository.NewRepository(db)
	return NewService(repo), db
}

// seedPlayerHistory는 3일 동안 1위, 3위, 2위, 4위를 기록한 플레이어를 만듭니다.
func seedPlayerHistory(t *testing.T, db *sql.DB) int {
	t.Helper()

	result, err := db.Exec(`INSERT INTO players (name) VALUES (?)`, "희재93")
	if err != nil {
		t.Fatalf("insert player: %v", err)
	}
	playerID, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("read player id: %v", err)
	}

	fixtures := []struct {
		date      string
		createdAt string
		rank      int
	}{
		{"2026-07-01", "2026-07-01 18:00:00", 1},
		{"2026-07-01", "2026-07-01 18:10:00", 3},
		{"2026-08-02", "2026-08-02 19:00:00", 2},
		{"2026-09-03", "2026-09-03 20:00:00", 4},
	}

	for _, fixture := range fixtures {
		roundResult, insertErr := db.Exec(
			`INSERT INTO rounds (date, created_at) VALUES (?, ?)`,
			fixture.date,
			fixture.createdAt,
		)
		if insertErr != nil {
			t.Fatalf("insert round: %v", insertErr)
		}

		roundID, idErr := roundResult.LastInsertId()
		if idErr != nil {
			t.Fatalf("read round id: %v", idErr)
		}
		if _, insertErr = db.Exec(
			`INSERT INTO round_results (round_id, player_id, rank) VALUES (?, ?, ?)`,
			roundID,
			playerID,
			fixture.rank,
		); insertErr != nil {
			t.Fatalf("insert round result: %v", insertErr)
		}
	}

	return int(playerID)
}

func TestGetPlayerStatsHistory(t *testing.T) {
	svc, db := newHistoryTestService(t)
	defer db.Close()
	playerID := seedPlayerHistory(t, db)

	history, err := svc.GetPlayerStatsHistory(playerID, "", "")
	if err != nil {
		t.Fatalf("GetPlayerStatsHistory returned error: %v", err)
	}

	if history.Player.ID != playerID || history.Player.Name != "희재93" {
		t.Fatalf("unexpected player: %+v", history.Player)
	}
	if history.Summary.Games != 4 || history.Summary.AvgRank != 2.5 {
		t.Fatalf("unexpected summary: %+v", history.Summary)
	}
	if history.Summary.FirstRate != 0.25 ||
		history.Summary.Top2Rate != 0.5 ||
		history.Summary.FourthRate != 0.25 {
		t.Fatalf("unexpected rates: %+v", history.Summary)
	}
	if history.RankCounts.First != 1 ||
		history.RankCounts.Second != 1 ||
		history.RankCounts.Third != 1 ||
		history.RankCounts.Fourth != 1 {
		t.Fatalf("unexpected rank counts: %+v", history.RankCounts)
	}

	if len(history.Daily) != 3 {
		t.Fatalf("expected 3 daily rows, got %d", len(history.Daily))
	}
	if history.Daily[0].Date != "2026-07-01" ||
		history.Daily[0].Games != 2 ||
		history.Daily[0].AvgRank != 2 {
		t.Fatalf("unexpected first daily row: %+v", history.Daily[0])
	}

	if len(history.RecentRounds) != 4 {
		t.Fatalf("expected 4 recent rounds, got %d", len(history.RecentRounds))
	}
	wantRecentRanks := []int{4, 2, 3, 1}
	for index, wantRank := range wantRecentRanks {
		if history.RecentRounds[index].Rank != wantRank {
			t.Fatalf(
				"recent round %d: want rank %d, got %d",
				index,
				wantRank,
				history.RecentRounds[index].Rank,
			)
		}
	}
}

func TestGetPlayerStatsHistoryDateRange(t *testing.T) {
	svc, db := newHistoryTestService(t)
	defer db.Close()
	playerID := seedPlayerHistory(t, db)

	history, err := svc.GetPlayerStatsHistory(
		playerID,
		"2026-08-01",
		"2026-09-30",
	)
	if err != nil {
		t.Fatalf("GetPlayerStatsHistory returned error: %v", err)
	}

	if history.Summary.Games != 2 || history.Summary.AvgRank != 3 {
		t.Fatalf("unexpected ranged summary: %+v", history.Summary)
	}
	if len(history.Daily) != 2 || len(history.RecentRounds) != 2 {
		t.Fatalf(
			"unexpected ranged rows: daily=%d recent=%d",
			len(history.Daily),
			len(history.RecentRounds),
		)
	}
}

func TestGetPlayerStatsHistoryValidation(t *testing.T) {
	svc, db := newHistoryTestService(t)
	defer db.Close()
	playerID := seedPlayerHistory(t, db)

	tests := []struct {
		name      string
		playerID  int
		startDate string
		endDate   string
		wantError error
	}{
		{"invalid player id", 0, "", "", ErrInvalidPlayerID},
		{"invalid start date", playerID, "2026-13-01", "", ErrInvalidDateFormat},
		{"invalid end date", playerID, "", "2026/09/01", ErrInvalidDateFormat},
		{"reversed range", playerID, "2026-09-02", "2026-09-01", ErrInvalidDateRange},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := svc.GetPlayerStatsHistory(
				test.playerID,
				test.startDate,
				test.endDate,
			)
			if !errors.Is(err, test.wantError) {
				t.Fatalf("want %v, got %v", test.wantError, err)
			}
		})
	}
}

func TestGetAllPlayerTotalStatsValidation(t *testing.T) {
	svc, db := newHistoryTestService(t)
	defer db.Close()

	tests := []struct {
		name      string
		startDate string
		endDate   string
		wantError error
	}{
		{"invalid start date", "2026-13-01", "", ErrInvalidDateFormat},
		{"invalid end date", "", "2026/09/01", ErrInvalidDateFormat},
		{"reversed range", "2026-09-02", "2026-09-01", ErrInvalidDateRange},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := svc.GetAllPlayerTotalStats(test.startDate, test.endDate)
			if !errors.Is(err, test.wantError) {
				t.Fatalf("want %v, got %v", test.wantError, err)
			}
		})
	}
}

func TestGetPlayerStatsHistoryRecentRoundLimit(t *testing.T) {
	svc, db := newHistoryTestService(t)
	defer db.Close()

	playerResult, err := db.Exec(`INSERT INTO players (name) VALUES (?)`, "희재93")
	if err != nil {
		t.Fatalf("insert player: %v", err)
	}
	playerID, err := playerResult.LastInsertId()
	if err != nil {
		t.Fatalf("read player id: %v", err)
	}

	var latestRoundID int64
	for index := 0; index < 55; index++ {
		roundResult, insertErr := db.Exec(
			`INSERT INTO rounds (date, created_at) VALUES (?, ?)`,
			"2026-09-18",
			"2026-09-18 18:00:00",
		)
		if insertErr != nil {
			t.Fatalf("insert round %d: %v", index, insertErr)
		}
		latestRoundID, insertErr = roundResult.LastInsertId()
		if insertErr != nil {
			t.Fatalf("read round id %d: %v", index, insertErr)
		}

		if _, insertErr = db.Exec(
			`INSERT INTO round_results (round_id, player_id, rank) VALUES (?, ?, ?)`,
			latestRoundID,
			playerID,
			index%4+1,
		); insertErr != nil {
			t.Fatalf("insert round result %d: %v", index, insertErr)
		}
	}

	history, err := svc.GetPlayerStatsHistory(int(playerID), "", "")
	if err != nil {
		t.Fatalf("GetPlayerStatsHistory returned error: %v", err)
	}
	if len(history.RecentRounds) != 50 {
		t.Fatalf("want 50 recent rounds, got %d", len(history.RecentRounds))
	}
	if history.RecentRounds[0].RoundID != int(latestRoundID) {
		t.Fatalf(
			"want latest round %d first, got %d",
			latestRoundID,
			history.RecentRounds[0].RoundID,
		)
	}
}
