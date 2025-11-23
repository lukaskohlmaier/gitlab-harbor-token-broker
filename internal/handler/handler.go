package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/lukaskohlmaier/gitlab-harbor-token-broker/internal/harbor"
	"github.com/lukaskohlmaier/gitlab-harbor-token-broker/internal/jwt"
	"github.com/lukaskohlmaier/gitlab-harbor-token-broker/internal/logging"
	"github.com/lukaskohlmaier/gitlab-harbor-token-broker/internal/policy"
)

// Handler handles HTTP requests
type Handler struct {
	jwtValidator *jwt.Validator
	policyEngine *policy.Engine
	harborClient *harbor.Client
	logger       *logging.Logger
	robotTTL     int
}

// TokenRequest represents the request body for /token endpoint
type TokenRequest struct {
	HarborProject string `json:"harbor_project"`
	Permissions   string `json:"permissions"`
}

// TokenResponse represents the response for /token endpoint
type TokenResponse struct {
	Username  string `json:"username"`
	Password  string `json:"password"`
	ExpiresAt string `json:"expires_at"`
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error string `json:"error"`
}

// NewHandler creates a new HTTP handler
func NewHandler(jwtValidator *jwt.Validator, policyEngine *policy.Engine, harborClient *harbor.Client, logger *logging.Logger, robotTTL int) *Handler {
	return &Handler{
		jwtValidator: jwtValidator,
		policyEngine: policyEngine,
		harborClient: harborClient,
		logger:       logger,
		robotTTL:     robotTTL,
	}
}

// HandleToken handles POST /token requests
func (h *Handler) HandleToken(w http.ResponseWriter, r *http.Request) {
	// Only allow POST method
	if r.Method != http.MethodPost {
		h.respondError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	// Extract JWT from Authorization header
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		h.respondError(w, http.StatusUnauthorized, "missing authorization header")
		return
	}

	tokenString := strings.TrimPrefix(authHeader, "Bearer ")
	if tokenString == authHeader {
		h.respondError(w, http.StatusUnauthorized, "invalid authorization header format")
		return
	}

	// Validate JWT
	claims, err := h.jwtValidator.ValidateToken(tokenString)
	if err != nil {
		h.logger.Error("JWT validation failed", err)
		h.respondError(w, http.StatusUnauthorized, "invalid or expired token")
		return
	}

	// Parse request body
	var req TokenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Validate request
	if req.HarborProject == "" {
		h.respondError(w, http.StatusBadRequest, "harbor_project is required")
		return
	}
	if req.Permissions == "" {
		h.respondError(w, http.StatusBadRequest, "permissions is required")
		return
	}

	// Validate permission format
	if err := policy.ValidatePermission(req.Permissions); err != nil {
		h.respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Check authorization policy
	if err := h.policyEngine.AuthorizeRequest(claims.ProjectPath, req.HarborProject, req.Permissions); err != nil {
		h.logger.AuditRequestDenied(claims.ProjectPath, req.HarborProject, req.Permissions, err.Error())
		h.respondError(w, http.StatusForbidden, "access denied by policy")
		return
	}

	// Generate robot account name
	robotName := fmt.Sprintf("ci-temp-%s-%d", claims.JobID, time.Now().Unix())

	// Create Harbor robot account
	robot, err := h.harborClient.CreateRobotAccount(req.HarborProject, robotName, req.Permissions, h.robotTTL)
	if err != nil {
		h.logger.Error("Failed to create robot account", err)
		h.respondError(w, http.StatusInternalServerError, "failed to create credentials")
		return
	}

	// Log audit event
	h.logger.AuditTokenIssued(claims.ProjectPath, req.HarborProject, req.Permissions, robot.ID, robot.Name, robot.ExpiresAt, claims.PipelineID, claims.JobID)

	// Return response
	response := TokenResponse{
		Username:  robot.Name,
		Password:  robot.Secret,
		ExpiresAt: robot.ExpiresAt.Format(time.RFC3339),
	}

	h.respondJSON(w, http.StatusOK, response)
}

// HandleHealth handles GET /health requests
func (h *Handler) HandleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.respondError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

// respondJSON sends a JSON response
func (h *Handler) respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// respondError sends an error response
func (h *Handler) respondError(w http.ResponseWriter, status int, message string) {
	h.respondJSON(w, status, ErrorResponse{Error: message})
}
