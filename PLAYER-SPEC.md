# Player Specification

## Design System (Outpost Theme)

```css
/* Colors */
--bg-primary: #0a0a0a;
--text-primary: #F5E6C8;      /* Cream */
--text-secondary: rgba(245, 230, 200, 0.7);
--accent: #E8A849;            /* Amber */
--glass-bg: rgba(255, 255, 255, 0.06);
--glass-bg-hover: rgba(255, 255, 255, 0.1);
--glass-border: rgba(255, 255, 255, 0.1);
--success: #4ade80;
--danger: #ef4444;
```

All player UI elements use cream (#F5E6C8) as primary, amber (#E8A849) for active/progress states, and glass blur for panels.

---

## Interactions

### Click/Tap Behaviors
- **Click video area** → Toggle play/pause
- **Double-click left side** → Skip back 10s
- **Double-click right side** → Skip forward 10s
- **Double-click center** → Toggle fullscreen
- **Mouse move** → Show controls (hide after 3s idle)
- **Tap (mobile)** → Toggle controls visibility

### Keyboard Shortcuts
| Key | Action |
|-----|--------|
| Space | Play/pause |
| K | Play/pause |
| F | Toggle fullscreen |
| M | Toggle mute |
| ← | Skip back 10s |
| → | Skip forward 10s |
| J | Skip back 10s |
| L | Skip forward 10s |
| ↑ | Volume up 5% |
| ↓ | Volume down 5% |
| C | Toggle subtitles |
| S | Cycle subtitle track |
| A | Cycle audio track |
| 0-9 | Seek to 0%-90% |
| Home | Seek to start |
| End | Seek to end |
| Esc | Exit fullscreen |

---

## Controls Layout

```
┌─────────────────────────────────────────────────────────────┐
│                                                             │
│                                                             │
│                      [VIDEO AREA]                           │
│                   click to play/pause                       │
│                                                             │
│                                                             │
├─────────────────────────────────────────────────────────────┤
│  advancement button advancement button                                              │
│                                                             │
│ advancement button advancement button  advancement button advancement button advancement button  advancement button advancement button advancement button │
└─────────────────────────────────────────────────────────────┘
```

### Bottom Controls (left to right):

**Row 1: Progress**
```
advancement button advancement button advancement button advancement button advancement button advancement button advancement button  
```
- Elapsed time (left)
- Scrubber bar (fills center)
- Total duration (right)
- Hover shows timestamp preview
- Chapter markers as dots on bar (if available)

**Row 2: Controls**
```
advancement button advancement button  advancement button advancement button advancement button advancement button  advancement button advancement button advancement button advancement button advancement button  advancement button advancement button advancement button  ⛶
```

| Position | Element | Notes |
|----------|---------|-------|
| Left | ◀◀ | Skip back 10s |
| Left | ▶ / ⏸ | Play/pause toggle |
| Left | ▶▶ | Skip forward 10s |
| Left | 🔊━━━ | Volume slider (click icon to mute) |
| Center | Title | Movie/episode name |
| Right | Skip Intro | Only visible during intro (gold button) |
| Right | CC | Subtitle selector dropdown |
| Right | Audio | Audio track selector dropdown |
| Right | ⚙️ | Quality/playback settings |
| Right | ⛶ | Fullscreen toggle |

---

## Subtitle Styling

### Current Problem
- Subtitles look cheap/default
- Poor contrast
- Bad positioning

### Fixed Styling
```css
.subtitle-container {
  position: absolute;
  bottom: 80px; /* Above controls */
  left: 0;
  right: 0;
  text-align: center;
  pointer-events: none;
  z-index: 10;
}

.subtitle-text {
  display: inline-block;
  max-width: 80%;
  padding: 8px 16px;
  
  /* Text - Cream on dark */
  font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif;
  font-size: 2.5vw; /* Scales with screen */
  font-weight: 600;
  color: #F5E6C8; /* Cream */
  text-shadow: 
    2px 2px 4px rgba(0, 0, 0, 0.9),
    -1px -1px 2px rgba(0, 0, 0, 0.5),
    0 0 20px rgba(0, 0, 0, 0.8);
  
  /* Optional glass background for hard-to-read scenes */
  /* background: rgba(10, 10, 10, 0.7); */
  /* backdrop-filter: blur(8px); */
  /* border-radius: 4px; */
}

/* Responsive sizing */
@media (max-width: 768px) {
  .subtitle-text {
    font-size: 4vw;
  }
}

@media (min-width: 1920px) {
  .subtitle-text {
    font-size: 32px; /* Cap max size */
  }
}
```

### Subtitle Options (Settings Menu)
- **Size**: Small / Medium / Large
- **Background**: None / Glass / Solid
- **Position**: Bottom / Top
- **Color**: Cream / White / Yellow

---

## Skip Intro Button

### Behavior
- Appears when video enters intro timestamp range
- Auto-hides after 5 seconds if not used
- Clicking skips to end of intro

### Styling
```css
.skip-intro-btn {
  position: absolute;
  bottom: 100px;
  right: 40px;
  
  padding: 12px 24px;
  background: rgba(232, 168, 73, 0.2); /* Amber glass */
  backdrop-filter: blur(12px);
  border: 1px solid rgba(232, 168, 73, 0.4);
  border-radius: 8px;
  
  color: #F5E6C8; /* Cream */
  font-size: 14px;
  font-weight: 600;
  
  cursor: pointer;
  transition: all 0.2s;
}

.skip-intro-btn:hover {
  background: rgba(232, 168, 73, 0.4);
  border-color: #E8A849;
  transform: scale(1.02);
}
```

---

## Progress Bar / Scrubber

### Styling
```css
.progress-container {
  position: relative;
  height: 4px;
  background: rgba(245, 230, 200, 0.2); /* Cream at 20% */
  border-radius: 2px;
  cursor: pointer;
  transition: height 0.2s;
}

.progress-container:hover {
  height: 8px;
}

.progress-buffered {
  position: absolute;
  height: 100%;
  background: rgba(245, 230, 200, 0.4); /* Cream at 40% */
  border-radius: 2px;
}

.progress-played {
  position: absolute;
  height: 100%;
  background: #E8A849; /* Amber accent */
  border-radius: 2px;
}

.progress-handle {
  position: absolute;
  width: 16px;
  height: 16px;
  background: #E8A849; /* Amber */
  border: 2px solid #F5E6C8; /* Cream border */
  border-radius: 50%;
  top: 50%;
  transform: translate(-50%, -50%);
  opacity: 0;
  transition: opacity 0.2s;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.4);
}

.progress-container:hover .progress-handle {
  opacity: 1;
}

/* Chapter markers */
.chapter-marker {
  position: absolute;
  width: 4px;
  height: 4px;
  background: #F5E6C8; /* Cream */
  border-radius: 50%;
  top: 50%;
  transform: translate(-50%, -50%);
}
```

### Timestamp Preview (on hover)
```css
.timestamp-preview {
  position: absolute;
  bottom: 100%;
  margin-bottom: 8px;
  transform: translateX(-50%);
  padding: 6px 12px;
  background: rgba(10, 10, 10, 0.9);
  backdrop-filter: blur(8px);
  border: 1px solid rgba(245, 230, 200, 0.2);
  border-radius: 6px;
  color: #F5E6C8; /* Cream */
  font-size: 13px;
  font-weight: 500;
  white-space: nowrap;
  pointer-events: none;
}
```

---

## Settings Menu (⚙️)

### Dropdown Styling
```css
.settings-dropdown {
  position: absolute;
  bottom: 100%;
  right: 0;
  margin-bottom: 12px;
  min-width: 200px;
  
  background: rgba(10, 10, 10, 0.95);
  backdrop-filter: blur(20px);
  border: 1px solid rgba(245, 230, 200, 0.1);
  border-radius: 12px;
  padding: 8px 0;
  
  box-shadow: 0 8px 32px rgba(0, 0, 0, 0.5);
}

.settings-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 12px 16px;
  color: #F5E6C8;
  font-size: 14px;
  cursor: pointer;
  transition: background 0.2s;
}

.settings-item:hover {
  background: rgba(255, 255, 255, 0.06);
}

.settings-item-value {
  color: rgba(245, 230, 200, 0.6);
  font-size: 13px;
}

.settings-item.active .settings-item-value {
  color: #E8A849;
}

.settings-divider {
  height: 1px;
  background: rgba(245, 230, 200, 0.1);
  margin: 8px 0;
}

.settings-header {
  padding: 8px 16px;
  color: rgba(245, 230, 200, 0.5);
  font-size: 12px;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.5px;
}
```

### Options
- **Playback Speed**: 0.5x, 0.75x, 1x, 1.25x, 1.5x, 2x
- **Quality**: Auto, 1080p, 720p, 480p, 360p
- **Subtitle Settings** → Opens subtitle customization
- **Audio Sync**: -500ms to +500ms slider

---

## Track Selector Dropdowns (CC / Audio)

```css
.track-dropdown {
  position: absolute;
  bottom: 100%;
  right: 0;
  margin-bottom: 12px;
  min-width: 220px;
  max-height: 300px;
  overflow-y: auto;
  
  background: rgba(10, 10, 10, 0.95);
  backdrop-filter: blur(20px);
  border: 1px solid rgba(245, 230, 200, 0.1);
  border-radius: 12px;
  padding: 8px 0;
  
  box-shadow: 0 8px 32px rgba(0, 0, 0, 0.5);
}

.track-item {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px 16px;
  color: #F5E6C8;
  font-size: 14px;
  cursor: pointer;
  transition: background 0.2s;
}

.track-item:hover {
  background: rgba(255, 255, 255, 0.06);
}

.track-item.active {
  color: #E8A849;
}

.track-item-check {
  width: 16px;
  color: #E8A849;
  opacity: 0;
}

.track-item.active .track-item-check {
  opacity: 1;
}

.track-item-label {
  flex: 1;
}

.track-item-meta {
  color: rgba(245, 230, 200, 0.5);
  font-size: 12px;
}

/* Scrollbar */
.track-dropdown::-webkit-scrollbar {
  width: 6px;
}

.track-dropdown::-webkit-scrollbar-track {
  background: transparent;
}

.track-dropdown::-webkit-scrollbar-thumb {
  background: rgba(245, 230, 200, 0.2);
  border-radius: 3px;
}

.track-dropdown::-webkit-scrollbar-thumb:hover {
  background: rgba(245, 230, 200, 0.3);
}
```

### Track Display Format
- **Subtitles**: `English (SRT)`, `English (Forced)`, `Spanish`, `Off`
- **Audio**: `English (Atmos)`, `English (5.1)`, `English (Stereo)`, `Commentary`

---

## Volume Control

```css
.volume-container {
  display: flex;
  align-items: center;
  gap: 8px;
}

.volume-icon {
  cursor: pointer;
  width: 24px;
  color: #F5E6C8; /* Cream */
  transition: color 0.2s;
}

.volume-icon:hover {
  color: #E8A849; /* Amber on hover */
}

.volume-slider {
  width: 0;
  overflow: hidden;
  transition: width 0.2s;
}

.volume-container:hover .volume-slider {
  width: 80px;
}

.volume-bar {
  height: 4px;
  background: rgba(245, 230, 200, 0.2);
  border-radius: 2px;
}

.volume-level {
  height: 100%;
  background: #F5E6C8; /* Cream */
  border-radius: 2px;
}
```

---

## Player Chrome (overall container)

```css
.player-container {
  position: relative;
  width: 100%;
  height: 100%;
  background: #0a0a0a;
  overflow: hidden;
}

.player-controls {
  position: absolute;
  bottom: 0;
  left: 0;
  right: 0;
  padding: 20px 24px;
  background: linear-gradient(transparent, rgba(10, 10, 10, 0.95));
  opacity: 0;
  transition: opacity 0.3s;
}

.player-container:hover .player-controls,
.player-container.controls-visible .player-controls {
  opacity: 1;
}

.player-top-bar {
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  padding: 20px 24px;
  background: linear-gradient(rgba(10, 10, 10, 0.95), transparent);
  display: flex;
  align-items: center;
  gap: 16px;
  opacity: 0;
  transition: opacity 0.3s;
}

.player-container:hover .player-top-bar,
.player-container.controls-visible .player-top-bar {
  opacity: 1;
}

.player-back-btn {
  color: #F5E6C8; /* Cream */
  font-size: 24px;
  cursor: pointer;
  transition: color 0.2s;
  background: none;
  border: none;
  padding: 8px;
  border-radius: 50%;
}

.player-back-btn:hover {
  color: #E8A849; /* Amber */
  background: rgba(255, 255, 255, 0.06);
}

.player-title {
  color: #F5E6C8; /* Cream */
  font-size: 18px;
  font-weight: 500;
}

.player-subtitle {
  color: rgba(245, 230, 200, 0.7); /* Cream secondary */
  font-size: 14px;
}

/* Control buttons */
.player-btn {
  background: none;
  border: none;
  color: #F5E6C8;
  font-size: 20px;
  padding: 10px;
  cursor: pointer;
  border-radius: 50%;
  transition: all 0.2s;
  display: flex;
  align-items: center;
  justify-content: center;
}

.player-btn:hover {
  color: #E8A849;
  background: rgba(255, 255, 255, 0.06);
}

.player-btn-play {
  font-size: 28px;
  padding: 12px;
}

/* Time display */
.player-time {
  color: rgba(245, 230, 200, 0.7);
  font-size: 13px;
  font-weight: 500;
  font-variant-numeric: tabular-nums;
}

/* Control row layout */
.player-controls-row {
  display: flex;
  align-items: center;
  gap: 8px;
}

.player-controls-left {
  display: flex;
  align-items: center;
  gap: 4px;
}

.player-controls-center {
  flex: 1;
  text-align: center;
}

.player-controls-right {
  display: flex;
  align-items: center;
  gap: 4px;
}
```

---

## Loading State

```css
.player-loading {
  position: absolute;
  inset: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  background: rgba(10, 10, 10, 0.6);
}

.loading-spinner {
  width: 48px;
  height: 48px;
  border: 3px solid rgba(245, 230, 200, 0.2);
  border-top-color: #E8A849; /* Amber */
  border-radius: 50%;
  animation: spin 1s linear infinite;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}
```

---

## Mobile Considerations

- Larger touch targets (44px minimum)
- Tap to show/hide controls (not hover)
- Swipe left/right to seek
- Swipe up/down on right side for volume
- Swipe up/down on left side for brightness (if supported)
- No hover states

---

## Summary of Changes Needed

**Theming:**
- All text: Cream (#F5E6C8)
- Active/progress states: Amber (#E8A849)
- Backgrounds: #0a0a0a with glass blur
- Dropdowns: Glass morphism (blur + dark bg + subtle border)

**Functionality:**
1. ✅ Click video to pause/play
2. ✅ Double-click to seek/fullscreen
3. ✅ Keyboard shortcuts
4. ✅ Clean subtitle styling with text shadow
5. ✅ Subtitle customization options
6. ✅ Skip intro button (amber glass)
7. ✅ Improved progress bar with hover preview
8. ✅ Volume slider expands on hover
9. ✅ Settings menu for speed/quality (glass dropdown)
10. ✅ Track selectors for audio/subs (glass dropdown)
11. ✅ Gradient fade on controls background
12. ✅ Hide controls after 3s idle
13. ✅ Back button + title in top bar
14. ✅ All colors match Outpost theme
