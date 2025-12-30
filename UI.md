# UI.md - Design Specification

---

## Design Philosophy

**Content First** - The UI is invisible; media is the star. Posters and artwork provide all color. Chrome is monochrome.

**Glass Aesthetic** - Translucent panels with backdrop blur. Layered depth. Subtle borders.

**Warm Beacon** - The only accent color is warm amber (#fbbf24) used sparingly for: logo window glow, progress bars, active states.

---

## Color Palette

```css
/* Chrome - monochrome only */
--black: #000000;
--white: #ffffff;
--gray-90: rgba(255, 255, 255, 0.9);
--gray-70: rgba(255, 255, 255, 0.7);
--gray-60: rgba(255, 255, 255, 0.6);
--gray-40: rgba(255, 255, 255, 0.4);
--gray-20: rgba(255, 255, 255, 0.2);
--gray-10: rgba(255, 255, 255, 0.1);
--gray-05: rgba(255, 255, 255, 0.05);

/* Accent - used sparingly */
--amber: #fbbf24;
--amber-light: #fef3c7;

/* Semantic */
--success: #22c55e;
--error: #ef4444;
--warning: #f59e0b;
```

---

## Typography

- **Headings:** System font stack, semibold/bold
- **Body:** System font stack, regular
- **Small:** 12px, gray-60
- **Monospace:** For technical info (codecs, bitrates)

No custom fonts. System fonts for performance.

```css
font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Oxygen, Ubuntu, sans-serif;
```

---

## Spacing Scale

```
4px  - xs
8px  - sm
12px - md
16px - base
24px - lg
32px - xl
48px - 2xl
64px - 3xl
```

---

## Components

### Glass Panel

```css
.glass-panel {
  background: rgba(0, 0, 0, 0.6);
  backdrop-filter: blur(24px);
  -webkit-backdrop-filter: blur(24px);
  border: 1px solid rgba(255, 255, 255, 0.1);
  border-radius: 24px;
}
```

### Sidebar (Desktop)

- Width: 64px fixed
- Position: Fixed left, full height
- Background: black/50 with backdrop blur
- Border: white/5 right border
- Z-index: 30

Structure:
```
┌────────┐
│  Logo  │  ← 40x40, rounded-xl, bg-white/10
├────────┤
│  Home  │  ← 40x40 icon button
│Discover│
│Library │
├────────┤
│        │  ← Spacer (flex-1)
├────────┤
│Settings│  ← Bottom
└────────┘
```

Icon buttons:
- Size: 40x40
- Active: bg-white/20, text-white
- Inactive: text-white/50, hover:text-white hover:bg-white/10
- Transition: 200ms

### Top Bar

- Height: 64px
- Position: Fixed top, starts after sidebar (left: 64px)
- Background: transparent (content scrolls behind)
- Z-index: 20

Structure:
```
┌──────────────────────────────────────────────────────────────┐
│                    [====Search Bar====]     🔔  👤           │
└──────────────────────────────────────────────────────────────┘
```

### Search Bar

- Max width: 600px
- Centered in top bar
- Background: white/10
- Border: white/10
- Border-radius: 9999px (full)
- Padding: 12px 20px
- Placeholder: "Search movies, shows, music..."

On focus:
- Expands search overlay
- Shows live suggestions

### Search Overlay

Full screen overlay with:
- Centered search input (larger)
- Results grouped: "In Your Library" (✓), "From TMDB" (+)
- Keyboard navigation
- ESC to close

### Poster Card

- Aspect ratio: 2:3 (150w x 225h base)
- Border-radius: 12px (xl)
- Overflow: hidden

States:
- Default: Poster image
- Hover: scale(1.05), play button overlay appears
- Shadow on hover

Structure:
```
┌─────────────┐
│             │
│   Poster    │
│    Image    │
│             │
│     ▶       │  ← Play button on hover
└─────────────┘
  Title
  Year
```

### Continue Watching Card

- Aspect ratio: 16:9
- Width: 288px
- Border-radius: 12px

Structure:
```
┌─────────────────────────────────┐
│                                 │
│       Backdrop Image            │
│                                 │
│  Title                    ▶     │  ← Play button on hover
│  S1 E4 · Episode Name           │
├─────────────────────────────────┤
│████████████░░░░░░░░░░░░░░░░░░░░│  ← Progress bar
└─────────────────────────────────┘
```

Play button behavior:
- Click play → direct play (opens player)
- Click card → opens detail page

### Episode Card

For TV detail page:

```
┌──────────────────────────────────────────────────────────────┐
│ 4  │  The Title of Episode                      45m    ✓/▶  │
│    │  Episode description text goes here...               │
└──────────────────────────────────────────────────────────────┘
```

- Left: Episode number (large)
- Middle: Title + description
- Right: Duration + watched checkmark or progress bar

### Stat Card (Settings)

For admin dashboard:

```
┌─────────────────────────────┐
│  📺  TV Shows               │
│                             │
│  847                        │  ← Large number
│  12,456 episodes            │  ← Subtitle
│  2.4 TB                     │  ← Size
└─────────────────────────────┘
```

- Background: white/5
- Border: white/10
- Icon: Colored (content type)
- Rounded: 2xl

### Buttons

**Primary (filled):**
```css
background: white;
color: black;
border-radius: 9999px;
padding: 12px 24px;
font-weight: 600;
```
Hover: opacity 90%

**Secondary (outline):**
```css
background: rgba(255, 255, 255, 0.1);
color: white;
border: 1px solid rgba(255, 255, 255, 0.2);
border-radius: 9999px;
```
Hover: bg white/20

**Icon Button:**
```css
width: 40px;
height: 40px;
background: rgba(255, 255, 255, 0.1);
border-radius: 9999px;
display: flex;
align-items: center;
justify-content: center;
```

---

## Page Layouts

### Home / Discover

1. **Hero Section** (if continue watching exists)
   - Full width
   - Height: 280px
   - Gradient background from poster dominant color
   - Poster/backdrop on right (faded)
   - Content on left: Title, episode info, progress bar
   - Buttons: Resume, Details

2. **Continue Watching Row**
   - Horizontal scroll
   - 288px cards with 16px gap

3. **Recently Added**
   - 6-column grid of poster cards
   - Row title with "See All" link

4. **Trending / Popular** (from TMDB)
   - Horizontal scroll
   - Items not in library show "+" to request

### Detail Page

Full page (not modal) with:

1. **Hero Section** (70vh height)
   - Background: Gradient from poster color
   - Large poster centered (faded)
   - Gradient overlays: bottom 50% black fade, left side fade
   - Back button: top-left, icon button

2. **Content Overlay**
   - Position: absolute, bottom of hero
   - Title: 5xl, bold
   - Metadata row: Rating, year, runtime, quality badge
   - Description: max 3 lines
   - Buttons: Play/Resume, Add to List, Download

3. **Episodes Section** (TV only)
   - Season selector dropdown
   - Episode list with cards
   - Progress indicators

4. **More Like This**
   - 4-column grid of related posters

### Library Page

- Page title
- Filter/sort controls
- Grid of poster cards (6 columns on desktop)
- Pagination or infinite scroll

### Settings Page

Tabbed interface:

**Tab 1: Overview (Dashboard)**
- 6 stat cards in 2x3 grid (Movies, TV, Anime, Music, Books, Audiobooks)
- Storage breakdown bar
- Recent activity feed

**Tab 2: Libraries**
- List of libraries grouped by type
- Scan buttons
- Add library button

**Tab 3: Downloads**
- Active download queue
- Progress bars with speed/ETA

**Tab 4: Connections**
- Download clients list
- Indexers list
- Test buttons

**Tab 5: Users**
- User list with roles
- Add user button

---

## Player

Full screen with auto-hiding controls (hide after 3s of no mouse movement).

### Top Bar
- Back button (left)
- Title + episode info (center)
- Background: gradient from top

### Center
- Play/pause button (only visible when paused)
- Large, 80x80

### Bottom Bar

**Progress Bar:**
- Full width
- Click/drag to seek
- Scrub preview thumbnail (future)
- Time: elapsed / -remaining

**Controls Row:**
```
[◀◀] [⏸] [▶▶]  🔊━━━━    |    📊 🎤 👥 🔊 💬 ⚙️ ⛶
 Skip  Play Skip  Volume       Stats Audio Cast Audio Sub Settings Full
 -10s      +10s                       Sync      Track
```

### Right-Side Panels

Slide in from right when icon clicked:

**Stats Panel:**
```
Video
  Codec: HEVC
  Resolution: 3840×2160
  Bitrate: 15.2 Mbps
  HDR: Dolby Vision

Audio
  Codec: TrueHD Atmos
  Channels: 7.1
  Bitrate: 4.2 Mbps

Playback
  Method: Direct Play
  Buffer: 30s
```

**Audio Sync Panel:**
```
Audio Delay
[-] ━━━━━●━━━━━ [+]
      -50ms

[Reset to 0]
```

**Cast Panel:**
```
Cast & Crew

[Photo] Actor Name
        Character Name

[Photo] Actor Name
        Character Name
        
... scrollable list
```

**Audio Track Panel:**
```
Audio Tracks

● English (TrueHD Atmos 7.1)
○ English (AC3 5.1)
○ Commentary (AAC 2.0)
```

**Subtitle Panel:**
```
Subtitles

● None
○ English (SRT)
○ English SDH (SRT)
○ Spanish (ASS)
```

### Overlays

**Skip Intro Button:**
- Appears at 5-15% progress (configurable)
- Position: bottom-right above controls
- White button with "Skip Intro" text
- Clicking skips to 16% (configurable end point)

**Up Next Card:**
- Appears at >90% progress
- Bottom-right
- Shows next episode poster, title
- "Play Next" button
- Auto-advances after countdown

---

## TV App (10-foot UI)

Separate experience for remote/gamepad navigation.

### Key Differences
- Everything 2-3x larger
- Horizontal scroll rows (no grids)
- No sidebar - top navigation
- Focus rings (2px white border) for D-pad navigation
- Larger hero (85vh)

### Top Navigation
```
[Logo]    Home   Movies   TV Shows   Library        🔔  👤
```

### Light Admin
- Bell icon: Notification panel with approve/deny
- Context menu on items: Mark watched, Delete
- No full settings access

---

## Icons

Use Lucide icons exclusively:

**Navigation:**
- Home, Search, Compass (Discover), Folder (Library), Settings

**Player:**
- Play, Pause, RotateCcw (skip back), RotateCw (skip forward)
- Volume2, VolumeX, Maximize, Minimize

**Actions:**
- Plus, Check, X, ChevronRight, ChevronLeft, ChevronDown
- Download, Clock, Bell

**Media Types:**
- Film, Tv, Music, BookOpen, Headphones, MonitorPlay

**Player Panels:**
- BarChart3 (stats), AudioLines (sync), Users (cast)
- Volume2 (audio tracks), Captions (subtitles)

---

## Animations

- **All transitions:** 300ms ease
- **Hover scale:** 1.05
- **Control fade:** opacity 0→1
- **Panel slide:** translateX + opacity
- **Focus ring:** 2px solid white, offset 2px

---

## Responsive Breakpoints

```css
sm: 640px   /* Mobile landscape */
md: 768px   /* Tablet */
lg: 1024px  /* Desktop */
xl: 1280px  /* Large desktop */
2xl: 1536px /* Ultra-wide */
```

Poster grid columns:
- sm: 2
- md: 3
- lg: 4
- xl: 6

---

## Accessibility

- Focus visible outlines
- Keyboard navigation
- ARIA labels on icon buttons
- Sufficient color contrast (white on dark)
- Reduced motion option (prefers-reduced-motion)
