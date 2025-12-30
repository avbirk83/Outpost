# DATABASE.md - Schema Reference

## Overview

SQLite database with WAL mode enabled for better concurrent read performance.

Database file: `/data/outpost.db`

---

## Tables

### users

```sql
CREATE TABLE users (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    username TEXT NOT NULL UNIQUE,
    email TEXT,
    password_hash TEXT NOT NULL,
    role TEXT NOT NULL DEFAULT 'user', -- 'admin', 'user', 'kid'
    auto_approve BOOLEAN NOT NULL DEFAULT 0,
    content_rating TEXT DEFAULT 'all', -- 'all', 'pg13', 'pg', 'g'
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
```

### sessions

```sql
CREATE TABLE sessions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token TEXT NOT NULL UNIQUE,
    expires_at DATETIME NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_sessions_token ON sessions(token);
CREATE INDEX idx_sessions_expires ON sessions(expires_at);
```

### libraries

```sql
CREATE TABLE libraries (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    path TEXT NOT NULL,
    type TEXT NOT NULL, -- 'movies', 'tv', 'anime', 'music', 'books', 'audiobooks', 'comics'
    scan_interval INTEGER DEFAULT 60, -- minutes
    last_scanned_at DATETIME,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
```

### movies

```sql
CREATE TABLE movies (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    library_id INTEGER NOT NULL REFERENCES libraries(id) ON DELETE CASCADE,
    tmdb_id INTEGER,
    imdb_id TEXT,
    title TEXT NOT NULL,
    original_title TEXT,
    year INTEGER,
    overview TEXT,
    tagline TEXT,
    runtime INTEGER, -- minutes
    rating REAL,
    content_rating TEXT, -- 'G', 'PG', 'PG-13', 'R', 'NC-17'
    genres TEXT, -- JSON array
    cast TEXT, -- JSON array
    director TEXT,
    poster_path TEXT,
    backdrop_path TEXT,
    file_path TEXT NOT NULL,
    file_size INTEGER,
    video_codec TEXT,
    audio_codec TEXT,
    resolution TEXT,
    added_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_movies_library ON movies(library_id);
CREATE INDEX idx_movies_tmdb ON movies(tmdb_id);
CREATE INDEX idx_movies_title ON movies(title);
CREATE INDEX idx_movies_year ON movies(year);
```

### shows

```sql
CREATE TABLE shows (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    library_id INTEGER NOT NULL REFERENCES libraries(id) ON DELETE CASCADE,
    tmdb_id INTEGER,
    tvdb_id INTEGER,
    imdb_id TEXT,
    title TEXT NOT NULL,
    original_title TEXT,
    year INTEGER,
    overview TEXT,
    status TEXT, -- 'continuing', 'ended', 'canceled'
    rating REAL,
    content_rating TEXT,
    genres TEXT, -- JSON array
    cast TEXT, -- JSON array
    network TEXT,
    poster_path TEXT,
    backdrop_path TEXT,
    folder_path TEXT NOT NULL,
    added_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_shows_library ON shows(library_id);
CREATE INDEX idx_shows_tmdb ON shows(tmdb_id);
CREATE INDEX idx_shows_title ON shows(title);
```

### seasons

```sql
CREATE TABLE seasons (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    show_id INTEGER NOT NULL REFERENCES shows(id) ON DELETE CASCADE,
    season_number INTEGER NOT NULL,
    name TEXT,
    overview TEXT,
    poster_path TEXT,
    air_date DATE,
    UNIQUE(show_id, season_number)
);

CREATE INDEX idx_seasons_show ON seasons(show_id);
```

### episodes

```sql
CREATE TABLE episodes (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    season_id INTEGER NOT NULL REFERENCES seasons(id) ON DELETE CASCADE,
    episode_number INTEGER NOT NULL,
    title TEXT,
    overview TEXT,
    air_date DATE,
    runtime INTEGER,
    still_path TEXT,
    file_path TEXT NOT NULL,
    file_size INTEGER,
    video_codec TEXT,
    audio_codec TEXT,
    resolution TEXT,
    added_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(season_id, episode_number)
);

CREATE INDEX idx_episodes_season ON episodes(season_id);
```

