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
	"github.com/stkevintan/miko/config"
	"github.com/stkevintan/miko/models"
	"github.com/stkevintan/miko/pkg/bookmarks"
	"github.com/stkevintan/miko/pkg/browser"
	"github.com/stkevintan/miko/pkg/cookiecloud"
	"github.com/stkevintan/miko/pkg/crypto"
	"github.com/stkevintan/miko/pkg/di"
	"github.com/stkevintan/miko/pkg/log"
	"github.com/stkevintan/miko/pkg/scanner"
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
		&models.SystemSetting{},
		&cookiecloud.Identity{},
		&models.MusicFolder{},
		&models.ArtistID3{},
		&models.AlbumID3{},
		&models.Child{},
		&models.Genre{},
		&models.PlaylistRecord{},
		&models.PlaylistSong{},
		&models.BookmarkRecord{},
		&models.PlayQueueRecord{},
		&models.PlayQueueSong{},
	)
	if err != nil {
		log.Fatalf("Failed to migrate database: %v", err)
	}

	// Initialize Injector
	ctx := di.NewContext(context.Background())
	di.Provide(ctx, cfg)
	di.Provide(ctx, db)

	// Create default user if none exists
	var count int64
	db.Model(&models.User{}).Count(&count)
	if count == 0 {
		password := "adminpassword"
		// Try to encrypt if secret is available
		if secret := crypto.ResolvePasswordSecret(ctx); secret != nil {
			if encrypted, err := crypto.Encrypt(password, secret); err == nil {
				password = encrypted
			}
		}
		defaultUser := models.User{
			Username:  "admin",
			Password:  password,
			AdminRole: true,
		}
		if err := db.Create(&defaultUser).Error; err != nil {
			log.Error("Failed to create default user: %v", err)
		} else {
			log.Info("Created default admin user: admin / adminpassword")
		}
	}

	// Register services
	di.Provide(ctx, cfg.CookieCloud)
	di.Provide(ctx, browser.New(db))
	di.Provide(ctx, bookmarks.New(db))
	// TODO: factory pattern for scanner with config
	di.Provide(ctx, scanner.New(db, cfg))

	// Initialize HTTP handler
	h := server.New(ctx)
	r := h.Routes()

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
