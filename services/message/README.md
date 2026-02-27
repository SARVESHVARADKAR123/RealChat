# Message Service

## 1. Service Overview

The **Message Service** is a critical component of the RealChat distributed backend, serving as the source of truth for chat history and message persistence. It is responsible for durably storing messages, ensuring strong ordering via conversational sequences, and reliably broadcasting message events to downstream consumers without data loss.

**Core Responsibilities:**
- **Persist messages to PostgreSQL:** Securely store message content, sender information, and sequence metadata.
- **Implement Outbox pattern within DB transaction:** Guarantee atomic saves of message data and outbound events.
- **Publish events to Kafka topic "messages-topic":** Act as a high-throughput producer to fan out real-time events to the Delivery Service and other consuming bounded contexts.

---

## 2. Data Model Overview

The service utilizes PostgreSQL, optimized for sequential access and transactional integrity. 

### Core Tables

* **`messages`**: Stores the actual chat messages.
  * Fields: `id`, `conversation_id`, `sender_id`, `sequence`, `type`, `content`, `metadata`, `sent_at`, `deleted_at`.
  * **Constraint**: A `UNIQUE(conversation_id, sequence)` index prevents sequence collisions and enforces strictly ordered chat histories.
* **`outbox_events`**: The staging table for the Transactional Outbox pattern.
  * Fields: `id` (BIGSERIAL), `aggregate_type`, `aggregate_id`, `event_type`, `payload`, `processed_at`, `created_at`.
  * **Index**: An index on `created_at` where `processed_at IS NULL` ensures extremely fast polling for unpublished events.
* **`idempotency_keys`**: Ensures safe retries for message ingestion.
  * Fields: `key`, `user_id`, `payload`, `created_at`, `expires_at`.
  * **Constraint**: `PRIMARY KEY(key, user_id)` prevents a user from duplicating a specific message send request.

---

## 3. Transaction Flow (Step-by-Step)

When a user sends a message, the service executes a strict sequence of operations to guarantee consistency across the database and the message broker:

1. **Idempotency Check:** The gRPC request is intercepted to check if the `(idempotency_key, sender_user_id)` exists. If yes, the cached `payload` response is returned immediately.
2. **Conversation Sequence Generation:** The service requests the next monotonic `sequence` number for the specific `conversation_id`.
3. **Begin DB Transaction:** A PostgreSQL transaction (`BEGIN`) is initiated.
4. **Insert Message:** The record is inserted into the `messages` table with the newly acquired sequence number.
5. **Insert Outbox Event:** A `MessageSentEvent` (containing the message payload) is serialized into bytes and inserted into the `outbox_events` table as `unprocessed`.
6. **Save Idempotency Key:** The result payload is saved to `idempotency_keys` under the current transaction.
7. **Commit Transaction:** The transaction is committed (`COMMIT`). At this point, the message is durably saved, and the system is guaranteed to eventually produce the Kafka event.
8. **Return Response:** A success response is dispatched to the client.

---

## 4. Outbox Pattern Implementation Details

To solve the dual-write problem (where writing to DB and publishing to Kafka sequentially can leave the system in an inconsistent state if one fails), the Message Service strictly adheres to the **Transactional Outbox Pattern**:

- **Atomicity:** The `messages` insert and the `outbox_events` insert share a single ACID database transaction.
- **Relay Mechanism:** A dedicated background worker continuously polls the `outbox_events` table for rows where `processed_at IS NULL`.
- **Publishing:** The relay extracts the `payload`, publishes it to the Kafka topic `"messages-topic"`, and waits for the Kafka broker acknowledgement (ACK).
- **Marking as Processed:** Once ACK'd, the relay executes an `UPDATE outbox_events SET processed_at = NOW()` to mark the event as successfully dispatched.

---

## 5. Producer Guarantees

As the primary producer of chat events, the service provides the following operational guarantees:

- **At-Least-Once Delivery:** Events are guaranteed to reach the `"messages-topic"` Kafka topic. In rare cases (e.g., worker crash after publishing to Kafka but before updating the DB), a duplicate event may be published. Consumers must be idempotent.
- **Strict Ordering per Conversation:** Because sequence numbers are monotonically assigned prior to the DB transaction, and messages are logically grouped by `conversation_id`, downstream consumers can strictly order events client-side based on the `sequence` field.
- **No Data Loss:** Acknowledgement configurations (`acks=all`) ensure that once an event is accepted by the Kafka partition leader, it is replicated to in-sync replicas before marking the outbox row as processed.

---

## 6. Failure Handling

| Failure Scenario | System Behavior | Resolution / Recovery |
| :--- | :--- | :--- |
| **Kafka Broker Down** | The message is safely persisted in the DB and Outbox. The sync API returns success to the sender. The Outbox Relay fails to publish and enters an exponential backoff retry loop. | Once Kafka recovers, the relay resumes processing the accumulated outbox events. Delivery is delayed, but **no messages are lost**. |
| **Database Down** | The API fails immediately. No transaction can begin. | The client receives a `500 Internal Server Error` (or `Unavailable`) and relies on standard client-side retries. |
| **Outbox Relay Crash** | DB transactions continue succeeding. Unprocessed events build up in the `outbox_events` table. | Upon relay restart, it automatically picks up from the oldest row where `processed_at IS NULL`. |
| **Sequence Collision** | The DB transaction will fail due to the `UNIQUE(conversation_id, sequence)` constraint. | The transaction is rolled back, and the service can retry sequence acquisition prior to fulfilling the request. |

---

## 7. Idempotency Strategy

In distributed mobile networks, clients often retry requests identically due to timeouts. To prevent duplicate messages:

1. **Client-Provided Keys:** Clients must supply a unique `idempotency_key` (e.g. UUID v4) with every `SendMessage` request.
2. **Compound Uniqueness:** Keys are scoped to `(key, user_id)` to mitigate accidental cross-user collisions.
3. **Short-Lived Caching:** Keys are stored in the `idempotency_keys` table with an `expires_at` timestamp.
4. **Early Exit:** If a request matches an existing key, the DB transaction is skipped, and the previously computed gRPC response (stored in `payload`) is directly retrieved and returned.

---

## 8. Scalability Considerations

- **Write-Heavy Outbox:** The `outbox_events` table experiences high write and update churn. To maintain performance, partitioned tables or aggressive vacuuming strategies are required to keep the table index footprint small.
- **Database Connection Pooling:** Robust connection pooling (e.g. PgBouncer) limits concurrent transactions to prevent PostgreSQL from being overwhelmed during heavy traffic spikes.
- **Kafka Partitioning:** The `"messages-topic"` should ideally be partitioned by `conversation_id`. This ensures that messages belonging to the same chat are routed to the same partition, preserving sequence order for consumers and maximizing parallel throughput.

---

## 9. Tradeoffs

* **Read-after-Write Latency vs Consistency:** The outbox pattern introduces a slight background delay (milliseconds) between the user pressing "Send" and Kafka receiving the event. This is an explicit tradeoff favoring absolute data consistency and durability over instantaneous push.
* **Storage Overhead:** Storing every outgoing event in PostgreSQL before passing it to Kafka increases disk IO and requires diligent cleanup/archival of the `outbox_events` table to prevent table bloat over time.
* **At-Least-Once Duplicates:** Downstream services (like Delivery or Push Notifications) bear the burden of deduplicating events, as the outbox relay pattern guarantees *at-least-once* delivery, not *exactly-once*.
