# Conversation Service

The **Conversation** service is the central authority for managing the lifecycles, permissions, and participants of all chat groups and direct messages within RealChat. 

## 🚀 Responsibilities & Features

- **Conversation Lifecycles**: Creates and fetches both 1-on-1 and Group chats.
- **Participant Tracking**: Manages the roster of users in groups and controls access.
- **Atomic Sequence Management**: Generates incrementing, gapless sequence numbers for each message appended to a conversation to guarantee precise ordering across all clients.
- **Read Receipts**: Tracks the read watermarks for each user per conversation to power UI indicators.

## 🔄 Event-Driven Architecture (EDA) Integration

The Conversation service participates in the broader EDA by emitting lifecycle events that other services or clients can react to:

- **Lifecycle Events**: Publishes events to **Apache Kafka** (e.g., `ConversationCreated`, `ParticipantAdded`, `ParticipantRemoved`) when chat groups or direct messages are structurally mutated.
- **Read Receipt Events**: Emits changes to users' read watermarks so that Delivery can push these updates in real-time to other participants in the conversation.

## 📡 API Contract (gRPC)

The service exposes the following RPC methods defined in `conversation_api.proto`:

| RPC Method | Request payload | Response payload | Description |
| :--- | :--- | :--- | :--- |
| `CreateConversation` | `CreateConversationRequest` (`conversation_id`, `type`, `display_name`, `avatar_url`, `participant_user_ids`) | `CreateConversationResponse` (Conversation) | Initiates a new group or direct conversation. |
| `ListConversations` | `ListConversationsRequest` (`user_id`) | `ListConversationsResponse` (Conversations) | Returns a user's active conversations. |
| `GetConversation` | `GetConversationRequest` (`conversation_id`) | `GetConversationResponse` (Conversation, participants) | Returns deep details of a specific conversation and its participant list. |
| `AddParticipant` | `AddParticipantRequest` (`conversation_id`, `actor_user_id`, `target_user_id`) | `AddParticipantResponse` | Adds a user to an existing conversation. |
| `RemoveParticipant` | `RemoveParticipantRequest` (`conversation_id`, `actor_user_id`, `target_user_id`) | `RemoveParticipantResponse` | Removes/kicks a user from a conversation. |
| `UpdateReadReceipt` | `UpdateReadReceiptRequest` (`conversation_id`, `user_id`, `read_sequence`) | `UpdateReadReceiptResponse` | Updates the high-water mark of a user's read state. |
| `NextSequence` | `NextSequenceRequest` (`conversation_id`) | `NextSequenceResponse` (sequence) | Atomically increments and returns the next message sequence number for strong ordering. |

## 🛠 Tech Stack & Architecture

- **Language**: Go
- **Communication Protocol**: gRPC (`realchat.conversation.v1.ConversationApi`)
- **Database**: PostgreSQL (Stores conversation metadata and participant mappings).

## ⚙️ Running Locally

Started primarily via Docker Compose:
```bash
cd infra/local
docker compose up -d
```

To run independently:
```bash
cd services/conversation
go run cmd/main.go
```
