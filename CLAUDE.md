# CLAUDE.md - Outpost Development Guidelines

## CRITICAL: Use Existing Components

Before creating ANY new UI element, check if a shared component already exists.

### Component Inventory

Location: `frontend/src/lib/components/`

| Component | Use For |
|-----------|---------|
| `PosterCard.svelte` | Any media poster with title, rating, badges |
| `GlassPanel.svelte` | Any frosted glass container |
| `IconButton.svelte` | Circular icon buttons (watchlist, watched, etc.) |
| `Dropdown.svelte` | Select menus, season pickers, quality selectors |
| `Badge.svelte` | Status badges, tags |
| `TypeBadge.svelte` | Movie/Series/Anime type indicators |
| `Toast.svelte` | Notifications (use via `toast` store) |
| `ScrollableRow.svelte` | Horizontal scroll with title and arrow buttons |
| `PersonCard.svelte` | Cast/crew member cards with photo |
| `InfoPanel.svelte` | Right sidebar with metadata |
| `RatingsRow.svelte` | TMDB/RT/Metacritic rating row |
| `ExternalLinks.svelte` | TMDB/IMDb/Trakt/Letterboxd icons |
| `MediaRow.svelte` | Horizontal row of PosterCards |
| `CastRow.svelte` | Horizontal cast member row |
| `EpisodeCard.svelte` | Episode cards with still image |
| `EpisodeGrid.svelte` | Grid of episode cards |
| `ContinueCard.svelte` | "Continue watching" cards |
| `DetailHero.svelte` | Full-width hero with backdrop |
| `SearchOverlay.svelte` | Search modal |
| `PersonModal.svelte` | Actor/crew detail modal |
| `TrailerModal.svelte` | YouTube trailer modal |
| `VideoPlayer.svelte` | Media playback |
| `Sidebar.svelte` | Main navigation sidebar |
| `TopBar.svelte` | Top navigation bar |

### NEVER Create These Inline

❌ **Do not create inline:**
- Scroll containers with left/right buttons → Use `ScrollableRow`
- Glass/frosted containers → Use `GlassPanel`
- Poster cards with hover effects → Use `PosterCard`
- Icon buttons → Use `IconButton`
- Cast/crew photo circles → Use `PersonCard`

✅ **Always import from:**
```svelte
import ComponentName from '$lib/components/ComponentName.svelte';
```

---

## Design System

### Colors (use Tailwind classes)

```
Background:     bg-bg-primary, bg-bg-card, bg-bg-elevated
Text:           text-text-primary, text-text-secondary, text-text-muted
Accent:         text-outpost-500, bg-outpost-500, border-outpost-500
Success:        text-green-400, bg-green-600
Warning:        text-yellow-400, text-amber-500
Error:          text-red-400, bg-red-600
Glass:          bg-white/10, bg-white/5, backdrop-blur-md
```

### Spacing

- Section padding: `px-6`
- Between sections: `space-y-6`
- Card gaps: `gap-4`
- Button gaps: `gap-2` or `gap-3`

### Border Radius

- Cards: `rounded-lg` or `rounded-xl`
- Pills/Buttons: `rounded-full`
- Small badges: `rounded`

### Standard Classes

```
liquid-card     - Glass card with border
liquid-tag      - Clickable pill tag
liquid-badge-sm - Small floating badge
liquid-btn-sm   - Small pill button
hover-lift      - Subtle lift on hover
scrollbar-thin  - Thin custom scrollbar
```

---

## Utilities

### Import from `$lib/utils`

```typescript
import {
  formatRuntime,
  parseGenres,
  parseCast,
  parseCrew,
  formatMoney,
  getLanguageName,
  getCountryName,
  getCountryFlag,
  getStatusColor,
  formatResolution,
  formatAudioChannels,
  getOfficialTrailer
} from '$lib/utils';
```

**NEVER** define these functions inline in page components.

### Import from `$lib/api`

