# API.md - Endpoint Specifications

---

## Overview

All endpoints prefixed with `/api`. JSON request/response bodies.

Authentication via session cookie or `Authorization: Bearer <token>` header.

---

## Auth

### POST /api/auth/login
Login and create session.

```json
// Request
{ "username": "admin", "password": "password" }

// Response
{ "user": { "id": 1, "username": "admin", "role": "admin" }, "token": "abc123..." }
```

### POST /api/auth/logout
Destroy current session.

### GET /api/auth/me
Get current user.

```json
// Response
{ "id": 1, "username": "admin", "email": "admin@example.com", "role": "admin" }
```

### POST /api/auth/setup
Initial setup (create first admin). Only works if no users exist.

```json
// Request
{ "username": "admin", "password": "password", "email": "admin@example.com" }
```

---

## Libraries

### GET /api/libraries
List all libraries.

### POST /api/libraries
Create library.

```json
// Request
{ "name": "Movies", "type": "movies", "path": "/media/movies" }
```

### GET /api/libraries/:id
Get library details.

### PUT /api/libraries/:id
Update library.

### DELETE /api/libraries/:id
Delete library and all its media.

### POST /api/libraries/:id/scan
Trigger library scan.

---

## Movies

### GET /api/movies
List movies. Query params: `library_id`, `sort`, `order`, `page`, `limit`

### GET /api/movies/:id
Get movie details.

### PUT /api/movies/:id
Update movie metadata.

### DELETE /api/movies/:id
Delete movie from library.

### POST /api/movies/:id/refresh
Re-fetch metadata from TMDB.

---

## TV Shows

### GET /api/shows
List shows. Query params: `library_id`, `sort`, `order`, `page`, `limit`

### GET /api/shows/:id
Get show details with seasons.

### GET /api/shows/:id/seasons/:season
Get season with episodes.

### GET /api/episodes/:id
Get episode details.

### PUT /api/shows/:id
Update show metadata.

### DELETE /api/shows/:id
Delete show and all episodes.

---

## Music

### GET /api/artists
List artists.

### GET /api/artists/:id
Get artist with albums.

### GET /api/albums
List albums.

### GET /api/albums/:id
Get album with tracks.

### GET /api/tracks
List tracks.

### GET /api/tracks/:id
Get track details.

---

## Books

### GET /api/books
List books. Query params: `library_id`, `type` (book/comic/audiobook)

### GET /api/books/:id
Get book details.

---

## Streaming

### GET /api/stream/:type/:id
Stream media file. Supports range requests.

- `:type` = movie, episode, track, book
- Query params: `transcode=true`, `start=<seconds>`

### GET /api/media-info/:type/:id
Get media file info (codecs, resolution, etc.)

```json
// Response
{
  "video_codec": "h264",
  "audio_codec": "aac",
  "resolution": "1920x1080",
  "duration": 7200,
  "bitrate": 8000000,
  "container": "mkv",
  "audio_tracks": [
    { "index": 0, "codec": "aac", "channels": 6, "language": "eng" }
  ],
  "subtitles": [
    { "index": 0, "codec": "srt", "language": "eng" }
  ]
}
```

---

## Progress

### GET /api/progress
Get all progress for current user.

### GET /api/progress/:type/:id
Get progress for specific item.

### PUT /api/progress/:type/:id
Update progress.

```json
// Request
{ "position": 3600, "duration": 7200, "completed": false }
```

### DELETE /api/progress/:type/:id
Clear progress (mark as unwatched).

---

## Discovery (TMDB)

### GET /api/discover/movies
Get trending/popular movies from TMDB.

Query params: `category` (trending, popular, upcoming, top_rated), `page`

### GET /api/discover/shows
Get trending/popular shows from TMDB.

### GET /api/discover/movie/:tmdb_id
Get movie details from TMDB.

### GET /api/discover/show/:tmdb_id
Get show details from TMDB.

### GET /api/search
Search TMDB. Query params: `q`, `type` (movie/tv/all)

