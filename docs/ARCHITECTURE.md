# ARCHITECTURE.md - System Design

## Overview

Outpost is a monolithic application with a Go backend and SvelteKit frontend, bundled into a single Docker container.

```
┌─────────────────────────────────────────────────────────────┐
│                        Docker Container                      │
│  ┌─────────────────────────────────────────────────────┐   │
│  │                    Outpost Binary                    │   │
│  │  ┌─────────────┐  ┌─────────────┐  ┌────────────┐  │   │
│  │  │   HTTP API  │  │  Services   │  │  Workers   │  │   │
│  │  │   (Gin)     │  │             │  │            │  │   │
│  │  └─────────────┘  └─────────────┘  └────────────┘  │   │
│  │         │                │               │          │   │
│  │         └────────────────┼───────────────┘          │   │
│  │                          │                          │   │
│  │                   ┌──────┴──────┐                   │   │
│  │                   │   SQLite    │                   │   │
│  │                   │   Database  │                   │   │
│  │                   └─────────────┘                   │   │
│  └─────────────────────────────────────────────────────┘   │
│                                                             │
│  ┌─────────────────────────────────────────────────────┐   │
│  │              Static Frontend (SvelteKit)             │   │
│  └─────────────────────────────────────────────────────┘   │
│                                                             │
│  ┌─────────────────────────────────────────────────────┐   │
│  │                      FFmpeg                          │   │
│  └─────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────┘
         │                    │                    │
         ▼                    ▼                    ▼
   ┌──────────┐        ┌──────────┐        ┌──────────┐
   │  TMDB    │        │ Prowlarr │        │ Download │
   │   API    │        │   API    │        │  Client  │
   └──────────┘        └──────────┘        └──────────┘
```

---

## Tech Stack Details

### Backend: Go

**Why Go:**
- Single binary deployment
- Excellent concurrency (goroutines for streaming, background jobs)
- Fast compilation
- Low memory footprint
- Strong standard library for HTTP, JSON, file I/O
- Easier learning curve than Rust

**Key packages:**
- `net/http` or `gin-gonic/gin` - HTTP server
- `mattn/go-sqlite3` - SQLite driver
- `robfig/cron` - Background job scheduling
- Standard library for most other needs

### Frontend: SvelteKit

**Why SvelteKit:**
- No virtual DOM = faster on low-power devices (TVs, Pis)
- Smaller bundle sizes (60-70% smaller than React)
- Simpler component model
- Built-in routing
- SSR capable but we'll use SPA mode

**Key packages:**
- `tailwindcss` - Styling
- `lucide-svelte` - Icons
- `svelte-french-toast` - Notifications

### Database: SQLite

**Why SQLite:**
- Single file = easy backup
- No separate server process
- Fast enough for home use
- Plex uses it successfully
- Portable across systems

**Considerations:**
- Single writer at a time (fine for home use)
- Enable WAL mode for better concurrency
- Regular VACUUM for maintenance

### Streaming: FFmpeg

**Direct Play:** When the client supports the file format, stream the file directly with range request support.

**Transcoding:** When the client doesn't support the format:
1. Detect client capabilities
2. Transcode to HLS with FFmpeg
3. Adaptive bitrate based on connection

---

## Directory Structure

```
outpost/
├── main.go                 # Entry point
├── internal/
│   ├── api/
│   │   ├── server.go       # HTTP server setup
│   │   ├── routes.go       # Route definitions
│   │   ├── middleware.go   # Auth, logging, etc.
│   │   └── handlers/
│   │       ├── auth.go
│   │       ├── libraries.go
│   │       ├── movies.go
│   │       ├── shows.go
│   │       ├── streaming.go
│   │       ├── downloads.go
│   │       ├── search.go
│   │       ├── discover.go
│   │       ├── requests.go
│   │       ├── users.go
│   │       └── settings.go
│   ├── config/
│   │   └── config.go       # Configuration loading
│   ├── database/
│   │   ├── database.go     # Connection, migrations
│   │   ├── migrations/     # SQL migration files
│   │   └── queries/        # SQL queries
│   ├── models/
│   │   ├── library.go
│   │   ├── movie.go
│   │   ├── show.go
│   │   ├── user.go
│   │   └── ...
│   ├── services/
│   │   ├── scanner/        # Library scanning
│   │   ├── metadata/       # TMDB, MusicBrainz clients
│   │   ├── streaming/      # Video streaming, transcoding
│   │   ├── downloader/     # Download client integrations
│   │   ├── indexer/        # Prowlarr, Torznab
│   │   ├── quality/        # Quality profile scoring
│   │   └── scheduler/      # Background jobs
│   └── util/
│       ├── parser.go       # Filename parsing
│       └── ...
├── frontend/
│   ├── src/
│   │   ├── app.html
│   │   ├── app.css
│   │   ├── routes/
│   │   │   ├── +layout.svelte
│   │   │   ├── +page.svelte          # Home/Discover
│   │   │   ├── movies/
│   │   │   │   ├── +page.svelte      # Movie list
│   │   │   │   └── [id]/+page.svelte # Movie detail
│   │   │   ├── tv/
│   │   │   ├── anime/
│   │   │   ├── music/
│   │   │   ├── books/
│   │   │   ├── requests/
│   │   │   ├── downloads/
│   │   │   ├── settings/
│   │   │   └── login/
│   │   ├── lib/
│   │   │   ├── api.ts        # API client
│   │   │   ├── stores.ts     # Svelte stores
│   │   │   └── utils.ts
│   │   └── components/
│   │       ├── Sidebar.svelte
│   │       ├── MediaCard.svelte
│   │       ├── VideoPlayer.svelte
│   │       ├── SearchBar.svelte
│   │       └── ...
│   └── static/
│       └── favicon.ico
├── data/                    # Runtime data (mounted volume)
│   ├── outpost.db          # SQLite database
│   ├── cache/              # Image cache
│   └── logs/               # Log files
└── docs/
```

