# INTEGRATIONS.md - External Services

---

## Overview

Outpost integrates with external services for:
- Metadata (TMDB, AniList, Audnexus)
- Download automation (torrent/usenet clients)
- Indexer search (Torznab/Newznab)

---

## TMDB (The Movie Database)

### Purpose
Movie and TV show metadata, images, search, discovery.

### API
- Base URL: `https://api.themoviedb.org/3`
- Auth: API key as query param or Bearer token
- Rate limit: 40 requests/10 seconds

### Endpoints Used

**Search:**
```
GET /search/movie?query=Movie%20Name&year=2024
GET /search/tv?query=Show%20Name
GET /search/multi?query=anything
```

**Details:**
```
GET /movie/{id}?append_to_response=credits,videos,images
GET /tv/{id}?append_to_response=credits,videos,images
GET /tv/{id}/season/{season}
GET /tv/{id}/season/{season}/episode/{episode}
```

**Discovery:**
```
GET /trending/movie/week
GET /trending/tv/week
GET /movie/popular
GET /movie/upcoming
GET /tv/popular
```

**Images:**
- Base URL: `https://image.tmdb.org/t/p/`
- Sizes: w92, w154, w185, w342, w500, w780, original
- Example: `https://image.tmdb.org/t/p/w500/poster.jpg`

### Matching Strategy
1. Parse filename for title, year, season, episode
2. Search TMDB
3. If year matches and title similarity > 80%, auto-match
4. Otherwise, prompt for manual selection

---

## AniList (Future)

### Purpose
Anime metadata with Japanese/Romaji/English titles, AniDB/MAL IDs.

### API
- GraphQL: `https://graphql.anilist.co`
- No API key required for public data

### Query Example
```graphql
query ($search: String) {
  Media(search: $search, type: ANIME) {
    id
    title {
      romaji
      english
      native
    }
    episodes
    status
    coverImage {
      large
    }
    description
  }
}
```

### Why AniList over TMDB for Anime
- Better Japanese title handling
- More accurate episode counts
- Release group/fansub awareness
- Airing schedule tracking

---

## Audnexus (Future)

### Purpose
Audiobook metadata from Audible (unofficial).

### API
- Base URL: `https://api.audnex.us`
- No API key required

### Endpoints
```
GET /books/{asin}
GET /authors/{asin}
GET /books/{asin}/chapters
```

### Matching
- ISBN to ASIN mapping
- Title + author search

---

## Download Clients

### qBittorrent

**API Base:** `http://host:port/api/v2`

**Authentication:**
```
POST /auth/login
Content-Type: application/x-www-form-urlencoded

username=admin&password=adminadmin
```
Returns cookie for session.

**Add Torrent:**
```
POST /torrents/add
Content-Type: multipart/form-data

urls=magnet:?xt=urn:btih:...
category=movies
savepath=/downloads/movies
```

**Get Torrents:**
```
GET /torrents/info?category=movies
```

**Response:**
```json
[{
  "hash": "abc123",
  "name": "Movie.Name.2024.2160p.WEB-DL",
  "progress": 0.45,
  "size": 15000000000,
  "dlspeed": 10000000,
  "eta": 1200,
  "state": "downloading"
}]
```

### Transmission

**API:** JSON-RPC at `http://host:port/transmission/rpc`

**Headers:**
- `X-Transmission-Session-Id`: Get from initial request

**Add Torrent:**
```json
{
  "method": "torrent-add",
  "arguments": {
    "filename": "magnet:?xt=...",
    "download-dir": "/downloads/movies"
  }
}
```

**Get Torrents:**
```json
{
  "method": "torrent-get",
  "arguments": {
    "fields": ["id", "name", "percentDone", "totalSize", "rateDownload", "eta", "status"]
  }
}
```

### SABnzbd

**API:** `http://host:port/api`

**Add NZB:**
```
GET /api?mode=addurl&name=http://indexer.com/nzb/123&apikey=xxx
```

**Get Queue:**
```
GET /api?mode=queue&output=json&apikey=xxx
```

### NZBGet

**API:** JSON-RPC at `http://host:port/jsonrpc`

**Auth:** Basic auth or in URL

**Add NZB:**
```json
{
  "method": "append",
  "params": ["Movie.Name.nzb", "base64_content", "movies", 0, false, false, "", 0, "SCORE"]
}
```

---

## Indexers

### Torznab (Torrent)

Jackett/Prowlarr-compatible API.

**Capabilities:**
```
GET /api/v2.0/indexers/all/results/torznab?t=caps&apikey=xxx
```

**Search:**
```
GET /api/v2.0/indexers/all/results/torznab?t=search&q=Movie%20Name&apikey=xxx
GET /api/v2.0/indexers/all/results/torznab?t=movie&q=Movie%20Name&imdbid=tt1234567
GET /api/v2.0/indexers/all/results/torznab?t=tvsearch&q=Show&season=1&ep=5
```

**Response (RSS XML):**
```xml
<rss>
  <channel>
    <item>
      <title>Movie.Name.2024.2160p.WEB-DL.DV.HDR.DDP5.1.Atmos</title>
      <size>15000000000</size>
      <link>magnet:?xt=urn:btih:...</link>
      <torznab:attr name="seeders" value="50"/>
      <torznab:attr name="peers" value="100"/>
    </item>
  </channel>
</rss>
```

### Newznab (Usenet)

Same API structure as Torznab but returns NZB download links.

**Search:**
```
GET /api?t=search&q=Movie%20Name&apikey=xxx
GET /api?t=movie&imdbid=tt1234567&apikey=xxx
```

---

## Prowlarr Integration (Optional)

Instead of configuring indexers manually, connect to Prowlarr.

**Get Indexers:**
```
GET /api/v1/indexer
X-Api-Key: prowlarr_api_key
```

**Sync:** Pull indexer configs and use them directly.

---

## Webhooks (Notifications)

### Discord

```
POST https://discord.com/api/webhooks/{id}/{token}
Content-Type: application/json

{
  "content": "Movie Name (2024) is now available!",
  "embeds": [{
    "title": "Movie Name",
    "thumbnail": {"url": "https://image.tmdb.org/t/p/w185/poster.jpg"}
  }]
}
```

### Custom Webhook

```
POST {user_configured_url}
Content-Type: application/json

{
  "event": "media.available",
  "media": {
    "type": "movie",
    "title": "Movie Name",
    "year": 2024,
    "tmdb_id": 12345
  }
}
```

---

## Rate Limiting

| Service | Limit |
|---------|-------|
| TMDB | 40 req/10s |
| AniList | 90 req/min |
| Indexers | Varies |

Implement:
- Request queuing
- Exponential backoff on 429
- Cache responses where appropriate

---

## Error Handling

### Connection Failures
- Retry with backoff
- Mark service as unavailable
- Show status in UI

### API Errors
- Log full response
- Show user-friendly message
- Don't retry immediately

### Timeout
- 10 second default
- 30 seconds for search operations
