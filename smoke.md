# ECS Smoke Tests Guide

This guide covers how to run `smoke-ecs.sh` against the API Gateway endpoint deployed from Serverless Compose.

## Prerequisites

- Infrastructure deployed from `infrastructure/serverless-compose.yml`.
- Reachable API Gateway HTTP API endpoint.
- Optional API key and/or JWT depending on your auth mode.

## Required environment variables

```bash
export API_URL=<http_api_endpoint>
```

## Common optional variables

```bash
export BASE_PATH=
export API_KEY=<optional_api_key>
export JWT=<optional_jwt>
export APP_ID=app-ecs-1
export USER_ID=user-ecs-1
export ROLE_ID=admin
export PERM_READ=perm:read
export PERM_WRITE=perm:write
```

## Authorization expectation variables

Use these when `AUTHORIZE_TEST_MODE=true` or when you want custom checks:

```bash
export EXPECT_READ_ALLOWED=true
export EXPECT_WRITE_ALLOWED=false
```

## Run

```bash
./smoke-ecs.sh
```

## What it validates

1. `GET /health` returns expected status (`EXPECT_HEALTH_STATUS`, default `200`).
2. Creates application, permission, role, and user-role assignment.
3. Calls `POST /authorize` for read and validates `allowed`.
4. Calls `POST /authorize` for write and validates `allowed`.
5. Reads back application, roles, permissions, and user-role data.

## Notes

- If you use a non-default stage path, set it explicitly (example: `BASE_PATH=/dev`).
- If `AUTH_MODE=none`, API key and JWT are not required.
- If `AUTHORIZE_TEST_MODE=true`, set both expected values to `true`.
