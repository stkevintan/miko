package main

// @title           Miko Service API
// @version         1.0
// @description     A Go service library with HTTP API endpoints
// @termsOfService  http://swagger.io/terms/

// @contact.name   API Support
// @contact.url    http://www.swagger.io/support
// @contact.email  support@swagger.io

// @license.name  MIT
// @license.url   http://opensource.org/licenses/MIT

// @host
// @BasePath  /api

// @securityDefinitions.apikey ApiKeyAuth
// @in header
// @name Authorization

// @schemes http https

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/samber/do/v2"
	"github.com/stkevintan/miko/config"
	_ "github.com/stkevintan/miko/docs" // This line is important for swagger docs
	"github.com/stkevintan/miko/pkg/cookiecloud"
	"github.com/stkevintan/miko/pkg/log"
	"github.com/stkevintan/miko/pkg/netease"
	"github.com/stkevintan/miko/server"
	"github.com/stkevintan/miko/server/models"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Initialize global logger from config.
	log.Default = log.New(cfg.Log)

	// Initialize Database
	var db *gorm.DB
	if cfg.Database.Driver == "sqlite" {
		db, err = gorm.Open(sqlite.Open(cfg.Database.DSN), &gorm.Config{})
		if err != nil {
			log.Fatalf("Failed to connect to database: %v", err)
		}
	} else {
		log.Fatalf("Unsupported database driver: %s", cfg.Database.Driver)
	}

	// Auto-migrate models
	err = db.AutoMigrate(&models.User{}, &cookiecloud.Identity{})
	if err != nil {
		log.Fatalf("Failed to migrate database: %v", err)
	}

	// Create default user if none exists
	var count int64
	db.Model(&models.User{}).Count(&count)
	if count == 0 {
		defaultUser := models.User{
			Username: "admin",
			Password: "adminpassword",
			IsAdmin:  true,
		}
		if err := db.Create(&defaultUser).Error; err != nil {
			log.Error("Failed to create default user: %v", err)
		} else {
			log.Info("Created default admin user: admin / adminpassword")
		}
	}

	// Pretty-print loaded config for debugging.
	if b, err := json.MarshalIndent(cfg, "", "  "); err == nil {
		log.Debug("Loaded config:\n%s", string(b))
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

	// Register Providers
	do.ProvideNamed(injector, "netease", netease.NewNetEaseProvider)

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
