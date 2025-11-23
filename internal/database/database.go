package database

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/lib/pq"
)

// DB wraps the database connection
type DB struct {
	conn *sql.DB
}

// AccessLog represents an access log entry
type AccessLog struct {
	ID            int64     `json:"id"`
	Timestamp     time.Time `json:"timestamp"`
	GitLabProject string    `json:"gitlab_project"`
	HarborProject string    `json:"harbor_project"`
	Permission    string    `json:"permission"`
	RobotID       *int64    `json:"robot_id,omitempty"`
	RobotName     *string   `json:"robot_name,omitempty"`
	ExpiresAt     *time.Time `json:"expires_at,omitempty"`
	PipelineID    *string   `json:"pipeline_id,omitempty"`
	JobID         *string   `json:"job_id,omitempty"`
	Status        string    `json:"status"`
	ErrorMessage  *string   `json:"error_message,omitempty"`
}

// PolicyRule represents a policy rule
type PolicyRule struct {
	ID                 int64     `json:"id"`
	GitLabProject      string    `json:"gitlab_project"`
	HarborProjects     []string  `json:"harbor_projects"`
	AllowedPermissions []string  `json:"allowed_permissions"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

// NewDB creates a new database connection
func NewDB(connectionString string) (*DB, error) {
	conn, err := sql.Open("postgres", connectionString)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Test connection
	if err := conn.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// Set connection pool settings
	conn.SetMaxOpenConns(25)
	conn.SetMaxIdleConns(5)
	conn.SetConnMaxLifetime(5 * time.Minute)

	return &DB{conn: conn}, nil
}

// Close closes the database connection
func (db *DB) Close() error {
	return db.conn.Close()
}

// RunMigrations executes database migrations
func (db *DB) RunMigrations(migrationSQL string) error {
	_, err := db.conn.Exec(migrationSQL)
	if err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}
	return nil
}

// LogAccess stores an access log entry
func (db *DB) LogAccess(log *AccessLog) error {
	query := `
		INSERT INTO access_logs 
		(timestamp, gitlab_project, harbor_project, permission, robot_id, robot_name, 
		 expires_at, pipeline_id, job_id, status, error_message)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING id
	`

	err := db.conn.QueryRow(
		query,
		log.Timestamp,
		log.GitLabProject,
		log.HarborProject,
		log.Permission,
		log.RobotID,
		log.RobotName,
		log.ExpiresAt,
		log.PipelineID,
		log.JobID,
		log.Status,
		log.ErrorMessage,
	).Scan(&log.ID)

	if err != nil {
		return fmt.Errorf("failed to insert access log: %w", err)
	}

	return nil
}

// GetAccessLogs retrieves access logs with pagination and optional filters
func (db *DB) GetAccessLogs(limit, offset int, filters map[string]string) ([]AccessLog, int, error) {
	// Build WHERE clause based on filters
	whereClause := ""
	args := []interface{}{}
	argCount := 1

	if gitlabProject, ok := filters["gitlab_project"]; ok && gitlabProject != "" {
		whereClause += fmt.Sprintf(" AND gitlab_project = $%d", argCount)
		args = append(args, gitlabProject)
		argCount++
	}

	if harborProject, ok := filters["harbor_project"]; ok && harborProject != "" {
		whereClause += fmt.Sprintf(" AND harbor_project = $%d", argCount)
		args = append(args, harborProject)
		argCount++
	}

	if status, ok := filters["status"]; ok && status != "" {
		whereClause += fmt.Sprintf(" AND status = $%d", argCount)
		args = append(args, status)
		argCount++
	}

	if whereClause != "" {
		whereClause = "WHERE 1=1" + whereClause
	}

	// Get total count
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM access_logs %s", whereClause)
	var total int
	err := db.conn.QueryRow(countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count access logs: %w", err)
	}

	// Get paginated results
	args = append(args, limit, offset)
	query := fmt.Sprintf(`
		SELECT id, timestamp, gitlab_project, harbor_project, permission, 
		       robot_id, robot_name, expires_at, pipeline_id, job_id, status, error_message
		FROM access_logs
		%s
		ORDER BY timestamp DESC
		LIMIT $%d OFFSET $%d
	`, whereClause, argCount, argCount+1)

	rows, err := db.conn.Query(query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query access logs: %w", err)
	}
	defer rows.Close()

	var logs []AccessLog
	for rows.Next() {
		var log AccessLog
		err := rows.Scan(
			&log.ID,
			&log.Timestamp,
			&log.GitLabProject,
			&log.HarborProject,
			&log.Permission,
			&log.RobotID,
			&log.RobotName,
			&log.ExpiresAt,
			&log.PipelineID,
			&log.JobID,
			&log.Status,
			&log.ErrorMessage,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan access log: %w", err)
		}
		logs = append(logs, log)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("error iterating access logs: %w", err)
	}

	return logs, total, nil
}

// GetPolicies retrieves all policy rules
func (db *DB) GetPolicies() ([]PolicyRule, error) {
	query := `
		SELECT id, gitlab_project, harbor_projects, allowed_permissions, created_at, updated_at
		FROM policy_rules
		ORDER BY gitlab_project
	`

	rows, err := db.conn.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query policies: %w", err)
	}
	defer rows.Close()

	var policies []PolicyRule
	for rows.Next() {
		var policy PolicyRule
		err := rows.Scan(
			&policy.ID,
			&policy.GitLabProject,
			&policy.HarborProjects,
			&policy.AllowedPermissions,
			&policy.CreatedAt,
			&policy.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan policy: %w", err)
		}
		policies = append(policies, policy)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating policies: %w", err)
	}

	return policies, nil
}

// CreatePolicy creates a new policy rule
func (db *DB) CreatePolicy(policy *PolicyRule) error {
	query := `
		INSERT INTO policy_rules (gitlab_project, harbor_projects, allowed_permissions)
		VALUES ($1, $2, $3)
		RETURNING id, created_at, updated_at
	`

	err := db.conn.QueryRow(
		query,
		policy.GitLabProject,
		policy.HarborProjects,
		policy.AllowedPermissions,
	).Scan(&policy.ID, &policy.CreatedAt, &policy.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to create policy: %w", err)
	}

	return nil
}

// UpdatePolicy updates an existing policy rule
func (db *DB) UpdatePolicy(policy *PolicyRule) error {
	query := `
		UPDATE policy_rules
		SET gitlab_project = $1, harbor_projects = $2, allowed_permissions = $3, updated_at = NOW()
		WHERE id = $4
		RETURNING updated_at
	`

	err := db.conn.QueryRow(
		query,
		policy.GitLabProject,
		policy.HarborProjects,
		policy.AllowedPermissions,
		policy.ID,
	).Scan(&policy.UpdatedAt)

	if err == sql.ErrNoRows {
		return fmt.Errorf("policy not found")
	}
	if err != nil {
		return fmt.Errorf("failed to update policy: %w", err)
	}

	return nil
}

// DeletePolicy deletes a policy rule
func (db *DB) DeletePolicy(id int64) error {
	query := `DELETE FROM policy_rules WHERE id = $1`

	result, err := db.conn.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to delete policy: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("policy not found")
	}

	return nil
}

// GetPolicyByGitLabProject retrieves a policy by GitLab project
func (db *DB) GetPolicyByGitLabProject(gitlabProject string) (*PolicyRule, error) {
	query := `
		SELECT id, gitlab_project, harbor_projects, allowed_permissions, created_at, updated_at
		FROM policy_rules
		WHERE gitlab_project = $1
	`

	var policy PolicyRule
	err := db.conn.QueryRow(query, gitlabProject).Scan(
		&policy.ID,
		&policy.GitLabProject,
		&policy.HarborProjects,
		&policy.AllowedPermissions,
		&policy.CreatedAt,
		&policy.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get policy: %w", err)
	}

	return &policy, nil
}
