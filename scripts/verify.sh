#!/usr/bin/env bash
# Verification script for Milestone 1 Steps 1-3
set -euo pipefail

API_URL="${API_URL:-http://localhost:8080}"
PASS=0
FAIL=0
SKIP=0

pass() { echo "  ✅ $1"; PASS=$((PASS + 1)); }
fail() { echo "  ❌ $1"; FAIL=$((FAIL + 1)); }
skip() { echo "  ⏭️  $1 (not implemented yet)"; SKIP=$((SKIP + 1)); }

check_status() {
    local method="$1" path="$2" expected="$3" desc="$4"
    local extra_args="${5:-}"
    local actual
    actual=$(curl -s -o /dev/null -w "%{http_code}" -X "$method" $extra_args "$API_URL$path")
    if [ "$actual" = "$expected" ]; then
        pass "$desc → $expected"
    else
        fail "$desc → expected $expected, got $actual"
    fi
}

check_json_field() {
    local path="$1" field="$2" expected="$3" desc="$4"
    local actual
    actual=$(curl -s "$API_URL$path" | python3 -c "import sys,json; print(json.load(sys.stdin).get('$field',''))" 2>/dev/null || echo "PARSE_ERROR")
    if [ "$actual" = "$expected" ]; then
        pass "$desc → $expected"
    else
        fail "$desc → expected '$expected', got '$actual'"
    fi
}

echo "============================================"
echo "CareerOS Verification — Milestone 1 API (Steps 1-7)"
echo "============================================"
echo ""

# --- Unit & Integration Tests ---
echo "📋 Automated Tests"
export PATH="$HOME/.local/go/bin:$PATH"
cd "$(dirname "$0")/.."
if make api-test > /tmp/careeros-test.log 2>&1; then
    pass "make api-test — all tests pass"
else
    fail "make api-test — tests failed (see /tmp/careeros-test.log)"
fi
echo ""

# --- Database ---
echo "🗄️  Database"
if docker exec careeros-postgres pg_isready -U careeros -d careeros > /dev/null 2>&1; then
    pass "PostgreSQL is healthy"
else
    fail "PostgreSQL is not reachable"
fi

MIG_VERSION=$(docker exec careeros-postgres psql -U careeros -d careeros -tAc "SELECT version FROM schema_migrations")
if [ "$MIG_VERSION" = "3" ]; then
    pass "Migrations at version 3"
else
    fail "Migration version expected 3, got $MIG_VERSION"
fi

TABLE_COUNT=$(docker exec careeros-postgres psql -U careeros -d careeros -tAc "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema='public' AND table_type='BASE TABLE' AND table_name != 'schema_migrations'")
if [ "$TABLE_COUNT" = "9" ]; then
    pass "9 application tables exist"
else
    fail "Expected 9 tables, got $TABLE_COUNT"
fi

VERIFIED_COUNT=$(docker exec careeros-postgres psql -U careeros -d careeros -tAc "SELECT COUNT(*) FROM opportunities WHERE verification_status='verified' AND status='open'")
if [ "$VERIFIED_COUNT" -ge 1 ]; then
    pass "At least 1 source-verified opportunity in catalog ($VERIFIED_COUNT)"
else
    fail "Expected at least 1 verified opportunity, got $VERIFIED_COUNT (run: make ingest)"
fi

SEED_UNVERIFIED=$(docker exec careeros-postgres psql -U careeros -d careeros -tAc "SELECT COUNT(*) FROM opportunities WHERE source='dev_seed' AND verification_status='unverified'")
if [ "$SEED_UNVERIFIED" = "10" ]; then
    pass "10 dev seed opportunities marked unverified"
else
    fail "Expected 10 unverified seed opportunities, got $SEED_UNVERIFIED"
fi

# Test unique constraint (run as separate statements)
docker exec careeros-postgres psql -U careeros -d careeros -c "INSERT INTO users (email, password_hash) VALUES ('dup@test.com', 'hash');" > /dev/null 2>&1
DUP_RESULT=$(docker exec careeros-postgres psql -U careeros -d careeros -c "INSERT INTO users (email, password_hash) VALUES ('dup@test.com', 'hash2');" 2>&1 || true)
if echo "$DUP_RESULT" | grep -qi "unique"; then
    pass "Unique email constraint enforced"
    docker exec careeros-postgres psql -U careeros -d careeros -c "DELETE FROM users WHERE email = 'dup@test.com';" > /dev/null 2>&1
