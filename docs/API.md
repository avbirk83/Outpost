# API.md - Endpoint Reference

## Base URL

All endpoints are prefixed with `/api/`

---

## Authentication

### POST /api/auth/login

Login and create session.

**Request:**
```json
{
  "username": "admin",
  "password": "password123"
}
```

**Response:**
```json
{
  "data": {
    "user": {
      "id": 1,
      "username": "admin",
      "role": "admin"
    }
  }
}
```

Sets HTTP-only cookie `outpost_session`.

### POST /api/auth/logout

Destroy session.

**Response:**
```json
{
  "data": { "success": true }
}
```

### GET /api/auth/me

Get current user.

**Response:**
```json
{
  "data": {
    "id": 1,
    "username": "admin",
    "email": "admin@example.com",
    "role": "admin"
  }
}
```

---

## Health

### GET /api/health

Health check endpoint. No auth required.

**Response:**
```json
{
  "data": {
    "status": "ok",
    "version": "0.1.0",
    "uptime": 3600
  }
}
```

---

## Libraries

### GET /api/libraries

List all libraries.

**Response:**
```json
{
  "data": [
    {
      "id": 1,
      "name": "Movies",
      "path": "/media/movies",
      "type": "movies",
      "itemCount": 150,
      "lastScanned": "2024-12-27T10:00:00Z"
    }
  ]
}
```

### POST /api/libraries

Add a library. Admin only.

**Request:**
```json
{
  "name": "Movies",
  "path": "/media/movies",
  "type": "movies"
}
```

### PUT /api/libraries/:id

Update library. Admin only.

### DELETE /api/libraries/:id

Remove library. Admin only.

### POST /api/libraries/:id/scan

Trigger library scan. Admin only.

**Response:**
```json
{
  "data": {
    "status": "scanning",
    "jobId": "abc123"
  }
}
```

---

## Movies

### GET /api/movies

List movies with pagination.

**Query params:**
- `page` (default: 1)
- `limit` (default: 20)
- `sort` (title, year, added_at, rating)
- `order` (asc, desc)
- `search` (title search)
- `genre` (filter by genre)
- `year` (filter by year)

**Response:**
```json
{
  "data": [
    {
      "id": 1,
      "title": "Inception",
      "year": 2010,
      "overview": "...",
      "rating": 8.8,
      "runtime": 148,
      "posterPath": "/cache/posters/movie_1.jpg",
      "genres": ["Action", "Sci-Fi"],
      "hasFile": true
    }
  ],
  "pagination": {
    "page": 1,
    "limit": 20,
    "total": 150,
    "pages": 8
  }
}
```

### GET /api/movies/:id

Get movie details.

**Response:**
```json
{
  "data": {
    "id": 1,
    "tmdbId": 27205,
    "title": "Inception",
    "originalTitle": "Inception",
    "year": 2010,
    "overview": "...",
    "tagline": "...",
    "rating": 8.8,
    "runtime": 148,
    "contentRating": "PG-13",
    "genres": ["Action", "Sci-Fi"],
    "cast": [
      { "name": "Leonardo DiCaprio", "character": "Cobb" }
    ],
    "director": "Christopher Nolan",
    "posterPath": "/cache/posters/movie_1.jpg",
    "backdropPath": "/cache/backdrops/movie_1.jpg",
    "file": {
      "path": "/media/movies/Inception (2010)/Inception (2010).mkv",
      "size": 15000000000,
      "videoCodec": "HEVC",
      "audioCodec": "TrueHD",
      "resolution": "2160p"
    },
    "watchProgress": {
      "progress": 3600,
      "duration": 8880,
      "completed": false
    }
  }
}
```

### PUT /api/movies/:id

Update movie metadata. Admin only.

### DELETE /api/movies/:id

Remove movie from library. Admin only.

### POST /api/movies/:id/refresh

Refresh metadata from TMDB.

---

## TV Shows

### GET /api/shows

List shows with pagination. Same query params as movies.

### GET /api/shows/:id

Get show details including seasons.

**Response:**
```json
{
  "data": {
    "id": 1,
    "title": "Breaking Bad",
    "year": 2008,
    "overview": "...",
    "status": "ended",
    "seasons": [
      {
        "id": 1,
        "seasonNumber": 1,
        "episodeCount": 7,
        "availableEpisodes": 7
      }
    ]
  }
}
```

### GET /api/shows/:id/seasons/:seasonNum

Get season details with episodes.

### GET /api/shows/:id/seasons/:seasonNum/episodes/:epNum

Get episode details.

---

## Streaming

### GET /api/stream/:type/:id

