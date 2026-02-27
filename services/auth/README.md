# 🔐 Auth Service

The **Auth Service** is the identity management core of the RealChat distributed system. 
It securely handles user registration, authentication, JWT token issuance, and password management.

---

## 🎯 1. Responsibilities

| Responsibility | Description |
| :--- | :--- |
| **Registration** | Creates new accounts, hashes passwords, and saves credentials. |
| **Authentication** | Validates email/password and issues access & refresh tokens. |
| **JWT Issuance** | Generates short-lived stateless JSON Web Tokens. |
| **Token Validation** | Provides keys to the API Gateway to validate tokens independently. |
| **Session Control** | Manages refresh token rotation and handles secure user logouts. |
| **Event Publishing** | Emits `USER_CREATED` events to provision resources in other services asynchronously. |

---

## 🔑 2. JWT Strategy

| Feature | Details |
| :--- | :--- |
| **Algorithm** | Symmetric **HS256** (HMAC + SHA-256) |
| **Access Tokens** | Short-lived & stateless. Expires in **60 mins** (by default). Contains `sub`, `iss`, `aud`, `iat`, `exp` claims. |
| **Refresh Tokens** | Long-lived & stateful. Expires in **24 hours** (by default). Consumed and rotated on each use. |

*Keys and expiration times are fully configurable via environment variables.*

---

## 🛡️ 3. Password Security

- **Algorithm**: `bcrypt`
- **Work Factor (Cost)**: `12`
- **Zero Plaintext Principle**: Passwords are never logged, exposed via APIs, or kept in memory longer than necessary.

---

## ⚠️ 4. Failure Cases

How the service gracefully handles edge cases:

- **Invalid Credentials**: Returns a generic `InvalidCredentials` error for both wrong passwords and missing users (prevents user enumeration).
- **Expired/Invalid Refresh Tokens**: Returns `InvalidToken`, forcing the user to re-login.
- **Database Outages**: Fails securely with an internal error, never leaking infrastructure details.

---

## 🔒 5. Security Considerations

- **Secure Storage**: Refresh tokens are cryptographically hashed (**SHA-256**) *before* they hit the database. A database dump does not expose usable refresh tokens.
- **Instant Revocation**: Logging out instantly revokes exactly one refresh token family, letting users kick stolen devices off the platform.

---

## 🌐 6. Integration with Gateway

- **Public Routes**: The API Gateway blindly proxies unauthenticated routes directly to this service (`/register`, `/login`).
- **Decentralized Validation**: The Gateway holds the `JWT_SECRET` and validates tokens on its own. The Auth Service acts purely as the **issuer**, dramatically cutting down internal network traffic.

---

## 📈 7. Scalability

- **100% Stateless**: No session data is stored in memory. The service can be horizontally scaled infinitely behind a load balancer.
- **Edge Offloading**: Access token validation happens at the edge (API Gateway). The Auth database is only queried during initial logins, registrations, and token refreshes.
- **Event-Driven**: Post-registration side-effects are delegated to Kafka via the **Transactional Outbox**, keeping the registration endpoint extremely fast.
