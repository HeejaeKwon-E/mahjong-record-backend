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
