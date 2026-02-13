APP_NAME=rbac-service
BIN_DIR=bin
AWS_REGION?=us-east-1
AWS_ACCOUNT_ID?=
IMAGE_TAG?=latest
ECR_REPOSITORY?=rbac-service
ECR_REGISTRY=$(AWS_ACCOUNT_ID).dkr.ecr.$(AWS_REGION).amazonaws.com
ECR_IMAGE=$(ECR_REGISTRY)/$(ECR_REPOSITORY):$(IMAGE_TAG)

.PHONY: build docker-build ecr-login ecr-tag ecr-push ecr-release test clean fmt

build:
	mkdir -p $(BIN_DIR)
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o $(BIN_DIR)/$(APP_NAME) ./cmd/bootstrap

docker-build:
	docker build -t rbac-service:$(IMAGE_TAG) .

ecr-login:
	@if [ -z "$(AWS_ACCOUNT_ID)" ]; then echo "AWS_ACCOUNT_ID is required"; exit 1; fi
	aws ecr get-login-password --region $(AWS_REGION) | docker login --username AWS --password-stdin $(ECR_REGISTRY)

ecr-tag:
	@if [ -z "$(AWS_ACCOUNT_ID)" ]; then echo "AWS_ACCOUNT_ID is required"; exit 1; fi
	docker tag rbac-service:$(IMAGE_TAG) $(ECR_IMAGE)

ecr-push:
	@if [ -z "$(AWS_ACCOUNT_ID)" ]; then echo "AWS_ACCOUNT_ID is required"; exit 1; fi
	docker push $(ECR_IMAGE)

ecr-release: docker-build ecr-login ecr-tag ecr-push

test:
	go test ./internal/application -coverprofile=coverage.out
	go tool cover -func=coverage.out
	@coverage=$$(go tool cover -func=coverage.out | awk '/^total:/ {print substr($$3, 1, length($$3)-1)}'); \
	if [ "$${coverage%.*}" -lt 80 ]; then \
		echo "Coverage below 80%: $$coverage"; \
		exit 1; \
	fi

fmt:
	gofmt -w $$(rg --files -g '*.go')

clean:
	rm -rf $(BIN_DIR) coverage.out
