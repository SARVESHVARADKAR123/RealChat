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

## 💻 Running Locally

You can easily spin up the entire system locally utilizing Docker Compose.

### Prerequisites
- Docker and Docker Compose

### Start the Services
```bash
docker-compose up -d --build
```
*This command initializes PostgreSQL, Redis, Kafka, Zookeeper, and all RealChat microservices.*
