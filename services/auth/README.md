# Auth Service

The **Auth** service is a core microservice in the RealChat application responsible for identity management. It handles user registration, authentication (login/logout), and secure token lifecycle management using JSON Web Tokens (JWT).

## 🚀 Responsibilities & Features

- **User Registration**: Securely registers new users, hashes passwords using bcrypt, and persists credentials.
- **Authentication**: Validates credentials and issues short-lived access tokens and long-lived refresh tokens.
- **Token Management**: Implements secure refresh flows to maintain seamless user sessions without requiring repeated logins.
- **Logout Support**: Invalidates refresh tokens to securely terminate user sessions.

## 🔄 Event-Driven Architecture (EDA) Integration

While primarily a synchronous service handling direct client requests, the Auth service contributes to the EDA ecosystem:

- **Lifecycle Events**: Upon successful registration, the service may publish a `UserRegistered` event to **Apache Kafka**.
- **Asynchronous Onboarding**: Downstream services (like Profile) can asynchronously consume these events to initialize default sub-records (e.g., creating a blank profile) without slowing down the initial registration request.

## 📡 API Contract (gRPC)

The service exposes the following RPC methods defined in `auth_api.proto`:

| RPC Method | Request payload | Response payload | Description |
| :--- | :--- | :--- | :--- |
| `Register` | `RegisterRequest` (email, password) | `RegisterResponse` (user_id) | Creates a new user account. |
| `Login` | `LoginRequest` (email, password) | `LoginResponse` (access_token, refresh_token) | Authenticates a user and returns JWTs. |
| `Refresh` | `RefreshRequest` (refresh_token) | `RefreshResponse` (access_token, refresh_token) | Issues a new access token using a valid refresh token. |
| `Logout` | `LogoutRequest` (refresh_token) | `LogoutResponse` | Revokes the provided refresh token. |

## 🛠 Tech Stack & Architecture

- **Language**: Go
- **Communication Protocol**: gRPC (`realchat.auth.v1.AuthApi`)
- **Database**: PostgreSQL (Stores user credentials and hashed passwords securely)
- **Security**: JWT (Access & Refresh tokens), bcrypt (Password hashing)

## ⚙️ Running Locally

The Auth service is typically started alongside its PostgreSQL dependency using Docker Compose:
```bash
cd infra/local
docker compose up -d
```

To run independently during development:
```bash
cd services/auth
go run cmd/main.go
```
*Note: Ensure PostgreSQL is running and the database connection string is properly configured.*
