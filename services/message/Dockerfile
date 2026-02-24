# Standardized Dockerfile for separated Message Service (CGO Enabled)
FROM golang:1.25-alpine AS builder

WORKDIR /app

# Install build dependencies (CGO and librdkafka dependencies)
RUN apk add --no-cache git build-base

# Copy workspace configuration and only necessary modules
# Copy workspace configuration and all module definitions
COPY go.work go.work.sum ./
COPY contracts/go.mod contracts/go.sum ./contracts/
COPY edge/gateway/go.mod edge/gateway/go.sum ./edge/gateway/
COPY services/auth/go.mod services/auth/go.sum ./services/auth/
COPY services/delivery/go.mod services/delivery/go.sum ./services/delivery/
COPY services/message/go.mod services/message/go.sum ./services/message/
COPY services/profile/go.mod services/profile/go.sum ./services/profile/

# Download dependencies
RUN go mod download

# Copy source code for the service and its dependencies
COPY contracts ./contracts
COPY services/message ./services/message

# Build binary with CGO and musl for Alpine
RUN CGO_ENABLED=1 GOOS=linux go build -tags musl -o server ./services/message/cmd/server/main.go

# Run Stage
FROM alpine:3.19
RUN apk --no-cache add ca-certificates
WORKDIR /app
COPY --from=builder /app/server .
EXPOSE 50051
ENTRYPOINT ["./server"]
