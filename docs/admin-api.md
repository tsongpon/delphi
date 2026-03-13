# Admin API Manual

## Overview

Admin endpoints allow privileged operations — creating teams, managing user roles, and generating password reset links — that regular users cannot perform. All admin endpoints are grouped under the `/admin` path prefix and are completely separate from JWT-based user authentication.

**Base path:** `/admin`

---

## Authentication

Every admin request must include the `X-Admin-Secret` header with the value of the `ADMIN_SECRET` environment variable configured on the server.

```
X-Admin-Secret: <your-admin-secret>
```

If the header is missing or the value does not match, the server returns **401 Unauthorized**.

> **Security note:** Treat `ADMIN_SECRET` like a password. Do not embed it in client-side code or commit it to version control. Rotate it if it is ever exposed.

---

## Endpoints

### 1. Create Team

Create a new team. The team is persisted in GCP Firestore under the `teams` collection.

```
POST /admin/teams
```

**Headers**

| Header | Required | Description |
|--------|----------|-------------|
| `X-Admin-Secret` | Yes | Admin secret |
| `Content-Type` | Yes | `application/json` |

**Request Body**

```json
{
  "name": "Engineering"
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | string | Yes | Team name (non-empty) |

**Responses**

| Status | Description |
|--------|-------------|
| `201 Created` | Team created successfully |
| `400 Bad Request` | `name` is missing or blank |
| `401 Unauthorized` | Missing or incorrect `X-Admin-Secret` |
| `500 Internal Server Error` | Firestore write failed |

**201 Response Body**

```json
{
  "id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
  "name": "Engineering",
  "created_at": "2026-03-13T10:00:00Z",
  "updated_at": "2026-03-13T10:00:00Z"
}
```

**curl Example**

```bash
curl -X POST https://api.example.com/admin/teams \
  -H "X-Admin-Secret: your-admin-secret" \
  -H "Content-Type: application/json" \
  -d '{"name": "Engineering"}'
```

---

### 2. Update User Role

Change the role of an existing user. Valid roles are `member` and `manager`.

```
PUT /admin/users/:userID/role
```

**Path Parameters**

| Parameter | Description |
|-----------|-------------|
| `userID` | Firestore document ID of the user |

**Headers**

| Header | Required | Description |
|--------|----------|-------------|
| `X-Admin-Secret` | Yes | Admin secret |
| `Content-Type` | Yes | `application/json` |

**Request Body**

```json
{
  "role": "manager"
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `role` | string | Yes | One of: `member`, `manager` |

**Responses**

| Status | Description |
|--------|-------------|
| `200 OK` | Role updated successfully |
| `400 Bad Request` | `role` is not `member` or `manager` |
| `401 Unauthorized` | Missing or incorrect `X-Admin-Secret` |
| `500 Internal Server Error` | Firestore update failed |

**200 Response Body**

```json
{
  "role": "manager"
}
```

**curl Example**

```bash
# Promote a user to manager
curl -X PUT https://api.example.com/admin/users/USER_ID/role \
  -H "X-Admin-Secret: your-admin-secret" \
  -H "Content-Type: application/json" \
  -d '{"role": "manager"}'

# Demote a user to member
curl -X PUT https://api.example.com/admin/users/USER_ID/role \
  -H "X-Admin-Secret: your-admin-secret" \
  -H "Content-Type: application/json" \
  -d '{"role": "member"}'
```

---

### 3. Generate Password Reset Link

Generate a one-time password reset link for a user. Send this link to the user via a secure channel so they can reset their password.

```
POST /admin/users/:userID/reset-link
```

**Path Parameters**

| Parameter | Description |
|-----------|-------------|
| `userID` | Firestore document ID of the user |

**Headers**

| Header | Required | Description |
|--------|----------|-------------|
| `X-Admin-Secret` | Yes | Admin secret |

**Request Body**

None required.

**Responses**

| Status | Description |
|--------|-------------|
| `200 OK` | Reset link generated successfully |
| `400 Bad Request` | `userID` path parameter is missing |
| `401 Unauthorized` | Missing or incorrect `X-Admin-Secret` |
| `500 Internal Server Error` | Failed to generate reset token |

**200 Response Body**

```json
{
  "reset_link": "https://api.example.com/reset-password?token=abc123...",
  "expires_at": "2026-03-13T11:00:00Z"
}
```

**curl Example**

```bash
curl -X POST https://api.example.com/admin/users/USER_ID/reset-link \
  -H "X-Admin-Secret: your-admin-secret"
```

---

## Common Error Responses

All endpoints return errors in this format:

```json
{
  "error": "description of the error"
}
```

| Status | Cause |
|--------|-------|
| `400 Bad Request` | Invalid or missing request fields |
| `401 Unauthorized` | `X-Admin-Secret` header is missing or incorrect |
| `500 Internal Server Error` | Unexpected server or database error |

---

## Role Reference

| Role | Description |
|------|-------------|
| `member` | Regular user — can give feedback and view their own received feedback |
| `manager` | Elevated user — can view team feedback dashboards |

Use `PUT /admin/users/:userID/role` to move users between these roles.
