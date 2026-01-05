# CLAUDE.md - Outpost Development Guidelines

## Project Overview
Outpost is a unified self-hosted media server for movies and TV shows.
- **Frontend**: SvelteKit + TailwindCSS v4
- **Backend**: Go
- **Database**: SQLite with WAL mode

---

## BEFORE CREATING ANY COMPONENT

**STOP. Search first.**

```
/src/lib/components/ui/        ← Base primitives (GlassPanel, Select, Badge, ProgressBar)
/src/lib/components/media/     ← Media cards (MediaCard, EpisodeCard, CastCard, FileCard)
/src/lib/components/layout/    ← App shell (Topbar, StatusBar)
/src/lib/components/modals/    ← Dialogs (RequestModal, TrailerModal, PersonModal)
/src/lib/components/containers/← Layout (ScrollSection)
/src/lib/components/detail/    ← Detail page parts (ActionButtons, DetailInfoPanel)
```

If something similar exists, **extend it with props** — do not duplicate.

---

## Design System

### Color Tokens (USE THESE)
```css
/* Primary accent */
--color-cream: #F5E6C8        /* Logo, headings, primary text */
--color-amber: #E8A849        /* Hover states, CTAs, highlights */

/* Backgrounds */
--color-bg-primary: #0a0a0a   /* Main background */
--color-bg-card: #0f0f0f      /* Card backgrounds */
--color-bg-elevated: #141414  /* Elevated surfaces */
--color-glass: rgba(255, 255, 255, 0.06)  /* Glass panels */

/* Text */
--color-text-primary: #F5E6C8
--color-text-secondary: rgba(245, 230, 200, 0.7)
--color-text-muted: rgba(245, 230, 200, 0.5)

/* Borders */
--color-border-subtle: rgba(255, 255, 255, 0.1)
```

### ❌ NEVER USE THESE
| Wrong | Correct |
|-------|---------|
| `text-white` | `text-text-primary` |
| `bg-white/5`, `bg-white/8`, `bg-white/10` | `bg-glass` |
| `border-white/10`, `border-white/12` | `border-border-subtle` |
| `style="background: rgba(255,255,255,0.06)"` | `class="bg-glass"` |
| `style="color: #F5E6C8"` | `class="text-cream"` |

### Utility Classes (from app.css)
```
/* Buttons */
liquid-btn           Primary button
liquid-btn-sm        Small button  
liquid-btn-icon      Icon button (square)
btn-icon-circle      Circular icon button (topbar style)
btn-icon-circle-sm   Small circular icon
btn-icon-glass-lg    Large glass icon button (hero actions)

/* Inputs */
liquid-input         Text input
form-select          Dropdown select
form-select-sm       Small dropdown
form-checkbox        Checkbox
form-toggle          Toggle switch
form-input           Alternative input style

/* Containers */
glass                Glass panel background
liquid-card          Card with border
liquid-panel         Larger panel surface
liquid-dropdown      Dropdown menu container

/* Badges */
liquid-badge         Standard badge
liquid-badge-sm      Small badge
liquid-tag           Pill-shaped tag
```

---

## File Size Limits

| Type | Max Lines | Action if exceeded |
|------|-----------|-------------------|
| Component | 300 | Extract sub-components |
| Route page | 500 | Extract to components |
| Go file | 500 | Split by responsibility |

---

## Svelte 5 Patterns

### Props
```svelte
<script lang="ts">
  interface Props {
    title: string;
    variant?: 'default' | 'outlined';
    onclick?: () => void;
  }
  let { title, variant = 'default', onclick }: Props = $props();
</script>
```

### State & Derived
```svelte
let count = $state(0);
let items = $state<Item[]>([]);
let filtered = $derived(items.filter(i => i.active));
```

### Children (Snippets)
```svelte
<script lang="ts">
  import type { Snippet } from 'svelte';
  interface Props { children: Snippet; }
  let { children }: Props = $props();
</script>
<div>{@render children()}</div>
```

---

## Known Duplicates (DO NOT USE)

| Deprecated | Use Instead |
|------------|-------------|
| `/components/TopBar.svelte` | `/components/layout/Topbar.svelte` |
| `/components/GlassPanel.svelte` | `/components/ui/GlassPanel.svelte` |
| `/components/EpisodeCard.svelte` | `/components/media/EpisodeCard.svelte` |

---

## Backend Patterns

### Parser (single source of truth)
```go
import "github.com/outpost/outpost/internal/parser"
parsed := parser.Parse(releaseName)
```

### Quality Scoring
```go
import "github.com/outpost/outpost/internal/quality"
tier := quality.ComputeQualityTier(parsed)
```

### Database
- Always parameterized queries
- Check errors immediately
- Use transactions for multi-step ops
- `defer rows.Close()`

---

## Git Commit Rules

- **NO co-authoring** — Never add Co-Authored-By lines
- **NO AI attribution** — Never mention Claude, Anthropic, or AI in commits
- **NO markdown files** — Never create .md files except README.md
- Keep commit messages clean and focused on the changes
