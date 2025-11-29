package api

import (
	"database/sql"
	"errors"
	"log"
	"mahjong-stat-back/internal/domain"
	"mahjong-stat-back/internal/service"
	"net/http"
	"strconv"
	"time"

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

// GET /api/players/by-date?date=YYYY-MM-DD
func GetPlayersByDateHandler(svc *service.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		date := c.Query("date")
		if date == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "date query is required"})
			return
		}

		players, err := svc.GetPlayersByDate(date)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, players)
	}
}

// DeleteRoundHandler 는 라운드 삭제 요청을 처리합니다.
//
// 매개변수:
//   - svc: Service 포인터
//
// 반환값:
//   - gin.HandlerFunc: Gin 에서 사용 가능한 핸들러
func DeleteRoundHandler(svc *service.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		idStr := c.Param("id")
		id, err := strconv.ParseInt(idStr, 10, 0)
		if err != nil || id <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "invalid round id",
			})
			return
		}

		round, err := svc.DeleteRound(int(id))
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				c.JSON(http.StatusNotFound, gin.H{
					"error": "round not found",
				})
				return
			}

			log.Printf("failed to delete round (id=%d): %v", id, err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "failed to delete round",
			})
			return
		}

		// 삭제된 라운드 정보를 그대로 돌려줌
		c.JSON(http.StatusOK, round)
	}
}

// GetServerDateHandler 는 서버 기준 '오늘 날짜'를 반환합니다.
//
// 매개변수:
//   - 없음 (Gin 컨텍스트로부터 자동 주입)
//
// 반환값:
//   - JSON: {"today": "YYYY-MM-DD"}
func GetServerDateHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 필요하면 여기서 KST 고정도 가능:
		// loc, _ := time.LoadLocation("Asia/Seoul")
		// now := time.Now().In(loc)
		now := time.Now()

		c.JSON(http.StatusOK, gin.H{
			"today": now.Format("2006-01-02"),
		})
	}
}
