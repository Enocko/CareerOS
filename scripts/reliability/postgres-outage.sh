#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
COMPOSE_FILE="$ROOT/infrastructure/docker/docker-compose.yml"
BASE_URL="${BASE_URL:-http://localhost:8080}"

log() { printf '[reliability] %s\n' "$*"; }

log "1) check liveness while DB up"
curl -sf "$BASE_URL/health" | tee /tmp/careeros-health-up.json
curl -sf "$BASE_URL/ready" | tee /tmp/careeros-ready-up.json

log "2) stop PostgreSQL"
docker compose -f "$COMPOSE_FILE" stop postgres >/dev/null

sleep 2
log "3) observe degraded readiness"
set +e
curl -s -o /tmp/careeros-ready-down.json -w "%{http_code}" "$BASE_URL/ready" | tee /tmp/careeros-ready-down.code
set -e
curl -s -o /tmp/careeros-health-down.json -w "%{http_code}" "$BASE_URL/health" | tee /tmp/careeros-health-down.code

log "4) restore PostgreSQL"
docker compose -f "$COMPOSE_FILE" start postgres >/dev/null
for i in $(seq 1 30); do
  if docker exec careeros-postgres pg_isready -U careeros -d careeros >/dev/null 2>&1; then
    break
  fi
  sleep 1
done

sleep 2
log "5) observe recovery"
curl -sf "$BASE_URL/ready" | tee /tmp/careeros-ready-recovered.json

log "done — inspect /tmp/careeros-*.json and *.code"
