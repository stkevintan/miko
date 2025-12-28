package main

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path"
	"syscall"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/samber/do/v2"
	"github.com/stkevintan/miko/config"
	"github.com/stkevintan/miko/models"
	"github.com/stkevintan/miko/pkg/cookiecloud"
	"github.com/stkevintan/miko/pkg/log"
	"github.com/stkevintan/miko/server"
	"gorm.io/gorm"
)

//go:embed VERSION
var version string

func main() {
	// Load configuration
	cfg, err := config.Load(version)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Initialize global logger from config.
	log.Default = log.New(cfg.Log)

	// Pretty-print loaded config for debugging.
	if b, err := json.MarshalIndent(cfg, "", "  "); err == nil {
		log.Debug("Loaded config:\n%s", string(b))
	}

	// Initialize Database
	var db *gorm.DB
	if cfg.Database.Driver == "sqlite" {
		// Ensure directory exists
		dir := path.Dir(cfg.Database.DSN)
		if err := os.MkdirAll(dir, 0755); err != nil {
			log.Fatalf("Failed to create database directory: %v", err)
		}
		db, err = gorm.Open(sqlite.Open(cfg.Database.DSN), &gorm.Config{})
		if err != nil {
			log.Fatalf("Failed to connect to database: %v", err)
		}
	} else {
		log.Fatalf("Unsupported database driver: %s", cfg.Database.Driver)
	}

	// Auto-migrate models
	err = db.AutoMigrate(
		&models.User{},
		&cookiecloud.Identity{},
		&models.MusicFolder{},
		&models.ArtistID3{},
		&models.AlbumID3{},
		&models.Child{},
		&models.Genre{},
		&models.PlaylistRecord{},
		&models.PlaylistSong{},
	)
	if err != nil {
		log.Fatalf("Failed to migrate database: %v", err)
	}

	// Create default user if none exists
	var count int64
	db.Model(&models.User{}).Count(&count)
	if count == 0 {
		defaultUser := models.User{
			Username:  "admin",
			Password:  "adminpassword",
			AdminRole: true,
		}
		if err := db.Create(&defaultUser).Error; err != nil {
			log.Error("Failed to create default user: %v", err)
		} else {
			log.Info("Created default admin user: admin / adminpassword")
		}
	}

	// Initialize Injector
	injector := do.New()

	// Register services
	do.Provide(injector, func(i do.Injector) (*config.Config, error) {
		return cfg, nil
	})
	do.Provide(injector, func(i do.Injector) (*gorm.DB, error) {
		return db, nil
	})
	do.Provide(injector, func(i do.Injector) (*cookiecloud.Config, error) {
		return cfg.CookieCloud, nil
	})

	// Initialize HTTP handler
	h := server.New(injector)
	r := h.Routes()

	ctx := context.Background()

	// Create HTTP server
	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.Server.Port),
		Handler: r,
	}

	// Start server in a goroutine
	go func() {
		log.Printf("Server starting on port %d", cfg.Server.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	// Create a deadline to wait for
	ctx2, cancel2 := context.WithTimeout(ctx, 30*time.Second)
	defer cancel2()

	// Attempt graceful shutdown
	if err := server.Shutdown(ctx2); err != nil {
		log.Fatal("Server forced to shutdown:", err)
	}

	log.Println("Server exited")
}
