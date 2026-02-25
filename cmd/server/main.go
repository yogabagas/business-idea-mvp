package main

import (
	"log"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/yogabagas/business-idea-mvp/internal/api"
	"github.com/yogabagas/business-idea-mvp/internal/auth"
	"github.com/yogabagas/business-idea-mvp/internal/config"
	"github.com/yogabagas/business-idea-mvp/internal/filters"
)

func main() {
	_ = godotenv.Load()
	cfg := config.Load()
	if cfg.GoogleClientID == "" || cfg.GoogleSecret == "" {
		log.Fatal("GOOGLE_CLIENT_ID and GOOGLE_CLIENT_SECRET are required")
	}
	redirectURL := "http://localhost:" + cfg.ServerPort + "/api/auth/callback"
	ga := auth.NewGoogleAuth(cfg.GoogleClientID, cfg.GoogleSecret, redirectURL)
	fe := filters.NewEngine(100)
	handlers := api.NewHandlers(cfg, ga, fe)

	r := gin.Default()
	handlers.Register(r)
	log.Printf("Server listening on :%s", cfg.ServerPort)
	if err := r.Run(":" + cfg.ServerPort); err != nil {
		log.Fatal(err)
	}
}
