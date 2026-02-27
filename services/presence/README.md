# Presence Service

## 1. Service Overview

The **Presence Service** acts as the high-speed, centralized authority for user online/offline status and device tracking within the RealChat ecosystem. By aggregating connection events from various Delivery nodes, it provides an instantaneous view of a user's active devices and broadcasts global status changes.

**Core Responsibilities:**
- **State Aggregation:** Consolidate connection data from distributed Delivery Service nodes to determine if a user is truly "online" or just disconnected from one specific device.
- **Session Tracking:** Maintain an ephemeral registry of active `(user_id, device_id)` pairs and map them to the specific Delivery `instance_id` they are connected to (crucial for targeted routing).
- **Heartbeat & Expiry:** Enforce Time-To-Live (TTL) mechanisms to automatically mark users offline if their device loses network without cleanly disconnecting.
- **Event Broadcasting:** Publish presence changes (e.g., `UserOnline`, `UserOffline`) to the broader system using high-throughput Redis Pub/Sub.

---

## 2. Data Model Overview

Because presence data is inherently ephemeral (it is irrelevant if a user was online 5 days ago if we only care about real-time routing), this service entirely avoids persistent relational databases like PostgreSQL. 

### Core Datastore: Redis

The service leverages Redis for atomic, microsecond-latency operations.

* **`session:{user_id}:{device_id}`**: A Redis `String` storing the ID of the Delivery node handling the connection.
  * **TTL**: Set to 60 seconds. Requires continuous refreshing (heartbeats from Delivery).
* **`presence:user:{user_id}:devices`**: A Redis `Set` (unordered collection) holding all active `device_id`s for a specific user.
  * **TTL**: Slightly longer than individual sessions to ensure the set isn't wiped out before final cleanup.
* **`presence:updates` (Pub/Sub)**: A Redis channel used to instantly broadcast binary Protocol Buffer events (`PresenceUpdateEvent`) to downstream consumers (like the Profile service or caching layers).

---

## 3. Transaction Flow: Going Online / Offline

Presence relies on pipelined Redis transactions to ensure device sets and session keys stay strictly synchronized.

### The "Online" Flow (Register / Refresh)
1. Delivery Service accepts a WebSocket connection.
2. It sends a gRPC `Register` or `Refresh` request to the Presence Service.
3. **Atomic Pipeline:** Presence opens a Redis `TxPipeline`:
   - `SET session:{uID}:{dID} {instance_id} EX 60`
   - `SADD presence:user:{uID}:devices {dID}`
4. If this is a new device, a `PresenceStatus_ONLINE` event is serialized and published to the `presence:updates` channel.

### The "Offline" Flow (Unregister / Timeout)
1. Delivery Service detects a dropped WebSocket (or a client gracefully disconnects) and calls `Unregister`.
2. **Atomic Pipeline:**
   - `DEL session:{uID}:{dID}`
   - `SREM presence:user:{uID}:devices {dID}`
3. **Aggregation Check:** The service checks if the user has *any other* active devices left in their `Set`.
4. If the device set is now empty, a `PresenceStatus_OFFLINE` event is published to the `presence:updates` channel.

---

## 4. Why Redis Pub/Sub Over Kafka?

While Message, Profile, and Conversation services use Kafka with the Outbox pattern for durable, guaranteed event delivery, the Presence Service intentionally uses **Redis Pub/Sub** for broadcasting `PresenceUpdateEvent`s.

- **Ephemeral Nature:** If a downstream consumer misses an "Online" event, they can simply query the Presence gRPC API to get the current state. Durable replay (Kafka's main strength) is unnecessary for real-time presence.
- **Latency:** Redis Pub/Sub operates entirely in RAM, offering slightly lower latency than Kafka for instant UI updates (like the green dot next to a user's avatar).

---

## 5. Delivery Guarantees

- **At-Most-Once (Broadcasting):** Redis Pub/Sub does not persist messages. If a consumer is temporarily offline or crashes during publication, it misses the presence update.
- **Eventual Consistency via Polling:** To compensate for the lack of durable publish, services that care deeply about status (like Profile) occasionally poll the `/Get` gRPC endpoint to self-heal missed Pub/Sub events.

---

## 6. Failure Handling

| Failure Scenario | System Behavior | Resolution / Recovery |
| :--- | :--- | :--- |
| **Silent Device Crash** | The user's phone dies. No `Unregister` is sent. | The 60-second Redis TTL expires naturally. On the next read, the service detects the missing session key, cleans up the stale element in the `Set`, and eventually broadcasts `Offline`. |
| **Delivery Node Crash** | Thousands of web sockets drop without sending `Unregister`. | Redis TTL handles mass expiry automatically. No custom cleanup logic is required in the Presence Service. |
| **Redis Down** | The service cannot read/write status. All gRPC calls fail. | RealChat operates in a degraded state (everyone appears offline). Heartbeats fail but Delivery nodes keep WebSockets alive. Recovers instantly when Redis boots. |

---

## 7. Scalability Considerations

- **Redis Bottlenecks:** A single Redis instance can typically handle ~100k ops/sec. Given continual 60-second heartbeats for potentially millions of concurrent users, a single instance will bottleneck. 
- **Redis Cluster:** To scale, the Redis layer must be deployed as a Redis Cluster. The keys (`session:{uID}:*` and `presence:user:{uID}:*`) are carefully constructed to allow Redis Hash Tags (`{uID}`) in the future, ensuring all data for a specific user lands on the same shard for atomic pipeline operations.
- **Stateless API Horizon:** The Presence Go application itself is entirely stateless. It can be horizontally scaled infinitely behind a load balancer, as all state mutation is pushed down to Redis.

---

## 8. Tradeoffs

* **Redis Cluster Complexity vs Single Node:** While a single node is easy to manage, high-scale presence *requires* sharding. 
* **Heartbeat Overhead vs Accuracy:** A 60-second TTL means a user might appear "Online" for up to a minute after their phone dies. Lowering the TTL (e.g., 15 seconds) increases tracking accuracy but quadruples the network and Redis write load. 60 seconds represents a balanced tradeoff for battery/network efficiency.
