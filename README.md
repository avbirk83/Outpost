<p align="center">
  <img src="logos/outpost-banner-transparent.png" alt="Outpost" width="500">
</p>

<p align="center">
  <strong>A unified self-hosted media server for movies, TV shows, anime, music, and books.</strong>
</p>

<p align="center">
  <a href="#features">Features</a> •
  <a href="#installation">Installation</a> •
  <a href="#configuration">Configuration</a> •
  <a href="#tech-stack">Tech Stack</a>
</p>

---

## Features

### Library Management
- **Unified Library** — All your media in one place with smart organization
- **Multi-Media Support** — Movies, TV shows, anime, music, and books
- **Automatic Metadata** — Rich metadata from TMDB with posters, backdrops, and cast info
- **Watch Status Tracking** — Track what you've watched across all media types
- **Quality Detection** — Automatic parsing of resolution, source, HDR, audio format, and more
- **Collections** — Organize media into collections (auto-generated from TMDB or custom)

### Explore & Discovery
- **Explore** — Browse trending, popular, and top-rated content from TMDB
- **Calendar** — View upcoming releases and track air dates
- **Person Details** — Browse filmography and discover related content
- **Request System** — Users can request content with admin approval workflow
- **Watchlist** — Save content to watch later

### Quality Management
- **Quality Presets** — Built-in presets for movies, TV, and anime with customizable options
- **Custom Formats** — Create custom scoring rules based on resolution, source, codec, audio, HDR, release groups
- **Auto-Upgrade** — Automatically upgrade to better quality when available
- **Upgrade Search** — View all upgradeable items with one-click search for better quality
- **Cutoff System** — Stop upgrading after reaching target quality
- **Per-Item Monitoring** — Toggle monitoring for individual movies or TV seasons
- **Delay Profiles** — Wait for preferred quality before grabbing

### Automation
- **Prowlarr Integration** — Sync indexers directly from Prowlarr with auto-sync
- **Indexer Support** — Connect to Newznab and Torznab indexers
- **Download Clients** — Integration with qBittorrent, Transmission, SABnzbd, NZBGet
- **Automated Search** — Scheduled searches for missing and wanted content
- **RSS Feeds** — Monitor indexer RSS feeds for new releases
- **Import Queue** — Automatic import of completed downloads with rename support
- **Blocklist** — Automatically block failed releases and groups
- **Naming Templates** — Customizable folder and file naming

### Playback
- **Built-in Player** — Stream directly in browser with full playback controls
- **Track Selection** — Choose video, audio, and subtitle tracks
- **Skip Segments** — Configure intro/credits skip for TV shows
- **Chapter Support** — Navigate by chapters
- **Progress Sync** — Resume where you left off across devices
- **OpenSubtitles** — Search and download subtitles from OpenSubtitles with auto-download on import

### Integrations
- **Trakt.tv** — Sync watch history and ratings with Trakt (bidirectional sync)
- **OpenSubtitles** — Automatic subtitle downloading with language preferences

### Multi-User
- **User Accounts** — Create accounts for family and friends
- **Profiles** — Multiple profiles per user with individual watch history and preferences
- **Role-Based Access** — Admin and user roles with different permissions
- **Content Ratings** — Restrict content by rating (G, PG, PG-13, R, NC-17)
- **PIN Protection** — Optional PIN for user profiles
- **Per-User Tracking** — Individual watch history and watchlists
- **Smart Playlists** — Dynamic playlists with custom rules (recently added, unwatched, 4K, top rated, etc.)

### System Administration
- **Health Monitoring** — Real-time health checks for database, indexers, download clients, disk space
- **Storage Analytics** — View storage usage by library, quality, year with duplicate detection
- **Logs Viewer** — Browse and download application logs
- **Scheduled Tasks** — Configure and monitor background tasks
- **Backup & Restore** — Export and import configuration with merge/replace modes
- **Notifications** — In-app notification system for events

---

## Installation

### Docker Compose (Recommended)

```yaml
services:
  outpost:
    image: ghcr.io/avbirk83/outpost:latest
    container_name: outpost
    ports:
      - "8080:8080"
    volumes:
      - ./config:/config
      - /path/to/movies:/movies
      - /path/to/tv:/tv
      - /path/to/downloads:/downloads
    environment:
      - TMDB_API_KEY=your_api_key
    restart: unless-stopped
```

### Docker Run

```bash
docker run -d \
  --name outpost \
  -p 8080:8080 \
  -v /path/to/config:/config \
  -v /path/to/movies:/movies \
  -v /path/to/tv:/tv \
  -v /path/to/downloads:/downloads \
  -e TMDB_API_KEY=your_api_key \
  ghcr.io/avbirk83/outpost:latest
```

---

## Configuration

### Environment Variables

| Variable | Description | Required |
|----------|-------------|----------|
| `TMDB_API_KEY` | Your TMDB API key for metadata | Yes |
| `PORT` | Server port (default: 8080) | No |

### Initial Setup

1. Navigate to `http://localhost:8080`
2. Complete the setup wizard to create your admin account
3. Add libraries, download clients, and configure Prowlarr
4. Start scanning your media

### Libraries

Add your media libraries through Settings:

1. Go to **Settings** → **General**
2. Click **Add Library**
3. Select type (Movies, TV, Anime, Music, Books)
4. Enter the path to your media folder
5. Save and scan

---

## Automation

### Prowlarr Integration

The recommended way to manage indexers:

1. Go to **Settings** → **Sources** → **Prowlarr Sync**
2. Enter your Prowlarr URL and API key
3. Test the connection
4. Enable auto-sync and save
5. Click **Sync Now** to import indexers

### Manual Indexers

Add indexers manually:

1. Go to **Settings** → **Sources** → **Indexers**
2. Click **Add Indexer**
3. Select type (Torznab or Newznab)
4. Enter the indexer URL and API key
5. Test and save

### Download Clients

Supported download clients:

| Client | Type |
|--------|------|
| qBittorrent | Torrent |
| Transmission | Torrent |
| SABnzbd | Usenet |
| NZBGet | Usenet |

Configure in **Settings** → **Sources** → **Download Clients**.

### Quality Presets

Built-in presets organized by media type:

**Movies** — 4K Remux, 4K HDR, 4K, 1080p Remux, 1080p BluRay, 1080p, 720p, 480p

**TV Shows** — 4K HDR, 4K, 1080p, 1080p HDTV, 720p, Any

**Anime** — 4K, 1080p, 720p, 480p with preferences for dual audio, dubbed, and language

Customize or create new presets in **Settings** → **Quality**.

---

## System Management

### Backup & Restore

Export your configuration for backup or migration:

1. Go to **Settings** → **General** → **Backup & Restore**
2. Click **Download Backup** to export settings
3. To restore, upload a backup file and choose merge or replace mode

Backups include: settings, libraries, download clients, indexers, quality presets, collections, naming templates, and more.

### Health Monitoring

Monitor system health in **Settings** → **Health**:

- Database connectivity
- Download client status
- Indexer availability
- Disk space usage
- TMDB API connectivity

### Storage Analytics

View detailed storage information in **Settings** → **Storage**:

- Usage by library, quality, and year
- Largest items
- Duplicate detection

---

## Tech Stack

| Component | Technology |
|-----------|------------|
| Frontend | SvelteKit 5, TailwindCSS |
| Backend | Go |
| Database | SQLite |
| Metadata | TMDB API |
| Streaming | FFmpeg |

---

## Support

If you find Outpost useful, consider [sponsoring the project](https://github.com/sponsors/avbirk83).

---

<p align="center">
  <sub>Built with care for the self-hosted community.</sub>
</p>
