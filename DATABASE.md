# DATABASE.md - Schema Details

---

## Overview

SQLite database with WAL mode. 19 tables across these domains:

- Users & Auth
- Libraries & Media
- Progress & Playback
- Downloads & Automation
- Requests

---

## Schema

### Users & Auth

```sql
CREATE TABLE users (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    username TEXT UNIQUE NOT NULL,
    email TEXT UNIQUE,
    password_hash TEXT NOT NULL,
    role TEXT NOT NULL DEFAULT 'user',  -- admin, user, kid
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE sessions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token TEXT UNIQUE NOT NULL,
    expires_at DATETIME NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_sessions_token ON sessions(token);
CREATE INDEX idx_sessions_user ON sessions(user_id);
```

### Libraries

```sql
CREATE TABLE libraries (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    type TEXT NOT NULL,  -- movies, tv, anime, music, books, audiobooks
    path TEXT NOT NULL,
    scan_interval INTEGER DEFAULT 3600,  -- seconds
    last_scan DATETIME,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
```

### Movies

```sql
CREATE TABLE movies (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    library_id INTEGER NOT NULL REFERENCES libraries(id) ON DELETE CASCADE,
    tmdb_id INTEGER,
    imdb_id TEXT,
    title TEXT NOT NULL,
    original_title TEXT,
    year INTEGER,
    runtime INTEGER,  -- minutes
    overview TEXT,
    tagline TEXT,
    poster_path TEXT,
    backdrop_path TEXT,
    rating REAL,
    genres TEXT,  -- JSON array
    file_path TEXT NOT NULL,
    file_size INTEGER,
    video_codec TEXT,
    audio_codec TEXT,
    resolution TEXT,
    container TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_movies_library ON movies(library_id);
CREATE INDEX idx_movies_tmdb ON movies(tmdb_id);
```

### TV Shows

```sql
CREATE TABLE shows (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    library_id INTEGER NOT NULL REFERENCES libraries(id) ON DELETE CASCADE,
    tmdb_id INTEGER,
    imdb_id TEXT,
    title TEXT NOT NULL,
    original_title TEXT,
    year INTEGER,
    status TEXT,  -- Returning Series, Ended, Canceled
    overview TEXT,
    poster_path TEXT,
    backdrop_path TEXT,
    rating REAL,
    genres TEXT,  -- JSON array
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE seasons (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    show_id INTEGER NOT NULL REFERENCES shows(id) ON DELETE CASCADE,
    season_number INTEGER NOT NULL,
    name TEXT,
    overview TEXT,
    poster_path TEXT,
    air_date DATE,
    episode_count INTEGER,
    UNIQUE(show_id, season_number)
);

CREATE TABLE episodes (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    show_id INTEGER NOT NULL REFERENCES shows(id) ON DELETE CASCADE,
    season_id INTEGER REFERENCES seasons(id) ON DELETE CASCADE,
    season_number INTEGER NOT NULL,
    episode_number INTEGER NOT NULL,
    title TEXT,
    overview TEXT,
    still_path TEXT,
    air_date DATE,
    runtime INTEGER,
    file_path TEXT NOT NULL,
    file_size INTEGER,
    video_codec TEXT,
    audio_codec TEXT,
    resolution TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(show_id, season_number, episode_number)
);

CREATE INDEX idx_shows_library ON shows(library_id);
CREATE INDEX idx_episodes_show ON episodes(show_id);
```

### Music

```sql
CREATE TABLE artists (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    library_id INTEGER NOT NULL REFERENCES libraries(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    sort_name TEXT,
    musicbrainz_id TEXT,
    overview TEXT,
    image_path TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE albums (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    artist_id INTEGER REFERENCES artists(id) ON DELETE SET NULL,
    library_id INTEGER NOT NULL REFERENCES libraries(id) ON DELETE CASCADE,
    title TEXT NOT NULL,
    year INTEGER,
    musicbrainz_id TEXT,
    genre TEXT,
    cover_path TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE tracks (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    album_id INTEGER REFERENCES albums(id) ON DELETE CASCADE,
    artist_id INTEGER REFERENCES artists(id) ON DELETE SET NULL,
    library_id INTEGER NOT NULL REFERENCES libraries(id) ON DELETE CASCADE,
    title TEXT NOT NULL,
    track_number INTEGER,
    disc_number INTEGER DEFAULT 1,
    duration INTEGER,  -- seconds
    file_path TEXT NOT NULL,
    file_size INTEGER,
    codec TEXT,
    bitrate INTEGER,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_albums_artist ON albums(artist_id);
CREATE INDEX idx_tracks_album ON tracks(album_id);
```

