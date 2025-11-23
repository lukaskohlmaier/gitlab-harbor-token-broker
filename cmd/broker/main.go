package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/lukaskohlmaier/gitlab-harbor-token-broker/internal/config"
	"github.com/lukaskohlmaier/gitlab-harbor-token-broker/internal/database"
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

	// Initialize database if enabled
	var db *database.DB
	var apiHandler *handler.APIHandler
	if cfg.Database.Enabled {
		logger.Info("Database enabled, connecting...")
		var err error
		db, err = database.NewDB(cfg.Database.ConnectionString)
		if err != nil {
			logger.Error("Failed to connect to database", err)
			os.Exit(1)
		}
		defer db.Close()

		logger.Info("Database connected successfully")

		// Run migrations
		migrationSQL, err := os.ReadFile("migrations/001_initial_schema.sql")
		if err != nil {
			logger.Error("Failed to read migration file", err)
			os.Exit(1)
		}

		if err := db.RunMigrations(string(migrationSQL)); err != nil {
			logger.Error("Failed to run migrations", err)
			os.Exit(1)
		}

		logger.Info("Database migrations completed")

		// Update logger to use database storage
		accessLogStore := database.NewAccessLogStoreAdapter(db)
		logger = logging.NewLoggerWithStore(accessLogStore)

		// Initialize API handler for UI
		apiHandler = handler.NewAPIHandler(db, logger)
	}

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
	var policyEngine *policy.Engine
	if cfg.Database.Enabled {
		// Use database-backed policy store
		policyStore := database.NewPolicyStoreAdapter(db)
		policyEngine = policy.NewEngineWithStore(policyStore)
		logger.Info("Policy engine initialized with database storage")
	} else {
		// Use config-based policies
		policyEngine = policy.NewEngine(cfg.Policies)
		logger.Info(fmt.Sprintf("Policy engine initialized with %d rules from config", len(cfg.Policies)))
	}

	// Initialize Harbor client
	harborClient := harbor.NewClient(cfg.Harbor.URL, cfg.Harbor.Username, cfg.Harbor.Password)
	logger.Info("Harbor client initialized")

	// Initialize HTTP handler
	httpHandler := handler.NewHandler(jwtValidator, policyEngine, harborClient, logger, cfg.Security.RobotTTLMinutes)

	// Setup HTTP routes
	mux := http.NewServeMux()
	mux.HandleFunc("/token", httpHandler.HandleToken)
	mux.HandleFunc("/health", httpHandler.HandleHealth)

	// Add API endpoints if database is enabled
	if cfg.Database.Enabled && apiHandler != nil {
		mux.HandleFunc("/api/access-logs", corsMiddleware(apiHandler.HandleGetAccessLogs))
		mux.HandleFunc("/api/policies", corsMiddleware(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodGet {
				apiHandler.HandleGetPolicies(w, r)
			} else if r.Method == http.MethodPost {
				apiHandler.HandleCreatePolicy(w, r)
			} else {
				w.WriteHeader(http.StatusMethodNotAllowed)
			}
		}))
		mux.HandleFunc("/api/policies/", corsMiddleware(func(w http.ResponseWriter, r *http.Request) {
			if strings.HasPrefix(r.URL.Path, "/api/policies/") && len(r.URL.Path) > len("/api/policies/") {
				if r.Method == http.MethodPut {
					apiHandler.HandleUpdatePolicy(w, r)
				} else if r.Method == http.MethodDelete {
					apiHandler.HandleDeletePolicy(w, r)
				} else {
					w.WriteHeader(http.StatusMethodNotAllowed)
				}
			} else {
				w.WriteHeader(http.StatusNotFound)
			}
		}))
		// Serve static files from ui/dist directory
		fs := http.FileServer(http.Dir("ui/dist"))
		mux.Handle("/", fs)
		logger.Info("API endpoints and static file server enabled")
	}

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

// corsMiddleware adds CORS headers for frontend development
func corsMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		next(w, r)
	}
}
