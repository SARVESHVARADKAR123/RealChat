# RealChat

RealChat is a modern, high-performance, microservices-based real-time chat application built with Go. It leverages a robust architecture featuring gRPC for internal communication, Kafka for event-driven messaging, and WebSockets for real-time delivery.

## 📋 Table of Contents

- [Features](#-features)
- [Architecture](#-architecture)
- [Tech Stack](#-tech-stack)
- [Services](#-services)
- [Getting Started](#-getting-started)
- [Project Structure](#-project-structure)

## ✨ Features

- **Real-Time Messaging**: Low-latency message delivery using WebSockets.
- **Group & Direct Chats**: Support for 1-on-1 conversations and multi-user groups with admin controls.
- **Event-Driven**: Reliable message processing and delivery backed by Apache Kafka outbox pattern.
- **Scalable Microservices**: Independent, specialized services communicating efficiently via gRPC.
- **Presence Tracking**: Real-time online/offline status indicators.
- **Comprehensive Observability**: Built-in distributed tracing (Jaeger) and metrics (Prometheus/Grafana).

## 🚀 Architecture

The system is designed with a focus on scalability, observability, and decoupled interactions through an Event-Driven Architecture (EDA). It consists of multiple independent services coordinated via an API Gateway and Apache Kafka.

### Core Architecture
- **API Gateway**: The entry point for all HTTP and WebSocket requests, providing unified routing, rate-limiting, and authentication validation.
- **Microservices**: Specialized services handling authentication, user profiles, messaging, conversations, and presence via synchronous gRPC.
- **Real-Time Delivery**: A dedicated delivery service manages WebSocket connections for pushing live updates to clients.

### Event-Driven Architecture (EDA)
RealChat heavily relies on an Event-Driven Architecture to decouple services and ensure reliable, asynchronous processing:
- **Transactional Outbox Pattern**: The Message service persists chat messages to a database and atomically writes an event to an `outbox` table in the same transaction.
- **Event Streaming**: Background processes stream outbox events into **Apache Kafka** topics, guaranteeing at-least-once delivery without dropping messages.
- **Asynchronous Fan-out**: The Delivery service continuously consumes these Kafka topics and fans out the events to globally distributed WebSocket clients.
- **Presence & Lifecycle Events**: User connect/disconnect events, typing indicators, and online status changes are modeled as asynchronous events, reducing synchronous bottlenecks.

## 🛠 Tech Stack

- **Language**: [Go](https://go.dev/)
- **Communication**: [gRPC](https://grpc.io/), [Protobuf](https://protobuf.dev/)
- **Real-time**: WebSockets
- **Event Streaming**: [Apache Kafka](https://kafka.apache.org/) (Confluent Platform)
- **Databases**: [PostgreSQL](https://www.postgresql.org/), [Redis](https://redis.io/)
- **Observability**: [Jaeger](https://www.jaegertracing.io/), [Prometheus](https://prometheus.io/), [Grafana](https://grafana.com/)
- **Infrastructure**: Docker & Docker Compose

## 📦 Services

| Service | Description | Ports | Readme |
| :--- | :--- | :--- | :--- |
| **Gateway** | API Routing & Aggregation | `8080` (HTTP), `8090` (Internal) | [Docs](./edge/gateway/README.md) |
| **Auth** | User Auth & JWT Management | `8081` (HTTP), `50051` (gRPC), `8091` (Internal) | [Docs](./services/auth/README.md) |
| **Profile** | User Profiles & Relationships | `8082` (HTTP), `50052` (gRPC), `8092` (Internal) | [Docs](./services/profile/README.md) |
| **Message** | Message Persistence & History | `50053` (gRPC), `8094` (Internal) | [Docs](./services/message/README.md) |
| **Conversation**| Group & Direct Chat Logic | `50055` (gRPC), `8095` (Internal) | [Docs](./services/conversation/README.md) |
| **Presence** | Real-time Online/Offline Status | `50056` (gRPC), `8096` (Internal) | [Docs](./services/presence/README.md) |
| **Delivery** | WebSocket Event Delivery | `8083` (WS), `8093` (Internal) | [Docs](./services/delivery/README.md) |

## 🚀 Getting Started

### Prerequisites
- [Docker](https://www.docker.com/) & Docker Compose
- [Go](https://go.dev/) 1.25+

### Running the Project

1. **Clone the repository**:
   ```bash
   git clone <repository-url>
   cd RealChat
   ```

2. **Spin up the infrastructure**:
   Navigate to the local infra directory and start the services:
   ```bash
   cd infra/local
   docker compose up -d
   ```
   This will start all databases, message brokers, observability tools, and the microservices themselves.

3. **Verify running services**:
   ```bash
   docker ps
   ```

### Observability Dashboards
- **Jaeger (Traces)**: [http://localhost:16686](http://localhost:16686)
- **Prometheus (Metrics)**: [http://localhost:9090](http://localhost:9090)
- **Grafana (Dashboards)**: [http://localhost:3000](http://localhost:3000)

## 📂 Project Structure

- `/contracts`: Protobuf definitions and generated code.
- `/edge/gateway`: The API Gateway service.
- `/services`: Core microservices (auth, conversation, delivery, message, presence, profile).
- `/infra`: Docker Compose and infrastructure configuration (Postgres, Kafka, etc.).
- `/shared`: Shared Go libraries and utilities.
- `go.work`: Go Workspace file for multi-module development.