else
    fail "Unique email constraint not working"
    docker exec careeros-postgres psql -U careeros -d careeros -c "DELETE FROM users WHERE email = 'dup@test.com';" > /dev/null 2>&1
fi
echo ""

# --- Implemented Endpoints ---
echo "🌐 API Endpoints (implemented)"
check_json_field "/health" "status" "ok" "GET /health status"
check_json_field "/health" "database" "connected" "GET /health database"
check_status "GET" "/health" "200" "GET /health HTTP status"

# Request ID header
REQ_ID=$(curl -s -D - -o /dev/null "$API_URL/health" | grep -i "x-request-id" | tr -d '\r')
if [ -n "$REQ_ID" ]; then
    pass "X-Request-ID header present"
else
    fail "X-Request-ID header missing"
fi

# CORS preflight
CORS_STATUS=$(curl -s -o /dev/null -w "%{http_code}" -X OPTIONS \
    -H "Origin: http://localhost:5173" \
    -H "Access-Control-Request-Method: GET" \
    "$API_URL/health")
if [ "$CORS_STATUS" = "204" ]; then
    pass "CORS preflight → 204"
else
    fail "CORS preflight → expected 204, got $CORS_STATUS"
fi

CORS_HEADER=$(curl -s -D - -o /dev/null \
    -H "Origin: http://localhost:5173" \
    "$API_URL/health" | grep -i "access-control-allow-origin" | tr -d '\r')
if echo "$CORS_HEADER" | grep -q "localhost:5173"; then
    pass "CORS Allow-Origin header set"
else
    fail "CORS Allow-Origin header missing"
fi
echo ""

# --- Auth Endpoints ---
echo "🔐 Auth Endpoints"
TEST_EMAIL="verify-$(date +%s)@gram.edu"
TEST_PASS="securepass123"

REGISTER_RESP=$(curl -s -w "\n%{http_code}" -X POST "$API_URL/api/v1/auth/register" \
    -H "Content-Type: application/json" \
    -d "{\"email\":\"$TEST_EMAIL\",\"password\":\"$TEST_PASS\"}")
REGISTER_CODE=$(echo "$REGISTER_RESP" | tail -1)
REGISTER_BODY=$(echo "$REGISTER_RESP" | sed '$d')

if [ "$REGISTER_CODE" = "201" ]; then
    pass "POST /api/v1/auth/register → 201"
else
    fail "POST /api/v1/auth/register → expected 201, got $REGISTER_CODE"
fi

TOKEN=$(echo "$REGISTER_BODY" | python3 -c "import sys,json; print(json.load(sys.stdin).get('token',''))" 2>/dev/null)
if [ -n "$TOKEN" ]; then
    pass "Register returns JWT token"
else
    fail "Register response missing token"
fi

DUP_CODE=$(curl -s -o /dev/null -w "%{http_code}" -X POST "$API_URL/api/v1/auth/register" \
    -H "Content-Type: application/json" \
    -d "{\"email\":\"$TEST_EMAIL\",\"password\":\"$TEST_PASS\"}")
if [ "$DUP_CODE" = "409" ]; then
    pass "Duplicate register → 409"
else
    fail "Duplicate register → expected 409, got $DUP_CODE"
fi

LOGIN_CODE=$(curl -s -o /dev/null -w "%{http_code}" -X POST "$API_URL/api/v1/auth/login" \
    -H "Content-Type: application/json" \
    -d "{\"email\":\"$TEST_EMAIL\",\"password\":\"$TEST_PASS\"}")
if [ "$LOGIN_CODE" = "200" ]; then
    pass "POST /api/v1/auth/login → 200"
else
    fail "POST /api/v1/auth/login → expected 200, got $LOGIN_CODE"
fi

BAD_LOGIN_CODE=$(curl -s -o /dev/null -w "%{http_code}" -X POST "$API_URL/api/v1/auth/login" \
    -H "Content-Type: application/json" \
    -d "{\"email\":\"$TEST_EMAIL\",\"password\":\"wrongpass1\"}")
if [ "$BAD_LOGIN_CODE" = "401" ]; then
    pass "Invalid login → 401"
else
    fail "Invalid login → expected 401, got $BAD_LOGIN_CODE"
fi

ME_CODE=$(curl -s -o /dev/null -w "%{http_code}" -X GET "$API_URL/api/v1/auth/me" \
    -H "Authorization: Bearer $TOKEN")