### artists

```sql
CREATE TABLE artists (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    library_id INTEGER NOT NULL REFERENCES libraries(id) ON DELETE CASCADE,
    musicbrainz_id TEXT,
    name TEXT NOT NULL,
    sort_name TEXT,
    overview TEXT,
    image_path TEXT,
    folder_path TEXT NOT NULL,
    added_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_artists_library ON artists(library_id);
CREATE INDEX idx_artists_name ON artists(name);
```

### albums

```sql
CREATE TABLE albums (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    artist_id INTEGER NOT NULL REFERENCES artists(id) ON DELETE CASCADE,
    musicbrainz_id TEXT,
    title TEXT NOT NULL,
    year INTEGER,
    genres TEXT, -- JSON array
    cover_path TEXT,
    folder_path TEXT NOT NULL,
    added_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_albums_artist ON albums(artist_id);
```

### tracks

```sql
CREATE TABLE tracks (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    album_id INTEGER NOT NULL REFERENCES albums(id) ON DELETE CASCADE,
    track_number INTEGER,
    disc_number INTEGER DEFAULT 1,
    title TEXT NOT NULL,
    duration INTEGER, -- seconds
    file_path TEXT NOT NULL,
    file_size INTEGER,
    bitrate INTEGER,
    format TEXT,
    added_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_tracks_album ON tracks(album_id);
```

### books

```sql
CREATE TABLE books (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    library_id INTEGER NOT NULL REFERENCES libraries(id) ON DELETE CASCADE,
    isbn TEXT,
    title TEXT NOT NULL,
    author TEXT,
    publisher TEXT,
    year INTEGER,
    description TEXT,
    genres TEXT, -- JSON array
    cover_path TEXT,
    file_path TEXT NOT NULL,
    file_size INTEGER,
    format TEXT, -- 'epub', 'pdf', 'mobi', 'cbz', 'cbr'
    page_count INTEGER,
    type TEXT DEFAULT 'book', -- 'book', 'audiobook', 'comic'
    added_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_books_library ON books(library_id);
CREATE INDEX idx_books_title ON books(title);
CREATE INDEX idx_books_author ON books(author);
```

### quality_profiles

```sql
CREATE TABLE quality_profiles (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    upgrade_allowed BOOLEAN DEFAULT 1,
    upgrade_until_quality TEXT,
    min_format_score INTEGER DEFAULT 0,
    upgrade_until_score INTEGER,
    min_score_increment INTEGER DEFAULT 1,
    qualities TEXT NOT NULL, -- JSON array of enabled qualities in order
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
```

### custom_formats

```sql
CREATE TABLE custom_formats (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    profile_id INTEGER NOT NULL REFERENCES quality_profiles(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    score INTEGER NOT NULL,
    conditions TEXT NOT NULL -- JSON array of match conditions
);

CREATE INDEX idx_custom_formats_profile ON custom_formats(profile_id);
```

### wanted

```sql
CREATE TABLE wanted (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    type TEXT NOT NULL, -- 'movie', 'show'
    tmdb_id INTEGER NOT NULL,
    title TEXT NOT NULL,
    year INTEGER,
    quality_profile_id INTEGER REFERENCES quality_profiles(id),
    monitored BOOLEAN DEFAULT 1,
    seasons TEXT, -- JSON array for shows, null for movies
    added_by INTEGER REFERENCES users(id),
    added_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_wanted_type ON wanted(type);
CREATE INDEX idx_wanted_tmdb ON wanted(tmdb_id);
```

### requests

```sql
CREATE TABLE requests (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    type TEXT NOT NULL, -- 'movie', 'show'
    tmdb_id INTEGER NOT NULL,
    title TEXT NOT NULL,
    year INTEGER,
    poster_path TEXT,
    requested_by INTEGER NOT NULL REFERENCES users(id),
    status TEXT NOT NULL DEFAULT 'requested', -- 'requested', 'approved', 'denied', 'available'
    approved_by INTEGER REFERENCES users(id),
    seasons TEXT, -- JSON array for shows
    notes TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_requests_user ON requests(requested_by);
CREATE INDEX idx_requests_status ON requests(status);
```

