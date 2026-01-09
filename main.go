package main

import (
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/outpost/outpost/internal/acquisition"
	"github.com/outpost/outpost/internal/api"
	"github.com/outpost/outpost/internal/auth"
	"github.com/outpost/outpost/internal/config"
	"github.com/outpost/outpost/internal/database"
	"github.com/outpost/outpost/internal/downloadclient"
	"github.com/outpost/outpost/internal/indexer"
	"github.com/outpost/outpost/internal/metadata"
	"github.com/outpost/outpost/internal/scanner"
	"github.com/outpost/outpost/internal/scheduler"
)

func main() {
	cfg := config.Load()

	// Ensure data directory exists
	dataDir := filepath.Dir(cfg.DBPath)
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		log.Fatalf("Failed to create data directory: %v", err)
	}

	// Ensure images directory exists
	imageDir := filepath.Join(dataDir, "images")
	if err := os.MkdirAll(imageDir, 0755); err != nil {
		log.Fatalf("Failed to create images directory: %v", err)
	}

	// Initialize database
	db, err := database.New(cfg.DBPath)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	// Initialize auth service
	authSvc := auth.New(db)

	// Get TMDB API key from settings (may be empty initially)
	apiKey, _ := db.GetSetting("tmdb_api_key")

	// Initialize metadata service
	meta := metadata.NewService(db, apiKey, imageDir)

	// Initialize scanner with metadata service
	scan := scanner.New(db, meta, dataDir)

	// Initialize shared managers
	downloads := downloadclient.NewManager(db)
	indexers := indexer.NewManager()

	// Initialize scheduler
	sched := scheduler.New(db, indexers, downloads, scan)

	// Initialize acquisition service for download tracking and import
	acqSvc := acquisition.NewService(db, db.DB(), downloads, indexers, nil)

	// Initialize server with scheduler and acquisition service
	server := api.NewServer(cfg, db, scan, meta, authSvc, downloads, indexers, sched, acqSvc)

	// Start scheduler
	sched.Start()

	// Start acquisition service
	acqSvc.Start()
	log.Println("Acquisition service started")

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start server in goroutine
	go func() {
		log.Printf("Starting Outpost server on port %s", cfg.Port)
		if err := server.Start(); err != nil {
			log.Fatalf("Server failed to start: %v", err)
		}
	}()

	// Wait for shutdown signal
	<-sigChan
	log.Println("Shutting down...")

	// Stop services
	acqSvc.Stop()
	sched.Stop()

	log.Println("Goodbye!")
}
