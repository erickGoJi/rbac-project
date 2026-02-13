FROM golang:1.22-alpine AS builder
WORKDIR /src
RUN apk add --no-cache ca-certificates git
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/rbac-service ./cmd/bootstrap

FROM gcr.io/distroless/static-debian12
WORKDIR /
COPY --from=builder /out/rbac-service /rbac-service
EXPOSE 8080
ENTRYPOINT ["/rbac-service"]
