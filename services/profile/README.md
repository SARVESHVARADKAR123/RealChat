# Profile Service

The **Profile** service manages user identities, metadata, and social graphs within the RealChat ecosystem. It acts as the central repository for user-facing information.

## 🚀 Responsibilities & Features

- **Profile Management**: Stores and serves user details including display names, avatars, and bios.
- **Batch Processing**: Highly optimized to fetch multiple user profiles at once (essential for rendering group chats and friend lists efficiently).
- **Data Synchronization**: Ensures timestamps (created/updated) are consistently maintained for profile modifications.

## 🔄 Event-Driven Architecture (EDA) Integration

The Profile service interacts with the broader event-driven system to maintain eventual consistency:

- **Event Consumption**: It can consume events like `UserRegistered` (from Auth) to automatically provision default profiles seamlessly in the background.
- **Profile Updates**: When a user changes their avatar or display name, it can publish `ProfileUpdated` events, allowing other services (like Delivery or Client UIs) to invalidate stale cached data locally or push updates.

## 📡 API Contract (gRPC)

The service exposes the following RPC methods defined in `profile_api.proto`:

| RPC Method | Request payload | Response payload | Description |
| :--- | :--- | :--- | :--- |
| `GetProfile` | `GetProfileRequest` (user_id) | `Profile` | Retrieves the public profile for a single user. |
| `UpdateProfile` | `UpdateProfileRequest` (user_id, optional fields) | `Profile` | Updates an existing user's profile metadata. |
| `BatchGetProfiles` | `BatchGetProfilesRequest` (list of user_ids) | `BatchGetProfilesResponse` (list of Profiles) | Efficiently fetches profile data for multiple users in one call. |

## 🛠 Tech Stack & Architecture

- **Language**: Go
- **Communication Protocol**: gRPC (`realchat.profile.v1.ProfileApi`)
- **Database**: PostgreSQL (Stores user metadata, display names, and avatars)

## ⚙️ Running Locally

The Profile service is typically started via the main Docker Compose infrastructure setup:
```bash
cd infra/local
docker compose up -d
```

To run independently during development:
```bash
cd services/profile
go run cmd/main.go
```
*Note: Ensure PostgreSQL is running and the environment variables are correctly set.*
