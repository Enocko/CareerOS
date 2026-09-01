#!/usr/bin/env bash
# Populate the production opportunity catalog from your machine.
# Requires the Render *external* database URL (Dashboard → careeros-db → Connect → External).
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

set -a
if [[ -f "$ROOT/.env" ]]; then
  # shellcheck disable=SC1091
  source "$ROOT/.env"
fi
set +a

if [[ -z "${PRODUCTION_DATABASE_URL:-}" ]]; then
  echo "Set PRODUCTION_DATABASE_URL in .env or pass it on the command line." >&2
  echo "Example:" >&2
  echo "  ./scripts/ingest-production.sh" >&2
  exit 1
fi

if [[ -z "${USAJOBS_API_KEY:-}" || -z "${USAJOBS_USER_AGENT:-}" ]]; then
  echo "USAJOBS_API_KEY and USAJOBS_USER_AGENT must be set (in .env or the environment)." >&2
  exit 1
fi

# make ingest sources .env and would overwrite DATABASE_URL with localhost — run ingest directly.
export DATABASE_URL="$PRODUCTION_DATABASE_URL"

echo "Running opportunity ingest against production database..."
cd "$ROOT/apps/api"
GO="${GO:-$(command -v go)}"
if [[ -z "$GO" && -x "$HOME/.local/go/bin/go" ]]; then
  GO="$HOME/.local/go/bin/go"
fi
"$GO" run ./cmd/ingest

echo ""
echo "Ingest complete. Browse should show live opportunities after the API reads the updated catalog."
