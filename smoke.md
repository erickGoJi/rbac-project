# Smoke Tests Guide

This guide covers how to run the `smoke.sh` script after deployment.

## Prerequisites

- Deployed stack with `sls deploy --stage dev`
- HTTP API endpoint URL
- Valid Cognito JWT token

## Required environment variables

```bash
export API_URL=<your_http_api_url>
export JWT=<your_jwt>
export API_KEY=<your_api_key>
```

## Optional overrides

```bash
export APP_ID=app-1
export USER_ID=user-123
export ROLE_ID=admin
export PERM_READ=perm:read
export PERM_WRITE=perm:write
```

## Run

```bash
./smoke.sh
```

## What it does

1. Verifies auth is enforced (expect 401 when no JWT).
2. Creates an application.
3. Creates a permission.
4. Creates a role with permission.
5. Assigns role to user.
6. Authorizes read permission (expect `allowed=true`).
7. Authorizes write permission (expect `allowed=false`).
8. Gets application.
9. Lists roles.
10. Lists permissions.
11. Gets user roles.

## Expected output

- Step 1 prints `401`.
- Step 6 returns JSON with `allowed: true`.
- Step 7 returns JSON with `allowed: false`.