```typescript
import {
  getImageUrl,
  getTmdbImageUrl,
  // ... other API functions
} from '$lib/api';
```

---

## Page Structure

### Detail Pages (movies/[id], tv/[id], etc.)

```svelte
<script lang="ts">
  // 1. Imports (components, api, utils, stores)
  // 2. State declarations
  // 3. Derived values
  // 4. Event handlers
  // 5. onMount
</script>

{#if loading}
  <!-- Loading spinner -->
{:else if error}
  <!-- Error message -->
{:else if data}
  <div class="space-y-6 -mt-22 -mx-6">
    <!-- Hero Section -->
    <section class="relative min-h-[500px]">
      <!-- Backdrop + gradients -->
      <!-- 3-column layout: Poster | Content | InfoPanel -->
    </section>

    <!-- Content Sections using ScrollableRow -->
    <ScrollableRow title="Cast">...</ScrollableRow>
    <ScrollableRow title="Crew">...</ScrollableRow>
    <ScrollableRow title="More Like This">...</ScrollableRow>
  </div>
{/if}
```

---

## Component Creation Rules

### When to Create a New Component

Create a new shared component when:
1. The same UI pattern appears 2+ times
2. The element has 50+ lines of markup
3. The element has its own state management

### New Component Checklist

1. Place in `frontend/src/lib/components/`
2. Use TypeScript: `<script lang="ts">`
3. Define Props interface
4. Use `$props()` rune for props
5. Use `$state()` for local state
6. Use `$derived()` for computed values
7. Export from `$lib/index.ts` if widely used

### Component Template

```svelte
<script lang="ts">
  interface Props {
    requiredProp: string;
    optionalProp?: number;
    class?: string;
  }

  let { requiredProp, optionalProp = 0, class: className = '' }: Props = $props();

  // Local state
  let isOpen = $state(false);

  // Derived
  const displayValue = $derived(requiredProp.toUpperCase());

  // Handlers
  function handleClick() {
    isOpen = !isOpen;
  }
</script>

<div class="component-base {className}">
  <!-- markup -->
</div>
```

---

## Common Patterns

### Scrollable Row with Items

```svelte
<ScrollableRow title="Section Title">
  {#each items as item}
    <ItemCard {item} />
  {/each}
</ScrollableRow>
```

### Icon Button Row

```svelte
<div class="flex items-center gap-2">
  <IconButton onclick={handleAction1} title="Action 1">
    <svg>...</svg>
  </IconButton>
  <IconButton onclick={handleAction2} active={isActive} title="Action 2">
    <svg>...</svg>
  </IconButton>
</div>
```

### Glass Info Card

```svelte
<GlassPanel class="p-4 space-y-2">
  <div class="flex justify-between">
    <span class="text-text-muted">Label</span>
    <span>{value}</span>
  </div>
</GlassPanel>
```

### Conditional Badge

```svelte
{#if condition}
  <Badge variant="success">Text</Badge>
{/if}
```

---

## File Organization

```
frontend/src/
├── lib/
│   ├── components/     # Shared UI components
│   ├── stores/         # Svelte stores (auth, toast)
│   ├── utils/          # Utility functions
│   │   ├── formatters.ts
│   │   ├── trailers.ts
│   │   └── index.ts    # Re-exports
│   ├── api.ts          # API client
│   └── index.ts        # Component re-exports
├── routes/
│   ├── +layout.svelte
│   ├── +page.svelte    # Home
│   ├── movies/[id]/    # Movie detail
│   ├── tv/[id]/        # TV detail
│   └── ...
└── app.d.ts
```

---

## Don't Forget

1. **Check components first** before writing any UI code
2. **Import utilities** instead of redefining them
3. **Use design tokens** (Tailwind classes) not raw colors
4. **Keep pages thin** — logic in components, formatting in utils
5. **Consistent spacing** — px-6 for sections, gap-4 for cards
