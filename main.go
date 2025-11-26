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

	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"
)

type Config struct {
	Mode string `json:"mode"` // "debug" | "release"
	Port int    `json:"port"`
}

func loadConfig() (*Config, error) {
	f, err := os.ReadFile("config.json")
	if err != nil {
		return nil, err
	}
	var cfg Config
	err = json.Unmarshal(f, &cfg)
	return &cfg, err
}

func main() {
	cfg, err := loadConfig()
	if err != nil {
		panic(err)
	}

	db, err := sql.Open("sqlite3", "./mahjong.db?_foreign_keys=on")
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
