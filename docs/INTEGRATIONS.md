# INTEGRATIONS.md - External Services

## Overview

Outpost integrates with external services for metadata, indexers, and download clients. All integrations are optional and user-configured.

---

## Metadata Providers

### TMDB (The Movie Database)

**Used for:** Movies, TV Shows, Anime

**API:** https://api.themoviedb.org/3

**Required:** API key (free registration)

**Endpoints used:**
- `/search/movie` - Search movies
- `/search/tv` - Search TV shows
- `/movie/{id}` - Movie details
- `/tv/{id}` - TV show details
- `/tv/{id}/season/{num}` - Season details
- `/trending/movie/week` - Trending movies
- `/trending/tv/week` - Trending TV
- `/movie/popular` - Popular movies
- `/tv/popular` - Popular TV
- `/configuration` - Image base URLs

**Data fetched:**
- Title, original title
- Overview, tagline
- Release date, runtime
- Rating (vote_average)
- Genres
- Cast and crew
- Poster and backdrop images
- Content rating (certifications)

**Rate limits:** 40 requests per 10 seconds

**Caching:** Cache responses for 24 hours minimum

### MusicBrainz

**Used for:** Music

**API:** https://musicbrainz.org/ws/2

**Required:** No API key, but user-agent required

**Endpoints used:**
- `/artist` - Search/lookup artists
- `/release-group` - Albums
- `/release` - Specific releases
- `/recording` - Tracks

**Rate limits:** 1 request per second

### Cover Art Archive

**Used for:** Album artwork

**API:** https://coverartarchive.org

**Required:** None

### Open Library

**Used for:** Books

**API:** https://openlibrary.org

**Required:** None

**Endpoints used:**
- `/search.json` - Search books
- `/works/{id}.json` - Book details
- `/authors/{id}.json` - Author details

### Audnexus

**Used for:** Audiobooks

**API:** https://api.audnex.us

**Required:** None

### AniList (Future)

**Used for:** Anime-specific metadata, tracking

**API:** https://graphql.anilist.co

---

## Indexers

### Torznab

**Protocol:** REST/XML

**Standard endpoints:**
- `?t=caps` - Get capabilities
- `?t=search&q=query` - Search
- `?t=movie&imdbid=tt0000000` - Movie search
- `?t=tvsearch&tvdbid=000&season=1&ep=1` - TV search

**Required params:**
- `apikey` - API key

**Response format:**
```xml
<rss>
  <channel>
    <item>
      <title>Release.Name.2024</title>
      <guid>...</guid>
      <link>download_url</link>
      <size>1500000000</size>
      <pubDate>...</pubDate>
      <attr name="seeders" value="100"/>
      <attr name="peers" value="150"/>
      <attr name="category" value="2000"/>
    </item>
  </channel>
</rss>
```

### Newznab

**Protocol:** REST/XML (same as Torznab)

**Same endpoints and format, different categories for usenet**

### Prowlarr Integration

**Used for:** Centralized indexer management

**API:** REST/JSON

**Base URL:** User configured

**Required:** API key

**Endpoints used:**
- `GET /api/v1/indexer` - List indexers
- `GET /api/v1/search` - Search all indexers
- `GET /api/v1/indexerstats` - Statistics

**Sync behavior:**
1. Fetch indexer list from Prowlarr
2. Store indexer configs locally
3. Search directly via Prowlarr API
4. Re-sync periodically or on demand

---

## Download Clients

### qBittorrent

**Protocol:** REST/JSON

**Default port:** 8080

**Authentication:** Cookie-based session

**Endpoints used:**
- `POST /api/v2/auth/login` - Login
- `GET /api/v2/torrents/info` - List torrents
- `POST /api/v2/torrents/add` - Add torrent
- `POST /api/v2/torrents/delete` - Delete torrent
- `POST /api/v2/torrents/pause` - Pause
- `POST /api/v2/torrents/resume` - Resume

**Add torrent params:**
- `urls` or `torrents` (file)
- `category` - Category/label
- `savepath` - Download path
- `paused` - Start paused

### Transmission

**Protocol:** RPC/JSON

**Default port:** 9091

**Authentication:** Basic auth

**RPC endpoint:** `/transmission/rpc`

**Methods used:**
- `torrent-get` - List torrents
- `torrent-add` - Add torrent
- `torrent-remove` - Remove torrent
- `torrent-start` - Start
- `torrent-stop` - Stop

### Deluge

**Protocol:** JSON-RPC

**Default port:** 8112

**Authentication:** Session-based

**Methods used:**
- `auth.login`
- `core.get_torrents_status`
- `core.add_torrent_url`
- `core.remove_torrent`

### rTorrent

**Protocol:** XML-RPC

**Endpoint:** User configured (SCGI or HTTP)

**Methods used:**
- `d.multicall2` - List torrents
- `load.start` - Add torrent
- `d.erase` - Remove torrent

### SABnzbd

**Protocol:** REST/JSON

**Default port:** 8080

**Authentication:** API key

**Endpoints used:**
- `?mode=queue&output=json` - Get queue
- `?mode=addurl&name=URL` - Add NZB
- `?mode=history&output=json` - Get history
- `?mode=delete&value=nzo_id` - Delete item

### NZBGet

**Protocol:** JSON-RPC

**Default port:** 6789

**Authentication:** Basic auth

**Methods used:**
- `listgroups` - List downloads
- `append` - Add NZB
- `editqueue` - Manage queue
- `history` - Get history

---

## Integration Interface

All download clients implement a common interface:

```go
type DownloadClient interface {
    // Test connection
    TestConnection() error
    
    // Get active downloads
    GetDownloads() ([]Download, error)
    
    // Add download
    AddDownload(url string, category string) (string, error)
    
    // Remove download
    RemoveDownload(id string, deleteFiles bool) error
    
    // Pause/resume
    PauseDownload(id string) error
    ResumeDownload(id string) error
    
    // Get completed downloads for import
    GetCompleted() ([]CompletedDownload, error)
}

type Download struct {
    ID         string
    Name       string
    Status     string  // queued, downloading, paused, completed, error
    Progress   float64 // 0-100
    Size       int64
    Downloaded int64
    Speed      int64   // bytes/sec
    ETA        int     // seconds
}
```

All indexers implement:

```go
type Indexer interface {
    // Test connection
    TestConnection() error
    
    // Get capabilities
    GetCapabilities() (*Capabilities, error)
    
    // Search
    Search(query string, categories []int) ([]Release, error)
    
    // Search by ID
    SearchMovie(imdbID string) ([]Release, error)
    SearchTV(tvdbID int, season int, episode int) ([]Release, error)
}

type Release struct {
    Title       string
    GUID        string
    DownloadURL string
    Size        int64
    Seeders     int
    Peers       int
    Age         time.Duration
    Indexer     string
    Categories  []int
}
```

---

## Error Handling

All integrations should:

1. Retry transient failures (network errors, 5xx responses)
2. Respect rate limits with backoff
3. Log errors with context
4. Return user-friendly error messages
5. Not expose API keys in logs

---

## Configuration Storage

Credentials are stored in the `settings` table:

```sql
-- Download clients stored in download_clients table
-- Indexers stored in indexers table
-- API keys in settings table

INSERT INTO settings (key, value) VALUES
('tmdb_api_key', 'encrypted_value'),
('prowlarr_url', 'http://localhost:9696'),
('prowlarr_api_key', 'encrypted_value');
```

API keys should be encrypted at rest.
