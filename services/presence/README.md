# Presence Service

The **Presence** service is a highly responsive, high-throughput microservice in RealChat responsible for tracking the online/offline statuses of all connected users across their devices in real time.

## 🚀 Responsibilities & Features

- **Live Status Tracking**: Maintains accurate mappings of which user is online, and from which specific device.
- **Session Refreshes**: Heartbeat mechanisms keep user sessions alive and proactively detect disconnections or dropped websockets to update offline status.
- **Status Broadcasting**: Emits state transitions (Online/Offline) enabling client UIs to instantly show green/grey presence dots next to user avatars.
- **Multi-Device Support**: Efficiently handles cases where a single user is logged in onto multiple devices simultaneously.

## 🔄 Event-Driven Architecture (EDA) Integration

The Presence service relies heavily on asynchronous event-driven flows to minimize overhead during frequent status changes:

- **State Broadcasting**: Rather than having clients explicitly poll for presence changes, the service publishes `UserOnline` and `UserOffline` events directly to **Apache Kafka**.
- **Delivery Fan-out Hub**: The Delivery Service consumes these Kafka presence events and pushes the live status updates to any client who shares a conversation or friend-list with the state-changed user.
- **Ephemeral Event Flow**: Presence events are treated as ephemeral and high-throughput, bypassing slow database persistence in favor of Redis and rapid Kafka propagation.

## 📡 API Contract (gRPC)

The service exposes the following RPC methods defined in `presence.proto`:

| RPC Method | Request payload | Response payload | Description |
| :--- | :--- | :--- | :--- |
| `GetPresence` | `GetPresenceRequest` (list of user_ids) | `GetPresenceResponse` (List of UserPresence) | Fetches the current online/offline status for a batch of users. |
| `RegisterSession` | `RegisterSessionRequest` (user, device, instance) | `RegisterSessionResponse` | Called by Delivery Service upon a WebSocket connect. |
| `UnregisterSession` | `UnregisterSessionRequest` (user, device) | `UnregisterSessionResponse` | Called by Delivery Service upon a WebSocket disconnect. |
| `RefreshSession` | `RefreshSessionRequest` (user, device) | `RefreshSessionResponse` | Updates the TTL heartbeat of a session to keep it alive. |
| `GetUserDevices` | `GetUserDevicesRequest` (user_id) | `GetUserDevicesResponse` (List of DeviceInfo) | Returns all currently active websocket sessions for a user. |

## 🛠 Tech Stack & Architecture

- **Language**: Go
- **Communication Protocol**: gRPC (`realchat.presence.v1.PresenceApi`)
- **Database**: Redis (Provides the high-throughput, low-latency KV store required for storing transient ephemeral TTL-based user sessions).

## ⚙️ Running Locally

Typically started using the Docker Compose setup:
```bash
cd infra/local
docker compose up -d
```

To run independently during development:
```bash
cd services/presence
go run cmd/main.go
```
*Note: Ensure Redis is running locally since presence state heavily relies on it.*
