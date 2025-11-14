package domain

import "time"

type Player struct {
	ID        int       `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}

type Round struct {
	ID        int       `json:"id"`
	Date      string    `json:"date"`       // YYYY-MM-DD
	CreatedAt time.Time `json:"created_at"` // DB 측에서 넣어주는 시간
	Ranking   []int     `json:"ranking"`    // player ids (1위부터 순서대로)
}

// 요청 페이로드용 DTO

type CreatePlayerRequest struct {
	Name string `json:"name" binding:"required"`
}

type CreateRoundRequest struct {
	Date    string `json:"date" binding:"required"`    // YYYY-MM-DD
	Ranking []int  `json:"ranking" binding:"required"` // len == 4
}