---

## Download Clients

### GET /api/download-clients
List configured download clients.

### POST /api/download-clients
Add download client.

```json
// Request
{
  "name": "qBittorrent",
  "type": "qbittorrent",
  "host": "192.168.1.100",
  "port": 8080,
  "username": "admin",
  "password": "adminadmin"
}
```

### PUT /api/download-clients/:id
Update download client.

### DELETE /api/download-clients/:id
Remove download client.

### POST /api/download-clients/:id/test
Test connection to download client.

---

## Indexers

### GET /api/indexers
List configured indexers.

### POST /api/indexers
Add indexer.

```json
// Request
{
  "name": "Jackett",
  "type": "torznab",
  "url": "http://localhost:9117/api/v2.0/indexers/all/results/torznab",
  "api_key": "abc123"
}
```

### PUT /api/indexers/:id
Update indexer.

### DELETE /api/indexers/:id
Remove indexer.

### POST /api/indexers/:id/test
Test indexer connection.

### POST /api/indexers/search
Search all indexers.

```json
// Request
{ "query": "Movie Name 2024", "categories": [2000, 5000] }

// Response
{
  "results": [
    {
      "indexer": "Jackett",
      "title": "Movie.Name.2024.2160p.WEB-DL",
      "size": 15000000000,
      "seeders": 50,
      "download_url": "magnet:?xt=..."
    }
  ]
}
```

---

## Quality Profiles

### GET /api/quality-profiles
List quality profiles.

### POST /api/quality-profiles
Create quality profile.

### PUT /api/quality-profiles/:id
Update quality profile.

### DELETE /api/quality-profiles/:id
Delete quality profile.

---

## Custom Formats

### GET /api/custom-formats
List custom formats.

### POST /api/custom-formats
Create custom format.

### PUT /api/custom-formats/:id
Update custom format.

### DELETE /api/custom-formats/:id
Delete custom format.

---

## Wanted

### GET /api/wanted
List wanted items.

### POST /api/wanted
Add to wanted list.

```json
// Request
{
  "media_type": "movie",
  "tmdb_id": 12345,
  "title": "Movie Name",
  "year": 2024,
  "quality_profile_id": 1
}
```

### DELETE /api/wanted/:id
Remove from wanted list.

### POST /api/wanted/:id/search
Manually search indexers for item.

---

## Requests

### GET /api/requests
List requests. Admins see all, users see their own.

### POST /api/requests
Create request.

```json
// Request
{
  "media_type": "movie",
  "tmdb_id": 12345,
  "title": "Movie Name",
  "year": 2024
}
```

### PUT /api/requests/:id
Update request (admin only).

```json
// Request
{ "status": "approved", "notes": "Added to queue" }
```

### DELETE /api/requests/:id
Cancel request.

---

## Users (Admin)

### GET /api/users
List all users.

### POST /api/users
Create user.

### PUT /api/users/:id
Update user.

### DELETE /api/users/:id
Delete user.

---

## Settings

### GET /api/settings
Get all settings.

### PUT /api/settings
Update settings.

```json
// Request
{ "tmdb_api_key": "abc123", "scan_interval": 3600 }
```

---

## Downloads (Queue)

### GET /api/downloads
Get active downloads from all clients.

```json
// Response
{
  "downloads": [
    {
      "client": "qBittorrent",
      "name": "Movie.Name.2024.2160p.WEB-DL",
      "progress": 45.5,
      "size": 15000000000,
      "speed": 10000000,
      "eta": 1200,
      "status": "downloading"
    }
  ]
}
```

---

## Error Responses

All errors return:

```json
{
  "error": "Error message",
  "code": "ERROR_CODE"
}
```

Common codes:
- `UNAUTHORIZED` - Not logged in
- `FORBIDDEN` - Not enough permissions
- `NOT_FOUND` - Resource doesn't exist
- `VALIDATION_ERROR` - Invalid request data
- `INTERNAL_ERROR` - Server error
