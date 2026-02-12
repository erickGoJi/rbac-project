#!/usr/bin/env bash
set -euo pipefail

if [[ -z "${API_URL:-}" ]]; then
  echo "API_URL is required" >&2
  exit 1
fi

if [[ -z "${JWT:-}" ]]; then
  echo "JWT is required" >&2
  exit 1
fi

if [[ -z "${API_KEY:-}" ]]; then
  echo "API_KEY is required" >&2
  exit 1
fi

APP_ID=${APP_ID:-app-1}
USER_ID=${USER_ID:-user-123}
ROLE_ID=${ROLE_ID:-admin}
PERM_READ=${PERM_READ:-perm:read}
PERM_WRITE=${PERM_WRITE:-perm:write}

request() {
  local method=$1
  local path=$2
  local body=${3:-}
  if [[ -n "$body" ]]; then
    curl -sS -X "$method" "$API_URL$path" \
      -H "x-api-key: $API_KEY" \
      -H "Authorization: Bearer $JWT" \
      -H "Content-Type: application/json" \
      -d "$body"
  else
    curl -sS -X "$method" "$API_URL$path" \
      -H "x-api-key: $API_KEY" \
      -H "Authorization: Bearer $JWT"
  fi
}

echo "1) Health check for auth (expect 401)"
set +e
curl -sS -o /dev/null -w "%{http_code}\n" -X GET "$API_URL/applications/$APP_ID"
set -e

echo "2) Create application"
request POST "/applications" "{\"id\":\"$APP_ID\",\"name\":\"SmokeApp\",\"description\":\"Smoke test app\"}"

echo "3) Create permission"
request POST "/applications/$APP_ID/permissions" "{\"id\":\"$PERM_READ\",\"name\":\"Read\",\"description\":\"Read permission\"}"

echo "4) Create role with permission"
request POST "/applications/$APP_ID/roles" "{\"id\":\"$ROLE_ID\",\"name\":\"Admin\",\"permissions\":[\"$PERM_READ\"]}"

echo "5) Assign role to user"
request POST "/applications/$APP_ID/users/$USER_ID/roles" "{\"role_id\":\"$ROLE_ID\"}"

echo "6) Authorize read"
request POST "/authorize" "{\"app_id\":\"$APP_ID\",\"user_id\":\"$USER_ID\",\"permission\":\"$PERM_READ\"}"

echo "7) Authorize write (expect allowed=false)"
request POST "/authorize" "{\"app_id\":\"$APP_ID\",\"user_id\":\"$USER_ID\",\"permission\":\"$PERM_WRITE\"}"

echo "8) Get app"
request GET "/applications/$APP_ID"

echo "9) List roles"
request GET "/applications/$APP_ID/roles"

echo "10) List permissions"
request GET "/applications/$APP_ID/permissions"

echo "11) Get user roles"
request GET "/applications/$APP_ID/users/$USER_ID"

echo "Smoke tests completed"
