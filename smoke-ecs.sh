#!/usr/bin/env bash
set -euo pipefail

if [[ -z "${API_URL:-}" ]]; then
  echo "API_URL is required" >&2
  exit 1
fi

APP_ID=${APP_ID:-app-ecs-1}
USER_ID=${USER_ID:-user-ecs-1}
ROLE_ID=${ROLE_ID:-admin}
PERM_READ=${PERM_READ:-perm:read}
PERM_WRITE=${PERM_WRITE:-perm:write}
EXPECT_HEALTH_STATUS=${EXPECT_HEALTH_STATUS:-200}

headers=()
if [[ -n "${API_KEY:-}" ]]; then
  headers+=("-H" "x-api-key: ${API_KEY}")
fi
if [[ -n "${JWT:-}" ]]; then
  headers+=("-H" "Authorization: Bearer ${JWT}")
fi

request() {
  local method=$1
  local path=$2
  local body=${3:-}
  if [[ -n "$body" ]]; then
    curl -sS -X "$method" "$API_URL$path" \
      "${headers[@]}" \
      -H "Content-Type: application/json" \
      -d "$body"
  else
    curl -sS -X "$method" "$API_URL$path" \
      "${headers[@]}"
  fi
}

echo "1) Health check /health"
health_status=$(curl -sS -o /dev/null -w "%{http_code}" -X GET "$API_URL/health")
echo "HTTP $health_status"
if [[ "$health_status" != "$EXPECT_HEALTH_STATUS" ]]; then
  echo "Unexpected /health status. expected=$EXPECT_HEALTH_STATUS got=$health_status" >&2
  exit 1
fi

echo "2) Create application"
request POST "/applications" "{\"id\":\"$APP_ID\",\"name\":\"Smoke ECS\",\"description\":\"Smoke ECS app\"}"

echo "3) Create permission"
request POST "/applications/$APP_ID/permissions" "{\"id\":\"$PERM_READ\",\"name\":\"Read\",\"description\":\"Read permission\"}"

echo "4) Create role with permission"
request POST "/applications/$APP_ID/roles" "{\"id\":\"$ROLE_ID\",\"name\":\"Admin\",\"permissions\":[\"$PERM_READ\"]}"

echo "5) Assign role to user"
request POST "/applications/$APP_ID/users/$USER_ID/roles" "{\"role_id\":\"$ROLE_ID\"}"

echo "6) Authorize read"
request POST "/authorize" "{\"app_id\":\"$APP_ID\",\"user_id\":\"$USER_ID\",\"permission\":\"$PERM_READ\"}"

echo "7) Authorize write"
request POST "/authorize" "{\"app_id\":\"$APP_ID\",\"user_id\":\"$USER_ID\",\"permission\":\"$PERM_WRITE\"}"

echo "8) Read back entities"
request GET "/applications/$APP_ID"
request GET "/applications/$APP_ID/roles"
request GET "/applications/$APP_ID/permissions"
request GET "/applications/$APP_ID/users/$USER_ID"

echo "ECS smoke tests completed"
