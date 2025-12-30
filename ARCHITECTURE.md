# ARCHITECTURE.md - System Design

---

## Overview

```
┌─────────────────────────────────────────────────────────────────┐
│                         Docker Container                         │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────────┐  │
│  │   SvelteKit │  │   Go API    │  │        SQLite           │  │
│  │   Frontend  │◄─┤   Backend   │◄─┤       Database          │  │
│  │   :3000     │  │   :8080     │  │   /app/data/outpost.db  │  │
│  └─────────────┘  └──────┬──────┘  └─────────────────────────┘  │
│                          │                                       │
│                    ┌─────┴─────┐                                 │
│                    │  FFmpeg   │                                 │
│                    │ Transcoder│                                 │
│                    └───────────┘                                 │
└─────────────────────────────────────────────────────────────────┘
         │                    │                    │
         ▼                    ▼                    ▼
    ┌─────────┐         ┌─────────┐         ┌─────────┐
    │  Media  │         │  TMDB   │         │ Download│
    │ Folders │         │   API   │         │ Clients │
    └─────────┘         └─────────┘         └─────────┘
```

---

## Technology Choices

### Backend: Go 1.24

**Why Go:**
- Fast compilation and execution
- Single binary deployment
- Excellent concurrency (goroutines)
- Strong standard library
- No runtime dependencies

**Key Packages:**
- `net/http` - HTTP server (Go 1.22+ routing)
- `database/sql` + `modernc.org/sqlite` - SQLite (pure Go)
- `encoding/json` - JSON handling
- `os/exec` - FFmpeg execution
- `golang.org/x/crypto/bcrypt` - Password hashing

### Frontend: SvelteKit

**Why SvelteKit:**
- Compiled output (small bundles)
- No virtual DOM overhead
- Server-side rendering capable
- Great developer experience
- Tailwind CSS integration

**Key Dependencies:**
- Tailwind CSS - Styling
- Lucide Icons - Icon library
- Video.js or native `<video>` - Playback

### Database: SQLite

**Why SQLite:**
- Single file database
- No separate process
- WAL mode for concurrent reads
- Easy backup (copy file)
- Sufficient for home server scale

**Configuration:**
- WAL mode enabled
- Foreign keys enabled
- Busy timeout: 5000ms

### Streaming: FFmpeg

**Why FFmpeg:**
- Industry standard
- All codec support
- Hardware acceleration support
- Reliable and well-documented

---

## Directory Structure

### Backend

```
backend/
├── main.go                 # Entry point
├── go.mod
├── go.sum
└── internal/
    ├── api/
    │   ├── server.go       # HTTP server setup
    │   ├── middleware.go   # Auth, CORS, logging
    │   ├── handlers/       # Route handlers by domain
    │   │   ├── auth.go
    │   │   ├── libraries.go
    │   │   ├── movies.go
    │   │   ├── shows.go
    │   │   ├── music.go
    │   │   ├── books.go
    │   │   ├── streaming.go
    │   │   ├── downloads.go
    │   │   ├── requests.go
    │   │   └── settings.go
    │   └── responses.go    # Standard response helpers
    ├── config/
    │   └── config.go       # Configuration loading
    ├── database/
    │   ├── database.go     # Connection management
    │   └── migrations/     # Schema migrations
    ├── models/
    │   └── models.go       # Data structures
    ├── library/
    │   ├── scanner.go      # File scanning
    │   └── matcher.go      # Metadata matching
    ├── metadata/
    │   ├── tmdb.go         # TMDB client
    │   ├── anilist.go      # AniList client (future)
    │   └── audnexus.go     # Audnexus client (future)
    ├── streaming/
    │   ├── direct.go       # Direct play
    │   └── transcode.go    # FFmpeg transcoding
    ├── downloads/
    │   ├── clients/        # Download client integrations
    │   │   ├── qbittorrent.go
    │   │   ├── transmission.go
    │   │   ├── sabnzbd.go
    │   │   └── nzbget.go
    │   ├── indexers/       # Indexer integrations
    │   │   ├── torznab.go
    │   │   └── newznab.go
    │   └── scheduler.go    # Background job runner
    └── auth/
        ├── sessions.go     # Session management
        └── passwords.go    # Password hashing
```

