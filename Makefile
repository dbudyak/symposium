NAS_HOST := dima@192.168.1.200
NAS_DIR := ~/symposium
VPS_HOST := symposium@212.147.239.16
VPS_DIR := /opt/symposium

# --- Orchestrator (NAS) ---

.PHONY: orch-deploy orch-restart orch-stop orch-logs orch-status

orch-deploy: ## Sync code and rebuild orchestrator on NAS
	rsync -avz --delete orchestrator.Dockerfile docker-compose.orchestrator.yml orchestrator $(NAS_HOST):$(NAS_DIR)/
	ssh $(NAS_HOST) "cd $(NAS_DIR) && docker compose -f docker-compose.orchestrator.yml build && docker compose -f docker-compose.orchestrator.yml up -d"

orch-restart: ## Restart orchestrator without rebuilding
	ssh $(NAS_HOST) "cd $(NAS_DIR) && docker compose -f docker-compose.orchestrator.yml restart"

orch-stop: ## Stop orchestrator
	ssh $(NAS_HOST) "cd $(NAS_DIR) && docker compose -f docker-compose.orchestrator.yml down"

orch-logs: ## Tail orchestrator logs
	ssh $(NAS_HOST) "docker compose -f $(NAS_DIR)/docker-compose.orchestrator.yml logs -f --tail 30"

orch-status: ## Show orchestrator container status
	ssh $(NAS_HOST) "docker compose -f $(NAS_DIR)/docker-compose.orchestrator.yml ps"

# --- VPS (backend + frontend + caddy) ---

.PHONY: vps-deploy vps-restart vps-logs vps-status

vps-deploy: ## Sync code and rebuild all VPS services
	rsync -avz --delete --exclude='node_modules' --exclude='dist' --exclude='.env' --exclude='tsconfig.tsbuildinfo' \
		backend.Dockerfile frontend.Dockerfile Caddyfile docker-compose.yml init.sql backend frontend \
		$(VPS_HOST):$(VPS_DIR)/
	ssh $(VPS_HOST) "cd $(VPS_DIR) && docker compose build backend && docker compose build caddy && docker compose up -d"

vps-restart: ## Restart all VPS services without rebuilding
	ssh $(VPS_HOST) "cd $(VPS_DIR) && docker compose restart"

vps-logs: ## Tail all VPS logs
	ssh $(VPS_HOST) "docker compose -f $(VPS_DIR)/docker-compose.yml logs -f --tail 30"

vps-status: ## Show VPS container status + API check
	ssh $(VPS_HOST) "docker compose -f $(VPS_DIR)/docker-compose.yml ps && echo '---' && curl -s https://symposium.kodatek.app/api/status"

# --- Convenience ---

.PHONY: deploy status logs help

deploy: orch-deploy vps-deploy ## Deploy everything

status: ## Quick status of all services
	@echo "=== NAS (orchestrator) ==="
	@ssh $(NAS_HOST) "docker compose -f $(NAS_DIR)/docker-compose.orchestrator.yml ps --format '{{.Name}}\t{{.Status}}'" 2>/dev/null || echo "  unreachable"
	@echo ""
	@echo "=== VPS (backend + caddy + db) ==="
	@ssh $(VPS_HOST) "docker compose -f $(VPS_DIR)/docker-compose.yml ps --format '{{.Name}}\t{{.Status}}'" 2>/dev/null || echo "  unreachable"
	@echo ""
	@echo "=== API ==="
	@ssh $(VPS_HOST) "curl -s https://symposium.kodatek.app/api/status" 2>/dev/null || echo "  unreachable"

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-16s\033[0m %s\n", $$1, $$2}'

.DEFAULT_GOAL := help
