# STREAMING.md - Video & Audio Playback

## Overview

Outpost supports both direct play (serving the original file) and transcoding (converting on-the-fly). Direct play is always preferred for quality and server load.

---

## Direct Play

### When to Direct Play

Direct play when the client supports:
- Container format (mkv, mp4, webm)
- Video codec (h264, h265/hevc, vp9, av1)
- Audio codec (aac, mp3, ac3, eac3)
- Resolution and bitrate

### Implementation

```go
func StreamDirect(w http.ResponseWriter, r *http.Request, filePath string) {
    file, _ := os.Open(filePath)
    defer file.Close()
    
    // Get file info
    stat, _ := file.Stat()
    
    // Set headers
    w.Header().Set("Content-Type", "video/mp4") // or appropriate type
    w.Header().Set("Accept-Ranges", "bytes")
    
    // Handle range requests
    http.ServeContent(w, r, stat.Name(), stat.ModTime(), file)
}
```

### Range Requests

Essential for seeking. Format:

```
Request:  Range: bytes=1000000-2000000
Response: Content-Range: bytes 1000000-2000000/50000000
          Content-Length: 1000001
          HTTP 206 Partial Content
```

---

## Transcoding

### When to Transcode

- Client doesn't support codec (e.g., HEVC on older devices)
- Client requests lower quality (bandwidth)
- Audio needs conversion
- Subtitles need to be burned in

### HLS Output

Convert to HLS (HTTP Live Streaming) for adaptive bitrate:

```
/stream/movie/123/
├── master.m3u8      # Master playlist
├── 1080p.m3u8       # 1080p variant playlist
├── 720p.m3u8        # 720p variant playlist
├── 480p.m3u8        # 480p variant playlist
└── segments/
    ├── 1080p_0001.ts
    ├── 1080p_0002.ts
    └── ...
```

### Master Playlist

```m3u8
#EXTM3U
#EXT-X-VERSION:3
#EXT-X-STREAM-INF:BANDWIDTH=5000000,RESOLUTION=1920x1080
1080p.m3u8
#EXT-X-STREAM-INF:BANDWIDTH=2500000,RESOLUTION=1280x720
720p.m3u8
#EXT-X-STREAM-INF:BANDWIDTH=1000000,RESOLUTION=854x480
480p.m3u8
```

### FFmpeg Commands

**1080p transcode:**
```bash
ffmpeg -i input.mkv \
  -c:v libx264 -preset fast -crf 22 \
  -c:a aac -b:a 192k \
  -f hls \
  -hls_time 6 \
  -hls_list_size 0 \
  -hls_segment_filename 'segments/1080p_%04d.ts' \
  1080p.m3u8
```

**720p transcode:**
```bash
ffmpeg -i input.mkv \
  -c:v libx264 -preset fast -crf 23 \
  -vf scale=1280:720 \
  -c:a aac -b:a 128k \
  -f hls \
  -hls_time 6 \
  -hls_list_size 0 \
  -hls_segment_filename 'segments/720p_%04d.ts' \
  720p.m3u8
```

### Hardware Acceleration

Support hardware transcoding when available:

| Platform | Technology | FFmpeg flags |
|----------|------------|--------------|
| NVIDIA | NVENC | `-c:v h264_nvenc` |
| Intel | QSV | `-c:v h264_qsv` |
| AMD | AMF | `-c:v h264_amf` |
| Raspberry Pi | V4L2 | `-c:v h264_v4l2m2m` |

Detection:
```bash
# Check for NVIDIA
ffmpeg -encoders 2>/dev/null | grep nvenc

# Check for Intel QSV
ffmpeg -encoders 2>/dev/null | grep qsv
```

---

## Subtitles

### Supported Formats

| Format | Extension | Notes |
|--------|-----------|-------|
| SRT | .srt | Most common, text-based |
| ASS/SSA | .ass, .ssa | Styled subtitles |
| VobSub | .sub, .idx | DVD subtitles (image-based) |
| PGS | embedded | Blu-ray subtitles (image-based) |
| WebVTT | .vtt | Web standard |

### Extraction

Extract embedded subtitles:
```bash
ffmpeg -i input.mkv -map 0:s:0 -c:s srt output.srt
```

### Delivery Options

1. **Side-load (preferred):** Deliver subtitle file separately, player renders
2. **Burn-in:** Encode subtitles into video stream (required for image-based)

### API

```
GET /api/stream/movie/123/subtitles
Response:
{
  "data": [
    { "index": 0, "language": "eng", "title": "English", "format": "srt" },
    { "index": 1, "language": "spa", "title": "Spanish", "format": "srt" }
  ]
}

GET /api/stream/movie/123/subtitles/0
Response: SRT or VTT file
```

---

## Audio Tracks

### Detection

```bash
ffprobe -v quiet -print_format json -show_streams input.mkv
```

Extract audio stream info:
- Codec (ac3, eac3, dts, truehd, aac)
- Channels (2.0, 5.1, 7.1)
- Language
- Title/description

### API

