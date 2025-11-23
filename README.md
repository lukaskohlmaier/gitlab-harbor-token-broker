# Harbor CI Credential Broker

A **Go-based credential brokering service** that issues **short-lived Harbor robot accounts** for use in **GitLab CI pipelines**. This eliminates the need to store long-lived Harbor credentials in CI/CD variables.

## 🎯 Purpose

Harbor requires long-lived robot account credentials or user accounts with elevated permissions. These tokens can be leaked, misused, and are difficult to rotate. The Harbor CI Credential Broker solves this by:

- **Dynamically creating** short-lived Harbor robot accounts (5-10 minute TTL)
- **Authenticating** CI pipelines using GitLab's built-in OIDC JWT (`CI_JOB_JWT_V2`)
- **Enforcing** fine-grained authorization policies
- **Auditing** all credential requests

## 🏗️ Architecture

```
GitLab CI Pipeline → [JWT Auth] → Broker → [Policy Check] → Harbor API → Robot Account
```

1. CI pipeline requests credentials with `CI_JOB_JWT_V2`
2. Broker validates JWT signature and claims
3. Broker checks authorization policy
4. Broker creates temporary Harbor robot account
5. Broker returns credentials to pipeline
6. Pipeline uses credentials to push/pull images
7. Credentials expire automatically after TTL

## ✨ Features

- **JWT Authentication**: Validates GitLab CI OIDC tokens (CI_JOB_JWT_V2)
- **Policy-Based Authorization**: Fine-grained control over which projects can access which Harbor projects
- **Least Privilege**: Robot accounts created with exact permissions requested (read/write/read-write)
- **Short-Lived Credentials**: Configurable TTL (default: 10 minutes)
- **Structured Audit Logging**: JSON logs with full audit trail
- **Graceful Shutdown**: Clean server shutdown on termination signals
- **Health Checks**: Built-in health endpoint for monitoring
- **Container Ready**: Docker image with non-root user

## 📋 Requirements

- Go 1.21 or later
- Harbor v2.x instance with admin credentials
- GitLab instance with OIDC support

## 🚀 Quick Start

### 1. Clone and Build

```bash
git clone https://github.com/lukaskohlmaier/gitlab-harbor-token-broker.git
cd gitlab-harbor-token-broker
make build
```

### 2. Configure

Edit `config.yaml`:

```yaml
server:
  port: 8080

gitlab:
  instance_url: "https://gitlab.example.com"
  audience: "https://broker.example.com"

harbor:
  url: "https://harbor.example.com"
  username: "admin"
  password: "Harbor12345"

security:
  robot_ttl_minutes: 10

policies:
  - gitlab_project: "mygroup/myproject"
    harbor_projects:
      - "backend-project"
    allowed_permissions:
      - "read"
      - "write"
```

**Important**: Store Harbor credentials securely! Use environment variables in production:

```bash
export HARBOR_USERNAME=admin
export HARBOR_PASSWORD=your-secure-password
```

### 3. Run

```bash
./broker -config config.yaml
```

Or using Make:

```bash
make run
```

## 🐳 Docker Deployment

### Build Image

```bash
make docker-build
```

Or manually:

```bash
docker build -t gitlab-harbor-token-broker:latest .
```

### Run Container

```bash
docker run -p 8080:8080 \
  -v $(pwd)/config.yaml:/app/config.yaml \
  -e HARBOR_USERNAME=admin \
  -e HARBOR_PASSWORD=your-password \
  gitlab-harbor-token-broker:latest
```

### Docker Compose Example

```yaml
version: '3.8'
services:
  broker:
    image: gitlab-harbor-token-broker:latest
    ports:
      - "8080:8080"
    volumes:
      - ./config.yaml:/app/config.yaml:ro
    environment:
      - HARBOR_USERNAME=admin
      - HARBOR_PASSWORD=${HARBOR_PASSWORD}
    restart: unless-stopped
```

## 📖 Usage in GitLab CI

### Example .gitlab-ci.yml

