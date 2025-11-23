package policy

import (
	"fmt"

	"github.com/lukaskohlmaier/gitlab-harbor-token-broker/internal/config"
)

// PolicyStore interface for policy storage backends
type PolicyStore interface {
	GetPolicyByGitLabProject(gitlabProject string) (PolicyRule, error)
}

// PolicyRule represents a policy rule (compatible with database)
type PolicyRule struct {
	GitLabProject      string
	HarborProjects     []string
	AllowedPermissions []string
}

// Engine enforces authorization policies
type Engine struct {
	rules       []config.PolicyRule
	policyStore PolicyStore
}

// NewEngine creates a new policy engine with config-based rules
func NewEngine(rules []config.PolicyRule) *Engine {
	return &Engine{
		rules: rules,
	}
}

// NewEngineWithStore creates a new policy engine with database-backed storage
func NewEngineWithStore(store PolicyStore) *Engine {
	return &Engine{
		policyStore: store,
	}
}

// AuthorizeRequest checks if a request is authorized
func (e *Engine) AuthorizeRequest(gitlabProject, harborProject, permission string) error {
	// If using database store, query from database
	if e.policyStore != nil {
		rule, err := e.policyStore.GetPolicyByGitLabProject(gitlabProject)
		if err != nil {
			return fmt.Errorf("failed to fetch policy: %w", err)
		}

		// Check if Harbor project is allowed
		if !contains(rule.HarborProjects, harborProject) {
			return fmt.Errorf("no policy found for GitLab project '%s' and Harbor project '%s'", gitlabProject, harborProject)
		}

		// Check if permission is allowed
		if !contains(rule.AllowedPermissions, permission) {
			return fmt.Errorf("permission '%s' not allowed for this project", permission)
		}

		return nil
	}

	// Otherwise, use config-based rules
	for _, rule := range e.rules {
		if rule.GitLabProject == gitlabProject {
			// Check if Harbor project is allowed
			if !contains(rule.HarborProjects, harborProject) {
				continue
			}

			// Check if permission is allowed
			if !contains(rule.AllowedPerms, permission) {
				return fmt.Errorf("permission '%s' not allowed for this project", permission)
			}

			// Authorization successful
			return nil
		}
	}

	return fmt.Errorf("no policy found for GitLab project '%s' and Harbor project '%s'", gitlabProject, harborProject)
}

// contains checks if a slice contains a string
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// ValidatePermission checks if a permission value is valid
func ValidatePermission(permission string) error {
	switch permission {
	case "read", "write", "read-write":
		return nil
	default:
		return fmt.Errorf("invalid permission: must be 'read', 'write', or 'read-write'")
	}
}
