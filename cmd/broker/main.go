package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/lukaskohlmaier/gitlab-harbor-token-broker/internal/config"
	"github.com/lukaskohlmaier/gitlab-harbor-token-broker/internal/handler"
	"github.com/lukaskohlmaier/gitlab-harbor-token-broker/internal/harbor"
	"github.com/lukaskohlmaier/gitlab-harbor-token-broker/internal/jwt"
	"github.com/lukaskohlmaier/gitlab-harbor-token-broker/internal/logging"
	"github.com/lukaskohlmaier/gitlab-harbor-token-broker/internal/policy"
)

func main() {
	// Parse command-line flags
	configPath := flag.String("config", "config.yaml", "path to configuration file")
	flag.Parse()

	// Initialize logger
	logger := logging.NewLogger()
	logger.Info("Starting Harbor CI Credential Broker")

	// Load configuration
	cfg, err := config.Load(*configPath)
	if err != nil {
		logger.Error("Failed to load configuration", err)
		os.Exit(1)
	}

	logger.Info(fmt.Sprintf("Configuration loaded successfully from %s", *configPath))

	// Construct JWKS URL if not provided
	jwksURL := cfg.GitLab.JWKSUrl
	if jwksURL == "" {
		jwksURL = fmt.Sprintf("%s/oauth/discovery/keys", cfg.GitLab.InstanceURL)
	}

	// Set default issuers if not provided
	issuers := cfg.GitLab.Issuers
	if len(issuers) == 0 {
		issuers = []string{cfg.GitLab.InstanceURL}
	}

	// Initialize JWT validator
	jwtValidator := jwt.NewValidator(cfg.GitLab.Audience, issuers, jwksURL)
	logger.Info("JWT validator initialized")

	// Initialize policy engine
	policyEngine := policy.NewEngine(cfg.Policies)
	logger.Info(fmt.Sprintf("Policy engine initialized with %d rules", len(cfg.Policies)))

	// Initialize Harbor client
	harborClient := harbor.NewClient(cfg.Harbor.URL, cfg.Harbor.Username, cfg.Harbor.Password)
	logger.Info("Harbor client initialized")

	// Initialize HTTP handler
	httpHandler := handler.NewHandler(jwtValidator, policyEngine, harborClient, logger, cfg.Security.RobotTTLMinutes)

	// Setup HTTP routes
	mux := http.NewServeMux()
	mux.HandleFunc("/token", httpHandler.HandleToken)
	mux.HandleFunc("/health", httpHandler.HandleHealth)

	// Create HTTP server
	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Server.Port),
		Handler:      mux,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
	}

	// Start server in a goroutine
	go func() {
		logger.Info(fmt.Sprintf("Server listening on port %d", cfg.Server.Port))
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("Server error", err)
			os.Exit(1)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down server...")

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logger.Error("Server forced to shutdown", err)
		os.Exit(1)
	}

	logger.Info("Server stopped")
}
