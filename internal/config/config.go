package config

import (
	"os"
)

type Config struct {
	ServerPort       string
	GoogleClientID  string
	GoogleSecret    string
	FrontendOrigin  string
	YouTubeAPIKey   string // optional: for videos.list to get liveChatId without OAuth
}

func Load() *Config {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	frontend := os.Getenv("FRONTEND_ORIGIN")
	if frontend == "" {
		frontend = "http://localhost:5173"
	}
	return &Config{
		ServerPort:      port,
		GoogleClientID:  os.Getenv("GOOGLE_CLIENT_ID"),
		GoogleSecret:    os.Getenv("GOOGLE_CLIENT_SECRET"),
		FrontendOrigin:  frontend,
		YouTubeAPIKey:   os.Getenv("YOUTUBE_API_KEY"),
	}
}
