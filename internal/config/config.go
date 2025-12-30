package config

import (
	"os"
)

type Config struct {
	Port      string
	StaticDir string
	DBPath    string
}

func Load() *Config {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	staticDir := os.Getenv("STATIC_DIR")
	if staticDir == "" {
		staticDir = "./frontend/build"
	}

	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "./data/outpost.db"
	}

	return &Config{
		Port:      port,
		StaticDir: staticDir,
		DBPath:    dbPath,
	}
}
