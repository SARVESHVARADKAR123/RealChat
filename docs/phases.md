# RealChat Implementation Phases

**Implementation-Grade Execution Sequencing**

This is not a demo. This is distributed infrastructure that must survive crashes, scale, and evolve.

Each phase is structured as:
1. Phase intent
2. Internal architecture evolution
3. Data model
4. Failure modes
5. Operational maturity
6. Exit criteria

No hand-waving.

---

## 🧱 Phase 0 — Architecture & Contract Freeze

### 🎯 Goal

Lock system invariants before code.

If invariants are wrong, everything downstream is garbage.

---

### 0.1 Define Non-Negotiables

You must explicitly define:

* **Message ordering guarantee**
  * Per conversation? (Recommended)
  * Per user?
* **Delivery semantics**
  * At-least-once (practical)
  * Exactly-once (illusion via dedup)
* **Latency SLA**
  * P99 < 150ms intra-region?
* **Availability target**
  * 99.9%?
* **Max room size**
  * Impacts fanout model

---

### 0.2 Message Envelope Contract

Immutable schema. Versioned.

```json
{
  "message_id": "UUID",
  "conversation_id": "UUID",
  "sender_id": "UUID",
  "sequence_number": "int64",
  "payload": "bytes",
  "created_at": "timestamp",
  "idempotency_key": "string"
}
```

**Why idempotency key?**  
Because retries are inevitable.

---

### 0.3 Partitioning Strategy (Critical)

If using Kafka:

**Partition key = `conversation_id`**

**Why?**  
Because ordering is preserved per partition.

**Tradeoff:**  
Hot conversations create hot partitions.

**Mitigation later via sharding.**

---

### 0.4 Exit Criteria

* ✓ API spec frozen
* ✓ Data model finalized
* ✓ Failure semantics documented
* ✓ Ordering semantics documented
* ✓ All edge cases enumerated

**If not, do not code.**

---

## 🧱 Phase 1 — Single Node, Correctness First

### 🎯 Goal

Deliver a correct end-to-end message pipeline on ONE machine.

---

### 1.1 Architecture

```
Client → WS Server → Message Handler → Postgres → In-memory fanout → Client
```

**No Kafka. No Redis.**

---

### 1.2 Core Components

#### WebSocket Manager

**Responsibilities:**
* Connection registry
* Heartbeat
* Backpressure detection
* Graceful disconnect

**Data structure:**
```go
map[userID] -> connection
```

**Must handle:**
* Concurrent writes
* Race during disconnect

---

#### Message Service

**Responsibilities:**
* Validate sender
* Validate conversation membership
* Generate sequence number
* Persist message
* Trigger fanout

**Sequence generation:**  
Use DB atomic increment per conversation.

**Do NOT compute in memory.**

---

### 1.3 Data Model

**Messages table:**

```sql
CREATE TABLE messages (
  id UUID PRIMARY KEY,
  conversation_id UUID NOT NULL,
  sender_id UUID NOT NULL,
  sequence BIGINT NOT NULL,
  payload JSONB NOT NULL,
  created_at TIMESTAMP NOT NULL DEFAULT NOW(),
  
  UNIQUE(conversation_id, sequence)
);

CREATE INDEX idx_messages_conversation ON messages(conversation_id, sequence);
```

**This guarantees ordering correctness.**

---

### 1.4 Failure Handling

**Case:** DB write succeeds but WS delivery fails.

**System behavior:**
* Message remains persisted.
* Client will fetch on reconnect.

**That's eventual delivery.**

---

### 1.5 Complexity

**Time:**
* Write: O(1)
* Fanout: O(n) where n = participants

**Space:**
* O(active_connections)

---

### 1.6 Exit Criteria

* ✓ No duplicate messages under retry
* ✓ Correct ordering under concurrent send
* ✓ Stable under 10k connections on single node
* ✓ Memory footprint predictable

**Only then move forward.**

---

## 🧱 Phase 2 — Introduce Event-Driven Decoupling

Now we introduce Kafka.

---

### 🎯 Goal

Separate ingestion from delivery.

---

### 2.1 New Flow

```
Client → WS → Produce to Kafka
Kafka → Message Consumer → DB
Kafka → Delivery Consumer → WS nodes
```

---

### 2.2 Why This Matters

Now:
* Writes are asynchronous
* You can replay
* You decouple compute domains

---

### 2.3 Idempotency

**Producer:**
* Enable idempotent producer
* Attach idempotency_key

**Consumer:**
* Maintain dedup table:

```sql
CREATE TABLE processed_messages (
  idempotency_key VARCHAR(255) PRIMARY KEY,
  processed_at TIMESTAMP NOT NULL DEFAULT NOW()
);
```

**Before writing:**  
Check existence.

---

### 2.4 Retry Strategy

Exponential backoff.

**If max retries exceeded:**  
Send to DLQ.

**Never silently drop.**

---

### 2.5 Ordering Guarantee

**Because partition key = conversation_id:**  
Ordering preserved per conversation.

**If you choose random partitioning:**  
You destroy ordering guarantee.

---

### 2.6 Failure Modes

#### Kafka down

Gateway must:
* Apply backpressure
* Reject new messages (HTTP 503)
* Or buffer in memory (risky)

**Never infinite buffer.**

---

### 2.7 Exit Criteria

