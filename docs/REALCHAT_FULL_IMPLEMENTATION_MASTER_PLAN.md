# 🚀 RealChat — Full Implementation Master Plan (Deep Technical Version)

---

# 0️⃣ System Definition

RealChat is a distributed, event-driven real-time messaging infrastructure designed for:

* Strict ordering per conversation
* No message loss
* At-least-once delivery semantics
* Idempotent processing
* Stateless horizontal scaling
* Backpressure enforcement
* Observable and testable behavior

Target (MVP full system):

* 10k–50k concurrent connections
* 1k–3k messages/sec
* Single region
* Horizontal gateway scaling

---

# 1️⃣ Architectural Invariants (Lock Before Coding)

These are **non-negotiable guarantees**.

## 1.1 Ordering

* Ordering is guaranteed per `conversation_id`.
* Kafka partition key = `conversation_id`.
* Sequence number allocated transactionally.
* Gateway never reorders.

## 1.2 Delivery Semantics

* Delivery is at-least-once.
* Consumers must be idempotent.
* Duplicate events must not cause duplicate user-visible messages.

## 1.3 Write Path Atomicity

* Message insert + outbox insert occur in the same DB transaction.
* No external calls inside transaction.

## 1.4 Gateway Statelessness

* No sticky sessions.
* All state reconstructable from DB + Kafka.

If any engineer cannot explain how these are enforced, do not start implementation.

---

# 2️⃣ Repository & Service Structure

```
realchat/
│
├── cmd/
│   ├── chat/
│   ├── gateway/
│   ├── presence/
│   └── auth/
│
├── services/
│   ├── chat/
│   ├── gateway/
│   ├── presence/
│   └── auth/
│
├── pkg/
│   ├── logger/
│   ├── metrics/
│   ├── tracing/
│   ├── kafka/
│   └── middleware/
│
├── internal/
│   ├── events/
│   ├── dto/
│   └── constants/
│
├── api/
│   └── proto/
│
├── deployments/
│   ├── docker/
│   └── docker-compose.yml
│
├── configs/
├── test/
└── docs/
```

---

# 3️⃣ Phase 1 — Chat Service (Deterministic Write Engine)

---

## 3.1 Database Schema Design

### messages table

```sql
CREATE TABLE messages (
    id UUID PRIMARY KEY,
    conversation_id UUID NOT NULL,
    sender_id UUID NOT NULL,
    sequence_number BIGINT NOT NULL,
    content TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT NOW(),
    idempotency_key VARCHAR(64),
    delivered BOOLEAN DEFAULT FALSE,
    UNIQUE(conversation_id, sequence_number),
    UNIQUE(idempotency_key)
);
```

Indexes:

```sql
CREATE INDEX idx_messages_conv_seq
ON messages(conversation_id, sequence_number);
```

---

### outbox table

```sql
CREATE TABLE outbox (
    id UUID PRIMARY KEY,
    aggregate_id UUID,
    event_type VARCHAR(100),
    payload JSONB,
    created_at TIMESTAMP DEFAULT NOW(),
    published BOOLEAN DEFAULT FALSE,
    retry_count INT DEFAULT 0
);
```

Index:

```sql
CREATE INDEX idx_outbox_unpublished
ON outbox(published);
```

---

## 3.2 Transactional Write Flow

Inside single transaction:

1. Validate request.
2. Check idempotency key uniqueness.
3. Lock conversation:

   ```sql
   SELECT MAX(sequence_number)
   FROM messages
   WHERE conversation_id = ?
   FOR UPDATE;
   ```
4. Compute next sequence.
5. Insert message row.
6. Insert outbox row.
7. Commit.

No Kafka calls allowed inside transaction.

---

## 3.3 Failure Analysis

### Case 1: DB Crash Before Commit

→ Nothing persisted. Safe.

### Case 2: Crash After Commit, Before Publish

→ Outbox row persists.
→ Publisher picks it up later.

### Case 3: Duplicate idempotency key

→ Unique constraint prevents duplication.

---

## 3.4 Outbox Publisher Worker

Loop:

* Fetch 100 unpublished rows.
* Publish to Kafka.
* Mark as published.
* Commit.

Producer config:

