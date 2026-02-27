# Conversation Service

## 1. Service Overview

The **Conversation Service** is responsible for managing the lifecycle, metadata, and membership of all chats within the RealChat ecosystem. It acts as the source of truth for conversational topology, determining who is allowed to participate in or read a specific chat, and provides monotonically increasing sequence numbers to guarantee strict message ordering.

**Core Responsibilities:**
- **Manage Chat Types:** Support both 1-on-1 (Direct) and Group conversations.
- **Maintain Membership:** Track participants, roles (admin vs. member), and read receipts (last read sequence).
- **Sequence Generation:** Provide atomic, monotonically increasing sequence numbers per conversation for the Message Service.
- **Publish Topology Events:** Emit Outbox events (e.g., `ConversationCreated`, `ParticipantAdded`) to keep downstream services (like Delivery or Push) synchronized with membership changes.

---

## 2. Data Model Overview

The service utilizes PostgreSQL to maintain strongly consistent relational data for conversations and their members.

### Core Tables

* **`conversations`**: The root entity for a chat.
  * Fields: `id`, `type` (`direct` or `group`), `display_name`, `avatar_url`, `lookup_key` (for deduplicating 1-on-1s), `created_at`, `updated_at`.
* **`conversation_participants`**: Tracks who is in which chat, and their progress.
  * Fields: `conversation_id`, `user_id`, `role`, `last_read_sequence`, `joined_at`.
  * **Constraint**: `PRIMARY KEY(conversation_id, user_id)` ensures a user cannot join the same conversation twice.
* **`conversation_sequences`**: An atomic counter table for message ordering.
  * Fields: `conversation_id`, `next_sequence`.
  * **Behavior**: Used via `SELECT ... FOR UPDATE` to strictly serialize sequence generation, preventing race conditions when concurrent messages are sent.
* **`outbox_events`** / **`outbox_dlq`**: Standard Transactional Outbox tables to reliably publish membership events to Kafka.

---

## 3. Transaction Flow: Generating a Sequence

When the Message Service needs to save a new message, it first synchronously calls the Conversation Service via gRPC to obtain a valid sequence. 

1. **gRPC Request:** Message service requests `NextSequence(conversation_id, sender_id)`.
2. **Membership Check:** The service queries `conversation_participants` to ensure `sender_id` is an active member.
3. **Pessimistic Lock:** A database transaction begins, executing `SELECT next_sequence FROM conversation_sequences WHERE conversation_id = ? FOR UPDATE`. This locks the row, preventing concurrent requests for the same chat from getting the same sequence.
4. **Increment and Update:** The sequence is incremented (`next_sequence + 1`) and saved back to the DB.
5. **Commit:** The transaction commits, releasing the lock.
6. **Return Response:** The sequence is returned to the Message Service.

---

## 4. Outbox Pattern Implementation Details

To ensure that downstream services are instantly aware when a user is added to or removed from a conversation, this service utilizes the **Transactional Outbox Pattern**:

- **Atomicity:** When a user is added to `conversation_participants`, an event (e.g., `ParticipantAddedEvent`) is simultaneously inserted into `outbox_events` within the strict boundaries of the same DB transaction.
- **Relay Mechanism:** A dedicated background worker continuously polls `outbox_events`.
- **Publishing:** It reads the `payload` and publishes to the Kafka topic `"conversations-topic"`.
- **Dead Letter Queue (DLQ):** If publishing fails repeatedly, the event is moved to an `outbox_dlq` table for manual intervention, ensuring the primary outbox does not back up permanently.

---

## 5. Producer Guarantees

- **At-Least-Once Delivery:** Membership and conversation events will reach Kafka at least once. Consumers must be idempotent.
- **Strict Ordering per Chat:** Sequence generation is strictly serialized per `conversation_id` via Postgres row-level locks.
- **Topology Source of Truth:** Downstream services (especially Delivery/WebSocket routers) rely entirely on events produced here to know which connections to route messages to.

---

## 6. Failure Handling

| Failure Scenario | System Behavior | Resolution / Recovery |
| :--- | :--- | :--- |
| **Database Down** | All gRPC APIs (Read, Sequence Gen, Member Add) fail immediately. | Clients receive `Unavailable`. Standard retry policies apply. |
| **Kafka Broker Down** | Conversation creation and membership updates succeed. Outbox relay goes into retries. | Events remain safe in postgres `outbox_events`. Automatically resumes when Kafka recovers. No events lost. |
| **Concurrent Sequence Requests** | Requests for the same conversation queue at the database level (`FOR UPDATE` lock). | Processed sequentially. Extremely high throughput in a single chat may experience backpressure. |
| **Outbox Relay Crash** | DB writes continue. Unprocessed events build up. | Relay recovers on restart by scanning for `processed_at IS NULL`. |

---

## 7. Scalability Considerations

- **Sequence Hotspots:** Large, highly active group chats can cause contention on their specific row in the `conversation_sequences` table due to row-level locking. For ultra-massive chats (e.g., thousands of messages per second), a different sharded sequence approach might be required in the future.
- **Read-Heavy Workloads:** `conversation_participants` is accessed frequently by both clients (listing their chats) and internal services (validating senders). Caching layers (e.g., Redis) may be necessary to offload DB reads.
- **Outbox Polling:** To prevent polling overhead on the primary read-write database, outbox polling should utilize indexing on `processed_at` and potentially be migrated to a CDC (Change Data Capture) tool like Debezium at extreme scale.

---

## 8. Tradeoffs

* **Pessimistic DB Locks vs Performance:** Using `SELECT FOR UPDATE` on sequences guarantees absolute strict ordering with no gaps, but slightly limits maximum write concurrency for a *single* conversation.
* **Separation of Concerns:** Abstracting "Conversations & Members" away from the "Message Service" creates a slight network hop overhead during message sending (Message uses gRPC to ask Conversation for a sequence), but cleanly decouples domain responsibilities, making both services easier to scale independently.
