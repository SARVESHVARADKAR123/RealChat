# Message Service

The **Message** service acts as the source of truth for all chat history and message persistence across conversations within RealChat. It serves both reads (sync requests) and writes, while abstracting messaging persistence from downstream processing (like push notifications) using the outbox pattern.

## 🚀 Responsibilities & Features

- **Store-and-Forward Processing**: Stores sent messages securely in a highly durable PostgreSQL database.
- **Transactional Outbox Pattern**: Upon saving a new message, an event is atomically appended to an 'outbox' table. A separate process streams these events to Apache Kafka ensuring message events are never lost and downstream services always receive them.
- **Chat History Retrieval**: Provides paginated, sequence-based message synchronization logic, allowing clients to cleanly fetch messages sent after their last known sequence ID.
- **Message Deletion**: Support for hard or soft message deletion from history.

## 📡 API Contract (gRPC)

The service exposes the following RPC methods defined in `message_api.proto`:

| RPC Method | Request payload | Response payload | Description |
| :--- | :--- | :--- | :--- |
| `SendMessage` | `SendMessageRequest` (conversation_id, sender_user_id, idempotency_key, content, etc.) | `SendMessageResponse` (Message) | Ingests and persists a new message, assigns a precise sequence number via Conversation API, and emits an event via Outbox. |
| `DeleteMessage` | `DeleteMessageRequest` (conversation_id, message_id, actor_user_id) | `DeleteMessageResponse` | Deletes a message if the actor is authorized. |
| `SyncMessages` | `SyncMessagesRequest` (conversation_id, after_sequence, page_size) | `SyncMessagesResponse` (Messages) | Pulls sequential chat history efficiently to sync disconnected clients. |

## 🛠 Tech Stack & Architecture

- **Language**: Go
- **Communication Protocol**: gRPC (`realchat.message.v1.MessageApi`)
- **Event Streaming**: Apache Kafka (Messages are published to Kafka topics for consumption by Delivery or Push Notification services).
- **Database**: PostgreSQL (Stores raw messages and the transactional outbox table).

## ⚙️ Running Locally

The Message service is typically started via `infra/local` using Docker Compose, as it is heavily reliant on Kafka and Postgres:
```bash
cd infra/local
docker compose up -d
```

To run independently during development:
```bash
cd services/message
go run cmd/main.go
```
*Note: Ensure PostgreSQL and Kafka (Zookeeper/Broker) are running locally.*
