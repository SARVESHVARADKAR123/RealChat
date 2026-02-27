# Delivery Service

## 1. Service Overview

The **Delivery Service** is the real-time edge of the RealChat ecosystem. It maintains long-lived WebSocket connections with client devices, acting as the primary push-notification layer for active sessions. It bridges the asynchronous, event-driven backend (Kafka) directly to synchronous user devices.

**Core Responsibilities:**
- **WebSocket Management:** Handle thousands of concurrent, long-lived bidirectional WebSocket connections.
- **Event Routing:** Consume real-time events from Kafka (e.g., new messages, profile updates) and fan them out to the correct, connected user devices.
- **Buffering & Delivery Guarantees:** Buffer incoming Kafka events for clients that are temporarily disconnected or handling sequence gaps, ensuring smooth real-time rendering on the UI.
- **Heartbeating & Presence:** Track active connections and publish connection state changes to the Presence Service.

---

## 2. Infrastructure Overview

Unlike the core domain services (Message, Profile), the Delivery Service is entirely stateless regarding permanent data storage. It acts as an intelligent router and relies on in-memory state and Redis.

### Core Components

* **WebSocket Registry (`internal/websocket/registry.go`)**: An in-memory data structure mapping `user_id` -> `[]Session`. A single user can have multiple concurrent active sessions (e.g., iPhone and Desktop).
* **Kafka Consumers (`internal/kafka/`)**: High-throughput consumers listening to various domain topics (`chat.messages.v1`, `chat.profiles.v1`).
* **Redis Pub/Sub & Presence (`internal/presencewatcher/`)**: Because clients can connect to *any* Delivery Service node behind the load balancer, Redis is used to track which node holds the active connection for a specific `user_id`.

---

## 3. Transaction Flow: Delivering a Message

When a user receives a new chat message, the delivery flow bypasses traditional databases entirely for speed:

1. **Kafka Consumption:** A Delivery Service node consumes a `MessageSentEvent` from the `"messages-topic"`.
2. **Local Registry Check:** The node checks its internal `WebSocket Registry` to see if the recipient `user_id` is currently connected to *this specific node*.
3. **Local Delivery:** If connected, the event payload is serialized to binary (Protobuf) and pushed into the specific WebSocket session's `SendQueue`.
4. **Buffering / Backpressure:** If the client's `SendQueue` is full (client is reading too slowly), the node applies backpressure. If the buffer overflows, the connection is intentionally dropped to protect the server's memory.
5. **Cross-Node Routing (Optional depending on architecture):** If the user is connected to a *different* Delivery node, the Kafka consumer group naturally ensures that the event is processed by a node. If partitioned by `user_id`, the correct node processes it. Alternatively, nodes use Redis Pub/Sub to forward the event to the node holding the connection.

---

## 4. Connection Lifecycle & Heartbeats

To keep load balancers from dropping idle connections and to accurately track online status:

- **Server-Side Pings:** The Delivery Service sends a WebSocket `PingMessage` to the client every `54 seconds` (`pingPeriod`).
- **Client-Side Pongs:** The client must respond with a `PongMessage` within `60 seconds` (`pongWait`).
- **Disconnection:** If a pong is missed, or a write deadline fails, the service forcefully closes the connection, removes the session from the registry, and emits a `UserDisconnected` event to the Presence Service.

---

## 5. Delivery Guarantees

- **At-Most-Once (Real-Time):** The Delivery Service only guarantees "fire and forget" push over the WebSocket. If a client is offline or drops packets instantly after the WebSocket frame is sent, the Delivery Service does *not* retry.
- **Reliability via Sync:** Clients are expected to hit the Message Service's `SyncMessages` API upon reconnection to fetch any missed events. The Delivery Service's sole job is *speed*, not durable persistence.
- **Ordered Buffering:** If events arrive slightly out of order from Kafka, the `Session` struct can buffer and sort them by `Sequence` before flushing them down the WebSocket, ensuring the client UI doesn't render messages out of order.

---

## 6. Failure Handling

| Failure Scenario | System Behavior | Resolution / Recovery |
| :--- | :--- | :--- |
| **Delivery Node Crash** | All WebSockets connected to that specific node drop instantly. | Clients detect the dropped TCP connection and automatically reconnect to the load balancer, which routes them to a healthy node. |
| **Kafka Broker Down** | WebSockets remain established. Real-time push stops. | Once Kafka recovers, consumers resume reading and push the backlog of events to connected clients. |
| **Client Network Drop** | Ping/Pong heartbeat fails. | Server drops the session and cleans up memory. Client reconnects when network is restored. |
| **Client Reads Too Slowly** | Server `SendQueue` (size 128) fills up. | Server logs `backpressure overflow` and gracefully terminates the session via `CloseWithReason`. Client must reconnect and sync. |

---

## 7. Scalability Considerations

- **Memory Bound:** The primary bottleneck of the Delivery Service is RAM. Each concurrent WebSocket connection (and its associated go-routines and `SendQueue` channel buffers) consumes memory. Scaling involves adding more horizontal instances.
- **OS File Descriptors:** Because every WebSocket is a TCP connection, the host OS must be configured to allow a massive number of open file descriptors (`ulimit -n`).
- **Kafka Partitioning Strategy:** To maximize efficiency, Kafka topics should be partitioned by `user_id`. This ensures that all events destined for a specific user go to the same consumer, preventing the need for complex cross-node Redis broadcasting.

---

## 8. Tradeoffs

* **Stateless Push vs Guaranteed Delivery:** RealChat explicitly trades strict delivery guarantees over the WebSocket for extreme low-latency and low-overhead push. Heavylifting for guaranteed consistency is completely offloaded to the client's reconnect-and-sync logic.
* **Go Channels per Session:** Allocating a `chan []byte` for every single connected user makes the code incredibly clean and concurrent-safe, but adds a small baseline memory footprint per user compared to lower-level epoll/kqueue event loop architectures (like in C or Rust).
