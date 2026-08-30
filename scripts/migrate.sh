#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
COMPOSE_FILE="$ROOT_DIR/infrastructure/docker/docker-compose.yml"

DATABASE_URL="${DATABASE_URL:-postgres://careeros:careeros@localhost:5433/careeros?sslmode=disable}"

usage() {
    echo "Usage: $0 [up|down|version|create <name>]"
    echo ""
    echo "Commands:"
    echo "  up       Apply all pending migrations"
    echo "  down     Roll back the last migration"
    echo "  version  Show current migration version"
    echo "  create   Create a new migration pair (e.g., create add_users_index)"
    exit 1
}

run_migrate() {
    docker run --rm \
        --network host \
        -v "$ROOT_DIR/apps/api/migrations:/migrations" \
        migrate/migrate:v4.18.1 \
        -path=/migrations \
        -database="$DATABASE_URL" \
        "$@"
}

case "${1:-}" in
    up)
        echo "Applying migrations..."
        run_migrate up
        echo "Migrations applied."
        ;;
    down)
        echo "Rolling back last migration..."
        run_migrate down 1
        echo "Rollback complete."
        ;;
    version)
        run_migrate version
        ;;
    create)
        if [ -z "${2:-}" ]; then
            echo "Error: migration name required"
            usage
        fi
        TIMESTAMP=$(date +%s)
        NAME="${2}"
        UP="$ROOT_DIR/apps/api/migrations/${TIMESTAMP}_${NAME}.up.sql"
        DOWN="$ROOT_DIR/apps/api/migrations/${TIMESTAMP}_${NAME}.down.sql"
        touch "$UP" "$DOWN"
        echo "Created:"
        echo "  $UP"
        echo "  $DOWN"
        ;;
    *)
        usage
        ;;
esac