if [ "$ME_CODE" = "200" ]; then
    pass "GET /api/v1/auth/me (authenticated) → 200"
else
    fail "GET /api/v1/auth/me → expected 200, got $ME_CODE"
fi

ME_UNAUTH_CODE=$(curl -s -o /dev/null -w "%{http_code}" -X GET "$API_URL/api/v1/auth/me")
if [ "$ME_UNAUTH_CODE" = "401" ]; then
    pass "GET /api/v1/auth/me (unauthenticated) → 401"
else
    fail "GET /api/v1/auth/me unauthenticated → expected 401, got $ME_UNAUTH_CODE"
fi

VALIDATION_CODE=$(curl -s -o /dev/null -w "%{http_code}" -X POST "$API_URL/api/v1/auth/register" \
    -H "Content-Type: application/json" \
    -d '{"email":"bad","password":"short"}')
if [ "$VALIDATION_CODE" = "400" ]; then
    pass "Invalid register input → 400"
else
    fail "Invalid register input → expected 400, got $VALIDATION_CODE"
fi
echo ""

# --- Profile Endpoints ---
echo "👤 Profile Endpoints"
PROFILE_GET_BEFORE=$(curl -s -o /dev/null -w "%{http_code}" -X GET "$API_URL/api/v1/profile" \
    -H "Authorization: Bearer $TOKEN")
if [ "$PROFILE_GET_BEFORE" = "404" ]; then
    pass "GET /api/v1/profile (no profile yet) → 404"
else
    fail "GET /api/v1/profile before create → expected 404, got $PROFILE_GET_BEFORE"
fi

PROFILE_PUT_CODE=$(curl -s -o /dev/null -w "%{http_code}" -X PUT "$API_URL/api/v1/profile" \
    -H "Authorization: Bearer $TOKEN" \
    -H "Content-Type: application/json" \
    -d '{"first_name":"Jordan","last_name":"Smith","major":"Computer Science","graduation_year":2027,"skills":["Python","Go"],"work_arrangement":"remote","experience_level":"intern"}')
if [ "$PROFILE_PUT_CODE" = "200" ]; then
    pass "PUT /api/v1/profile (create) → 200"
else
    fail "PUT /api/v1/profile → expected 200, got $PROFILE_PUT_CODE"
fi

PROFILE_GET_AFTER=$(curl -s -X GET "$API_URL/api/v1/profile" \
    -H "Authorization: Bearer $TOKEN")
PROFILE_MAJOR=$(echo "$PROFILE_GET_AFTER" | python3 -c "import sys,json; print(json.load(sys.stdin).get('major',''))" 2>/dev/null)
if [ "$PROFILE_MAJOR" = "Computer Science" ]; then
    pass "GET /api/v1/profile (after create) → 200 with data"
else
    fail "GET /api/v1/profile → expected major 'Computer Science', got '$PROFILE_MAJOR'"
fi

PROFILE_UNAUTH=$(curl -s -o /dev/null -w "%{http_code}" -X GET "$API_URL/api/v1/profile")
if [ "$PROFILE_UNAUTH" = "401" ]; then
    pass "GET /api/v1/profile (unauthenticated) → 401"
else
    fail "GET /api/v1/profile unauthenticated → expected 401, got $PROFILE_UNAUTH"
fi

PROFILE_INVALID=$(curl -s -o /dev/null -w "%{http_code}" -X PUT "$API_URL/api/v1/profile" \
    -H "Authorization: Bearer $TOKEN" \
    -H "Content-Type: application/json" \
    -d '{"graduation_year":2010}')
if [ "$PROFILE_INVALID" = "400" ]; then
    pass "Invalid profile input → 400"
else
    fail "Invalid profile input → expected 400, got $PROFILE_INVALID"
fi
echo ""

# --- Opportunity Endpoints ---
echo "💼 Opportunity Endpoints"
OPP_LIST_CODE=$(curl -s -o /dev/null -w "%{http_code}" -X GET "$API_URL/api/v1/opportunities" \
    -H "Authorization: Bearer $TOKEN")
if [ "$OPP_LIST_CODE" = "200" ]; then
    pass "GET /api/v1/opportunities → 200"
else
    fail "GET /api/v1/opportunities → expected 200, got $OPP_LIST_CODE"
fi

OPP_TOTAL=$(curl -s -X GET "$API_URL/api/v1/opportunities" \
    -H "Authorization: Bearer $TOKEN" \
    | python3 -c "import sys,json; print(json.load(sys.stdin)['pagination']['total'])" 2>/dev/null)
