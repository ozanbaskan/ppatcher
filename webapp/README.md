# PPatcher Web App

A web application for managing PPatcher applications with Google OAuth authentication.

## Quick Start

### 1. Set up Google OAuth

1. Go to [Google Cloud Console](https://console.cloud.google.com/apis/credentials)
2. Create a new project (or use an existing one)
3. Go to **APIs & Services → Credentials → Create Credentials → OAuth 2.0 Client ID**
4. Set **Application type** to "Web application"
5. Add `http://localhost:8080/auth/google/callback` as an **Authorized redirect URI**
6. Copy the Client ID and Client Secret

### 2. Configure environment

```bash
cd webapp
cp .env.example .env
# Edit .env with your Google OAuth credentials and a random session secret
```

### 3. Run with Docker Compose

From the project root:

```bash
docker compose up --build
```

The app will be available at **http://localhost:8080**.

### 4. Local development (without Docker)

You need a PostgreSQL instance running. Then:

```bash
cd webapp
export DATABASE_URL="postgres://ppatcher:ppatcher@localhost:5432/ppatcher?sslmode=disable"
export GOOGLE_CLIENT_ID="your-client-id"
export GOOGLE_CLIENT_SECRET="your-client-secret"
export APP_URL="http://localhost:8080"
export SESSION_SECRET="some-random-secret"
go run .
```

## Architecture

- **Go backend** with `gorilla/mux` router
- **PostgreSQL** for persistent storage (auto-migrates on startup)
- **Google OAuth2** for authentication
- **Go HTML templates** + Tailwind CSS (CDN) for responsive, mobile-friendly UI
- **Docker Compose** with PostgreSQL and the webapp container

## Project Structure

```
webapp/
├── main.go          # Server entry point, routing
├── db.go            # Database connection, migrations
├── models.go        # Data models (User, Application)
├── auth.go          # Google OAuth2 handlers
├── middleware.go     # Auth middleware
├── handlers.go      # Page and CRUD handlers
├── Dockerfile       # Multi-stage build
├── .env.example     # Environment template
├── static/          # Static assets (embedded)
└── templates/       # Go HTML templates (embedded)
    ├── helpers.html  # Shared nav, head, icons
    ├── index.html    # Landing page
    ├── dashboard.html
    ├── app_form.html # Create/edit application
    └── app_view.html # Application detail view
```

## Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `DATABASE_URL` | PostgreSQL connection string | `postgres://ppatcher:ppatcher@localhost:5432/ppatcher?sslmode=disable` |
| `GOOGLE_CLIENT_ID` | Google OAuth Client ID | (required) |
| `GOOGLE_CLIENT_SECRET` | Google OAuth Client Secret | (required) |
| `APP_URL` | Public URL of the app | `http://localhost:8080` |
| `SESSION_SECRET` | Cookie encryption key | `dev-secret-change-me` |
| `PORT` | HTTP listen port | `8080` |