### Books

```sql
CREATE TABLE books (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    library_id INTEGER NOT NULL REFERENCES libraries(id) ON DELETE CASCADE,
    title TEXT NOT NULL,
    author TEXT,
    year INTEGER,
    isbn TEXT,
    overview TEXT,
    cover_path TEXT,
    file_path TEXT NOT NULL,
    file_size INTEGER,
    format TEXT,  -- epub, pdf, mobi, cbz, cbr, m4b, mp3
    type TEXT DEFAULT 'book',  -- book, comic, audiobook
    duration INTEGER,  -- for audiobooks, in seconds
    chapters TEXT,  -- JSON array for audiobooks
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_books_library ON books(library_id);
CREATE INDEX idx_books_type ON books(type);
```

### Progress

```sql
CREATE TABLE progress (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    media_type TEXT NOT NULL,  -- movie, episode, track, book
    media_id INTEGER NOT NULL,
    position INTEGER NOT NULL,  -- seconds or page number
    duration INTEGER,  -- total duration/pages
    completed BOOLEAN DEFAULT FALSE,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(user_id, media_type, media_id)
);

CREATE INDEX idx_progress_user ON progress(user_id);
CREATE INDEX idx_progress_media ON progress(media_type, media_id);
```

### Download Automation

```sql
CREATE TABLE download_clients (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    type TEXT NOT NULL,  -- qbittorrent, transmission, sabnzbd, nzbget
    host TEXT NOT NULL,
    port INTEGER NOT NULL,
    username TEXT,
    password TEXT,
    use_ssl BOOLEAN DEFAULT FALSE,
    category TEXT,
    priority INTEGER DEFAULT 0,
    enabled BOOLEAN DEFAULT TRUE,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE indexers (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    type TEXT NOT NULL,  -- torznab, newznab
    url TEXT NOT NULL,
    api_key TEXT,
    categories TEXT,  -- JSON array of category IDs
    priority INTEGER DEFAULT 0,
    enabled BOOLEAN DEFAULT TRUE,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE quality_profiles (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    upgrade_allowed BOOLEAN DEFAULT TRUE,
    cutoff_quality TEXT,
    min_format_score INTEGER DEFAULT 0,
    cutoff_format_score INTEGER,
    upgrade_score_increment INTEGER DEFAULT 100,
    qualities TEXT NOT NULL,  -- JSON array of quality definitions
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE custom_formats (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    conditions TEXT NOT NULL,  -- JSON array of conditions
    score INTEGER DEFAULT 0,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE wanted (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    media_type TEXT NOT NULL,  -- movie, show
    tmdb_id INTEGER NOT NULL,
    title TEXT NOT NULL,
    year INTEGER,
    quality_profile_id INTEGER REFERENCES quality_profiles(id),
    monitored BOOLEAN DEFAULT TRUE,
    status TEXT DEFAULT 'wanted',  -- wanted, searching, downloading, downloaded
    added_by INTEGER REFERENCES users(id),
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(media_type, tmdb_id)
);

CREATE INDEX idx_wanted_status ON wanted(status);
```

### Requests

```sql
CREATE TABLE requests (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    media_type TEXT NOT NULL,  -- movie, show
    tmdb_id INTEGER NOT NULL,
    title TEXT NOT NULL,
    year INTEGER,
    poster_path TEXT,
    status TEXT DEFAULT 'pending',  -- pending, approved, denied, available
    approved_by INTEGER REFERENCES users(id),
    approved_at DATETIME,
    notes TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(user_id, media_type, tmdb_id)
);

CREATE INDEX idx_requests_user ON requests(user_id);
CREATE INDEX idx_requests_status ON requests(status);
```

### Settings

```sql
CREATE TABLE settings (
    key TEXT PRIMARY KEY,
    value TEXT,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
```

---

## Indexes

All foreign keys are indexed. Additional indexes for:
- Session token lookups
- Media by library
- Progress by user
- Wanted/requests by status

---

## Notes

- All timestamps are UTC
- JSON fields store arrays/objects as TEXT
- File paths are absolute paths within container
- Soft deletes not implemented (hard delete with CASCADE)
- WAL mode enables concurrent reads during writes
