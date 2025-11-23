# Quick Start Guide: Web UI Mode

This guide will help you quickly set up the Harbor Token Broker with the web UI for managing policies and viewing access logs.

## Prerequisites

- Docker and Docker Compose
- GitLab instance with OIDC support
- Harbor instance with admin credentials

## Step 1: Clone and Configure

```bash
git clone https://github.com/lukaskohlmaier/gitlab-harbor-token-broker.git
cd gitlab-harbor-token-broker
```

Create a `.env` file with your credentials:

```bash
cat > .env << EOF
HARBOR_USERNAME=admin
HARBOR_PASSWORD=your-harbor-password
POSTGRES_PASSWORD=secure-db-password
EOF
```

## Step 2: Update Configuration

Edit `config.db.yaml` and update the GitLab settings:

```yaml
gitlab:
  instance_url: "https://your-gitlab.com"
  audience: "https://your-broker-url.com"

harbor:
  url: "https://your-harbor.com"
  # Credentials from .env file
```

## Step 3: Start Services

```bash
# Start PostgreSQL and the broker
docker-compose up -d

# Check logs
docker-compose logs -f broker
```

The broker will:
1. Connect to PostgreSQL
2. Run database migrations automatically
3. Start the API server and UI

## Step 4: Access the Web UI

Open your browser to: **http://localhost:8080**

You'll see two main sections:

### Access Logs
- View all token requests
- Filter by GitLab project, Harbor project, or status
- See timestamps, permissions, and robot account details
- Paginated view for large datasets

### Policies
- Add new authorization policies
- Edit existing policies
- Delete policies
- Configure which GitLab projects can access which Harbor projects

## Step 5: Create Your First Policy

1. Click **"Policies"** in the navigation
2. Click **"Add Policy"**
3. Fill in the form:
   - **GitLab Project**: `mygroup/myproject`
   - **Harbor Projects**: `backend-project, frontend-project`
   - **Allowed Permissions**: Select `read`, `write`, or both
4. Click **"Save"**

## Step 6: Test from GitLab CI

Add to your `.gitlab-ci.yml`:

```yaml
variables:
  BROKER_URL: "http://your-broker:8080"

build:
  stage: build
  image: docker:latest
  services:
    - docker:dind
  id_tokens:
    CI_JOB_JWT_V2:
      aud: https://your-broker-url.com
  script:
    # Request credentials from broker
    - |
      RESPONSE=$(curl -X POST "$BROKER_URL/token" \
        -H "Authorization: Bearer $CI_JOB_JWT_V2" \
        -H "Content-Type: application/json" \
        -d '{"harbor_project": "backend-project", "permissions": "read-write"}')
    
    # Use credentials
    - export HARBOR_USERNAME=$(echo $RESPONSE | jq -r '.username')
    - export HARBOR_PASSWORD=$(echo $RESPONSE | jq -r '.password')
    - echo "$HARBOR_PASSWORD" | docker login $HARBOR_URL -u "$HARBOR_USERNAME" --password-stdin
```

## Monitoring Access Logs

After running CI jobs:

1. Go to **"Access Logs"** in the UI
2. You'll see all token requests with:
   - Timestamp
   - GitLab project that requested
   - Harbor project accessed
   - Permission level
   - Success/failure status
   - Robot account details

Use filters to:
- Find requests from specific projects
- Monitor failed access attempts
- Audit token usage patterns

## Advanced Features

### Database Management

Connect to the database directly:

```bash
docker-compose exec postgres psql -U broker -d harbor_broker
```

View tables:
```sql
\dt
SELECT * FROM access_logs ORDER BY timestamp DESC LIMIT 10;
SELECT * FROM policy_rules;
```

### Backup and Restore

Backup:
```bash
docker-compose exec postgres pg_dump -U broker harbor_broker > backup.sql
```

Restore:
```bash
cat backup.sql | docker-compose exec -T postgres psql -U broker harbor_broker
```

### Development Mode

To develop the UI locally:

```bash
cd ui
npm install
VITE_API_URL=http://localhost:8080 npm run dev
```

The UI will run on http://localhost:5173 with hot-reload enabled.

## Troubleshooting

### Database Connection Failed

Check if PostgreSQL is running:
```bash
docker-compose ps
docker-compose logs postgres
```

### Migration Failed

Manually run migrations:
```bash
docker-compose exec postgres psql -U broker harbor_broker < migrations/001_initial_schema.sql
```

### UI Not Loading

Check if the broker built the UI:
```bash
docker-compose exec broker ls -la /app/ui/dist
```

Rebuild if needed:
```bash
docker-compose build --no-cache
docker-compose up -d
```

## Next Steps

- Configure TLS with a reverse proxy (nginx, Traefik)
- Set up monitoring and alerting
- Review access logs regularly for security audits
- Implement backup automation for PostgreSQL

For more information, see the main [README.md](README.md).
