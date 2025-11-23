package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/lukaskohlmaier/gitlab-harbor-token-broker/internal/database"
	"github.com/lukaskohlmaier/gitlab-harbor-token-broker/internal/logging"
)

// APIHandler handles API requests for the UI
type APIHandler struct {
	db     *database.DB
	logger *logging.Logger
}

// NewAPIHandler creates a new API handler
func NewAPIHandler(db *database.DB, logger *logging.Logger) *APIHandler {
	return &APIHandler{
		db:     db,
		logger: logger,
	}
}

// AccessLogsResponse represents the response for access logs
type AccessLogsResponse struct {
	Logs  []database.AccessLog `json:"logs"`
	Total int                  `json:"total"`
	Page  int                  `json:"page"`
	Limit int                  `json:"limit"`
}

// HandleGetAccessLogs handles GET /api/access-logs
func (h *APIHandler) HandleGetAccessLogs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.respondError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	// Parse query parameters
	query := r.URL.Query()
	page, _ := strconv.Atoi(query.Get("page"))
	if page < 1 {
		page = 1
	}

	limit, _ := strconv.Atoi(query.Get("limit"))
	if limit < 1 || limit > 100 {
		limit = 20
	}

	offset := (page - 1) * limit

	// Build filters
	filters := make(map[string]string)
	if gitlabProject := query.Get("gitlab_project"); gitlabProject != "" {
		filters["gitlab_project"] = gitlabProject
	}
	if harborProject := query.Get("harbor_project"); harborProject != "" {
		filters["harbor_project"] = harborProject
	}
	if status := query.Get("status"); status != "" {
		filters["status"] = status
	}

	// Get logs from database
	logs, total, err := h.db.GetAccessLogs(limit, offset, filters)
	if err != nil {
		h.logger.Error("Failed to get access logs", err)
		h.respondError(w, http.StatusInternalServerError, "failed to retrieve access logs")
		return
	}

	response := AccessLogsResponse{
		Logs:  logs,
		Total: total,
		Page:  page,
		Limit: limit,
	}

	h.respondJSON(w, http.StatusOK, response)
}

// HandleGetPolicies handles GET /api/policies
func (h *APIHandler) HandleGetPolicies(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.respondError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	policies, err := h.db.GetPolicies()
	if err != nil {
		h.logger.Error("Failed to get policies", err)
		h.respondError(w, http.StatusInternalServerError, "failed to retrieve policies")
		return
	}

	h.respondJSON(w, http.StatusOK, policies)
}

// HandleCreatePolicy handles POST /api/policies
func (h *APIHandler) HandleCreatePolicy(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.respondError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var policy database.PolicyRule
	if err := json.NewDecoder(r.Body).Decode(&policy); err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Validate policy
	if policy.GitLabProject == "" {
		h.respondError(w, http.StatusBadRequest, "gitlab_project is required")
		return
	}
	if len(policy.HarborProjects) == 0 {
		h.respondError(w, http.StatusBadRequest, "harbor_projects must not be empty")
		return
	}
	if len(policy.AllowedPermissions) == 0 {
		h.respondError(w, http.StatusBadRequest, "allowed_permissions must not be empty")
		return
	}

	if err := h.db.CreatePolicy(&policy); err != nil {
		h.logger.Error("Failed to create policy", err)
		h.respondError(w, http.StatusInternalServerError, "failed to create policy")
		return
	}

	h.respondJSON(w, http.StatusCreated, policy)
}

// HandleUpdatePolicy handles PUT /api/policies/:id
func (h *APIHandler) HandleUpdatePolicy(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		h.respondError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	// Extract ID from URL path
	idStr := r.URL.Path[len("/api/policies/"):]
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid policy ID")
		return
	}

	var policy database.PolicyRule
	if err := json.NewDecoder(r.Body).Decode(&policy); err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	policy.ID = id

	// Validate policy
	if policy.GitLabProject == "" {
		h.respondError(w, http.StatusBadRequest, "gitlab_project is required")
		return
	}
	if len(policy.HarborProjects) == 0 {
		h.respondError(w, http.StatusBadRequest, "harbor_projects must not be empty")
		return
	}
	if len(policy.AllowedPermissions) == 0 {
		h.respondError(w, http.StatusBadRequest, "allowed_permissions must not be empty")
		return
	}

	if err := h.db.UpdatePolicy(&policy); err != nil {
		h.logger.Error("Failed to update policy", err)
		if err.Error() == "policy not found" {
			h.respondError(w, http.StatusNotFound, "policy not found")
		} else {
			h.respondError(w, http.StatusInternalServerError, "failed to update policy")
		}
		return
	}

	h.respondJSON(w, http.StatusOK, policy)
}

// HandleDeletePolicy handles DELETE /api/policies/:id
func (h *APIHandler) HandleDeletePolicy(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		h.respondError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	// Extract ID from URL path
	idStr := r.URL.Path[len("/api/policies/"):]
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid policy ID")
		return
	}

	if err := h.db.DeletePolicy(id); err != nil {
		h.logger.Error("Failed to delete policy", err)
		if err.Error() == "policy not found" {
			h.respondError(w, http.StatusNotFound, "policy not found")
		} else {
			h.respondError(w, http.StatusInternalServerError, "failed to delete policy")
		}
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// respondJSON sends a JSON response
func (h *APIHandler) respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// respondError sends an error response
func (h *APIHandler) respondError(w http.ResponseWriter, status int, message string) {
	h.respondJSON(w, status, ErrorResponse{Error: message})
}
