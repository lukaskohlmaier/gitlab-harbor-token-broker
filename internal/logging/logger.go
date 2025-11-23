package logging

import (
	"encoding/json"
	"log"
	"os"
	"time"
)

// AccessLogStore interface for storing access logs
type AccessLogStore interface {
	LogAccess(log interface{}) error
}

// Logger provides structured logging
type Logger struct {
	logger         *log.Logger
	accessLogStore AccessLogStore
}

// LogEntry represents a structured log entry
type LogEntry struct {
	Timestamp      time.Time              `json:"timestamp"`
	Level          string                 `json:"level"`
	Message        string                 `json:"message"`
	GitLabProject  string                 `json:"gitlab_project,omitempty"`
	HarborProject  string                 `json:"harbor_project,omitempty"`
	Permission     string                 `json:"permission,omitempty"`
	RobotID        int64                  `json:"robot_id,omitempty"`
	RobotName      string                 `json:"robot_name,omitempty"`
	ExpiresAt      string                 `json:"expires_at,omitempty"`
	PipelineID     string                 `json:"pipeline_id,omitempty"`
	JobID          string                 `json:"job_id,omitempty"`
	Error          string                 `json:"error,omitempty"`
	AdditionalData map[string]interface{} `json:"additional_data,omitempty"`
}

// NewLogger creates a new structured logger
func NewLogger() *Logger {
	return &Logger{
		logger: log.New(os.Stdout, "", 0),
	}
}

// NewLoggerWithStore creates a new structured logger with database storage
func NewLoggerWithStore(store AccessLogStore) *Logger {
	return &Logger{
		logger:         log.New(os.Stdout, "", 0),
		accessLogStore: store,
	}
}

// Info logs an info message
func (l *Logger) Info(message string) {
	l.log("INFO", message, LogEntry{})
}

// Error logs an error message
func (l *Logger) Error(message string, err error) {
	entry := LogEntry{}
	if err != nil {
		entry.Error = err.Error()
	}
	l.log("ERROR", message, entry)
}

// AuditTokenIssued logs when a token is issued
func (l *Logger) AuditTokenIssued(gitlabProject, harborProject, permission string, robotID int64, robotName string, expiresAt time.Time, pipelineID, jobID string) {
	entry := LogEntry{
		GitLabProject: gitlabProject,
		HarborProject: harborProject,
		Permission:    permission,
		RobotID:       robotID,
		RobotName:     robotName,
		ExpiresAt:     expiresAt.Format(time.RFC3339),
		PipelineID:    pipelineID,
		JobID:         jobID,
	}
	l.log("AUDIT", "Token issued", entry)

	// If database store is available, persist to database
	if l.accessLogStore != nil {
		dbLog := map[string]interface{}{
			"timestamp":      time.Now(),
			"gitlab_project": gitlabProject,
			"harbor_project": harborProject,
			"permission":     permission,
			"robot_id":       robotID,
			"robot_name":     robotName,
			"expires_at":     expiresAt,
			"pipeline_id":    pipelineID,
			"job_id":         jobID,
			"status":         "success",
		}
		_ = l.accessLogStore.LogAccess(dbLog)
	}
}

// AuditRequestDenied logs when a request is denied
func (l *Logger) AuditRequestDenied(gitlabProject, harborProject, permission, reason string) {
	entry := LogEntry{
		GitLabProject: gitlabProject,
		HarborProject: harborProject,
		Permission:    permission,
		Error:         reason,
	}
	l.log("AUDIT", "Request denied", entry)

	// If database store is available, persist to database
	if l.accessLogStore != nil {
		dbLog := map[string]interface{}{
			"timestamp":       time.Now(),
			"gitlab_project":  gitlabProject,
			"harbor_project":  harborProject,
			"permission":      permission,
			"status":          "denied",
			"error_message":   reason,
		}
		_ = l.accessLogStore.LogAccess(dbLog)
	}
}

// log writes a structured log entry
func (l *Logger) log(level, message string, entry LogEntry) {
	entry.Timestamp = time.Now()
	entry.Level = level
	entry.Message = message

	data, err := json.Marshal(entry)
	if err != nil {
		l.logger.Printf("Failed to marshal log entry: %v", err)
		return
	}

	l.logger.Println(string(data))
}

// LogWithFields logs a message with additional fields
func (l *Logger) LogWithFields(level, message string, fields map[string]interface{}) {
	entry := LogEntry{
		AdditionalData: fields,
	}
	l.log(level, message, entry)
}