if [ "$OPP_TOTAL" -ge 10 ] 2>/dev/null; then
    pass "Opportunities list has seed data (total=$OPP_TOTAL)"
else
    fail "Expected at least 10 seed opportunities, got $OPP_TOTAL"
fi

OPP_FILTER_CODE=$(curl -s -o /dev/null -w "%{http_code}" -X GET "$API_URL/api/v1/opportunities?category=internship" \
    -H "Authorization: Bearer $TOKEN")
if [ "$OPP_FILTER_CODE" = "200" ]; then
    pass "GET /api/v1/opportunities?category=internship → 200"
else
    fail "Filter by category → expected 200, got $OPP_FILTER_CODE"
fi

OPP_SEARCH_CODE=$(curl -s -o /dev/null -w "%{http_code}" -X GET "$API_URL/api/v1/opportunities?q=Google" \
    -H "Authorization: Bearer $TOKEN")
if [ "$OPP_SEARCH_CODE" = "200" ]; then
    pass "GET /api/v1/opportunities?q=Google → 200"
else
    fail "Search → expected 200, got $OPP_SEARCH_CODE"
fi

OPP_ID=$(curl -s -X GET "$API_URL/api/v1/opportunities" \
    -H "Authorization: Bearer $TOKEN" \
    | python3 -c "import sys,json; print(json.load(sys.stdin)['data'][0]['id'])" 2>/dev/null)
OPP_DETAIL_CODE=$(curl -s -o /dev/null -w "%{http_code}" -X GET "$API_URL/api/v1/opportunities/$OPP_ID" \
    -H "Authorization: Bearer $TOKEN")
if [ "$OPP_DETAIL_CODE" = "200" ]; then
    pass "GET /api/v1/opportunities/:id → 200"
else
    fail "GET /api/v1/opportunities/:id → expected 200, got $OPP_DETAIL_CODE"
fi

OPP_NOTFOUND_CODE=$(curl -s -o /dev/null -w "%{http_code}" -X GET "$API_URL/api/v1/opportunities/00000000-0000-4000-8000-000000000099" \
    -H "Authorization: Bearer $TOKEN")
if [ "$OPP_NOTFOUND_CODE" = "404" ]; then
    pass "GET /api/v1/opportunities/:id (not found) → 404"
else
    fail "Opportunity not found → expected 404, got $OPP_NOTFOUND_CODE"
fi

OPP_UNAUTH_CODE=$(curl -s -o /dev/null -w "%{http_code}" -X GET "$API_URL/api/v1/opportunities")
if [ "$OPP_UNAUTH_CODE" = "401" ]; then
    pass "GET /api/v1/opportunities (unauthenticated) → 401"
else
    fail "Opportunities unauthenticated → expected 401, got $OPP_UNAUTH_CODE"
fi
echo ""

# --- Saved & Application Endpoints ---
echo "📌 Saved & Application Endpoints"

SAVE_CODE=$(curl -s -o /dev/null -w "%{http_code}" -X POST "$API_URL/api/v1/opportunities/$OPP_ID/save" \
    -H "Authorization: Bearer $TOKEN")
if [ "$SAVE_CODE" = "200" ]; then
    pass "POST /api/v1/opportunities/:id/save → 200"
else
    fail "Save opportunity → expected 200, got $SAVE_CODE"
fi

SAVE_AGAIN_CODE=$(curl -s -o /dev/null -w "%{http_code}" -X POST "$API_URL/api/v1/opportunities/$OPP_ID/save" \
    -H "Authorization: Bearer $TOKEN")
if [ "$SAVE_AGAIN_CODE" = "200" ]; then
    pass "POST /api/v1/opportunities/:id/save (idempotent) → 200"
else
    fail "Idempotent save → expected 200, got $SAVE_AGAIN_CODE"
fi

SAVED_LIST_CODE=$(curl -s -o /dev/null -w "%{http_code}" -X GET "$API_URL/api/v1/saved-opportunities" \
    -H "Authorization: Bearer $TOKEN")
if [ "$SAVED_LIST_CODE" = "200" ]; then
    pass "GET /api/v1/saved-opportunities → 200"
else
    fail "List saved → expected 200, got $SAVED_LIST_CODE"
fi