### downloads

```sql
CREATE TABLE downloads (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    type TEXT NOT NULL, -- 'movie', 'episode'
    media_id INTEGER NOT NULL,
    download_id TEXT NOT NULL, -- ID from download client
    client TEXT NOT NULL, -- 'qbittorrent', 'transmission', 'sabnzbd', 'nzbget'
    name TEXT NOT NULL,
    status TEXT NOT NULL, -- 'queued', 'downloading', 'completed', 'failed', 'imported'
    progress REAL DEFAULT 0,
    size INTEGER,
    added_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    completed_at DATETIME
);

CREATE INDEX idx_downloads_status ON downloads(status);
```

### watch_progress

```sql
CREATE TABLE watch_progress (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    type TEXT NOT NULL, -- 'movie', 'episode', 'track', 'book'
    media_id INTEGER NOT NULL,
    progress INTEGER NOT NULL, -- seconds for video/audio, page for books
    duration INTEGER, -- total duration/pages
    completed BOOLEAN DEFAULT 0,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(user_id, type, media_id)
);

CREATE INDEX idx_progress_user ON watch_progress(user_id);
```

### settings

```sql
CREATE TABLE settings (
    key TEXT PRIMARY KEY,
    value TEXT NOT NULL,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
```

### download_clients

```sql
CREATE TABLE download_clients (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    type TEXT NOT NULL, -- 'qbittorrent', 'transmission', 'deluge', 'rtorrent', 'sabnzbd', 'nzbget'
    host TEXT NOT NULL,
    port INTEGER NOT NULL,
    username TEXT,
    password TEXT,
    use_ssl BOOLEAN DEFAULT 0,
    category TEXT,
    priority INTEGER DEFAULT 0,
    enabled BOOLEAN DEFAULT 1,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
```

### indexers

```sql
CREATE TABLE indexers (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    type TEXT NOT NULL, -- 'torznab', 'newznab'
    url TEXT NOT NULL,
    api_key TEXT,
    categories TEXT, -- JSON array
    priority INTEGER DEFAULT 0,
    enabled BOOLEAN DEFAULT 1,
    prowlarr_id INTEGER, -- if synced from Prowlarr
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
```

### blocked_extensions

```sql
CREATE TABLE blocked_extensions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    pattern TEXT NOT NULL UNIQUE
);

-- Pre-populate with defaults
INSERT INTO blocked_extensions (pattern) VALUES
('.exe'), ('.bat'), ('.cmd'), ('.com'), ('.dll'), ('.iso'), ('.bin'),
('.js'), ('.vbs'), ('.ps1'), ('.sh'), ('.py'), ('.msi'), ('.scr'),
('*sample.mkv'), ('*sample.avi'), ('*sample.mp4');
```

### notifications

```sql
CREATE TABLE notifications (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER REFERENCES users(id) ON DELETE CASCADE, -- null = all users
    type TEXT NOT NULL,
    title TEXT NOT NULL,
    message TEXT,
    read BOOLEAN DEFAULT 0,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_notifications_user ON notifications(user_id);
CREATE INDEX idx_notifications_read ON notifications(read);
```

---

## Migrations

Migrations are stored in `internal/database/migrations/` as numbered SQL files:

```
001_initial_schema.sql
002_add_anime_settings.sql
003_add_notifications.sql
...
```

Run migrations on startup. Track applied migrations in:

```sql
CREATE TABLE migrations (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL UNIQUE,
    applied_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
```

---

## Indexes Summary

All foreign keys should have indexes for JOIN performance.
Additional indexes on frequently queried columns (title, year, status).

---

## Backup Strategy

1. SQLite file can be copied while in WAL mode
2. Use `.backup` command for consistent snapshot
3. Backup location configurable
4. Optional upload to S3/Google Drive