```yaml
variables:
  BROKER_URL: "https://broker.example.com"
  HARBOR_URL: "harbor.example.com"

build-and-push:
  stage: build
  image: docker:latest
  services:
    - docker:dind
  id_tokens:
    CI_JOB_JWT_V2:
      aud: https://broker.example.com
  script:
    # Request temporary Harbor credentials
    - |
      RESPONSE=$(curl -X POST "$BROKER_URL/token" \
        -H "Authorization: Bearer $CI_JOB_JWT_V2" \
        -H "Content-Type: application/json" \
        -d "{
          \"harbor_project\": \"backend-project\",
          \"permissions\": \"read-write\"
        }")
    
    # Extract credentials
    - export HARBOR_USERNAME=$(echo $RESPONSE | jq -r '.username')
    - export HARBOR_PASSWORD=$(echo $RESPONSE | jq -r '.password')
    
    # Login to Harbor
    - echo "$HARBOR_PASSWORD" | docker login $HARBOR_URL -u "$HARBOR_USERNAME" --password-stdin
    
    # Build and push image
    - docker build -t $HARBOR_URL/backend-project/myapp:$CI_COMMIT_SHA .
    - docker push $HARBOR_URL/backend-project/myapp:$CI_COMMIT_SHA
```

### Using a Reusable Script

Create `.gitlab/scripts/harbor-login.sh`:

```bash
#!/bin/bash
set -e

BROKER_URL="${BROKER_URL:-https://broker.example.com}"
HARBOR_PROJECT="${1}"
PERMISSIONS="${2:-read}"

# Request credentials from broker
RESPONSE=$(curl -f -X POST "$BROKER_URL/token" \
  -H "Authorization: Bearer $CI_JOB_JWT_V2" \
  -H "Content-Type: application/json" \
  -d "{\"harbor_project\": \"$HARBOR_PROJECT\", \"permissions\": \"$PERMISSIONS\"}")

# Extract and set credentials
export HARBOR_USERNAME=$(echo $RESPONSE | jq -r '.username')
export HARBOR_PASSWORD=$(echo $RESPONSE | jq -r '.password')

# Login to Harbor
echo "$HARBOR_PASSWORD" | docker login $HARBOR_URL -u "$HARBOR_USERNAME" --password-stdin

echo "Successfully logged in to Harbor (expires: $(echo $RESPONSE | jq -r '.expires_at'))"
```

Then in `.gitlab-ci.yml`:

```yaml
build:
  script:
    - source .gitlab/scripts/harbor-login.sh backend-project read-write
    - docker build -t $HARBOR_URL/backend-project/myapp:latest .
    - docker push $HARBOR_URL/backend-project/myapp:latest
```

## 🔒 Security Considerations

### JWT Validation

The broker validates:
- **Signature**: Using GitLab's JWKS endpoint
- **Issuer**: Must match configured GitLab instance
- **Audience**: Must match broker's configured audience
- **Expiration**: Token must not be expired

### Policy Enforcement

Authorization policies define:
- Which GitLab projects can access which Harbor projects
- What permissions each project can request (read/write/read-write)

Example policy that allows read-only access:

```yaml
policies:
  - gitlab_project: "engineering/frontend"
    harbor_projects:
      - "frontend-images"
    allowed_permissions:
      - "read"  # Only pull allowed
```

### Credential Security

**Never log sensitive credentials!** The broker:
- ✅ Logs audit events with project names, permissions, robot IDs
- ❌ Never logs robot passwords/secrets
- ✅ Returns credentials only in HTTP response body
- ❌ Never includes credentials in error messages

### Production Deployment

For production:

1. **Use HTTPS**: Deploy behind a reverse proxy with TLS
2. **Secure Secrets**: Use environment variables or secret stores for Harbor credentials
3. **Network Isolation**: Restrict broker access to GitLab CI network
4. **Rate Limiting**: Add rate limiting at reverse proxy level
5. **Monitoring**: Monitor audit logs for suspicious activity

## 📊 API Reference

### POST /token

Request temporary Harbor credentials.