APP_CREATE=$(curl -s -w "\n%{http_code}" -X POST "$API_URL/api/v1/applications" \
    -H "Authorization: Bearer $TOKEN" \
    -H "Content-Type: application/json" \
    -d "{\"opportunity_id\":\"$OPP_ID\",\"notes\":\"Found via verify script\"}")
APP_CREATE_CODE=$(echo "$APP_CREATE" | tail -1)
APP_CREATE_BODY=$(echo "$APP_CREATE" | sed '$d')

if [ "$APP_CREATE_CODE" = "201" ]; then
    pass "POST /api/v1/applications → 201"
else
    fail "Create application → expected 201, got $APP_CREATE_CODE"
fi

APP_ID=$(echo "$APP_CREATE_BODY" | python3 -c "import sys,json; print(json.load(sys.stdin).get('id',''))" 2>/dev/null)

APP_DUP_CODE=$(curl -s -o /dev/null -w "%{http_code}" -X POST "$API_URL/api/v1/applications" \
    -H "Authorization: Bearer $TOKEN" \
    -H "Content-Type: application/json" \
    -d "{\"opportunity_id\":\"$OPP_ID\",\"notes\":\"duplicate\"}")
if [ "$APP_DUP_CODE" = "409" ]; then
    pass "Duplicate application → 409"
else
    fail "Duplicate application → expected 409, got $APP_DUP_CODE"
fi

APP_PATCH_CODE=$(curl -s -o /dev/null -w "%{http_code}" -X PATCH "$API_URL/api/v1/applications/$APP_ID" \
    -H "Authorization: Bearer $TOKEN" \
    -H "Content-Type: application/json" \
    -d '{"current_status":"applied","notes":"Submitted via company portal"}')
if [ "$APP_PATCH_CODE" = "200" ]; then
    pass "PATCH /api/v1/applications/:id → 200"
else
    fail "Update application → expected 200, got $APP_PATCH_CODE"
fi

APP_LIST_CODE=$(curl -s -o /dev/null -w "%{http_code}" -X GET "$API_URL/api/v1/applications" \
    -H "Authorization: Bearer $TOKEN")
if [ "$APP_LIST_CODE" = "200" ]; then
    pass "GET /api/v1/applications (dashboard) → 200"
else
    fail "Application dashboard → expected 200, got $APP_LIST_CODE"
fi

APP_DETAIL=$(curl -s -X GET "$API_URL/api/v1/applications/$APP_ID" \
    -H "Authorization: Bearer $TOKEN")
HISTORY_COUNT=$(echo "$APP_DETAIL" | python3 -c "import sys,json; print(len(json.load(sys.stdin).get('status_history',[])))" 2>/dev/null)
if [ "$HISTORY_COUNT" -ge 2 ] 2>/dev/null; then
    pass "GET /api/v1/applications/:id with status history → 200"
else
    fail "Application detail → expected 2+ history entries, got $HISTORY_COUNT"
fi

UNSAVED_CODE=$(curl -s -o /dev/null -w "%{http_code}" -X DELETE "$API_URL/api/v1/opportunities/$OPP_ID/save" \
    -H "Authorization: Bearer $TOKEN")
if [ "$UNSAVED_CODE" = "204" ]; then
    pass "DELETE /api/v1/opportunities/:id/save → 204"
else
    fail "Unsave opportunity → expected 204, got $UNSAVED_CODE"
fi
echo ""

# --- Degraded health check ---
echo "🔻 Failure Mode: database disconnected"
docker stop careeros-postgres > /dev/null 2>&1
sleep 1
DEGRADED=$(curl -s -o /dev/null -w "%{http_code}" "$API_URL/health")
if [ "$DEGRADED" = "503" ]; then
    pass "GET /health returns 503 when DB is down"
else
    fail "GET /health expected 503 when DB down, got $DEGRADED"
fi
DEGRADED_STATUS=$(curl -s "$API_URL/health" | python3 -c "import sys,json; print(json.load(sys.stdin).get('status',''))" 2>/dev/null)
if [ "$DEGRADED_STATUS" = "degraded" ]; then
    pass "Health status is 'degraded' when DB is down"
else
    fail "Health status expected 'degraded', got '$DEGRADED_STATUS'"
fi
docker start careeros-postgres > /dev/null 2>&1
sleep 2
echo ""

# --- Summary ---
echo "============================================"
echo "Results: $PASS passed, $FAIL failed, $SKIP skipped (not implemented)"
echo "============================================"

if [ "$FAIL" -gt 0 ]; then
    exit 1
fi
