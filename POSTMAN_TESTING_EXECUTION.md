# Detailed Postman Execution Guide for RealChat

---

## Prerequisites

Before running any request ensure you have:

1. Imported the `realchat.postman_collection.json` collection.
2. Imported the `realchat.postman_environment.json` environment and selected it.
3. Set the environment variable `{{base_url}}` to the address of your running gateway (e.g., `http://localhost:8080`).
4. (Optional) Run **Register** once to create a test user, then **Login** to populate `jwt_token`.

> **All requests route through the gateway.** The gateway listens on `{{base_url}}` and proxies to the appropriate backend service via gRPC. Never call backend services directly.

---

## Auth Endpoints (no JWT required)

### 1. Register (`POST /api/register`)

1. **Select** – **Auth → Register**.
2. **Method & URL** – `POST {{base_url}}/api/register`.
3. **Body** – raw JSON:
   ```json
   {
     "email": "testuser@example.com",
     "password": "StrongPass123"
   }
   ```
4. **Send**.
5. **Verify** –
   - Status **201**.
   - Response body contains `user_id`.
6. **Note** – No token is returned on registration; use Login next.

---

### 2. Login (`POST /api/login`)

1. **Select** – **Auth → Login**.
2. **Method & URL** – `POST {{base_url}}/api/login`.
3. **Body** – raw JSON:
   ```json
   {
     "email": "testuser@example.com",
     "password": "StrongPass123"
   }
   ```
4. **Send**.
5. **Verify** –
   - Status **200**.
   - Response contains `access_token` and `refresh_token`.
6. **Token capture** – Add to the **Tests** tab:
   ```javascript
   pm.environment.set("jwt_token", pm.response.json().access_token);
   pm.environment.set("refresh_token", pm.response.json().refresh_token);
   ```
   Confirm both variables are populated in the **Environment** quick-look (eye icon).
7. **Proceed** – All subsequent requests include `Authorization: Bearer {{jwt_token}}` automatically via the collection pre-request script.

---

### 3. Refresh Token (`POST /api/refresh`)

1. **Select** – **Auth → Refresh**.
2. **Method & URL** – `POST {{base_url}}/api/refresh`.
3. **Body** – raw JSON:
   ```json
   { "refresh_token": "{{refresh_token}}" }
   ```
4. **Send**.
5. **Verify** –
   - Status **200**.
   - New `access_token` in the response.
6. **Token update** – Same test script as Login overwrites `jwt_token`.

---

### 4. Logout (`POST /api/logout`)

1. **Select** – **Auth → Logout**.
2. **Method & URL** – `POST {{base_url}}/api/logout`.
3. **Headers** – Pre-request script adds `Authorization: Bearer {{jwt_token}}`.
4. **Body** – raw JSON:
   ```json
   { "refresh_token": "{{refresh_token}}" }
   ```
5. **Send**.
6. **Verify** – Status **204 No Content** (empty body).
7. **After logout** – The stored `jwt_token` is revoked server-side. Run **Login** again before testing protected endpoints.

---

## Profile Endpoints (JWT required)

### 5. Get Profile (`GET /api/profile`)

1. **Select** – **Profile → Get Profile**.
2. **Method & URL** – `GET {{base_url}}/api/profile`.
3. **Headers** – `Authorization: Bearer {{jwt_token}}` (added by pre-request script).
4. **Send**.
5. **Verify** –
   - Status **200**.
   - Response contains `user_id`, `email`, `display_name`, `avatar_url`, `bio`.

---

### 6. Update Profile (`PATCH /api/profile`)

1. **Select** – **Profile → Update Profile**.
2. **Method & URL** – `PATCH {{base_url}}/api/profile`.
3. **Headers** – `Authorization` added automatically; `Content-Type: application/json`.
4. **Body** – Include only the fields you want to update:
   ```json
   {
     "display_name": "Test User",
     "avatar_url": "https://example.com/avatar.png",
     "bio": "Hello world"
   }
   ```
   All three fields are optional pointers; omit any you do not want to change.
