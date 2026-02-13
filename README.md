# RBAC Service (Go + Hexagonal + ECS Fargate)

RBAC service in Go with hexagonal architecture, Echo HTTP server, DynamoDB single-table design, configurable authentication middleware, and AWS X-Ray instrumentation.

## Architecture

### Layers
- `internal/domain`: entities and domain errors.
- `internal/ports`: repository interfaces.
- `internal/application`: use cases and business services.
- `internal/infrastructure`: technical adapters (DynamoDB, Cognito JWT).
- `internal/adapters/http`: auth middleware (`AUTH_MODE`).
- `internal/interfaces/http`: Echo handlers and route registration.
- `cmd/bootstrap`: application composition and HTTP startup.

### Runtime model
- Standard HTTP app listening on `:8080`.
- Health endpoint: `GET /health`.
- All RBAC endpoints served by the same Echo app.
- `AUTHORIZE_TEST_MODE=true` forces `/authorize` to return `{"allowed": true}` for smoke/integration testing.

## Endpoints

- `GET /health`
- `POST /applications`
- `PUT /applications/{id}`
- `GET /applications/{id}`
- `POST /applications/{app_id}/roles`
- `PUT /applications/{app_id}/roles/{role_id}`
- `GET /applications/{app_id}/roles`
- `POST /applications/{app_id}/permissions`
- `GET /applications/{app_id}/permissions`
- `POST /applications/{app_id}/users/{user_id}/roles`
- `GET /applications/{app_id}/users/{user_id}`
- `POST /authorize`

## Authentication modes

Controlled by `AUTH_MODE`:
- `none`: no auth checks in middleware.
- `api_key`: no auth checks in app (gateway/infrastructure enforces it if configured).
- `cognito`: validates JWT with Cognito JWK and injects `user_id` from `sub`.

`AUTHORIZE_TEST_MODE`:
- `true`: `/authorize` short-circuits to allow requests.
- `false`: normal authorization flow using services/repositories.

## Local build and run

### Build binary
```bash
make build
```

### Docker build
```bash
docker build -t rbac-service .
```

### Docker run
```bash
docker run --rm -p 8080:8080 \
  -e TABLE_NAME=rbac-dev \
  -e AWS_REGION=us-east-1 \
  -e AUTH_MODE=none \
  -e AUTHORIZE_TEST_MODE=true \
  rbac-service
```

## Infrastructure stacks

Infrastructure is split into:
- `infrastructure/serverless-compose.yml`
- `infrastructure/network/serverless.yml`
- `infrastructure/dynamodb/serverless.yml`
- `infrastructure/ecr/serverless.yml`
- `infrastructure/alb/serverless.yml`
- `infrastructure/ecs/serverless.yml`
- `infrastructure/apigateway/serverless.yml`

### Infrastructure Deployment
```bash
cd infrastructure
serverless deploy
```

To remove all compose stacks:
```bash
cd infrastructure
serverless remove
```

### Deploy one stack independently
```bash
cd infrastructure/network
serverless deploy
```

### Build/push container image
```bash
make ecr-release AWS_REGION=us-east-1 AWS_ACCOUNT_ID=<account_id> IMAGE_TAG=latest DOCKER_PLATFORM=linux/arm64
```

## AWS components created by compose stacks

- VPC networking resources and security groups.
- DynamoDB table (`rbac-dev`).
- ECR repository (`rbac-dev-service`).
- Internal ALB + listener + target group.
- ECS Fargate cluster, task definition, service, autoscaling.
- API Gateway HTTP API + VPC Link to ALB listener.
- ECS target tracking autoscaling (`min=1`, `max=3`, `CPU=60%`).
- X-Ray sidecar daemon container in ECS task.

## X-Ray

- DynamoDB client is instrumented with `aws-xray-sdk-go`.
- App initializes X-Ray in `cmd/bootstrap/main.go`.
- ECS task definition includes an X-Ray daemon sidecar.

## Structured Logging

- Logging uses `log/slog` with JSON output through the adapter in `internal/adapters/logger/slog_logger.go`.
- The logger enriches records with `trace_id` when an X-Ray segment is present in request context.
- HTTP requests are logged via middleware with fields:
  - `method`
  - `path`
  - `status`
  - `duration`
- This format is CloudWatch-friendly and allows request-to-trace correlation in ECS.

## Tests

Existing domain/application tests remain unchanged.

```bash
make test
```

## ECS smoke test

```bash
export API_URL=<http_api_endpoint>
export BASE_PATH=
export API_KEY=<optional_api_key>
export JWT=<optional_jwt>
./smoke-ecs.sh
```

Detailed guide: `/Users/erickeduardogomezjimenez/projects/rbac-project/smoke.md`.
