# 💬 RealChat

> A distributed, real-time messaging backend system built for scale and reliability.

RealChat provides **low-latency message delivery**, **reliable presence tracking**, and **scalable user communication**. The architecture is designed to handle high concurrency while maintaining strict data consistency and fault tolerance through an event-driven microservices model.

---

## 🏗️ Architecture Overview

The system is composed of several decoupled microservices communicating synchronously via **gRPC/HTTP** and asynchronously via **Kafka**:

### Core Services
- 🚪 **API Gateway**: Unified entry point for clients, handling protocol translation and routing.
- 🔐 **Auth Service**: Manages issuing and validating JSON Web Tokens (JWT) for authentication & authorization.
- 👤 **Profile Service**: Manages user metadata and directory lookups.
- 📨 **Messaging Service**: Handles core message logic, persisting to PostgreSQL and publishing events using the Outbox pattern.
- 📡 **Delivery Service**: Maintains stateful WebSocket connections, consumes routing events from Kafka, and pushes messages to clients.

### Infrastructure
- 🐘 **PostgreSQL**: Durable relational storage for profiles, messages, and transactional outbox tables.
- 🚀 **Kafka**: Event backbone for inter-service communication and reliable message routing.
- ⚡ **Redis**: Ephemeral state for user presence, WebSocket session routing, and rate limiting.

---

## 💡 Design Principles

- **Event-Driven**: State changes are propagated asynchronously via Kafka, decoupling producers/consumers.
- **Reliability First**: Data is persisted to PostgreSQL *before* being published to Kafka (prioritizing durability).
- **Service Isolation**: Database-per-service model restricts failure domains to individual boundaries.
- **Idempotency**: All state-mutating APIs and message consumers safely handle network retries and at-least-once delivery.

---

## 🛡️ Consistency & Guarantees

- **Delivery**: **At-least-once** delivery semantics end-to-end. Clients must deduplicate received messages.
- **Ordering**: Causal ordering within a single conversation (Kafka topics partitioned by conversation ID).
- **Writing**: Messages are persisted via the **Transactional Outbox** pattern, ensuring atomicity.
- **Reading**: Consumers commit Kafka offsets only after successful processing and local state updates.

---

## 💥 Chaos & Failure Handling

RealChat's design explicitly accounts for infrastructure volatility:

- 🛑 **Kafka Offline**: Messaging Service accepts messages, persisting to DB/outbox. Outbox relay stalls until Kafka recovers, then publishes the backlog.
- 🐢 **Redis Down**: Rate limiting degrades openly. WebSocket reconnections may temporarily fail presence updates, but active connections remain open.
- ☠️ **Poison Messages**: Consumers use a **Dead Letter Queue (DLQ)**. Failed messages are routed to a DLQ topic after bounded retries to unblock the partition.
- 🔄 **Service Crashes**: Stateless services are automatically restarted. **Graceful Shutdown** ensures in-flight requests finish, Kafka consumers drain, and DB connections close cleanly.

---

## 🚀 Scalability & Security

### Scaling Strategy
- **Stateless Service Scaling**: API Gateway, Auth, Profile, and Messaging services scale horizontally behind a load balancer.
- **Event Consumers**: Delivery Service scales Kafka consumers up to the number of topic partitions.
- **Connection Routing**: Delivery Service uses Redis for session routing, allowing WebSocket connections across any replica.

### Security
- **Authentication**: Stateless, short-lived JWTs. Auth Service issues them; other services validate using shared public keys.
- **Rate Limiting**: Redis-backed distributed rate limiter per IP address and user ID protects public endpoints.
- **Transport Security**: External traffic mandates TLS. Internal communication uses private network segments.

---

## ⚖️ Tradeoffs & Limitations

- 🔥 **No Exactly-Once Delivery**: We guarantee at-least-once delivery. Exactly-once incurs prohibitive performance overhead.
- 🕰️ **Eventual Consistency**: Read operations may return stale data immediately following a write, pending Kafka replication.
- 🏗️ **Operational Complexity**: The Outbox pattern and Kafka dependencies introduce significant overhead compared to a monolith.

---

## ⚙️ Environment Configuration (`.env`)

Before running the system (locally or in production), you must define the necessary environment variables. The system relies heavily on these configurations for database connections, Kafka routing, internal gRPC addresses, and security keys.

### � Example `.env` for Local Development
Create a `.env` file in the root of the project and paste the following configuration. This setup operates perfectly with the local `docker-compose.yml`.

