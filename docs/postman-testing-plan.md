# RealChat — Combined Postman Testing Plan

> Auth Service `:8081` · Profile Service `:8082`

---

## Prerequisites

1. Run the full stack:
   ```
   docker compose up --build -d
   ```
2. Wait for health checks — confirm with:
   - `GET http://localhost:8081/health` → `200 ok`
   - `GET http://localhost:8082/health` → `200`
3. In Postman, create an **Environment** named `RealChat-Local` with these variables (leave initial values blank):

   | Variable          | Description                 |
   |-------------------|-----------------------------|
   | `auth_url`        | `http://localhost:8081`     |
   | `profile_url`     | `http://localhost:8082`     |
   | `access_token`    | _(auto-set by scripts)_     |
   | `refresh_token`   | _(auto-set by scripts)_     |
   | `user_email`      | _(auto-set by scripts)_     |
   | `user_email_2`    | _(second user for contacts/blocks)_ |
   | `user_id_2`       | _(auto-set by scripts)_     |

4. Select the **RealChat-Local** environment in Postman before running.

---

## Phase 1 — Health & Readiness

### Step 1: Auth Health Check

| Field   | Value                             |
|---------|-----------------------------------|
| Method  | `GET`                             |
| URL     | `{{auth_url}}/health`             |
| Body    | _none_                            |

**Expected Response:**
- Status: `200 OK`
- Body: `ok`

---

### Step 2: Auth Readiness Check

| Field   | Value                             |
|---------|-----------------------------------|
| Method  | `GET`                             |
| URL     | `{{auth_url}}/health/ready`       |
| Body    | _none_                            |

**Expected Response:**
- Status: `200 OK`
- Body: `ready`

---

### Step 3: Profile Health Check

| Field   | Value                             |
|---------|-----------------------------------|
| Method  | `GET`                             |
| URL     | `{{profile_url}}/health`          |
| Body    | _none_                            |

**Expected Response:**
- Status: `200`

---

### Step 4: Profile Readiness Check

| Field   | Value                             |
|---------|-----------------------------------|
| Method  | `GET`                             |
| URL     | `{{profile_url}}/health/ready`    |
| Body    | _none_                            |

**Expected Response:**
- Status: `200`

---

## Phase 2 — User Registration & Login

### Step 5: Register User A

| Field   | Value                                     |
|---------|-------------------------------------------|
| Method  | `POST`                                    |
| URL     | `{{auth_url}}/api/v1/auth/register`       |
| Headers | `Content-Type: application/json`          |
| Body    | _(raw JSON)_                              |

```json
{
  "email": "alice@example.com",
  "password": "P@ssw0rd123"
}
```

**Expected Response:**
- Status: `201 Created`
- Body: _empty_

**Post-response Script** (Postman → Scripts → Post-response):
```javascript
pm.environment.set("user_email", "alice@example.com");
```

---

### Step 6: Register Duplicate User (negative test)

Same request as Step 5 — send it again.

**Expected Response:**
- Status: `400 Bad Request`
- Body: `{"error": "..."}`  (duplicate email error)

---

### Step 7: Register With Missing Fields (negative test)

| Field   | Value                                     |
|---------|-------------------------------------------|
| Method  | `POST`                                    |
| URL     | `{{auth_url}}/api/v1/auth/register`       |
| Body    | _(raw JSON)_                              |

```json
{
  "email": ""
}
```

**Expected Response:**
- Status: `400 Bad Request`
- Body: `{"error": "email and password required"}`

---

### Step 8: Login User A

| Field   | Value                                     |
|---------|-------------------------------------------|
| Method  | `POST`                                    |
| URL     | `{{auth_url}}/api/v1/auth/login`          |
| Headers | `Content-Type: application/json`          |
| Body    | _(raw JSON)_                              |

```json
{
  "email": "alice@example.com",
  "password": "P@ssw0rd123"
}
```

**Expected Response:**
- Status: `200 OK`
- Body:
```json
{
  "access_token": "<jwt>",
  "refresh_token": "<uuid-or-opaque>"
}
```

**Post-response Script:**
```javascript
const res = pm.response.json();
pm.environment.set("access_token", res.access_token);
pm.environment.set("refresh_token", res.refresh_token);
```

---

### Step 9: Login With Wrong Password (negative test)

