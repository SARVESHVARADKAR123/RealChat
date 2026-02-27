# Profile Service

## 1. Service Overview

The **Profile Service** manages user identities, metadata, and social graphs within the RealChat platform. It serves as the central directory for user information, handling contact lists, presence statuses (online/offline), and user-to-user blocking mechanisms. 

**Core Responsibilities:**
- **User Profiles:** Manage display names, bios, avatars, and high-level presence status.
- **Social Graph (Contacts):** Maintain user contact lists, including "favorite" designations.
- **Privacy & Moderation (Blocks):** Enforce user block lists to prevent unwanted interactions.
- **Event Broadcasting:** Publish profile updates and social graph changes (e.g., `ProfileUpdated`, `ContactAdded`) to Kafka using the Transactional Outbox pattern, ensuring downstream services like Delivery and Message can react to identity changes.

---

## 2. Data Model Overview

The service uses PostgreSQL to manage user data and relationships efficiently.

### Core Tables

* **`profiles`**: Stores the core user identity.
  * Fields: `user_id` (UUID), `username`, `display_name`, `bio`, `avatar_url`, `status` (`online`, `offline`, `away`, `busy`), `last_seen`, `created_at`, `updated_at`.
* **`contacts`**: Represents the social graph.
  * Fields: `user_id`, `contact_user_id`, `nickname`, `is_favorite`.
  * **Constraint**: `PRIMARY KEY(user_id, contact_user_id)` ensures unique relationships. A check constraint prevents self-contacting.
* **`blocks`**: Represents the negative social graph (privacy).
  * Fields: `user_id`, `blocked_user_id`, `reason`.
* **`outbox`**: The staging table for the Transactional Outbox pattern.
  * Fields: `id`, `topic`, `key`, `payload` (JSONB), `created_at`, `published_at`.
  * **Index**: Indexed on `created_at` where `published_at IS NULL` for blazing-fast relay polling.

---

## 3. Transaction Flow (Step-by-Step)

When a user modifies their profile (e.g., changes their display name), the service ensures atomicity via the following flow:

1. **Validation:** The incoming gRPC request is validated (e.g., username character limits, bio length).
2. **Begin DB Transaction:** A PostgreSQL `BEGIN` statement opens the transaction.
3. **Update Profile:** An `UPDATE profiles SET display_name = ? WHERE user_id = ?` query is executed. A Postgres trigger automatically updates the `updated_at` column.
4. **Insert Outbox Event:** A `ProfileUpdated` event (containing the new display name and the target Kafka topic) is inserted into the `outbox` table as unpublished.
5. **Commit Transaction:** The `COMMIT` guarantees that the database reflects the new name *only* if the event is safely queued for broadcasting.
6. **Return Response:** A success response is sent back to the client.

---

## 4. Outbox Pattern Implementation Details

Like other core RealChat services, Profile utilizes the **Transactional Outbox Pattern** to prevent the dual-write problem:

- **Atomicity:** The core entity modification (e.g., `INSERT INTO contacts`) and the outbound event (`INSERT INTO outbox`) share a single ACID transaction.
- **Relay Mechanism:** A background worker consistently polls the `outbox` table for events where `published_at IS NULL`.
- **Publishing:** The worker reads the JSONB `payload`, extracts the routing `topic` and `key`, and publishes to Kafka.
- **Marking as Processed:** Once the Kafka broker acknowledges the message, the worker executes `UPDATE outbox SET published_at = NOW()`.

---

## 5. Producer Guarantees

- **At-Least-Once Delivery:** Events are guaranteed to reach Kafka. In rare crash scenarios, the outbox relay might publish the same event twice. Downstream consumers (like client caches) must be idempotent.
- **Event Ordering:** Events for a specific `user_id` are published to Kafka using the `user_id` as the Kafka message `key`. This guarantees that all updates for a single user land on the same Kafka partition and are processed in strict order by consumers.

---

## 6. Failure Handling

| Failure Scenario | System Behavior | Resolution / Recovery |
| :--- | :--- | :--- |
| **Database Down** | All gRPC APIs (Read, Update, Add Contact) fail. | Fast failure with `Unavailable` status. Clients rely on standard retry mechanisms. |
| **Kafka Broker Down** | Profile updates and contact additions succeed. The outbox relay handles the failure. | Events queue up in the `outbox` table. The relay uses exponential backoff and resumes publishing once Kafka is healthy. No data loss. |
| **Outbox Relay Crash** | Profile database writes complete successfully. Events remain unpublished. | Upon restart, the relay scans for rows where `published_at IS NULL` and resumes processing. |
| **Concurrent Profile Updates** | Postgres row-level locks resolve concurrency for the same `user_id`. | Processed strictly sequentially at the database level. |

---

## 7. Scalability Considerations

- **Read-Heavy Nature:** Profile data is overwhelmingly read-heavy (e.g., displaying avatars in a chat list). The service should be fronted by a distributed cache (like Redis) to intercept `GetProfile` requests and reduce Postgres load.
- **Cache Invalidation:** The outbox events (`ProfileUpdated`) published to Kafka can be consumed by cache-invalidation workers to ensure eventual consistency between Postgres and Redis.
- **Connection Pooling:** Utilizing a tool like PgBouncer is critical to manage the high volume of brief, read-only transactions spawned by clients loading their contact lists.

---

## 8. Tradeoffs

* **JSONB in Outbox:** Using `JSONB` for the outbox payload provides extreme flexibility and schema evolution speed for events, at the cost of slightly higher storage overhead and CPU usage compared to raw binary formats (like Protobuf).
* **Eventually Consistent Clients:** Because changes are propagated via Kafka topics, there is a microsecond-to-millisecond delay between a user updating their avatar and their contacts seeing the change. This favors high availability and system decoupling over strict global consistency.
