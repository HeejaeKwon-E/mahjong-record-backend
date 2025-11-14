package api

import (
	"mahjong-stat-back/internal/domain"
	"mahjong-stat-back/internal/service"
	"net/http"

	"github.com/gin-gonic/gin"
)

func GetPlayersHandler(svc *service.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		players, err := svc.GetPlayers()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, players)
	}
}

func CreatePlayerHandler(svc *service.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req domain.CreatePlayerRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
			return
		}

		player, err := svc.AddPlayer(req.Name)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, player)
	}
}

func CreateRoundHandler(svc *service.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req domain.CreateRoundRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
			return
		}

		roundID, err := svc.AddRound(req.Date, req.Ranking)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"round_id": roundID})
	}
}

// getRoundsByDateHandler는 /api/rounds?date=YYYY-MM-DD 요청을 처리합니다.
//
// query:
//   - date: 필수, 예) 2025-11-14
func GetRoundsByDateHandler(svc *service.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		date := c.Query("date")
		if date == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "date query is required"})
			return
		}

		rounds, err := svc.GetRoundsByDate(date)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, rounds)
	}
}