---

## API Design

RESTful JSON API. All endpoints under `/api/`.

### Authentication

- Session-based with HTTP-only cookies
- Login returns session token stored in cookie
- All API requests (except login) require valid session
- Sessions expire after 30 days of inactivity

### Standard Response Format

**Success:**
```json
{
  "data": { ... }
}
```

**Error:**
```json
{
  "error": {
    "code": "NOT_FOUND",
    "message": "Movie not found"
  }
}
```

### Pagination

```
GET /api/movies?page=1&limit=20

{
  "data": [...],
  "pagination": {
    "page": 1,
    "limit": 20,
    "total": 150,
    "pages": 8
  }
}
```

---

## Background Jobs

Using a simple scheduler (robfig/cron):

| Job | Interval | Description |
|-----|----------|-------------|
| Library Scan | 1 hour | Re-scan libraries for new files |
| Metadata Refresh | 24 hours | Update metadata for existing items |
| RSS Check | 15 minutes | Check indexers for new releases |
| Download Monitor | 1 minute | Check download client status |
| Cleanup | 24 hours | Remove orphaned files, old logs |

---

## Streaming Architecture

### Direct Play Flow

```
Client → GET /api/stream/movie/123
                    │
                    ▼
            Check file format
                    │
        ┌───────────┴───────────┐
        │                       │
   Supported              Not Supported
        │                       │
        ▼                       ▼
   Serve file              Transcode
   with Range              (see below)
   requests
```

### Transcoding Flow

```
Client → GET /api/stream/movie/123?transcode=true
                    │
                    ▼
            Start FFmpeg process
                    │
                    ▼
            Output HLS segments
            to temp directory
                    │
                    ▼
            Serve .m3u8 playlist
            and .ts segments
                    │
                    ▼
            Cleanup on completion
            or timeout
```

---

## Security Considerations

1. **Authentication:** bcrypt password hashing, secure session tokens
2. **Authorization:** Role-based access control on all endpoints
3. **Input validation:** Validate all user input
4. **SQL injection:** Use parameterized queries only
5. **Path traversal:** Validate file paths are within allowed directories
6. **CORS:** Restrict to known origins
7. **Rate limiting:** On auth endpoints
8. **HTTPS:** Required for production (handled by reverse proxy)

---

## Configuration

Environment variables with sensible defaults:

```bash
# Server
OUTPOST_PORT=8080
OUTPOST_HOST=0.0.0.0

# Database
OUTPOST_DB_PATH=/data/outpost.db

# Paths
OUTPOST_DATA_PATH=/data
OUTPOST_CACHE_PATH=/data/cache
OUTPOST_LOG_PATH=/data/logs

# External services (optional)
OUTPOST_TMDB_API_KEY=
OUTPOST_PROWLARR_URL=
OUTPOST_PROWLARR_API_KEY=

# Security
OUTPOST_SESSION_SECRET=  # Generated on first run if not set
```

---

## Deployment

### Docker (Recommended)

```yaml
version: "3.8"
services:
  outpost:
    image: outpost/outpost:latest
    container_name: outpost
    ports:
      - "8080:8080"
    volumes:
      - ./data:/data
      - /path/to/movies:/media/movies:ro
      - /path/to/tv:/media/tv:ro
      - /path/to/music:/media/music:ro
    environment:
      - OUTPOST_TMDB_API_KEY=your_key_here
    restart: unless-stopped
```

### Manual

1. Download binary for your platform
2. Create data directory
3. Set environment variables
4. Run `./outpost`

---

## Performance Targets

| Metric | Target |
|--------|--------|
| API response time | < 100ms (non-streaming) |
| Library scan (1000 files) | < 60 seconds |
| Memory usage (idle) | < 100MB |
| Memory usage (streaming) | < 500MB |
| Startup time | < 5 seconds |
| Docker image size | < 500MB |
