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
- `infrastructure/dynamodb.yml`
- `infrastructure/serverless.yml`

### Deploy DynamoDB stack
```bash
cd infrastructure
sls deploy --config dynamodb.yml --stage dev --region us-east-1
```

### Build/push container image
```bash
aws ecr get-login-password --region us-east-1 | docker login --username AWS --password-stdin <account_id>.dkr.ecr.us-east-1.amazonaws.com

docker build -t rbac-service .
docker tag rbac-service:latest <account_id>.dkr.ecr.us-east-1.amazonaws.com/rbac-service:latest
docker push <account_id>.dkr.ecr.us-east-1.amazonaws.com/rbac-service:latest
```

### Build/push using Makefile
```bash
make ecr-release AWS_REGION=us-east-1 AWS_ACCOUNT_ID=<account_id> IMAGE_TAG=latest
```

### Deploy ECS/ALB/API stack
```bash
cd infrastructure
ECR_IMAGE_URI=<account_id>.dkr.ecr.us-east-1.amazonaws.com/rbac-service:latest \
AUTH_MODE=none \
AUTHORIZE_TEST_MODE=false \
sls deploy --config serverless.yml --stage dev --region us-east-1
```

## AWS components created by `infrastructure/serverless.yml`

- ECR repository.
- VPC with public and private subnets.
- NAT gateway and routing.
- ECS Fargate cluster, task definition, and service.
- Internal ALB + target group.
- API Gateway HTTP API + VPC Link to ALB.
- ECS target tracking autoscaling (`min=1`, `max=3`, `CPU=60%`).
- X-Ray sidecar daemon container in ECS task.

## X-Ray

- DynamoDB client is instrumented with `aws-xray-sdk-go`.
- App initializes X-Ray in `cmd/bootstrap/main.go`.
- ECS task definition includes an X-Ray daemon sidecar.

## Tests

Existing domain/application tests remain unchanged.

```bash
make test
```

## ECS smoke test

```bash
export API_URL=<http_api_endpoint>
export API_KEY=<optional_api_key>
export JWT=<optional_jwt>
./smoke-ecs.sh
```
