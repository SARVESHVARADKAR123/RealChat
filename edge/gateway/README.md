# 🚪 API Gateway

> The unified entry point for RealChat, handling external traffic, authentication, and routing.

The API Gateway is the central entry point for the RealChat distributed messaging system. It handles all external client traffic, authenticates requests, enforces access policies, and securely routes traffic to the appropriate internal backend microservices.

---

## 🛠️ Core Responsibilities

- 🌍 **Single Entry Point**: The only external-facing interface for all HTTP and WebSocket traffic.
- 🔐 **Authentication**: Validates JSON Web Tokens (JWT) before allowing access to internal systems.
- 🔀 **Smart Routing**: Acts as a reverse proxy, mapping external routes to internal microservices.
- 🛡️ **Rate Limiting**: Protects backend services from abuse using a Redis-backed token bucket algorithm.
- 🔌 **WebSockets**: Detects HTTP-to-WebSocket upgrades and hands persistent connections to the Delivery Service.
- 🛑 **Graceful Shutdown**: Drains connections cleanly during service restarts or deployments.

---

## 🌊 Request Flow

1. **Client Request**: Client sends an HTTP request or initiates a WebSocket connection.
2. **Rate Limit Check**: Checks the Redis-backed token bucket (by IP or User ID) for quotas.
3. **Authentication**: Validates JWT signature and expiration for protected routes.
4. **Forwarding**: Injects user context (like User ID), strips potentially malicious headers, and proxies to the target service.
5. **Response**: Upstream service processes the request and responds through the Gateway to the client.

---

## 🚦 Rate Limiting & Security

### Rate Limiting (Token Bucket)
- **Redis-Backed**: Global rate limits maintained across the entire infrastructure, not just per instance.
- **Dynamic Limits**: Applied per User ID for authenticated users, and per IP for unauthenticated routes (like login) to prevent brute force attacks.

### Security Model
- **Zero Trust Perimeter**: The gateway forms the perimeter and validates all external input.
- **Default Deny**: All endpoints require authentication unless explicitly whitelisted.
- **Header Sanitization**: Strips spoofed internal headers (e.g., `X-Internal-User-Id`) from external clients.

---

## 💥 Failure Handling

- 🐢 **Redis Down**: Fails *open* for rate limiting to maintain availability, logging the failure heavily.
- ⏱️ **Downstream Timeout**: Aborts hanging requests to unresponsive services, returning `504 Gateway Timeout` or `502 Bad Gateway`.
- ❌ **Invalid Tokens**: Returns fast `401 Unauthorized` without hitting backend services.
- 🛑 **Rate Limit Exceeded**: Returns fast `429 Too Many Requests` with `Retry-After` headers.

---

## 🚀 Scalability

The API Gateway is **100% stateless**. It stores no user sessions in memory (JWT handles auth, Redis handles rate limits). As traffic increases, you can elastically scale instances horizontally behind a standard load balancer.

---

## ⚙️ Configuration Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `PORT` | Listening port for the gateway | `8080` |
| `JWT_SECRET` | Secret key for validating JWTs | *Required* |
| `REDIS_ADDR` | Address of the Redis instance | `localhost:6379` |
| `RATE_LIMIT_RPS` | Sustained requests per second | `10` |
| `RATE_LIMIT_BURST` | Maximum burst bucket size | `20` |
| `UPSTREAM_AUTH_URL` | Auth service address | `http://auth:5001` |
| `UPSTREAM_MESSAGE_URL`| Message service address | `http://message:5002` |
| `UPSTREAM_DELIVERY_URL`| Delivery service address | `http://delivery:5003` |
| `SHUTDOWN_TIMEOUT` | Grace period for connection draining | `15s` |

---

## 🛣️ Example Routes

### Public (Rate Limited by IP)
- `POST /api/v1/auth/register` ➡️ Proxied to Auth Service
- `POST /api/v1/auth/login` ➡️ Proxied to Auth Service

### Protected (Rate Limited by User ID)
- `GET /api/v1/users/me` ➡️ Proxied to Profile Service
- `GET /api/v1/conversations` ➡️ Proxied to Conversation Service
- `POST /api/v1/messages` ➡️ Proxied to Message Service

### Real-Time
- `GET /ws` ➡️ Upgrades to WebSocket, proxies to Delivery Service.