| Field   | Value                                     |
|---------|-------------------------------------------|
| Method  | `POST`                                    |
| URL     | `{{auth_url}}/api/v1/auth/login`          |
| Body    | _(raw JSON)_                              |

```json
{
  "email": "alice@example.com",
  "password": "WrongPassword"
}
```

**Expected Response:**
- Status: `401 Unauthorized`
- Body: `{"error": "invalid credentials"}`

---

## Phase 3 — Token Management

### Step 10: Refresh Tokens

| Field   | Value                                     |
|---------|-------------------------------------------|
| Method  | `POST`                                    |
| URL     | `{{auth_url}}/api/v1/auth/refresh`        |
| Headers | `Content-Type: application/json`          |
| Body    | _(raw JSON)_                              |

```json
{
  "refresh": "{{refresh_token}}"
}
```

**Expected Response:**
- Status: `200 OK`
- Body:
```json
{
  "access_token": "<new-jwt>",
  "refresh_token": "<new-refresh>"
}
```

**Post-response Script:**
```javascript
const res = pm.response.json();
pm.environment.set("access_token", res.access_token);
pm.environment.set("refresh_token", res.refresh_token);
```

---

### Step 11: Use Old Refresh Token (negative test)

> After Step 10, the previous refresh token should be revoked.

Send the same request from Step 10 but use the **old** refresh token (copy it before Step 10 runs).

**Expected Response:**
- Status: `401 Unauthorized`
- Body: `{"error": "invalid refresh token"}`

---

### Step 12: Refresh With Empty Token (negative test)

```json
{
  "refresh": ""
}
```

**Expected Response:**
- Status: `400 Bad Request`
- Body: `{"error": "refresh token required"}`

---

## Phase 4 — Profile Operations (cross-service)

> These requests use the `access_token` obtained from auth login/refresh.
> The JWT `sub` claim (user ID) is extracted by the profile's JWT middleware.

### Step 13: Get Profile (first time, profile may not exist yet)

| Field   | Value                                          |
|---------|-------------------------------------------------|
| Method  | `GET`                                           |
| URL     | `{{profile_url}}/api/v1/profile/me`            |
| Headers | `Authorization: Bearer {{access_token}}`        |
| Body    | _none_                                          |

**Expected Response (if profile auto-created via Kafka event):**
- Status: `200 OK`
- Body: profile object with `user_id`, `display_name`, `bio`, `avatar_url`

**Expected Response (if profile NOT yet created):**
- Status: `404 Not Found`
- Body: `{"error": "profile not found"}`

> **Note:** The auth publishes a `auth.user.created` Kafka event on registration. If the profile consumes this event and auto-creates a profile, you'll get `200`. Otherwise `404`.

---

### Step 14: Get Profile Without Auth (negative test)

| Field   | Value                                          |
|---------|-------------------------------------------------|
| Method  | `GET`                                           |
| URL     | `{{profile_url}}/api/v1/profile/me`            |
| Headers | _no Authorization header_                      |

**Expected Response:**
- Status: `401`

---

### Step 15: Update Profile

| Field   | Value                                          |
|---------|-------------------------------------------------|
| Method  | `PUT`                                           |
| URL     | `{{profile_url}}/api/v1/profile/me`            |
| Headers | `Authorization: Bearer {{access_token}}`        |
|         | `Content-Type: application/json`                |
| Body    | _(raw JSON)_                                    |

```json
{
  "display_name": "Alice",
  "bio": "Hello from RealChat!",
  "avatar_url": "https://example.com/alice.png"
}
```

**Expected Response:**
- Status: `204 No Content`

---

### Step 16: Verify Profile Update

Repeat Step 13 — `GET /api/v1/profile/me`.

**Expected Response:**
- Status: `200 OK`
- Body should contain the updated fields:
```json
{
  "user_id": "...",
  "display_name": "Alice",
  "bio": "Hello from RealChat!",
  "avatar_url": "https://example.com/alice.png"
}
```

---

## Phase 5 — Contacts (requires second user)

### Step 17: Register & Login User B

**Register:**

| Field   | Value                                     |
|---------|-------------------------------------------|
| Method  | `POST`                                    |
| URL     | `{{auth_url}}/api/v1/auth/register`       |
| Body    | _(raw JSON)_                              |

```json
{
  "email": "bob@example.com",
  "password": "B0bStr0ng!"
}
```

