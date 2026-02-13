APP_NAME=rbac-service
BIN_DIR=bin
AWS_REGION?=us-east-1
AWS_ACCOUNT_ID?=
AWS_PROFILE?=
IMAGE_TAG?=latest
ECR_REPOSITORY?=rbac-dev-service
GOARCH?=arm64
DOCKER_PLATFORM?=linux/arm64
AWS_PROFILE_FLAG=$(if $(AWS_PROFILE),--profile $(AWS_PROFILE),)
ECR_REGISTRY=$(AWS_ACCOUNT_ID).dkr.ecr.$(AWS_REGION).amazonaws.com
ECR_IMAGE=$(ECR_REGISTRY)/$(ECR_REPOSITORY):$(IMAGE_TAG)

.PHONY: build docker-build ecr-login ecr-tag ecr-push ecr-release test clean fmt

build:
	mkdir -p $(BIN_DIR)
	GOOS=linux GOARCH=$(GOARCH) CGO_ENABLED=0 go build -o $(BIN_DIR)/$(APP_NAME) ./cmd/bootstrap

docker-build:
	docker build --platform $(DOCKER_PLATFORM) -t rbac-service:$(IMAGE_TAG) .

ecr-login:
	@if [ -z "$(AWS_ACCOUNT_ID)" ]; then echo "AWS_ACCOUNT_ID is required"; exit 1; fi
	@aws sts get-caller-identity $(AWS_PROFILE_FLAG) >/dev/null || (echo "Invalid or expired AWS credentials. Run 'aws sso login --profile <profile>' or export valid AWS_* vars."; exit 1)
	@PASS="$$(aws ecr get-login-password --region $(AWS_REGION) $(AWS_PROFILE_FLAG))"; \
	if [ -z "$$PASS" ]; then echo "Failed to get ECR login password"; exit 1; fi; \
	echo "$$PASS" | docker login --username AWS --password-stdin $(ECR_REGISTRY)

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
