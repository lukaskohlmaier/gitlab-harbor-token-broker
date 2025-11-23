package policy

import (
	"fmt"

	"github.com/lukaskohlmaier/gitlab-harbor-token-broker/internal/config"
)

// Engine enforces authorization policies
type Engine struct {
	rules []config.PolicyRule
}

// NewEngine creates a new policy engine
func NewEngine(rules []config.PolicyRule) *Engine {
	return &Engine{
		rules: rules,
	}
}

// AuthorizeRequest checks if a request is authorized
func (e *Engine) AuthorizeRequest(gitlabProject, harborProject, permission string) error {
	// Find matching policy rule
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