```env
# ================= DATABASE =================
POSTGRES_USER=postgres
POSTGRES_PASSWORD=postgres
POSTGRES_DB=realchat
POSTGRES_HOST=postgres
POSTGRES_PORT=5432

AUTH_DATABASE_URL=postgres://${POSTGRES_USER}:${POSTGRES_PASSWORD}@${POSTGRES_HOST}:${POSTGRES_PORT}/auth?sslmode=disable
PROFILE_DATABASE_URL=postgres://${POSTGRES_USER}:${POSTGRES_PASSWORD}@${POSTGRES_HOST}:${POSTGRES_PORT}/profile?sslmode=disable
MESSAGING_DATABASE_URL=postgres://${POSTGRES_USER}:${POSTGRES_PASSWORD}@${POSTGRES_HOST}:${POSTGRES_PORT}/messaging?sslmode=disable
CONVERSATION_DATABASE_URL=postgres://${POSTGRES_USER}:${POSTGRES_PASSWORD}@${POSTGRES_HOST}:${POSTGRES_PORT}/conversation?sslmode=disable

# ================= INFRASTRUCTURE =================
REDIS_ADDR=redis:6379
ZOOKEEPER_CLIENT_PORT=2181
KAFKA_BROKER_ID=1
KAFKA_BROKER=kafka:9092
KAFKA_LISTENERS=PLAINTEXT://0.0.0.0:9092
KAFKA_ADVERTISED_LISTENERS=PLAINTEXT://kafka:9092

# ================= ROUTING & API GATEWAY =================
GATEWAY_PORT=8080
GATEWAY_HTTP_ADDR=8090

# ================= GRPC ADDRESSES =================
AUTH_GRPC_ADDR=auth:50051
PROFILE_GRPC_ADDR=profile:50052
MSG_GRPC_ADDR=messaging:50053
CONV_GRPC_ADDR=conversation:50055
PRESENCE_GRPC_ADDR=presence:50056

# ================= HTTP ADDRESSES & PORTS =================
AUTH_HTTP_ADDR=8081
AUTH_OBS_HTTP_ADDR=8091
PROFILE_HTTP_PORT=8082
PROFILE_HTTP_ADDR=8092
MESSAGING_HTTP_ADDR=8094
CONVERSATION_HTTP_ADDR=8095
PRESENCE_HTTP_ADDR=8096
DELIVERY_HTTP_PORT=8083
DELIVERY_HTTP_ADDR=8093

# ================= KAFKA TOPICS & INSTANCES =================
MESSAGING_KAFKA_TOPIC=messaging.events.v1
CONVERSATION_KAFKA_TOPIC=conversation.events.v1
DELIVERY_KAFKA_TOPICS=messaging.events.v1,conversation.events.v1
PRESENCE_INSTANCE_ID=presence-1
DELIVERY_INSTANCE_ID=delivery-1

# ================= APPLICATION & SECURITY =================
APP_VERSION=1.0.0
JWT_SECRET=dev-secret-change-me-in-prod
```

### 🔐 Detailed Breakdown for Production (`.env.prod`)

When moving to production, you will need to replace the local simulated values with robust and secure counterparts in a `.env.prod` file:

- **Database**: `POSTGRES_HOST` and `POSTGRES_PORT` should point to your managed database cluster. All `DATABASE_URL` strings must use strong passwords.
- **Infrastructure**: Update `KAFKA_BROKER`, `REDIS_ADDR` to point to production clusters. Update `KAFKA_ADVERTISED_LISTENERS` so brokers can be routed properly.
- **Security**: Generate a highly secure `JWT_SECRET` (e.g., using `openssl rand -base64 32`).

---

## 💻 Running Locally

You can easily spin up the entire system locally utilizing Docker Compose now that your `.env` is configured.

### Prerequisites
- Docker and Docker Compose
- `.env` file configured in the project root.

### Start the Services Locally
```bash
docker-compose up -d
```
*This command pulls pre-built images from Docker Hub and initializes PostgreSQL, Redis, Kafka, Zookeeper, and all RealChat microservices. There is no need to build images locally.*

---

## 🌐 Running in Production

To run the system in a production environment, use the provided `docker-compose.prod.yml` file. Ensure you load the production environment variables (`.env.prod`) mirroring the variable structure above but containing strong production secrets, managed databases, or external Kafka clusters.

### Start the Services in Production
```bash
docker-compose -f docker-compose.prod.yml --env-file .env.prod up -d
```
*This command spins up the entire RealChat stack using production configurations. Pre-built images are pulled directly from Docker Hub.*

---

## 🧪 Testing

All API endpoints are exposed through the **API Gateway** (`http://localhost:8080`). Use the guide below to test every endpoint step-by-step via Postman — no CLI tools required.

👉 **[Postman Execution Guide](./POSTMAN_TESTING_EXECUTION.md)**

The guide covers:
- Auth (register, login, refresh, logout)
- Profile (get, update)
- Conversations (create, list, get)
- Participants (add, remove)
- Messages (send, sync, delete)
- Read Receipts
- Presence