**Headers:**
```
Authorization: Bearer <CI_JOB_JWT_V2>
Content-Type: application/json
```

**Request Body:**
```json
{
  "harbor_project": "backend-project",
  "permissions": "read-write"
}
```

**Permissions:** `read`, `write`, or `read-write`

**Success Response (200):**
```json
{
  "username": "robot$ci-temp-12345-1234567890",
  "password": "eyJhbGci...",
  "expires_at": "2024-01-01T12:15:00Z"
}
```

**Error Responses:**
- `400` - Invalid request format
- `401` - Invalid or expired JWT
- `403` - Access denied by policy
- `500` - Internal server error

### GET /health

Health check endpoint.

**Response (200):**
```
OK
```

## 🔧 Configuration Reference

### Server Section

```yaml
server:
  port: 8080                    # HTTP server port
  read_timeout: 10s             # HTTP read timeout
  write_timeout: 10s            # HTTP write timeout
```

### GitLab Section

```yaml
gitlab:
  instance_url: "https://gitlab.example.com"  # GitLab instance URL
  audience: "https://broker.example.com"      # Expected JWT audience
  jwks_url: "https://..."                     # JWKS URL (optional, auto-constructed)
  issuers:                                     # Allowed JWT issuers (optional)
    - "https://gitlab.example.com"
```

### Harbor Section

```yaml
harbor:
  url: "https://harbor.example.com"  # Harbor instance URL
  username: "admin"                   # Admin username (or use HARBOR_USERNAME env)
  password: "password"                # Admin password (or use HARBOR_PASSWORD env)
```

### Security Section

```yaml
security:
  robot_ttl_minutes: 10  # Robot account TTL in minutes (default: 10)
```

### Policy Rules

```yaml
policies:
  - gitlab_project: "group/project"      # GitLab project path
    harbor_projects:                      # Allowed Harbor projects
      - "project1"
      - "project2"
    allowed_permissions:                  # Allowed permission types
      - "read"
      - "write"
      - "read-write"
```

## 📝 Logging

The broker outputs structured JSON logs:

```json
{
  "timestamp": "2024-01-01T12:00:00Z",
  "level": "AUDIT",
  "message": "Token issued",
  "gitlab_project": "mygroup/myproject",
  "harbor_project": "backend-project",
  "permission": "read-write",
  "robot_id": 12345,
  "robot_name": "robot$ci-temp-67890-1234567890",
  "expires_at": "2024-01-01T12:10:00Z",
  "pipeline_id": "67890",
  "job_id": "12345"
}
```

## 🧪 Development

### Running Tests

```bash
make test
```

### Code Formatting

```bash
make fmt
```

### Linting

```bash
make lint
```

### Full Verification

```bash
make verify
```

## 🗺️ Project Structure

```
.
├── cmd/
│   └── broker/           # Main application entry point
│       └── main.go
├── internal/
│   ├── config/           # Configuration management
│   │   └── config.go
│   ├── jwt/              # JWT validation
│   │   └── validator.go
│   ├── policy/           # Policy engine
│   │   └── engine.go
│   ├── harbor/           # Harbor API client
│   │   └── client.go
│   ├── handler/          # HTTP handlers
│   │   └── handler.go
│   └── logging/          # Structured logging
│       └── logger.go
├── config.yaml           # Example configuration
├── Dockerfile            # Container image definition
├── Makefile              # Build automation
├── go.mod                # Go module definition
└── README.md             # This file
```

## 🤝 Contributing

Contributions are welcome! Please:

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests
5. Run `make verify`
6. Submit a pull request

## 📄 License

[Add your license here]

## 🙏 Acknowledgments

This project implements the Harbor robot account pattern for secure CI/CD credential management.

## 📞 Support

For issues and questions:
- Open an issue on GitHub
- Check existing documentation
- Review audit logs for troubleshooting

## 🔄 Version History

### v1.0.0
- Initial release
- JWT authentication with GitLab OIDC
- Policy-based authorization
- Harbor robot account creation
- Structured audit logging
- Docker support