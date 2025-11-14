package main

import (
	"database/sql"
	"log"
	"mahjong-stat-back/internal/api"
	"mahjong-stat-back/internal/repository"
	"mahjong-stat-back/internal/schema"
	"mahjong-stat-back/internal/service"

	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"
)

func main() {
	// TODO: 플래그/환경변수로 경로 뺄 수 있음
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

	r := gin.Default()
	api.RegisterRoutes(r, svc)

	if err := r.Run(":8080"); err != nil {
		log.Fatal(err)
	}
}
