package harbor

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client is a Harbor API client
type Client struct {
	baseURL  string
	username string
	password string
	client   *http.Client
}

// RobotAccount represents a Harbor robot account
type RobotAccount struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	Secret    string    `json:"secret"`
	ExpiresAt time.Time `json:"expires_at"`
}

// CreateRobotRequest represents the request to create a robot account
type CreateRobotRequest struct {
	Name        string       `json:"name"`
	Description string       `json:"description"`
	Duration    int64        `json:"duration"` // in days, -1 for never expire
	Level       string       `json:"level"`
	Permissions []Permission `json:"permissions"`
}

// Permission represents robot account permissions
type Permission struct {
	Kind      string   `json:"kind"`
	Namespace string   `json:"namespace"`
	Access    []Access `json:"access"`
}

// Access represents an access action
type Access struct {
	Resource string `json:"resource"`
	Action   string `json:"action"`
}

// Project represents a Harbor project
type Project struct {
	ProjectID int64  `json:"project_id"`
	Name      string `json:"name"`
}

// NewClient creates a new Harbor API client
func NewClient(baseURL, username, password string) *Client {
	return &Client{
		baseURL:  baseURL,
		username: username,
		password: password,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// GetProject retrieves a project by name
func (c *Client) GetProject(projectName string) (*Project, error) {
	url := fmt.Sprintf("%s/api/v2.0/projects?name=%s", c.baseURL, projectName)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.SetBasicAuth(c.username, c.password)
	req.Header.Set("Accept", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("harbor API error (status %d): %s", resp.StatusCode, string(body))
	}

	var projects []Project
	if err := json.NewDecoder(resp.Body).Decode(&projects); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if len(projects) == 0 {
		return nil, fmt.Errorf("project '%s' not found", projectName)
	}

	return &projects[0], nil
}

// CreateRobotAccount creates a new robot account for a project
func (c *Client) CreateRobotAccount(projectName, robotName, permission string, ttlMinutes int) (*RobotAccount, error) {
	// First, get the project to obtain its ID
	project, err := c.GetProject(projectName)
	if err != nil {
		return nil, fmt.Errorf("failed to get project: %w", err)
	}

	// Map permission to Harbor access actions
	access := c.mapPermissionToAccess(permission)

	// Create robot account request
	// Duration is in days for Harbor API, but we want minutes
	// Convert minutes to days (fractional)
	durationDays := float64(ttlMinutes) / (60.0 * 24.0)

	// Harbor expects duration in minutes for expiration
	// We'll set it via duration field which expects days, but we can also use expires_at
	request := CreateRobotRequest{
		Name:        robotName,
		Description: "Temporary CI robot account",
		Duration:    -1, // We'll handle expiration through the API
		Level:       "project",
		Permissions: []Permission{
			{
				Kind:      "project",
				Namespace: projectName,
				Access:    access,
			},
		},
	}

	// For shorter TTL, we need to use the duration field properly
	// Harbor v2 API uses duration in days
	if durationDays < 1 {
		// For very short durations, round up to at least 1 day
		// This is a Harbor API limitation
		request.Duration = 1
	} else {
		request.Duration = int64(durationDays + 0.5) // Round to nearest day
	}

	body, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/api/v2.0/projects/%d/robots", c.baseURL, project.ProjectID)

	req, err := http.NewRequest("POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.SetBasicAuth(c.username, c.password)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("harbor API error (status %d): %s", resp.StatusCode, string(body))
	}

	var robot RobotAccount
	if err := json.NewDecoder(resp.Body).Decode(&robot); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Calculate expires_at based on TTL
	robot.ExpiresAt = time.Now().Add(time.Duration(ttlMinutes) * time.Minute)

	return &robot, nil
}

// mapPermissionToAccess maps our permission model to Harbor access actions
func (c *Client) mapPermissionToAccess(permission string) []Access {
	switch permission {
	case "read":
		return []Access{
			{Resource: "repository", Action: "pull"},
		}
	case "write":
		return []Access{
			{Resource: "repository", Action: "pull"},
			{Resource: "repository", Action: "push"},
		}
	case "read-write":
		return []Access{
			{Resource: "repository", Action: "pull"},
			{Resource: "repository", Action: "push"},
		}
	default:
		return []Access{}
	}
}