* enable.idempotence = true
* acks = all
* retries = unlimited

Retry policy:

* Exponential backoff.
* Move to dead-letter after threshold (e.g., 10 retries).

---

# 4️⃣ Phase 2 — Kafka Backbone

---

## 4.1 Topics

```
chat.message.created.v1
chat.message.acknowledged.v1
presence.user.online.v1
presence.user.offline.v1
```

---

## 4.2 Partition Strategy

Partition key:

```
conversation_id
```

Guarantee:
All messages of same conversation go to same partition → ordering preserved.

---

## 4.3 Consumer Settings

* Manual offset commit.
* Handle rebalance listener.
* Avoid committing before message delivered.

---

# 5️⃣ Phase 3 — Gateway (Real-Time Delivery Engine)

---

## 5.1 Connection Architecture

Shard count: 64 (configurable).

Shard selection:

```
hash(userID) % shardCount
```

Each shard:

* Dedicated goroutine.
* Own client registry.
* No shared locks.

---

## 5.2 Client Structure

```go
type Client struct {
    userID string
    conn *websocket.Conn
    send chan []byte // capacity 256
}
```

Memory per client ≈ 50KB.

---

## 5.3 Backpressure Strategy

If:

```go
len(send) == cap(send)
```

Then:

* Log warning.
* Close connection.

Never allow unbounded growth.

---

## 5.4 Kafka Consumer Flow

1. Consume event.
2. Identify recipient(s).
3. Locate shard.
4. Push message into client.send.
5. Commit offset after enqueue.

Duplicate events:

* Client-level idempotency required.
* Sequence number prevents re-display.

---

# 6️⃣ Phase 4 — Presence Service

---

## 6.1 Redis Model

```
SET user:{id}:{deviceID} connectionID EX 60
```

Heartbeat every 20 seconds.

Expiry triggers offline.

---

## 6.2 Failure Handling

Redis restart:

* Clients reconnect.
* Presence rebuilt via heartbeat.

---

# 7️⃣ Phase 5 — Observability

---

## 7.1 Metrics

Gateway:

* active_connections
* send_queue_depth
* kafka_lag
* delivery_latency_ms

Chat:

* db_query_latency
* outbox_pending_count
* kafka_publish_latency

---

## 7.2 Tracing

Instrument:

* gRPC
* Kafka producer
* Kafka consumer
* DB queries
* Redis calls

---

## 7.3 Logging

JSON structured logs.

Mandatory fields:

* trace_id
* service_name
* user_id
* conversation_id
* message_id
* error_code

---

# 8️⃣ Phase 6 — Dockerization

Each service:

* Multi-stage build
* Non-root user
* Liveness endpoint
* Readiness endpoint

docker-compose includes:

* postgres
* redis
* kafka
* zookeeper
* chat
* gateway
* presence

---

# 9️⃣ Phase 7 — Load Testing

---

## Simulations

* 10k concurrent connections.
* 1k–2k messages/sec.
* 20% slow clients.
* 10% reconnect churn.

---

## Metrics to Watch

* Heap growth.
* Goroutine count.
* Kafka lag.
* DB locks.
* P95 latency.

---

# 🔟 Failure Injection Testing

Simulate:

* Kill Kafka broker.
* Kill Chat mid-transaction.
* Kill Gateway during consumption.
* Redis restart.

Verify:

* No message loss.
* No ordering violation.
* System recovers.

---

# 1️⃣1️⃣ Scaling Expectations (MVP)

Memory estimate:

50KB × 10,000 connections ≈ 500MB

Gateway replicas scale horizontally.

Kafka partitions scale throughput.

---

# 1️⃣2️⃣ Production Readiness Criteria

System is ready only if:

* No duplicate sequence numbers.
* No lost messages.
* Gateway handles restart safely.
* Outbox crash-safe.
* Backpressure enforced.
* Load test stable ≥ 1 hour.
* Observability dashboards working.

---

# 1️⃣3️⃣ Known Risk Areas

* Transaction isolation misconfiguration.
* Incorrect Kafka partition key.
* Missing consumer rebalance handler.
* Unbounded WebSocket buffers.
* Memory leak in shard registry.
* Outbox publisher race condition.

Each must be explicitly tested.

---