* ✓ Replay from offset works
* ✓ Kill consumer → restart → no data loss
* ✓ Inject duplicate messages → dedup works
* ✓ Consumer lag observable

---

## 🧱 Phase 3 — Horizontal WebSocket Layer

Now complexity increases sharply.

---

### 🎯 Goal

Scale to multiple WS nodes.

---

### 3.1 Problem

* User A connected to node 1
* User B connected to node 2

**How does delivery happen?**

---

### 3.2 Solution Pattern

Use Redis:

```
user:{user_id} -> ws_node_id
node:{node_id} -> connection_list
```

**Delivery flow:**
1. Delivery consumer determines recipient
2. Lookup Redis
3. Publish to correct node

---

### 3.3 Pub/Sub vs Direct RPC

**Option 1:**  
Redis PubSub (simple, less reliable)

**Option 2:**  
Dedicated delivery topic per node (recommended)

---

### 3.4 Backpressure Handling

If WS node overloaded:
* Stop consuming from delivery topic
* Consumer lag increases
* Autoscaler triggers

---

### 3.5 Load Balancing

**Do NOT rely on sticky sessions alone.**

WebSocket is long-lived.  
Load imbalance happens over time.

**Implement:**
* Connection draining on deployment
* Graceful shutdown with timeout

---

### 3.6 Exit Criteria

* ✓ 100k+ concurrent connections
* ✓ Horizontal scaling works
* ✓ Killing 1 node does not drop messages permanently
* ✓ Redis failover tested

---

## 🧱 Phase 4 — Reliability Engineering

Now we make it production-worthy.

---

### 4.1 Timeouts Everywhere

Every network call:
* DB
* Redis
* Kafka

**Must have timeout.**

**Otherwise:**  
Thread exhaustion.

---

### 4.2 Circuit Breakers

If Redis latency spikes:
* Trip breaker.
* Fail fast.

---

### 4.3 Graceful Shutdown

When pod terminating:
1. Stop accepting new connections
2. Drain existing
3. Commit offsets
4. Close Kafka consumer cleanly

---

### 4.4 Observability

#### Metrics
* Active connections
* Kafka lag
* DB write latency
* P99 end-to-end latency
* Failed deliveries

#### Tracing
* Trace ID per message

#### Logs
* Structured JSON logs only

---

### 4.5 Chaos Testing

**Kill:**
* Kafka broker
* Redis node
* Random WS node

**System must degrade gracefully.**

---

## 🧱 Phase 5 — Product Complexity

Now comes the real scaling stress.

---

### 5.1 Large Groups (Fanout Explosion)

If group size = 10k:

**Naive O(n) fanout per message is expensive.**

**Optimizations:**
* Batch delivery
* Lazy pull model
* Notification pointer model

---

### 5.2 Offline Users

On reconnect:
1. Query DB by sequence
2. Deliver missing messages
3. Update last_seen_sequence

**This requires:**

```sql
CREATE TABLE user_conversation_state (
  user_id UUID NOT NULL,
  conversation_id UUID NOT NULL,
  last_delivered_sequence BIGINT NOT NULL DEFAULT 0,
  
  PRIMARY KEY (user_id, conversation_id)
);
```

---

### 5.3 Read Receipts

**Model:**
* Store read sequence per user

**Never store per-message read rows.**  
Explodes storage.

---

## 🧱 Phase 6 — Performance Engineering

Now think like infra.

---

### 6.1 Load Testing

**Simulate:**
* 1M connections
* 10k messages/sec

**Measure:**
* P50, P95, P99
* GC pauses
* CPU saturation
* Kafka lag

---

### 6.2 Optimization Levers

* Batch DB writes
* Kafka batch producer
* Connection pooling
* Reduce JSON allocations
* Use binary protocol internally

---

### 6.3 Memory Math

```
If 1 connection = 50KB
1M connections = 50GB RAM
```

**Can your infra afford that?**

Always compute memory per connection.

---

## 🧱 Phase 7 — Multi-Region

Now things get real.

**Questions:**
* Global ordering?
* Active-active?
* Conflict resolution?

**Tradeoffs:**
* Strong consistency → higher latency
* Eventual consistency → simpler

**Most systems choose:**  
Regional isolation + eventual sync.

---

## 🧠 What Most Engineers Do Wrong

They:
* Add Kafka before correctness
* Add Redis before reasoning
* Add Kubernetes before stability

**Correct order:**  
Correctness → Decoupling → Scalability → Reliability → Optimization.

---

## 📋 Phase Summary

| Phase | Focus | Key Technology | Exit Metric |
|-------|-------|----------------|-------------|
| 0 | Architecture | Design docs | Contract frozen |
| 1 | Correctness | Single node + Postgres | 10k connections stable |
| 2 | Decoupling | Kafka | Replay works |
| 3 | Scale | Redis + Multi-node | 100k connections |
| 4 | Reliability | Timeouts + Circuit breakers | Chaos tested |
| 5 | Product | Offline sync + Large groups | 10k user rooms |
| 6 | Performance | Load testing | P99 < 150ms |
| 7 | Global | Multi-region | Regional failover |

---

## 🚨 Critical Principles

1. **Never skip phases** — Each builds on the previous
2. **Exit criteria are mandatory** — No fuzzy "good enough"
3. **Failure modes first** — Design for failure, not success
4. **Measure everything** — No observability = no production
5. **Simplicity wins** — Add complexity only when forced

---

**This is how you build systems that last.**
