# Talks Indexer

A Go application that indexes conference talks from moresleep into Elasticsearch.

## Project Overview

This application fetches talk data from a moresleep instance and indexes it into two Elasticsearch indexes:
- `javazone_private`: All talks with full data (requires authentication)
- `javazone_public`: Only talks with status "APPROVED" (public data only)

## Commands

```bash
make templ      # Generate templ templates
make build      # Build the application (includes templ)
make test       # Run tests
make run        # Run the application locally (includes templ)
make fmt        # Format code
make lint       # Run linter
make docker     # Build Docker image
make up         # Start all services with docker compose
make down       # Stop all services
```

## Architecture

This project uses clean hexagonal architecture:

- `internal/adapters/` - Infrastructure implementations
  - `api/` - HTTP API handlers
  - `web/` - Web admin dashboard (templ + htmx)
    - `handlers/` - Web request handlers
    - `templates/` - templ templates
  - `auth/` - OIDC authentication (middleware, handlers)
  - `session/` - In-memory session storage
  - `moresleep/` - Client for fetching data from moresleep API
  - `elasticsearch/` - Elasticsearch bulk indexing client
- `internal/app/` - Business logic (indexing service)
- `internal/config/` - Centralized configuration
- `internal/domain/` - Domain models (Talk, Conference, Speaker)
- `internal/ports/` - Port interfaces (TalkSource, SearchIndex)

## Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `MODE` | Running mode (`production` or `development`). API disabled in production. | `production` |
| `HTTP_HOST` | HTTP server host | `0.0.0.0` |
| `HTTP_PORT` | HTTP server port | `8080` |
| `MORESLEEP_URL` | Base URL of moresleep instance | `http://localhost:8082` |
| `MORESLEEP_USER` | Username for moresleep authentication | (empty) |
| `MORESLEEP_PASSWORD` | Password for moresleep authentication | (empty) |
| `ELASTICSEARCH_URL` | Elasticsearch URL | `http://localhost:9200` |
| `ELASTICSEARCH_USER` | Username for Elasticsearch authentication | (empty) |
| `ELASTICSEARCH_PASSWORD` | Password for Elasticsearch authentication | (empty) |
| `PRIVATE_INDEX` | Name of private index | `javazone_private` |
| `PUBLIC_INDEX` | Name of public index | `javazone_public` |
| `OIDC_ISSUER_URL` | OIDC provider issuer URL (production only) | (empty) |
| `OIDC_CLIENT_ID` | OIDC client ID (production only) | (empty) |
| `OIDC_CLIENT_SECRET` | OIDC client secret (production only) | (empty) |
| `OIDC_REDIRECT_URL` | OIDC callback URL (production only) | (empty) |

## API Endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET | `/health` | Health check endpoint |
| POST | `/api/reindex` | Trigger full reindex of all conferences |
| POST | `/api/reindex/conference/{slug}` | Reindex a specific conference |
| POST | `/api/reindex/talk/{talkId}` | Reindex a specific talk |
| GET | `/admin` | Web admin dashboard (auth required in production) |
| GET | `/auth/callback` | OIDC callback handler (production only) |
| POST | `/auth/logout` | Logout and clear session (production only) |

## Testing

Tests use testify for assertions. Run with:

```bash
make test
```

For coverage report:

```bash
make coverage
```

## Development

1. Start Elasticsearch: `make up`
2. Run the application: `make run`
3. Trigger reindex: `curl -X POST http://localhost:8080/api/reindex`
