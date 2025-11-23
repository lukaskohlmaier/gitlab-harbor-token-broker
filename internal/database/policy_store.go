package database

import (
	"github.com/lukaskohlmaier/gitlab-harbor-token-broker/internal/policy"
)

// PolicyStoreAdapter adapts DB to policy.PolicyStore interface
type PolicyStoreAdapter struct {
	db *DB
}

// NewPolicyStoreAdapter creates a new PolicyStoreAdapter
func NewPolicyStoreAdapter(db *DB) *PolicyStoreAdapter {
	return &PolicyStoreAdapter{db: db}
}

// GetPolicyByGitLabProject retrieves a policy by GitLab project
func (p *PolicyStoreAdapter) GetPolicyByGitLabProject(gitlabProject string) (policy.PolicyRule, error) {
	dbPolicy, err := p.db.GetPolicyByGitLabProject(gitlabProject)
	if err != nil {
		return policy.PolicyRule{}, err
	}

	if dbPolicy == nil {
		return policy.PolicyRule{}, nil
	}

	return policy.PolicyRule{
		GitLabProject:      dbPolicy.GitLabProject,
		HarborProjects:     dbPolicy.HarborProjects,
		AllowedPermissions: dbPolicy.AllowedPermissions,
	}, nil
}
