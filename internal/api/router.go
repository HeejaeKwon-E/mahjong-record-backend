package api

import (
	"mahjong-stat-back/internal/service"
	"net/http"
	"path/filepath"
	"strings"

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
			//Todo: unuse now
			players.GET("/by-date", GetPlayersByDateHandler(svc))
		}

		rounds := api.Group("/rounds")
		{
			rounds.POST("", CreateRoundHandler(svc))
			rounds.GET("", GetRoundsByDateHandler(svc)) // 🔹 추가
			// TODO: GET /rounds?date=...
		}
	}
}

func RegisterWebRoutes(r *gin.Engine) {
	// 2) 정적 파일 경로 설정 (Vite 빌드 결과)
	distDir := "./web/dist"
	assetsDir := filepath.Join(distDir, "assets")

	// Vite가 만든 /assets/* 정적 파일
	r.Static("/assets", assetsDir)

	// 루트(/) index.html
	r.StaticFile("/", filepath.Join(distDir, "index.html"))

	// 3) SPA 라우터 지원: /record, /stats 같은 경로도 전부 index.html로 보내기
	r.NoRoute(func(c *gin.Context) {
		path := c.Request.URL.Path

		// /api/... 인데 라우트가 없다 → 진짜 404
		if strings.HasPrefix(path, "/api/") {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}

		// 그 외 경로는 모두 프론트 SPA에게 넘김
		c.File(filepath.Join(distDir, "index.html"))
	})
}