**Expected:** `201 Created`

**Login:**

| Field   | Value                                     |
|---------|-------------------------------------------|
| Method  | `POST`                                    |
| URL     | `{{auth_url}}/api/v1/auth/login`          |
| Body    | _(raw JSON)_                              |

```json
{
  "email": "bob@example.com",
  "password": "B0bStr0ng!"
}
```

**Post-response Script** — save Bob's token temporarily:
```javascript
const res = pm.response.json();
pm.environment.set("user_email_2", "bob@example.com");
// Decode JWT to get user ID (sub claim)
const payload = JSON.parse(atob(res.access_token.split('.')[1]));
pm.environment.set("user_id_2", payload.sub);
// Store Bob's token but keep Alice's as primary
pm.environment.set("access_token_b", res.access_token);
```

Then **re-login as Alice** (Step 8) to restore `access_token` to Alice's.

---

### Step 18: Add Contact

| Field   | Value                                          |
|---------|-------------------------------------------------|
| Method  | `POST`                                          |
| URL     | `{{profile_url}}/api/v1/profile/contacts`      |
| Headers | `Authorization: Bearer {{access_token}}`        |
|         | `Content-Type: application/json`                |
| Body    | _(raw JSON)_                                    |

```json
{
  "contact": "{{user_id_2}}"
}
```

**Expected Response:**
- Status: `204 No Content`

---

### Step 19: Add Duplicate Contact (negative test)

Repeat Step 18.

**Expected Response:**
- Status: `400 Bad Request`
- Body: `{"error": "..."}`

---

### Step 20: Add Contact With Empty Body (negative test)

```json
{
  "contact": ""
}
```

**Expected Response:**
- Status: `400 Bad Request`
- Body: `{"error": "invalid request body"}`

---

### Step 21: List Contacts

| Field   | Value                                                    |
|---------|-----------------------------------------------------------|
| Method  | `GET`                                                     |
| URL     | `{{profile_url}}/api/v1/profile/contacts?limit=20&offset=0` |
| Headers | `Authorization: Bearer {{access_token}}`                  |

**Expected Response:**
- Status: `200 OK`
- Body: array containing User B's ID

---

### Step 22: List Contacts With Pagination

| Field   | Value                                                    |
|---------|-----------------------------------------------------------|
| Method  | `GET`                                                     |
| URL     | `{{profile_url}}/api/v1/profile/contacts?limit=1&offset=0` |
| Headers | `Authorization: Bearer {{access_token}}`                  |

**Expected Response:**
- Status: `200 OK`
- Body: array with at most 1 entry

---

### Step 23: Remove Contact

| Field   | Value                                          |
|---------|-------------------------------------------------|
| Method  | `DELETE`                                        |
| URL     | `{{profile_url}}/api/v1/profile/contacts`      |
| Headers | `Authorization: Bearer {{access_token}}`        |
|         | `Content-Type: application/json`                |
| Body    | _(raw JSON)_                                    |

```json
{
  "contact": "{{user_id_2}}"
}
```

**Expected Response:**
- Status: `204 No Content`

---

### Step 24: Verify Contact Removed

Repeat Step 21 — list contacts.

**Expected:**
- Body should no longer contain User B's ID.

---

## Phase 6 — Block / Unblock

### Step 25: Re-Add Contact (setup for block test)

Repeat Step 18 — add User B as contact again.

**Expected:** `204 No Content`

---

### Step 26: Block User B

| Field   | Value                                          |
|---------|-------------------------------------------------|
| Method  | `POST`                                          |
| URL     | `{{profile_url}}/api/v1/profile/blocks`        |
| Headers | `Authorization: Bearer {{access_token}}`        |
|         | `Content-Type: application/json`                |
| Body    | _(raw JSON)_                                    |

```json
{
  "user": "{{user_id_2}}"
}
```

**Expected Response:**
- Status: `204 No Content`

---

### Step 27: Verify Contact Auto-Removed After Block

List contacts again (Step 21).

**Expected:**
- Body should **not** contain User B — blocking removes them from contacts.

---

### Step 28: Block With Empty Body (negative test)

```json
{
  "user": ""
}
```

**Expected Response:**
- Status: `400 Bad Request`
- Body: `{"error": "invalid request body"}`

---

### Step 29: Unblock User B

