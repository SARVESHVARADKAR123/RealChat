# RealChat

## 1. System Overview
RealChat is a distributed, real-time messaging backend system. It provides low-latency message delivery, reliable presence tracking, and scalable user communication. The architecture is designed to handle high concurrency while maintaining strict data consistency and fault tolerance through an event-driven microservices model.

## 2. High-Level Architecture Description
The system is composed of several decoupled services communicating synchronously via gRPC/HTTP and asynchronously via Kafka:
- **API Gateway**: Provides a unified entry point for clients, handling protocol translation and routing.
- **Auth Service**: Manages issuing and validating JSON Web Tokens (JWT) for authentication and authorization.
- **Profile Service**: Manages user metadata and directory lookups.
- **Messaging Service**: Handles core message domain logic, persisting messages to PostgreSQL and reliably publishing events using the Outbox pattern.
- **Delivery Service**: Maintains stateful WebSocket connections with clients, consuming routing events from Kafka and pushing messages to connected users.

Core infrastructure dependencies include:
- **Kafka**: Serves as the event backbone for inter-service communication and reliable message routing.
- **PostgreSQL**: Provides durable relational storage for profiles, messages, and transactional outbox tables.
- **Redis**: Maintains ephemeral state for user presence, WebSocket session routing, and distributed rate limiting.

## 3. Design Principles
- **Event-Driven**: State changes are propagated asynchronously via Kafka, decoupling producers from consumers and masking downstream latency.
- **Reliability**: The system prioritizes message durability over immediate delivery. Data is persisted to PostgreSQL before being published to Kafka.
- **Isolation**: Services are database-per-service isolated. Failure domains are restricted to individual service boundaries.
- **Idempotency**: All state-mutating APIs and message consumers are designed to be idempotent to safely handle network retries and at-least-once delivery semantics.

## 4. Consistency Guarantees
- **Producer Guarantees**: Messages are persisted via the Transactional Outbox pattern. A relational transaction commits both the domain entity and the outbox event. A background relay reliably publishes the outbox event to Kafka.
- **Consumer Guarantees**: Consumers commit Kafka offsets only after successful processing and local state updates.
- **Delivery Semantics**: The system provides at-least-once delivery semantics end-to-end. Clients must deduplicate received messages based on unique message identifiers.
- **Ordering Model**: Total ordering is not guaranteed across the system. Causal ordering within a single conversation is maintained by partitioning Kafka topics by conversation ID, ensuring events for a specific chat room are processed sequentially by a single consumer replica.

## 5. Failure Handling Strategy
- **Kafka Down**: The Messaging Service continues to accept messages, persisting them to the database and outbox table. The outbox relay will stall until Kafka recovers, at which point backlogged events are published.
- **Redis Down**: Rate limiting degrades openly. WebSocket reconnections may temporarily fail presence updates. Active WebSocket sessions remain open, but new connections may experience routing delays.
- **Poison Messages**: Consumers utilize a Dead Letter Queue (DLQ) strategy. Messages failing deserialization or repeated processing attempts are routed to a DLQ topic after a bounded number of retries, unblocking the partition.
- **Service Crash**: All services are stateless (excluding database/broker state) and managed by an orchestrator container. Crashed instances are restarted automatically. In-flight requests may fail and require client-side retries.

## 6. Scalability Model
- **Stateless Gateway and API Services**: The API Gateway, Auth Service, Profile Service, and Messaging Service are fully stateless and scale horizontally behind a load balancer.
- **Partition-Based Scaling**: Kafka consumers in the Delivery Service scale by adding replicas up to the number of topic partitions.
- **Connection Management**: The Delivery Service utilizes Redis for session routing, allowing WebSocket connections to be distributed across any Delivery Service replica.

## 7. Security Model
- **Authentication**: Stateless, short-lived JWTs are utilized for request authentication. The Auth Service issues tokens; other services validate signatures independently using shared public keys.
- **Rate Limiting**: A Redis-backed distributed rate limiter protects public endpoints against abuse and resource exhaustion, configured per IP address and user ID.
- **Transport Security**: All external client communication mandates TLS. Internal service-to-service communication occurs over private network segments.

## 8. Running Locally
The system can be executed locally using Docker Compose, which provisions all required dependencies and service containers.

Prerequisites:
- Docker and Docker Compose

Execution:
```bash
docker-compose up -d --build
```
This command initializes PostgreSQL, Redis, Kafka, Zookeeper, and all RealChat microservices.

## 9. Chaos Testing Summary
The system design explicitly accounts for infrastructure volatility.
- **Network Partitions**: Outbox relays retry indefinitely on connection loss.
- **Dependency Eviction**: Service readiness probes detect database/broker disconnections, routing traffic away from degraded instances.
- **Graceful Shutdown**: Services intercept termination signals (SIGTERM) to complete in-flight requests, drain Kafka consumers, and cleanly close database connections before exiting.

## 10. Tradeoffs & Limitations
- **No Exactly-Once Delivery**: The system guarantees at-least-once delivery. Exactly-once messaging in distributed systems incurs prohibitive performance overhead. Clients are responsible for idempotency.
- **Eventual Consistency**: Read operations may return stale data immediately following a write, pending Kafka replication and consumer processing.
- **Operational Complexity**: The Outbox pattern and Kafka dependencies introduce significant operational overhead compared to a monolithic architecture.
