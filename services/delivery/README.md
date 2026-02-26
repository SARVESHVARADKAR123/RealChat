# Delivery Service

The **Delivery** service is the stateless connection-holding engine of the RealChat architecture. It manages active WebSocket connections for thousands of concurrently connected clients, pushing real-time messages directly to users.

## 🚀 Responsibilities & Features

- **WebSocket Management**: Maintains thousands of simultaneous full-duplex WebSocket connections per node.
- **Presence Notification**: Informs the external Presence Service when clients gracefully or ungracefully connect, disconnect, or drop.
- **Heartbeat & Pinging**: Sends and expects ping/pong frames over WebSockets to identify inactive network drops.

## 🔄 Event-Driven Architecture (EDA) Integration

The Delivery service operates primarily as a **Consumer** of events within the RealChat ecosystem, bridging the asynchronous backend with synchronous real-time clients:

- **Kafka Event Consumption**: Subscribes to backend Apache Kafka topic streams (e.g., `chat.messages.v1` published by the Message service, or presence events).
- **Asynchronous Fan-out**: Consumes messages at high throughput from Kafka partition logs independently of the publishers, and accurately fans out these payloads to precisely the target user sessions currently connected to that node.
- **Decoupled Delivery**: Because Message ingestion and Delivery are decoupled by Kafka, a massive spike in concurrent users receiving messages will not overload the databases or write-paths.

## 📡 API Contract (gRPC & WebSocket)

The internal routing logic uses a stream-based gRPC API (`delivery_api.proto`), but the service's *primary* function is consuming from Kafka and exposing external WebSocket connections.

| RPC Method / Action | Flow Type | Description |
| :--- | :--- | :--- |
| `Delivery Connect` | gRPC Stream / WebSocket Upgrade | Accepts incoming connections mapping a `user_id` and `device_id` to an active duplex stream. Pushes `DeliveryEvent` packages containing sequences and payloads. |
| **Kafka Consumption** | Messaging Topic | Constantly polls Kafka partition logs to read new un-sent messages and push them towards the necessary WebSocket queues. |

## 🛠 Tech Stack & Architecture

- **Language**: Go
- **Communication**: gRPC (Internal stream management), WebSockets (Client-facing real-time sockets)
- **Event Streaming**: Apache Kafka (Consumes message events from other services like `Message` or `Presence`).
- **Dependencies**: Redis (Sometimes used for instance mapping or Pub/Sub fanouts when routing messages across multiple scaling Delivery nodes).

## ⚙️ Running Locally

Typically spun up via Docker Compose due to heavy Kafka dependencies:
```bash
cd infra/local
docker compose up -d
```

To run independently during development:
```bash
cd services/delivery
go run cmd/main.go
```
