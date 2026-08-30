#!/usr/bin/env bash
# Runs development seed data migration (000002) if not already applied.
# Seed data is labeled DEVELOPMENT ONLY in the migration file.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
"$SCRIPT_DIR/migrate.sh" up
echo "Seed data applied via migration 000002_seed_dev_data."
