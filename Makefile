.PHONY: up down logs migrate migrate-down db-shell clean api-run api-test api-build ingest web-install web-run web-build

COMPOSE_FILE = infrastructure/docker/docker-compose.yml
GO           = $(HOME)/.local/go/bin/go
API_DIR      = apps/api
WEB_DIR      = apps/web

up:
	docker compose -f $(COMPOSE_FILE) up -d postgres
	@echo "Waiting for PostgreSQL to be healthy..."
	@docker compose -f $(COMPOSE_FILE) up migrate
	@echo "CareerOS database is ready on localhost:5433"

down:
	docker compose -f $(COMPOSE_FILE) down

logs:
	docker compose -f $(COMPOSE_FILE) logs -f

migrate:
	./scripts/migrate.sh up

migrate-down:
	./scripts/migrate.sh down

db-shell:
	docker exec -it careeros-postgres psql -U careeros -d careeros

clean:
	docker compose -f $(COMPOSE_FILE) down -v
	@echo "Removed containers and database volume."

api-run:
	@bash -c 'set -a; [ -f .env ] && . ./.env; set +a; cd $(API_DIR) && $(GO) run ./cmd/server'

api-test:
	@bash -c 'set -a; [ -f .env ] && . ./.env; set +a; cd $(API_DIR) && $(GO) test ./...'

api-build:
	@bash -c 'set -a; [ -f .env ] && . ./.env; set +a; cd $(API_DIR) && $(GO) build -o bin/server ./cmd/server'

ingest:
	@bash -c 'set -a; [ -f .env ] && . ./.env; set +a; cd $(API_DIR) && $(GO) run ./cmd/ingest'

relevance-report:
	@bash -c 'set -a; [ -f .env ] && . ./.env; set +a; cd $(API_DIR) && $(GO) run ./cmd/relevance-report'

worker:
	@bash -c 'set -a; [ -f .env ] && . ./.env; set +a; cd $(API_DIR) && $(GO) run ./cmd/worker'

scheduler:
	@bash -c 'set -a; [ -f .env ] && . ./.env; set +a; cd $(API_DIR) && $(GO) run ./cmd/scheduler'

scheduler-once:
	@bash -c 'set -a; [ -f .env ] && . ./.env; set +a; cd $(API_DIR) && $(GO) run ./cmd/scheduler --once'

jobs-report:
	@bash -c 'set -a; [ -f .env ] && . ./.env; set +a; cd $(API_DIR) && $(GO) run ./cmd/jobs-report'

catalog-report:
	@bash -c 'set -a; [ -f .env ] && . ./.env; set +a; cd $(API_DIR) && $(GO) run ./cmd/catalog-report'

web-install:
	cd $(WEB_DIR) && npm install

web-run:
	cd $(WEB_DIR) && npm run dev

load-test:
	@mkdir -p tests/load/results
	docker run --rm -v $(PWD)/tests/load:/scripts -e BASE_URL=$${BASE_URL:-http://host.docker.internal:8080} grafana/k6 run /scripts/baseline.js

reliability-postgres-outage:
	@bash scripts/reliability/postgres-outage.sh

query-analysis:
	@bash scripts/reliability/query-analysis.sh

