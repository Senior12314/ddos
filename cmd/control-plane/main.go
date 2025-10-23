package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/cloudnordsp/minecraft-protection/internal/api"
	"github.com/cloudnordsp/minecraft-protection/internal/config"
	"github.com/cloudnordsp/minecraft-protection/internal/database"
	"github.com/cloudnordsp/minecraft-protection/internal/monitoring"
	"github.com/cloudnordsp/minecraft-protection/internal/node"
	"github.com/cloudnordsp/minecraft-protection/internal/proxy"
	"github.com/cloudnordsp/minecraft-protection/internal/storage"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	var (
		configFile = flag.String("config", "config.yaml", "Configuration file path")
		debug      = flag.Bool("debug", false, "Enable debug mode")
	)
	flag.Parse()

	// Load configuration
	cfg, err := config.Load(*configFile)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	if *debug {
		cfg.Debug = true
	}

	// Initialize database
	db, err := database.New(cfg.Database)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}

	// Get underlying sql.DB for closing
	sqlDB, err := db.DB()
	if err != nil {
		log.Fatalf("Failed to get underlying sql.DB: %v", err)
	}
	defer sqlDB.Close()

	// Initialize storage
	store := storage.New(db)

	// Initialize monitoring
	monitor := monitoring.New(cfg.Monitoring)

	// Initialize node manager
	nodeManager := node.NewManager(cfg.Node, store, monitor)

	// Initialize proxy manager
	proxyManager := proxy.NewManager(cfg.Proxy, nodeManager, monitor)

	// Initialize API server
	apiServer := api.NewServer(cfg.API, store, nodeManager, proxyManager, monitor)

	// Setup HTTP server
	router := gin.Default()
	
	// Add middleware
	router.Use(gin.Recovery())
	router.Use(monitor.Middleware())
	
	// Health check endpoint
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":    "healthy",
			"timestamp": time.Now().UTC(),
			"version":   "1.0.0",
		})
	})

	// Prometheus metrics endpoint
	router.GET("/metrics", gin.WrapH(promhttp.Handler()))

	// API routes
	apiGroup := router.Group("/api/v1")
	apiServer.SetupRoutes(apiGroup)

	// Start services
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start node manager
	go func() {
		if err := nodeManager.Start(ctx); err != nil {
			log.Printf("Node manager error: %v", err)
		}
	}()

	// Start proxy manager
	go func() {
		if err := proxyManager.Start(ctx); err != nil {
			log.Printf("Proxy manager error: %v", err)
		}
	}()

	// Start API server
	server := &http.Server{
		Addr:    cfg.API.Address,
		Handler: router,
	}

	go func() {
		log.Printf("Starting API server on %s", cfg.API.Address)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start API server: %v", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	// Graceful shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("Server forced to shutdown: %v", err)
	}

	cancel() // Cancel all background services
	log.Println("Server exited")
}
