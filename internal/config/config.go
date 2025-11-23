package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// Config represents the application configuration
type Config struct {
	Server   ServerConfig   `yaml:"server"`
	GitLab   GitLabConfig   `yaml:"gitlab"`
	Harbor   HarborConfig   `yaml:"harbor"`
	Security SecurityConfig `yaml:"security"`
	Database DatabaseConfig `yaml:"database"`
	Policies []PolicyRule   `yaml:"policies"`
}

// ServerConfig contains HTTP server settings
type ServerConfig struct {
	Port         int           `yaml:"port"`
	ReadTimeout  time.Duration `yaml:"read_timeout"`
	WriteTimeout time.Duration `yaml:"write_timeout"`
}

// GitLabConfig contains GitLab OIDC settings
type GitLabConfig struct {
	InstanceURL string   `yaml:"instance_url"`
	Audience    string   `yaml:"audience"`
	JWKSUrl     string   `yaml:"jwks_url"`
	Issuers     []string `yaml:"issuers"`
}

// HarborConfig contains Harbor API settings
type HarborConfig struct {
	URL      string `yaml:"url"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

// SecurityConfig contains security settings
type SecurityConfig struct {
	RobotTTLMinutes int `yaml:"robot_ttl_minutes"`
}

// DatabaseConfig contains database connection settings
type DatabaseConfig struct {
	ConnectionString string `yaml:"connection_string"`
	Enabled          bool   `yaml:"enabled"`
}

// PolicyRule defines authorization rules
type PolicyRule struct {
	GitLabProject  string   `yaml:"gitlab_project"`
	HarborProjects []string `yaml:"harbor_projects"`
	AllowedPerms   []string `yaml:"allowed_permissions"`
}

// Load reads and parses the configuration file
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	// Set defaults
	if cfg.Server.Port == 0 {
		cfg.Server.Port = 8080
	}
	if cfg.Server.ReadTimeout == 0 {
		cfg.Server.ReadTimeout = 10 * time.Second
	}
	if cfg.Server.WriteTimeout == 0 {
		cfg.Server.WriteTimeout = 10 * time.Second
	}
	if cfg.Security.RobotTTLMinutes == 0 {
		cfg.Security.RobotTTLMinutes = 10
	}

	// Override with environment variables if set
	if harborUser := os.Getenv("HARBOR_USERNAME"); harborUser != "" {
		cfg.Harbor.Username = harborUser
	}
	if harborPass := os.Getenv("HARBOR_PASSWORD"); harborPass != "" {
		cfg.Harbor.Password = harborPass
	}
	if dbConnStr := os.Getenv("DATABASE_URL"); dbConnStr != "" {
		cfg.Database.ConnectionString = dbConnStr
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return &cfg, nil
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	if c.GitLab.InstanceURL == "" {
		return fmt.Errorf("gitlab.instance_url is required")
	}
	if c.GitLab.Audience == "" {
		return fmt.Errorf("gitlab.audience is required")
	}
	if c.Harbor.URL == "" {
		return fmt.Errorf("harbor.url is required")
	}
	if c.Harbor.Username == "" {
		return fmt.Errorf("harbor.username is required")
	}
	if c.Harbor.Password == "" {
		return fmt.Errorf("harbor.password is required")
	}
	if c.Database.Enabled && c.Database.ConnectionString == "" {
		return fmt.Errorf("database.connection_string is required when database is enabled")
	}
	if !c.Database.Enabled && len(c.Policies) == 0 {
		return fmt.Errorf("at least one policy rule is required when database is disabled")
	}

	// Validate policy rules
	for i, rule := range c.Policies {
		if rule.GitLabProject == "" {
			return fmt.Errorf("policy[%d]: gitlab_project is required", i)
		}
		if len(rule.HarborProjects) == 0 {
			return fmt.Errorf("policy[%d]: harbor_projects must not be empty", i)
		}
		if len(rule.AllowedPerms) == 0 {
			return fmt.Errorf("policy[%d]: allowed_permissions must not be empty", i)
		}
		// Validate permissions
		for _, perm := range rule.AllowedPerms {
			if perm != "read" && perm != "write" && perm != "read-write" {
				return fmt.Errorf("policy[%d]: invalid permission '%s'", i, perm)
			}
		}
	}

	return nil
}
