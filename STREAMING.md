# STREAMING.md - Video/Audio Playback

---

## Overview

Outpost prioritizes **direct play** - serving the original file without modification. Transcoding is a fallback for incompatible formats.

---

## Direct Play

### When It Works
- Container: MP4, MKV, WebM
- Video: H.264, H.265/HEVC (browser support varies)
- Audio: AAC, MP3, Opus

### Implementation
1. Client requests `/api/stream/movie/123`
2. Server checks file format
3. If compatible → serve file with range request support
4. Client uses native `<video>` element

### Range Requests
HTTP range requests for:
- Seeking without downloading entire file
- Resume interrupted downloads
- Progressive playback

```
GET /api/stream/movie/123
Range: bytes=1000000-2000000

HTTP/1.1 206 Partial Content
Content-Range: bytes 1000000-2000000/50000000
Content-Length: 1000001
```

---

## Transcoding

### When Needed
- Incompatible video codec (MPEG-2, VC-1, AV1 on older browsers)
- Incompatible audio codec (DTS, TrueHD, Dolby Atmos)
- Client requests lower quality
- Audio-only extraction

### FFmpeg Pipeline

**Video transcode to H.264:**
```bash
ffmpeg -i input.mkv \
  -c:v libx264 \
  -preset fast \
  -crf 23 \
  -c:a aac \
  -b:a 192k \
  -movflags +faststart \
  -f mp4 \
  output.mp4
```

**Audio-only transcode (for incompatible audio):**
```bash
ffmpeg -i input.mkv \
  -c:v copy \
  -c:a aac \
  -b:a 192k \
  output.mkv
```

**Start from timestamp (for seeking):**
```bash
ffmpeg -ss 01:23:45 -i input.mkv ...
```

### Transcode Quality Presets

| Preset | Resolution | Video Bitrate | Audio |
|--------|------------|---------------|-------|
| Auto | Original | Original | AAC 192k |
| 4K | 3840x2160 | 20 Mbps | AAC 384k |
| 1080p | 1920x1080 | 8 Mbps | AAC 192k |
| 720p | 1280x720 | 4 Mbps | AAC 128k |
| 480p | 854x480 | 2 Mbps | AAC 128k |

---

## Hardware Acceleration (Future)

### NVIDIA NVENC
```bash
ffmpeg -hwaccel cuda -hwaccel_output_format cuda \
  -i input.mkv \
  -c:v h264_nvenc \
  -preset p4 \
  -b:v 8M \
  output.mp4
```

### Intel Quick Sync
```bash
ffmpeg -hwaccel qsv -hwaccel_output_format qsv \
  -i input.mkv \
  -c:v h264_qsv \
  -preset medium \
  -b:v 8M \
  output.mp4
```

### Detection
1. Check for GPU presence
2. Test encode capabilities
3. Store in settings
4. Fall back to CPU if unavailable

---

## Progress Tracking

### Save Progress
- Every 10 seconds during playback
- On pause
- On seek
- On close

```json
PUT /api/progress/movie/123
{
  "position": 3600,
  "duration": 7200,
  "completed": false
}
```

### Mark Complete
- When position > 90% of duration
- OR user manually marks watched

### Resume Logic
1. User opens media
2. Check for saved progress
3. If exists and < 90%: "Resume from X:XX?" or auto-resume
4. If > 90%: Start from beginning

---

## Subtitles

### Embedded Subtitles
Extract from MKV using FFmpeg:
```bash
ffmpeg -i input.mkv -map 0:s:0 -f webvtt subtitle.vtt
```

### External Subtitles
Detect files matching:
- `Movie.Name.2024.srt`
- `Movie.Name.2024.en.srt`
- `Movie.Name.2024.eng.srt`

### Supported Formats
- SRT → Convert to WebVTT
- ASS/SSA → Convert to WebVTT (lose styling)
- WebVTT → Use directly

### Subtitle Track Selection
- Store preference per user
- Default: None or first English
- Remember last selection per show

---

## Audio Tracks

### Multi-Audio Files
Detect all audio streams:
```bash
ffprobe -v error -select_streams a \
  -show_entries stream=index,codec_name,channels \
  -of json input.mkv
```

### Track Selection
- User selects in player
- Server transcodes selected track only
- Store preference per user

---

## Seeking

### Current Implementation
For transcoded content:
1. User seeks to new position
2. Stop current stream
3. Start new FFmpeg process from seek point
4. Player reloads from new URL with `?start=` param

### Future: True Seeking
Use HLS or DASH for segmented delivery:
- Pre-segment or on-demand segments
- Player switches segments without reload

---

## Music Playback

### Direct Play
- MP3, FLAC, AAC, Opus
- Use native `<audio>` element

### Transcode If Needed
```bash
ffmpeg -i input.flac \
  -c:a libmp3lame -q:a 2 \
  output.mp3
```

### Gapless Playback
- Preload next track
- Use Web Audio API for seamless transition

---

## Audiobook Playback

### Chapter Support
Extract chapter metadata:
```bash
ffprobe -v error -show_chapters \
  -print_format json input.m4b
```

### Player Features
- Chapter navigation
- Sleep timer (15m, 30m, 1h, custom)
- Playback speed (0.5x - 3x)
- Remember speed preference

---

## Book Reading

### EPUB
- Use epub.js or similar library
- Render in browser
- Track page/percentage progress

### PDF
- Use PDF.js
- Track page number

### CBZ/CBR (Comics)
- Extract images from archive
- Serve as image sequence
- Page/scroll mode toggle
- RTL reading for manga

---

## Chromecast (Future)

1. Detect Chromecast on network
2. User clicks cast button
3. Send media URL to Chromecast
4. Monitor playback from phone/browser
5. Sync progress back to server

### Requirements
- Media must be Chromecast-compatible (MP4/H.264/AAC)
- OR transcode stream URL

---

## AirPlay (Future)

Similar to Chromecast but for Apple devices.
- Detect AirPlay receivers
- Send compatible stream URL
- Control from iOS/macOS
