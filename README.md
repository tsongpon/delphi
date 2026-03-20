# Delphi

A backend API for 360-degree performance feedback. Teams can submit, collect, and review feedback across five dimensions: Communication, Leadership, Technical, Collaboration, and Delivery.

## Features

- User registration with role-based access (manager / member)
- Feedback submission with scores (1–5) and optional comments
- Named or anonymous feedback visibility
- Team dashboard with aggregated metrics (manager only)
- PDF export of personal feedback
- Invite link system for team onboarding
- Password reset via email
- Email digest notifications when feedback is received

## Tech Stack

- **Language:** Go 1.25
- **Framework:** [Echo v5](https://echo.labstack.com/)
- **Database:** Google Cloud Firestore
- **Email:** [Resend](https://resend.com/)
- **PDF generation:** gofpdf
- **Deployment:** Docker + Google Cloud Run

## Prerequisites

- Go 1.25+
- A GCP project with Firestore enabled
- A GCP service account key JSON file (for local development)
- A [Resend](https://resend.com/) account and API key

## Getting Started

```bash
# 1. Clone the repo
git clone https://github.com/tsongpon/delphi.git
cd delphi

# 2. Install dependencies
go mod download

# 3. Configure environment
cp .env.example .env
# Edit .env with your values (see Environment Variables below)

# 4. Run the server
go run ./cmd/api-server
```

The API will be available at `http://localhost:8080`.

## Environment Variables

| Variable | Required | Description |
|---|---|---|
| `GCP_PROJECT_ID` | Yes | GCP project ID |
| `GCP_FIRESTORE_DATABASE_ID` | No | Firestore database ID (defaults to `(default)`) |
| `JWT_SECRET` | Yes | Secret key for signing JWT tokens |
| `ADMIN_SECRET` | Yes | Secret header value for admin endpoints |
| `APP_BASE_URL` | Yes | Base URL used in generated links (no trailing slash) |
| `GOOGLE_APPLICATION_CREDENTIALS` | Yes (local) | Path to GCP service account key JSON |
| `RESEND_API_KEY` | Yes | Resend API key for sending emails |
| `RESEND_FROM_EMAIL` | Yes | Sender email address (e.g. `Feedback360 <notify@yourdomain.com>`) |
| `APP_ENV` | No | `development` or `production` (affects log format) |

## Running Tests

```bash
go test ./...
```

## Docker

```bash
# Build
docker build -t delphi:latest .

# Run
docker run -p 8080:8080 \
  -e GCP_PROJECT_ID=your-project \
  -e JWT_SECRET=your-secret \
  -e ADMIN_SECRET=your-admin-secret \
  -e APP_BASE_URL=http://localhost:8080 \
  -e RESEND_API_KEY=re_xxxx \
  -e RESEND_FROM_EMAIL="Feedback360 <notify@yourdomain.com>" \
  -e GOOGLE_APPLICATION_CREDENTIALS=/app/key.json \
  -v /path/to/key.json:/app/key.json \
  delphi:latest
```

## API Overview

### Public

| Method | Path | Description |
|---|---|---|
| `GET` | `/ping` | Health check |
| `POST` | `/register` | Register a new user |
| `POST` | `/login` | Login, returns JWT |
| `POST` | `/reset-password` | Reset password using a token |
| `GET` | `/invite-links/validate` | Validate an invite link token |

### User (JWT required)

| Method | Path | Description |
|---|---|---|
| `GET` | `/me/teammates` | List team members |
| `GET` | `/me/feedbacks` | Get received feedback (paginated) |
| `GET` | `/me/feedbacks/export` | Export received feedback as PDF |
| `GET` | `/me/given-feedbacks` | Get feedback given by current user |
| `POST` | `/feedbacks` | Submit feedback for a teammate |

### Team — Manager only (JWT required)

| Method | Path | Description |
|---|---|---|
| `GET` | `/teams/:teamId/feedbacks` | List all team feedback |
| `GET` | `/teams/:teamId/dashboard` | Get aggregated team dashboard metrics |
| `GET` | `/teams/:teamId/members/:memberId/feedbacks` | Get feedback for a specific member |
| `POST` | `/teams/:teamId/invite-links` | Create an invite link |
| `GET` | `/teams/:teamId/invite-links` | List invite links |
| `DELETE` | `/teams/:teamId/invite-links/:linkId` | Revoke an invite link |

### Admin (`X-Admin-Secret` header required)

| Method | Path | Description |
|---|---|---|
| `POST` | `/admin/teams` | Create a team |
| `PUT` | `/admin/users/:userID/role` | Update a user's role |
| `POST` | `/admin/users/:userID/reset-link` | Generate a password reset link |
| `POST` | `/admin/feedback-notify` | Send feedback digest emails |

See [docs/admin-api.md](docs/admin-api.md) for detailed admin API documentation with examples.

## Deployment

Pushes to `main` trigger a GitHub Actions workflow that builds a Docker image, pushes it to GCP Artifact Registry, and deploys to Cloud Run in `asia-southeast1`.

## License

MIT with Commons Clause — see [LICENSE](LICENSE).
