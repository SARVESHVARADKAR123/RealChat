# Gateway Service

The **Gateway** service (`edge/gateway`) serves as the single entry point (API Gateway) for all client HTTP and WebSocket upgrade requests in the RealChat application. It acts as a reverse proxy, interpreting outward-facing RESTful requests and routing them to the appropriate internal gRPC microservices.

## 🚀 Responsibilities & Features

- **API Routing & Aggregation**: Provides clean, unified HTTP REST endpoints that map to internal gRPC backend endpoints, reducing client complexity.
- **Authentication & Security**: Validates JWT tokens using middleware before passing any protected requests downstream. It rejects unauthorized requests directly at the edge.
- **Observability**: Automatically instruments tracing (via OpenTelemetry) and metrics (Prometheus) on all incoming requests.
- **Resilience**: Implements panic recovery and centralized request ID tracing.

## 📡 API Contract (REST HTTP)

The Gateway exposes the following RESTful endpoints:

### Public Endpoints (No Auth Required)
| Method | Endpoint | Internal Service Targeted |
| :--- | :--- | :--- |
| `POST` | `/api/login` | `Auth` (Login) |
| `POST` | `/api/register` | `Auth` (Register) |
| `POST` | `/api/refresh` | `Auth` (Refresh) |
| `POST` | `/api/logout` | `Auth` (Logout) |

### Protected Endpoints (Requires valid JWT `Authorization: Bearer <token>`)
| Method | Endpoint | Internal Service Targeted |
| :--- | :--- | :--- |
| `GET` | `/api/profile` | `Profile` (GetProfile) |
| `PATCH` | `/api/profile` | `Profile` (UpdateProfile) |
| `POST` | `/api/conversations` | `Conversation` (CreateConversation) |
| `GET` | `/api/conversations` | `Conversation` (ListConversations) |
| `GET` | `/api/conversations/{id}` | `Conversation` (GetConversation) |
| `GET` | `/api/messages` | `Message` (SyncMessages) |
| `POST` | `/api/messages` | `Message` (SendMessage) |
| `DELETE` | `/api/messages` | `Message` (DeleteMessage) |
| `POST` | `/api/participants` | `Conversation` (AddParticipant) |
| `DELETE` | `/api/participants`| `Conversation` (RemoveParticipant) |
| `POST` | `/api/read-receipt` | `Conversation` (UpdateReadReceipt) |
| `GET` | `/api/presence` | `Presence` (GetPresence) |

## 🛠 Tech Stack & Architecture

- **Language**: Go
- **Server**: HTTP Server using `go-chi/chi` for fast, idiomatic routing.
- **Clients**: gRPC Clients for communicating with internal backend services.
- **Telemetry**: OpenTelemetry (`otelhttp`)

## ⚙️ Running Locally

Typically started via the main Docker Compose file in `infra/local`. 

To run independently during development:
```bash
cd edge/gateway
go run cmd/main.go
```
*Make sure necessary backend dependencies are running and their gRPC ports are correctly mapped in the gateway's configuration.*