Stream media file.

**Params:**
- `type`: movie, episode, track
- `id`: media ID

**Query params:**
- `transcode`: true/false (force transcoding)
- `quality`: 1080p, 720p, 480p (for transcoding)

**Response:** Video stream with range request support.

### GET /api/stream/:type/:id/subtitles

List available subtitles.

### GET /api/stream/:type/:id/subtitles/:index

Get subtitle file.

---

## Progress

### GET /api/progress/:type/:id

Get watch progress.

### POST /api/progress/:type/:id

Update watch progress.

**Request:**
```json
{
  "progress": 3600,
  "duration": 8880
}
```

### DELETE /api/progress/:type/:id

Clear watch progress.

### GET /api/continue-watching

Get items in progress for current user.

---

## Search

### GET /api/search

Search across all media types.

**Query params:**
- `q`: search query (required)
- `type`: movie, show, music, book (optional filter)

**Response:**
```json
{
  "data": {
    "movies": [...],
    "shows": [...],
    "music": [...],
    "books": [...]
  }
}
```

### GET /api/search/releases

Search indexers for releases.

**Query params:**
- `q`: search query
- `type`: movie, show
- `tmdbId`: TMDB ID (for better matching)

**Response:**
```json
{
  "data": [
    {
      "title": "Inception 2010 2160p UHD BluRay REMUX",
      "indexer": "Example",
      "size": 65000000000,
      "seeders": 150,
      "age": "30 days",
      "qualityScore": 160000,
      "downloadUrl": "..."
    }
  ]
}
```

---

## Grab

### POST /api/grab

Send release to download client.

**Request:**
```json
{
  "type": "movie",
  "mediaId": 1,
  "releaseUrl": "...",
  "indexer": "Example"
}
```

---

## Downloads

### GET /api/downloads

List active downloads.

**Response:**
```json
{
  "data": [
    {
      "id": 1,
      "name": "Inception.2010.2160p.UHD.BluRay.REMUX",
      "status": "downloading",
      "progress": 45.5,
      "size": 65000000000,
      "speed": 10000000,
      "eta": 3600,
      "client": "qbittorrent"
    }
  ]
}
```

---

## Discovery

### GET /api/discover/movies

Get trending/popular movies from TMDB.

**Query params:**
- `type`: trending, popular, upcoming, now_playing

### GET /api/discover/shows

Get trending/popular shows from TMDB.

---

## Requests

### GET /api/requests

List requests. Admin sees all, users see their own.

**Query params:**
- `status`: requested, approved, denied, available

### POST /api/requests

Create a request.

**Request:**
```json
{
  "type": "movie",
  "tmdbId": 27205,
  "title": "Inception",
  "year": 2010
}
```

### PUT /api/requests/:id

Update request (approve/deny). Admin only.

**Request:**
```json
{
  "status": "approved",
  "notes": "Added to queue"
}
```

---

## Wanted

### GET /api/wanted

List monitored items.

### POST /api/wanted

Add to wanted list.

### PUT /api/wanted/:id

Update monitoring settings.

### DELETE /api/wanted/:id

Remove from wanted list.

---

## Users

### GET /api/users

List users. Admin only.

### POST /api/users

Create user. Admin only.

**Request:**
```json
{
  "username": "john",
  "email": "john@example.com",
  "password": "password123",
  "role": "user",
  "autoApprove": false,
  "contentRating": "all"
}
```

### GET /api/users/:id

Get user details.

### PUT /api/users/:id

Update user. Admin or self only.

### DELETE /api/users/:id

Delete user. Admin only.

---

## Settings

### GET /api/settings

Get all settings. Admin only.

### PUT /api/settings

Update settings. Admin only.

### GET /api/settings/download-clients

List download clients.

### POST /api/settings/download-clients

Add download client.

### POST /api/settings/download-clients/:id/test

Test download client connection.

### GET /api/settings/indexers

List indexers.

### POST /api/settings/indexers

Add indexer.

### POST /api/settings/indexers/prowlarr/sync

Sync indexers from Prowlarr.

### GET /api/settings/quality-profiles

List quality profiles.

### POST /api/settings/quality-profiles

Create quality profile.

---

## Notifications

### GET /api/notifications

Get notifications for current user.

### PUT /api/notifications/:id/read

Mark notification as read.

### DELETE /api/notifications/:id

Delete notification.

---

## System

### GET /api/system/status

System status including disk space, CPU, memory.

### GET /api/system/logs

Get recent logs. Admin only.

### POST /api/system/backup

Trigger backup. Admin only.

### GET /api/system/jobs

List background jobs and status.
