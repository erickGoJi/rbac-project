APP_NAME=bootstrap
BIN_DIR=bin

.PHONY: build test clean fmt

build:
	mkdir -p $(BIN_DIR)
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -tags lambda.norpc -o $(BIN_DIR)/$(APP_NAME) ./cmd/bootstrap
	cd $(BIN_DIR) && zip -q -j $(APP_NAME).zip $(APP_NAME)

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