| Field   | Value                                          |
|---------|-------------------------------------------------|
| Method  | `DELETE`                                        |
| URL     | `{{profile_url}}/api/v1/profile/blocks`        |
| Headers | `Authorization: Bearer {{access_token}}`        |
|         | `Content-Type: application/json`                |
| Body    | _(raw JSON)_                                    |

```json
{
  "user": "{{user_id_2}}"
}
```

**Expected Response:**
- Status: `204 No Content`

---

### Step 30: Add Contact Again After Unblock

Repeat Step 18 — should succeed now.

**Expected:** `204 No Content`

---

## Phase 7 — Logout & Token Invalidation

### Step 31: Logout

| Field   | Value                                     |
|---------|-------------------------------------------|
| Method  | `POST`                                    |
| URL     | `{{auth_url}}/api/v1/auth/logout`         |
| Headers | `Content-Type: application/json`          |
| Body    | _(raw JSON)_                              |

```json
{
  "refresh": "{{refresh_token}}"
}
```

**Expected Response:**
- Status: `204 No Content`

---

### Step 32: Refresh After Logout (negative test)

Use the same refresh token:

```json
{
  "refresh": "{{refresh_token}}"
}
```

**Expected Response:**
- Status: `401 Unauthorized`
- Body: `{"error": "invalid refresh token"}`

---

### Step 33: Access Profile With Expired/Revoked Token

> If your access token is still valid (15-min TTL), this test will still pass.
> To truly test expiry, wait 15 minutes or use a manually expired token.

| Field   | Value                                          |
|---------|-------------------------------------------------|
| Method  | `GET`                                           |
| URL     | `{{profile_url}}/api/v1/profile/me`            |
| Headers | `Authorization: Bearer {{access_token}}`        |

**Expected:** `200` if token is still within TTL, `401` after expiry.

---

## Quick-Reference: Complete Endpoint Map

### Auth Service — `localhost:8081`

| #  | Method | Endpoint                      | Auth? | Body Fields             |
|----|--------|-------------------------------|-------|-------------------------|
| 1  | GET    | `/health`                     | No    | —                       |
| 2  | GET    | `/health/ready`               | No    | —                       |
| 3  | POST   | `/api/v1/auth/register`       | No    | `email`, `password`     |
| 4  | POST   | `/api/v1/auth/login`          | No    | `email`, `password`     |
| 5  | POST   | `/api/v1/auth/refresh`        | No    | `refresh`               |
| 6  | POST   | `/api/v1/auth/logout`         | No    | `refresh`               |

### Profile Service — `localhost:8082`

| #  | Method | Endpoint                      | Auth?          | Body / Query              |
|----|--------|-------------------------------|----------------|---------------------------|
| 7  | GET    | `/health`                     | No             | —                         |
| 8  | GET    | `/health/ready`               | No             | —                         |
| 9  | GET    | `/api/v1/profile/me`          | `Bearer` token | —                         |
| 10 | PUT    | `/api/v1/profile/me`          | `Bearer` token | `display_name`, `bio`, `avatar_url` |
| 11 | POST   | `/api/v1/profile/contacts`    | `Bearer` token | `contact` (user ID)      |
| 12 | DELETE | `/api/v1/profile/contacts`    | `Bearer` token | `contact` (user ID)      |
| 13 | GET    | `/api/v1/profile/contacts`    | `Bearer` token | Query: `limit`, `offset` |
| 14 | POST   | `/api/v1/profile/blocks`      | `Bearer` token | `user` (user ID)         |
| 15 | DELETE | `/api/v1/profile/blocks`      | `Bearer` token | `user` (user ID)         |

---

## Postman Tips

1. **Use Collection Runner** — Create a Postman Collection with all steps above (in order). Run them sequentially via the Collection Runner for end-to-end validation.
2. **Auto-set tokens** — The post-response scripts automatically store `access_token` / `refresh_token` so downstream requests pick them up.
3. **`Authorization` tab** — Instead of manually setting the header, use Postman's Authorization tab → Type: `Bearer Token` → Token: `{{access_token}}`.
4. **Tests tab** — Add assertions in each request's Tests tab:
   ```javascript
   pm.test("Status is 201", () => pm.response.to.have.status(201));
   ```
5. **Negative tests first** — Run negative tests before moving to the next phase to catch regressions early.
