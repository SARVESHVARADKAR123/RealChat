# API Gateway

The API Gateway is the central entry point for the RealChat distributed messaging system. It handles all external client traffic, authenticates requests, enforces access policies, and securely routes traffic to the appropriate internal backend microservices.

## 1. Service Responsibilities

The API Gateway is responsible for ensuring the security, reliability, and proper routing of incoming traffic. Its core duties include:

- **Entry Point for All Clients**: Acts as the single external-facing interface for all HTTP and WebSocket traffic.
- **JWT Validation**: Authenticates incoming requests by cryptographically verifying JWT signatures and expiration times before allowing access to internal systems.
- **Request Forwarding**: Maps external routes to internal microservices (e.g., Auth, Profile, Message, Conversation) acting as a performant reverse proxy.
- **Rate Limiting**: Protects downstream services from localized spikes, abuse, and DDoS attacks using a robust Redis-backed token bucket algorithm.
- **WebSocket Upgrade Handling**: Detects and handles HTTP-to-WebSocket protocol upgrades, ensuring persistent real-time connections are properly handed off to the Delivery Service.
- **Graceful Shutdown**: Implements connection draining to ensure that in-flight requests and active persistent connections are handled cleanly during service restarts, deployments, or scaling events.

## 2. Request Flow

1. **Client Request**: A client (mobile, web app, etc.) makes an HTTP request or initiates a WebSocket connection to the gateway.
2. **Rate Limiting Check**: The gateway checks the Redis-backed token bucket using the client's IP address (for unauthenticated routes) or parsed Client ID/User ID to ensure they are within their allowed quotas.
3. **Authentication**: For protected routes, the gateway intercepts the request, extracts the JWT (from the `Authorization` header or query parameters), validates its signature against the known secret/public key, and verifies expiration.
4. **Context Enrichment & Forwarding**:
   - The gateway enriches the request by injecting validated user context (e.g., specific user ID headers) so internal services do not need to parse the JWT again.
   - It strips potentially malicious or reserved headers and forwards the request to the upstream target.
   - For WebSocket requests, it handles the upgrade handshake before proxying the TCP connection to the Delivery Service.
5. **Response**: The upstream service processes the payload and returns the response back through the gateway, which forwards it to the original client.

## 3. Rate Limiting Strategy

The service implements a **Redis-backed token bucket** rate limiting algorithm to constrain traffic dynamically:
- **Token Bucket**: Allows for a defined rate of requests over time while allowing short bursts. Each client has a "bucket" of tokens; requests consume tokens. The bucket refills at a constant rate.
- **Redis-Backed**: State is maintained externally in a highly available Redis cluster. This ensures that rate limits apply globally across the entire RealChat infrastructure, rather than on a per-gateway-instance basis.
- **Identifiers**: Limits are applied per User ID for authenticated users, preserving IP limits for unauthenticated interactions (e.g., Login/Registration endpoints) to prevent brute force attacks.

## 4. Security Model

- **Zero Trust Foundation**: The gateway forms the perimeter, validating all external input. Traffic behind the gateway is considered trusted for identity, but downstream services independently authorize resources.
- **Strict Validation**: All endpoints default to requiring authentication unless explicitly whitelisted in the routing configuration.
- **Transport Security**: The gateway (or an immediate load balancer in front of it) terminates TLS (HTTPS/WSS).
- **Header Sanitization**: Internal-only administrative headers (e.g., `X-Internal-User-Id`) cannot be spoofed by external clients; the gateway aggressively strips these out from incoming external queries.

## 5. Failure Behavior

The API Gateway is built to handle failure scenarios gracefully:
- **Redis Down**: If the rate-limiting Redis node is unreachable, the gateway fails open for rate limiting to maintain service availability, while aggressively logging the failure and triggering alerts.
- **Downstream Timeout**: If a destination service is unresponsive, the gateway enforces strict timeouts. It aborts the request and returns a standardized `504 Gateway Timeout` or `502 Bad Gateway` to prevent holding open connections and exhausting its own resources.
- **Invalid Tokens**: Returns immediate `401 Unauthorized` responses without dialing backend services.
- **Rate Limit Exceeded**: Returns a fast `429 Too Many Requests` response containing `Retry-After` headers.

## 6. Scalability Model

- **Stateless Horizontal Scaling**: The API Gateway is completely stateless. It does not store user sessions in memory. Authentication state is encoded in JWTs, and distributed state (rate limits) lives in Redis.
- **Elasticity**: As traffic increases, additional instances of the API Gateway can be deployed horizontally behind a Layer 4/Layer 7 load balancer without any coordination needed between gateway instances.

## 7. Configuration Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `PORT` | Listening port for the gateway | `8080` |
| `JWT_SECRET` | Secret key for validating incoming authorization tokens | *None (Required)* |
| `REDIS_ADDR` | Address of the Redis instance for rate limiting | `localhost:6379` |
| `RATE_LIMIT_RPS` | Sustained requests per second per client | `10` |
| `RATE_LIMIT_BURST` | Maximum burst bucket size per client | `20` |
| `UPSTREAM_AUTH_URL` | Address of the internal Auth service | `http://auth:5001` |
| `UPSTREAM_MESSAGE_URL`| Address of the internal Message service | `http://message:5002` |
| `UPSTREAM_DELIVERY_URL`| Address of the internal Delivery service | `http://delivery:5003` |
| `SHUTDOWN_TIMEOUT` | Grace period to drain connections on shutdown | `15s` |

## 8. Example API Endpoints

### Unauthenticated Routing (Rate Limited by IP)
- `POST /api/v1/auth/register` → Proxied to Auth Service
- `POST /api/v1/auth/login` → Proxied to Auth Service

### Authenticated REST Routing (Rate Limited by JWT Subject)
- `GET /api/v1/users/me` → Proxied to Profile Service
- `GET /api/v1/conversations` → Proxied to Conversation Service
- `POST /api/v1/messages` → Proxied to Message Service

### Real-Time Routing
- `GET /ws` → Upgrades to WebSocket. Validates auth token from query params (`?token=...`). Proxied to Delivery Service as a persistent stream.
