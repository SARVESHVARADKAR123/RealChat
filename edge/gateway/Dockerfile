# Standardized Dockerfile for API Gateway
FROM golang:1.25-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git

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
COPY edge/gateway ./edge/gateway

# Build binary
RUN CGO_ENABLED=0 GOOS=linux go build -o server ./edge/gateway/cmd/server

# Run Stage
FROM alpine:3.19
RUN apk --no-cache add ca-certificates
WORKDIR /app
COPY --from=builder /app/server .
EXPOSE 8080
ENTRYPOINT ["./server"]
