#!/usr/bin/env bash
# Post-deploy verification for CareerOS on Render.
set -euo pipefail

API_URL="${API_URL:-https://careeros-api.onrender.com}"
WEB_URL="${WEB_URL:-https://careeros-web.onrender.com}"

echo "Checking API health at $API_URL/health ..."
curl -fsS "$API_URL/health" | python3 -m json.tool

echo ""
echo "Checking API readiness at $API_URL/ready ..."
curl -fsS "$API_URL/ready" | python3 -m json.tool

echo ""
echo "Checking frontend at $WEB_URL ..."
status=$(curl -s -o /dev/null -w "%{http_code}" "$WEB_URL")
if [ "$status" = "200" ]; then
  echo "Frontend OK (HTTP $status)"
else
  echo "Frontend returned HTTP $status" >&2
  exit 1
fi

echo ""
echo "Live URLs:"
echo "  App:  $WEB_URL"
echo "  API:  $API_URL"
