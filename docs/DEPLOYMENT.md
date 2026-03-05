# Deployment

## Infrastructure

| Machine | Role | Address | User | Files |
|---------|------|---------|------|-------|
| UpCloud VPS | Backend + DB + Caddy | 212.147.239.16 | `symposium` | `/opt/symposium/` |
| Home NAS | Orchestrator + Ollama | 192.168.1.200 | `dima` | `~/symposium/` |

## Prerequisites

- SSH access to both machines (key-based)
- Docker and Docker Compose on both machines
- Ollama running on NAS with `deepseek-r1:8b` model pulled
- `.env` file on VPS at `/opt/symposium/.env` with `POSTGRES_PASSWORD` and `DOMAIN`
- `.env` file on NAS at `~/symposium/.env` with `POSTGRES_PASSWORD` and `VPS_HOST`

## Makefile Commands

Run from `lab/symposium/`:

```bash
make help              # Show all commands
make deploy            # Deploy everything (orchestrator + VPS)
make status            # Quick status of all services

# Orchestrator (NAS)
make orch-deploy       # Sync code, rebuild, restart orchestrator
make orch-restart      # Restart without rebuilding
make orch-stop         # Stop orchestrator
make orch-logs         # Tail orchestrator logs
make orch-status       # Show container status

# VPS (backend + frontend + caddy)
make vps-deploy        # Sync code, rebuild (sequentially), restart
make vps-restart       # Restart without rebuilding
make vps-logs          # Tail all VPS logs
make vps-status        # Container status + API health check
```

## How Deployment Works

### VPS (`make vps-deploy`)

1. `rsync` syncs source code to `/opt/symposium/` (excludes node_modules, dist, .env)
2. SSH builds Docker images **sequentially** (backend first, then caddy) to avoid OOM on 1GB RAM
3. `docker compose up -d` recreates changed containers

Services: `db` (PostgreSQL 16), `backend` (Go + Chi), `caddy` (Caddy + static frontend)

### Orchestrator (`make orch-deploy`)

1. `rsync` syncs orchestrator source + Dockerfiles to `~/symposium/`
2. SSH builds Docker image and restarts container

### Systemd Service (NAS)

The orchestrator runs as a systemd service for auto-start on boot:

```
/etc/systemd/system/symposium-orchestrator.service
```

- Waits 60 seconds after boot (for Ollama and Docker to start)
- Manages the Docker Compose stack
- Auto-restarts on failure

```bash
sudo systemctl start symposium-orchestrator
sudo systemctl stop symposium-orchestrator
sudo systemctl restart symposium-orchestrator
sudo systemctl status symposium-orchestrator
```

## Docker Compose Files

### `docker-compose.yml` (VPS)

Three services:
- **db**: PostgreSQL 16 with persistent volume, healthcheck, port 5432 exposed
- **backend**: Go binary, connects to db via Docker network, waits for db healthy
- **caddy**: Caddy 2 with built frontend static files, auto-TLS, reverse proxy to backend

### `docker-compose.orchestrator.yml` (NAS)

Single service:
- **orchestrator**: Go binary, connects to PostgreSQL on VPS remotely, reaches Ollama via `host.docker.internal`

## Environment Variables

### VPS `.env`

```
POSTGRES_PASSWORD=<secure-password>
DOMAIN=symposium.kodatek.app
```

### NAS `.env`

```
POSTGRES_PASSWORD=<same-password>
VPS_HOST=212.147.239.16
OLLAMA_MODEL=deepseek-r1:8b
```

## DNS

A record in Cloudflare (DNS only, not proxied):
```
symposium.kodatek.app -> 212.147.239.16
```

Caddy obtains TLS certificate automatically from Let's Encrypt.

## First-Time Setup

### VPS

```bash
# Create user
adduser symposium
usermod -aG docker symposium

# Create directory
mkdir -p /opt/symposium
chown symposium:symposium /opt/symposium

# Copy SSH keys
cp -r /root/.ssh /home/symposium/.ssh
chown -R symposium:symposium /home/symposium/.ssh

# Create .env
echo "POSTGRES_PASSWORD=<password>" > /opt/symposium/.env
echo "DOMAIN=symposium.kodatek.app" >> /opt/symposium/.env

# Deploy
make vps-deploy
```

### NAS

```bash
# Create .env
echo "POSTGRES_PASSWORD=<password>" > ~/symposium/.env
echo "VPS_HOST=212.147.239.16" >> ~/symposium/.env

# Deploy
make orch-deploy

# Install systemd service
sudo cp deploy/symposium-orchestrator.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable symposium-orchestrator
sudo systemctl start symposium-orchestrator
```