### Frontend

```
frontend/
├── package.json
├── svelte.config.js
├── tailwind.config.js
├── vite.config.ts
└── src/
    ├── app.html
    ├── app.css             # Tailwind imports
    ├── lib/
    │   ├── api.ts          # API client
    │   ├── stores/         # Svelte stores
    │   │   ├── auth.ts
    │   │   ├── player.ts
    │   │   └── ui.ts
    │   └── components/     # Reusable components
    │       ├── layout/
    │       │   ├── Sidebar.svelte
    │       │   ├── TopBar.svelte
    │       │   └── SearchOverlay.svelte
    │       ├── cards/
    │       │   ├── PosterCard.svelte
    │       │   ├── ContinueCard.svelte
    │       │   └── EpisodeCard.svelte
    │       ├── player/
    │       │   ├── VideoPlayer.svelte
    │       │   ├── Controls.svelte
    │       │   └── panels/
    │       └── ui/
    │           ├── Button.svelte
    │           ├── Input.svelte
    │           └── Modal.svelte
    └── routes/
        ├── +layout.svelte
        ├── +page.svelte          # Discover/home
        ├── login/
        ├── movies/
        │   ├── +page.svelte      # Movie library
        │   └── [id]/
        ├── tv/
        │   ├── +page.svelte      # TV library
        │   └── [id]/
        ├── music/
        │   ├── +page.svelte
        │   ├── artists/[id]/
        │   └── albums/[id]/
        ├── books/
        │   ├── +page.svelte
        │   └── [id]/
        ├── watch/
        │   ├── movie/[id]/
        │   └── episode/[id]/
        ├── discover/
        │   ├── movie/[id]/
        │   └── show/[id]/
        ├── requests/
        ├── downloads/
        ├── wanted/
        ├── search/
        ├── settings/
        └── users/
```

---

## Data Flow

### Library Scan
```
User adds library path
    ↓
Scanner walks directory
    ↓
File type detection
    ↓
Filename parsing (title, year, season, episode)
    ↓
TMDB search
    ↓
Best match selection
    ↓
Metadata + images download
    ↓
Database insert
```

### Playback
```
User clicks play
    ↓
Backend checks file codec
    ↓
Direct play compatible?
    ├─ Yes → Serve file directly with range support
    └─ No → FFmpeg transcode to HLS/MP4
    ↓
Frontend video player
    ↓
Progress updates sent to backend
    ↓
Resume position saved
```

### Request Flow
```
User requests content
    ↓
Admin approves
    ↓
Search indexers for release
    ↓
Select best match (quality profile)
    ↓
Send to download client
    ↓
Monitor download progress
    ↓
Download complete → rescan library
    ↓
Content available
```

---

## Configuration

### Environment Variables

```bash
# Server
OUTPOST_PORT=8080
OUTPOST_DATA_DIR=/app/data
OUTPOST_LOG_LEVEL=info

# Database
OUTPOST_DB_PATH=/app/data/outpost.db

# TMDB
TMDB_API_KEY=your_key_here

# Optional
OUTPOST_FFMPEG_PATH=/usr/bin/ffmpeg
OUTPOST_FFPROBE_PATH=/usr/bin/ffprobe
```

### Docker Compose

```yaml
version: '3.8'
services:
  outpost:
    image: outpost:latest
    container_name: outpost
    ports:
      - "8080:8080"
    volumes:
      - outpost_data:/app/data
      - /path/to/movies:/media/movies
      - /path/to/tv:/media/tv
      - /path/to/music:/media/music
      - /path/to/books:/media/books
    environment:
      - TMDB_API_KEY=your_key
    restart: unless-stopped

volumes:
  outpost_data:
```

---

## Security Considerations

- Passwords hashed with bcrypt (cost 12)
- Session tokens are random 32-byte strings
- Session expiry: 30 days
- CORS restricted to same origin in production
- No sensitive data in frontend bundle
- Media paths validated (no path traversal)
- Rate limiting on auth endpoints