5. **Send**.
6. **Verify** –
   - Status **200**.
   - Response reflects updated values.
7. **Confirm** – Run **Get Profile** again to verify persistence.

---

## Conversation Endpoints (JWT required)

### 7. Create Conversation (`POST /api/conversations`)

1. **Select** – **Conversations → Create**.
2. **Method & URL** – `POST {{base_url}}/api/conversations`.
3. **Body** – raw JSON:
   ```json
   {
     "type": "direct",
     "participant_user_ids": ["<other-user-uuid>"]
   }
   ```
   - `type` – `"direct"` or `"group"`.
   - `display_name` and `avatar_url` are optional (auto-resolved for direct chats).
   - `conversation_id` is optional; the gateway generates a UUID if omitted.
   - The authenticated user is added automatically; only supply **additional** participants.
4. **Send**.
5. **Verify** –
   - Status **201**.
   - Response contains `conversation.conversation_id`.
6. **Store ID** – In the **Tests** tab:
   ```javascript
   pm.environment.set("conv_id", pm.response.json().conversation.conversation_id);
   ```

---

### 8. List Conversations (`GET /api/conversations`)

1. **Select** – **Conversations → List**.
2. **Method & URL** – `GET {{base_url}}/api/conversations`.
3. **Headers** – `Authorization` added automatically.
4. **Send**.
5. **Verify** –
   - Status **200**.
   - Response contains a `conversations` array.

---

### 9. Get Conversation (`GET /api/conversations/{id}`)

1. **Select** – **Conversations → Get**.
2. **Method & URL** – `GET {{base_url}}/api/conversations/{{conv_id}}`.
3. **Headers** – `Authorization` added automatically.
4. **Send**.
5. **Verify** –
   - Status **200**.
   - Response contains full conversation details and participant list.

---

## Participant Endpoints (JWT required)

### 10. Add Participant (`POST /api/participants`)

1. **Select** – **Participants → Add**.
2. **Method & URL** – `POST {{base_url}}/api/participants`.
3. **Body** – raw JSON:
   ```json
   {
     "conversation_id": "{{conv_id}}",
     "user_id": "<target-user-uuid>"
   }
   ```
4. **Send**.
5. **Verify** –
   - Status **200**.
   - Response `{ "status": "ok" }`.

---

### 11. Remove Participant (`DELETE /api/participants`)

1. **Select** – **Participants → Remove**.
2. **Method & URL** – `DELETE {{base_url}}/api/participants`.
3. **Body** – raw JSON:
   ```json
   {
     "conversation_id": "{{conv_id}}",
     "user_id": "<target-user-uuid>"
   }
   ```
4. **Send**.
5. **Verify** –
   - Status **200**.
   - Response `{ "status": "ok" }`.

---

## Message Endpoints (JWT required)

### 12. Send Message (`POST /api/messages`)

1. **Select** – **Messages → Send**.
2. **Method & URL** – `POST {{base_url}}/api/messages`.
3. **Body** – raw JSON:
   ```json
   {
     "conversation_id": "{{conv_id}}",
     "content": "Hello from Postman!",
     "type": "text"
   }
   ```
   - `type` defaults to `"text"` if omitted.
   - `idempotency_key` is optional; the gateway generates a UUID if omitted.
4. **Send**.
5. **Verify** –
   - Status **200**.
   - Response includes `message_id` and `sequence`.
6. **Store** – Capture in **Tests** tab:
   ```javascript
   pm.environment.set("msg_id", pm.response.json().message_id);
   ```

---

### 13. Sync Messages (`GET /api/messages`)

1. **Select** – **Messages → Sync**.
2. **Method & URL** – `GET {{base_url}}/api/messages`.
3. **Params** (add in the **Params** tab):
   | Key | Value | Required |
   |-----|-------|----------|
   | `conversation_id` | `{{conv_id}}` | **Yes** |
   | `after` | `0` | No – sequence to page from (default 0) |
   | `limit` | `50` | No – max messages returned (default 50) |
