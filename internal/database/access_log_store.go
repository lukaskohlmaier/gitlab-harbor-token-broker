package database

import (
	"time"
)

// AccessLogStoreAdapter adapts DB to logging.AccessLogStore interface
type AccessLogStoreAdapter struct {
	db *DB
}

// NewAccessLogStoreAdapter creates a new AccessLogStoreAdapter
func NewAccessLogStoreAdapter(db *DB) *AccessLogStoreAdapter {
	return &AccessLogStoreAdapter{db: db}
}

// LogAccess stores an access log entry
func (a *AccessLogStoreAdapter) LogAccess(logData interface{}) error {
	data, ok := logData.(map[string]interface{})
	if !ok {
		// Log data is in unexpected format, but don't fail the main request
		// This is a best-effort logging mechanism
		return nil
	}

	log := &AccessLog{
		Status: "unknown",
	}

	if ts, ok := data["timestamp"].(time.Time); ok {
		log.Timestamp = ts
	} else {
		log.Timestamp = time.Now()
	}

	if gp, ok := data["gitlab_project"].(string); ok {
		log.GitLabProject = gp
	}

	if hp, ok := data["harbor_project"].(string); ok {
		log.HarborProject = hp
	}

	if perm, ok := data["permission"].(string); ok {
		log.Permission = perm
	}

	if rid, ok := data["robot_id"].(int64); ok {
		log.RobotID = &rid
	}

	if rn, ok := data["robot_name"].(string); ok {
		log.RobotName = &rn
	}

	if ea, ok := data["expires_at"].(time.Time); ok {
		log.ExpiresAt = &ea
	}

	if pid, ok := data["pipeline_id"].(string); ok {
		log.PipelineID = &pid
	}

	if jid, ok := data["job_id"].(string); ok {
		log.JobID = &jid
	}

	if status, ok := data["status"].(string); ok {
		log.Status = status
	}

	if errMsg, ok := data["error_message"].(string); ok {
		log.ErrorMessage = &errMsg
	}

	return a.db.LogAccess(log)
}