```
GET /api/stream/movie/123/audio
Response:
{
  "data": [
    { "index": 0, "codec": "truehd", "channels": "7.1", "language": "eng", "title": "TrueHD Atmos" },
    { "index": 1, "codec": "ac3", "channels": "5.1", "language": "eng", "title": "Commentary" }
  ]
}
```

### Audio Track Selection

When transcoding, select specific audio track:
```bash
ffmpeg -i input.mkv -map 0:v:0 -map 0:a:1 ...
```

---

## Audio Sync Adjustment

Sometimes audio is out of sync. Allow manual adjustment:

### Client-side (preferred)
JavaScript media element:
```javascript
video.currentTime = video.currentTime + offset; // Doesn't work well

// Better: Use AudioContext to delay audio
// Or: Use video.playbackRate briefly
```

### Server-side
Add delay during transcode:
```bash
ffmpeg -i input.mkv -itsoffset 0.5 -i input.mkv \
  -map 0:v -map 1:a ...
```

---

## Progress Tracking

### Save Progress

```
POST /api/progress/movie/123
{
  "progress": 3600,  // seconds
  "duration": 8880
}
```

Save every 10-30 seconds during playback.

### Resume Playback

```
GET /api/progress/movie/123
{
  "progress": 3600,
  "duration": 8880,
  "completed": false
}
```

Start player at `progress` seconds.

### Mark Complete

When progress > 90% of duration, mark as completed.

---

## Skip Intro/Credits (Future)

### Detection Methods

1. **Audio fingerprinting:** Match intro music across episodes
2. **Black frame detection:** Find scene boundaries
3. **Chapter markers:** Use if present in file
4. **Machine learning:** Train on known intros

### Storage

```sql
CREATE TABLE skip_segments (
    id INTEGER PRIMARY KEY,
    type TEXT NOT NULL,  -- 'movie', 'episode'
    media_id INTEGER NOT NULL,
    segment_type TEXT NOT NULL,  -- 'intro', 'credits', 'recap'
    start_time INTEGER NOT NULL,  -- seconds
    end_time INTEGER NOT NULL,
    UNIQUE(type, media_id, segment_type)
);
```

### UI

Show "Skip Intro" button when playback enters intro segment.

---

## Player Features

### Keyboard Shortcuts

| Key | Action |
|-----|--------|
| Space | Play/Pause |
| F | Fullscreen |
| M | Mute |
| ← | Seek back 10s |
| → | Seek forward 10s |
| ↑ | Volume up |
| ↓ | Volume down |
| C | Toggle captions |
| , | Previous frame (paused) |
| . | Next frame (paused) |

### Touch Gestures (Mobile)

- Tap: Show/hide controls
- Double-tap left: Seek back 10s
- Double-tap right: Seek forward 10s
- Swipe up/down (right side): Volume
- Swipe up/down (left side): Brightness

### Picture-in-Picture

```javascript
video.requestPictureInPicture();
```

### Chromecast

Use Google Cast SDK:
```javascript
const castContext = cast.framework.CastContext.getInstance();
castContext.setOptions({
    receiverApplicationId: chrome.cast.media.DEFAULT_MEDIA_RECEIVER_APP_ID,
    autoJoinPolicy: chrome.cast.AutoJoinPolicy.ORIGIN_SCOPED
});
```

### AirPlay

Native in Safari, use `x-webkit-airplay="allow"` attribute.

---

## Session Management

### Transcode Sessions

Track active transcodes:
```go
type TranscodeSession struct {
    ID        string
    MediaType string
    MediaID   int
    UserID    int
    Quality   string
    Started   time.Time
    Process   *exec.Cmd
}
```

### Cleanup

- Kill transcode process when client disconnects
- Delete temporary HLS segments
- Timeout inactive sessions (5 minutes)

### Limits

- Max concurrent transcodes (based on hardware)
- Per-user transcode limits
- Quality limits for free tier

---

## File Format Reference

### Containers

| Format | Extension | Direct Play Support |
|--------|-----------|---------------------|
| Matroska | .mkv | Most browsers via MSE |
| MP4 | .mp4 | Universal |
| WebM | .webm | Chrome, Firefox |
| AVI | .avi | Transcode required |

### Video Codecs

| Codec | Direct Play Support |
|-------|---------------------|
| H.264/AVC | Universal |
| H.265/HEVC | Safari, Edge, some Chrome |
| VP9 | Chrome, Firefox, Edge |
| AV1 | Chrome, Firefox (recent) |

### Audio Codecs

| Codec | Direct Play Support |
|-------|---------------------|
| AAC | Universal |
| MP3 | Universal |
| AC3 | Safari, some browsers |
| EAC3 | Safari |
| DTS | Transcode required |
| TrueHD | Transcode required |
| FLAC | Firefox, Chrome |

---

## Performance Optimization

### Caching

- Cache transcoded segments for repeat viewing
- Cache subtitle conversions
- Use CDN for static assets

### Preloading

- Start transcoding a few segments ahead
- Preload next episode

### Adaptive Bitrate

Adjust quality based on:
- Network conditions
- Buffer health
- Client capabilities