4. **Send**.
5. **Verify** –
   - Status **200**.
   - Response contains a `messages` array ordered by sequence.

---

### 14. Delete Message (`DELETE /api/messages`)

1. **Select** – **Messages → Delete**.
2. **Method & URL** – `DELETE {{base_url}}/api/messages`.
3. **Body** – raw JSON:
   ```json
   {
     "conversation_id": "{{conv_id}}",
     "message_id": "{{msg_id}}"
   }
   ```
4. **Send**.
5. **Verify** –
   - Status **200**.
   - Response `{ "status": "ok" }`.

---

## Read Receipt Endpoint (JWT required)

### 15. Update Read Receipt (`POST /api/read-receipt`)

1. **Select** – **Conversations → Read Receipt**.
2. **Method & URL** – `POST {{base_url}}/api/read-receipt`.
3. **Body** – raw JSON:
   ```json
   {
     "conversation_id": "{{conv_id}}",
     "sequence": 5
   }
   ```
   `sequence` must be ≥ 0 and should equal the highest sequence number the user has read.
4. **Send**.
5. **Verify** –
   - Status **200**.
   - Response `{ "status": "ok" }`.

---

## Presence Endpoint (JWT required)

### 16. Get Presence (`GET /api/presence`)

1. **Select** – **Presence → Get**.
2. **Method & URL** – `GET {{base_url}}/api/presence`.
3. **Params** (add in the **Params** tab):
   | Key | Value | Required |
   |-----|-------|----------|
   | `user_ids` | `<uuid1>,<uuid2>` | **Yes** – comma-separated list |
4. **Send**.
5. **Verify** –
   - Status **200**.
   - Response is a map of `user_id → { online, last_seen_at }`.

---

## General Tips

- **Console** – Open **View → Show Postman Console** to see script logs and debug failures.
- **Environment variables** – Click the eye icon to inspect `jwt_token`, `refresh_token`, `conv_id`, `msg_id`.
- **401 Unauthorized** – Run **Login** again to get a fresh token.
- **Pre-request scripts** – The collection-level script injects `Authorization: Bearer {{jwt_token}}` on every protected request. Do not add the header manually unless you remove the script first.
- **Saving changes** – After editing a request body or URL, press **Ctrl+S** (Windows) / **⌘S** (macOS) to persist the change.

---

## Complete Endpoint Reference

| # | Name | Method | Path | Auth |
|---|------|--------|------|------|
| 1 | Register | POST | `/api/register` | No |
| 2 | Login | POST | `/api/login` | No |
| 3 | Refresh Token | POST | `/api/refresh` | No |
| 4 | Logout | POST | `/api/logout` | No |
| 5 | Get Profile | GET | `/api/profile` | Bearer |
| 6 | Update Profile | PATCH | `/api/profile` | Bearer |
| 7 | Create Conversation | POST | `/api/conversations` | Bearer |
| 8 | List Conversations | GET | `/api/conversations` | Bearer |
| 9 | Get Conversation | GET | `/api/conversations/{id}` | Bearer |
| 10 | Add Participant | POST | `/api/participants` | Bearer |
| 11 | Remove Participant | DELETE | `/api/participants` | Bearer |
| 12 | Send Message | POST | `/api/messages` | Bearer |
| 13 | Sync Messages | GET | `/api/messages?conversation_id=...` | Bearer |
| 14 | Delete Message | DELETE | `/api/messages` | Bearer |
| 15 | Update Read Receipt | POST | `/api/read-receipt` | Bearer |
| 16 | Get Presence | GET | `/api/presence?user_ids=...` | Bearer |

---

## Running All Tests in One Click

1. Complete **Login** so `jwt_token` is set.
2. In the sidebar locate the top-level **RealChat** collection folder.
3. Click **Run** → **Collection Runner**.
4. Select the **RealChat** environment, set **Iterations = 1**, optional **Delay = 200 ms**.
5. Press **Start Run**.
6. Verify every request shows **Pass**. Expand any failures to read the console output.

---

*End of detailed execution guide.*
