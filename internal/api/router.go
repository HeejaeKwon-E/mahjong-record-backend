package api

import (
	"mahjong-stat-back/internal/service"

	"github.com/gin-gonic/gin"
)

// RegisterRoutes는 마작 관련 HTTP 라우트를 등록합니다.
//
// 매개변수:
//   - r: gin Engine
//   - svc: 도메인 서비스
func RegisterRoutes(r *gin.Engine, svc *service.Service) {
	api := r.Group("/api")
	{
		players := api.Group("/players")
		{
			players.GET("", GetPlayersHandler(svc))
			players.POST("", CreatePlayerHandler(svc))
		}

		rounds := api.Group("/rounds")
		{
			rounds.POST("", CreateRoundHandler(svc))
			rounds.GET("", GetRoundsByDateHandler(svc)) // 🔹 추가
			// TODO: GET /rounds?date=...
		}
	}
}
