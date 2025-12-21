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

	"github.com/stkevintan/miko/config"
	_ "github.com/stkevintan/miko/docs" // This line is important for swagger docs
	"github.com/stkevintan/miko/internal/handler"
	"github.com/stkevintan/miko/pkg/cookiecloud"
	"github.com/stkevintan/miko/pkg/log"
	"github.com/stkevintan/miko/pkg/netease"
	"github.com/stkevintan/miko/pkg/registry"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Initialize global logger from config.
	log.Default = log.New(cfg.Log)

	// Pretty-print loaded config for debugging.
	if b, err := json.MarshalIndent(cfg, "", "  "); err == nil {
		log.Debug("Loaded config:\n%s", string(b))
	}

	pr, err := registry.NewProviderRegistry(cfg.Registry)
	if err != nil {
		panic(fmt.Sprintf("Failed to create provider registry: %v", err))
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	// Initialize CookieCloud jar
	jar, err := cookiecloud.NewCookieCloudJar(ctx, cfg.CookieCloud)
	if err != nil {
		log.Fatalf("Failed to create CookieCloud jar: %v", err)
	}

	// add netease provider
	pr.RegisterFactory("netease", netease.NewNetEaseProviderFactory(jar))
	// add other providers here...

	// Initialize HTTP handler
	h := handler.New(pr)
	// Create HTTP server
	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.Server.Port),
		Handler: h.Routes(),
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
