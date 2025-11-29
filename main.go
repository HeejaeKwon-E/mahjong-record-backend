package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"mahjong-stat-back/internal/api"
	"mahjong-stat-back/internal/repository"
	"mahjong-stat-back/internal/schema"
	"mahjong-stat-back/internal/service"
	"os"
	"path/filepath"
	"time"

	"github.com/gin-gonic/gin"
	_ "modernc.org/sqlite"
)

type Config struct {
	Mode string `json:"mode"` // "debug" | "release"
	Port int    `json:"port"`
}

func loadConfig() (*Config, error) {
	// 현재 작업 디렉토리 가져오기
	wd, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	// OS에 맞게 경로 결합
	configPath := filepath.Join(wd, "config.json")

	// 파일 읽기
	f, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	// JSON 파싱
	var cfg Config
	if err := json.Unmarshal(f, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func main() {
	loc, _ := time.LoadLocation("Asia/Seoul")
	time.Local = loc
	cfg, err := loadConfig()
	if err != nil {
		panic(err)
	}

	db, err := sql.Open("sqlite", "./mahjong.db?_foreign_keys=on")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	if err := schema.InitSchema(db); err != nil {
		log.Fatal(err)
	}

	repo := repository.NewRepository(db)
	svc := service.NewService(repo)

	// 🔥 Config에 따라 동적으로 모드 변경
	switch cfg.Mode {
	case "debug":
		gin.SetMode(gin.DebugMode)
	case "test":
		gin.SetMode(gin.TestMode)
	default:
		gin.SetMode(gin.ReleaseMode)
	}
	r := gin.Default()
	api.RegisterRoutes(r, svc)
	api.RegisterWebRoutes(r)

	if err := r.Run(fmt.Sprintf(":%d", cfg.Port)); err != nil {
		log.Fatal(err)
	}
}
