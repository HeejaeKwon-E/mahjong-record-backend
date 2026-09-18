package domain

import "time"

type Player struct {
	ID        int        `json:"id"`
	Name      string     `json:"name"`
	CreatedAt *time.Time `json:"created_at"`
}

type Round struct {
	ID        int        `json:"id"`
	Date      string     `json:"date"`       // YYYY-MM-DD
	CreatedAt *time.Time `json:"created_at"` // DB 측에서 넣어주는 시간
	Ranking   []int      `json:"ranking"`    // player ids (1위부터 순서대로)
}

// 요청 페이로드용 DTO

type CreatePlayerRequest struct {
	Name string `json:"name" binding:"required"`
}

type CreateRoundRequest struct {
	Date    string `json:"date" binding:"required"`    // YYYY-MM-DD
	Ranking []int  `json:"ranking" binding:"required"` // len == 4
}

// PlayerTotalStats 는 전체 기간 기준 플레이어 통계를 나타냅니다.
type PlayerTotalStats struct {
	PlayerID   int64   `json:"player_id"`
	Name       string  `json:"name"`
	Games      int64   `json:"games"`
	First      int64   `json:"first"`
	Second     int64   `json:"second"`
	Third      int64   `json:"third"`
	Fourth     int64   `json:"fourth"`
	FirstRate  float64 `json:"first_rate"`  // 0.0 ~ 1.0
	Top2Rate   float64 `json:"top2_rate"`   // (1등+2등) / 국수
	FourthRate float64 `json:"fourth_rate"` // 4등 / 국수
	AvgRank    float64 `json:"avg_rank"`    // ← 추가!
}

// PlayerStatsSummary는 한 플레이어의 선택 기간 요약 통계입니다.
// 횟수와 비율을 함께 내려주므로 프론트엔드가 같은 계산을 중복하지 않습니다.
type PlayerStatsSummary struct {
	Games      int64   `json:"games"`
	AvgRank    float64 `json:"avg_rank"`
	FirstRate  float64 `json:"first_rate"`
	Top2Rate   float64 `json:"top2_rate"`
	FourthRate float64 `json:"fourth_rate"`
}

// RankCounts는 선택 기간에 기록한 등수별 횟수입니다.
type RankCounts struct {
	First  int64 `json:"first"`
	Second int64 `json:"second"`
	Third  int64 `json:"third"`
	Fourth int64 `json:"fourth"`
}

// DailyPlayerStats는 기존 API 호환성을 위해 유지하는 플레이어 한 명의 날짜별 집계입니다.
type DailyPlayerStats struct {
	Date    string  `json:"date"`
	Games   int64   `json:"games"`
	First   int64   `json:"first"`
	Second  int64   `json:"second"`
	Third   int64   `json:"third"`
	Fourth  int64   `json:"fourth"`
	AvgRank float64 `json:"avg_rank"`
}

// PlayerRoundResult는 개인 통계 계산에 사용하는 라운드 단위 원천 결과입니다.
// API에는 직접 노출하지 않고 Repository에서 Service로 전달할 때 사용합니다.
type PlayerRoundResult struct {
	RoundID   int
	Date      string
	Rank      int
	CreatedAt *time.Time
}

// PlayerRecentRound는 상세 화면의 최근 성적에 노출되는 최소 라운드 정보입니다.
type PlayerRecentRound struct {
	RoundID   int        `json:"round_id"`
	Date      string     `json:"date"`
	Rank      int        `json:"rank"`
	CreatedAt *time.Time `json:"created_at"`
}

// PlayerStatsHistory는 개인 통계 상세 화면에 필요한 데이터를 한 번에 반환합니다.
// recent_rounds는 선택 기간 안에서 최신 50게임까지만 포함합니다.
type PlayerStatsHistory struct {
	Player       Player              `json:"player"`
	Summary      PlayerStatsSummary  `json:"summary"`
	RankCounts   RankCounts          `json:"rank_counts"`
	Daily        []DailyPlayerStats  `json:"daily"`
	RecentRounds []PlayerRecentRound `json:"recent_rounds"`
}
