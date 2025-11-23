-- Create access_logs table
CREATE TABLE IF NOT EXISTS access_logs (
    id SERIAL PRIMARY KEY,
    timestamp TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    gitlab_project VARCHAR(500) NOT NULL,
    harbor_project VARCHAR(255) NOT NULL,
    permission VARCHAR(20) NOT NULL,
    robot_id BIGINT,
    robot_name VARCHAR(255),
    expires_at TIMESTAMPTZ,
    pipeline_id VARCHAR(100),
    job_id VARCHAR(100),
    status VARCHAR(50) NOT NULL DEFAULT 'success',
    error_message TEXT
);

-- Create indexes for access_logs
CREATE INDEX IF NOT EXISTS idx_access_logs_timestamp ON access_logs (timestamp);
CREATE INDEX IF NOT EXISTS idx_access_logs_gitlab_project ON access_logs (gitlab_project);
CREATE INDEX IF NOT EXISTS idx_access_logs_harbor_project ON access_logs (harbor_project);
CREATE INDEX IF NOT EXISTS idx_access_logs_status ON access_logs (status);

-- Create policy_rules table
CREATE TABLE IF NOT EXISTS policy_rules (
    id SERIAL PRIMARY KEY,
    gitlab_project VARCHAR(500) NOT NULL UNIQUE,
    harbor_projects TEXT[] NOT NULL,
    allowed_permissions TEXT[] NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Create index for policy_rules
CREATE INDEX IF NOT EXISTS idx_policy_rules_gitlab_project ON policy_rules (gitlab_project);
