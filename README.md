<p align="center">
  <img src="logos/outpost-banner-transparent.png" alt="Outpost" width="500">
</p>

<p align="center">
  <strong>A unified self-hosted media server for movies and TV shows.</strong>
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
- **Automatic Metadata** — Rich metadata from TMDB with posters, backdrops, and cast info
- **Watch Status Tracking** — Track what you've watched across movies and TV shows
- **Quality Detection** — Automatic parsing of resolution, source, HDR, audio format, and more

### Explore & Requests
- **Explore** — Browse trending, popular, and top-rated content from TMDB with detailed views
- **Request System** — Users can request content with admin approval workflow and season selection for TV
- **Watchlist** — Save content to watch later, even before it's in your library

### Quality Management
- **Quality Presets** — Built-in presets (Best Quality, High Quality, Balanced, Storage Saver, Anime)
- **Custom Formats** — Create custom scoring rules based on resolution, source, codec, audio, HDR, release groups
- **Auto-Upgrade** — Automatically upgrade to better quality when available
- **Cutoff System** — Stop upgrading after reaching target quality
- **Per-Item Monitoring** — Toggle monitoring for individual movies or TV seasons

### Automation
- **Prowlarr Integration** — Sync indexers directly from Prowlarr with auto-sync support
- **Indexer Support** — Connect to Newznab and Torznab indexers
- **Download Clients** — Integration with qBittorrent, Transmission, SABnzbd, NZBGet
- **Automated Search** — Scheduled searches for missing and wanted content
- **RSS Feeds** — Monitor indexer RSS feeds for new releases
- **Import Queue** — Automatic import of completed downloads with rename support
- **Blocklist** — Automatically block failed releases and groups

### Playback
- **Built-in Player** — Stream directly in browser with full playback controls
- **Track Selection** — Choose video, audio, and subtitle tracks
- **Progress Sync** — Resume where you left off across devices

### Multi-User
- **User Accounts** — Create accounts for family and friends
- **Role-Based Access** — Admin and user roles with different permissions
- **Per-User Tracking** — Individual watch history and watchlists

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
2. Create your admin account
3. Add your TMDB API key in Settings

### Libraries

Add your media libraries through Settings:

1. Go to **Settings** → **Libraries**
2. Click **Add Library**
3. Select type (Movies or TV)
4. Enter the path to your media folder
5. Save and scan

---

## Automation

### Indexers

Outpost supports Newznab and Torznab indexers:

1. Go to **Settings** → **Indexers**
2. Click **Add Indexer**
3. Enter the indexer URL and API key
4. Test the connection and save

### Download Clients

Supported download clients:

| Client | Type |
|--------|------|
| qBittorrent | Torrent |
| Transmission | Torrent |
| SABnzbd | Usenet |
| NZBGet | Usenet |

Configure in **Settings** → **Download Clients**.

### Quality Presets

Built-in presets organized by media type:

**Movies** — 4K Remux, 4K HDR, 4K, 1080p Remux, 1080p BluRay, 1080p, 720p, 480p

**TV Shows** — 4K HDR, 4K, 1080p, 1080p HDTV, 720p, Any

**Anime** — 4K, 1080p, 720p, 480p with editable preferences for:
- Dual Audio toggle
- Dubbed toggle
- Language preference (Any, English, Japanese)

---

## Tech Stack

| Component | Technology |
|-----------|------------|
| Frontend | SvelteKit, TailwindCSS |
| Backend | Go |
| Database | SQLite |
| Metadata | TMDB API |
| Transcoding | FFmpeg |

---

## Support

If you find Outpost useful, consider [sponsoring the project](https://github.com/sponsors/avbirk83).

---

<p align="center">
  <sub>Built with care for the self-hosted community.</sub>
</p>
