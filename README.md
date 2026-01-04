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
- **Quality Management** — Automatic quality detection and upgrade support

### Discovery & Requests
- **Discover** — Browse trending, popular, and top-rated content
- **Request System** — Users can request content with admin approval workflow
- **Watchlist** — Save content to watch later, even before it's in your library

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

### Docker (Recommended)

```bash
docker run -d \
  --name outpost \
  -p 8080:8080 \
  -v /path/to/config:/config \
  -v /path/to/movies:/movies \
  -v /path/to/tv:/tv \
  -e TMDB_API_KEY=your_api_key \
  ghcr.io/avbirk83/outpost:latest
```

### Docker Compose

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
    environment:
      - TMDB_API_KEY=your_api_key
    restart: unless-stopped
```

---

## Configuration

### Environment Variables

| Variable | Description | Required |
|----------|-------------|----------|
| `TMDB_API_KEY` | Your TMDB API key for metadata | Yes |
| `PORT` | Server port (default: 8080) | No |

### Libraries

After starting Outpost, add your media libraries through Settings:

1. Go to **Settings** → **Libraries**
2. Click **Add Library**
3. Select type (Movies or TV)
4. Enter the path to your media folder
5. Save and scan

---

## Tech Stack

| Component | Technology |
|-----------|------------|
| Frontend | SvelteKit, TailwindCSS |
| Backend | Go |
| Database | SQLite |
| Metadata | TMDB API |

---

## Support

If you find Outpost useful, consider [sponsoring the project](https://github.com/sponsors/avbirk83).

---

<p align="center">
  <sub>Built with care for the self-hosted community.</sub>
</p>
