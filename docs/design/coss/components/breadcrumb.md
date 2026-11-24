# Breadcrumb

Source: `apps/ui/registry/default/ui/breadcrumb.tsx` @ `b2336c45f2b7` · docs: `apps/ui/content/docs/components/breadcrumb.mdx`

> Displays the path to the current resource using a hierarchy of links.

Depends on: `@base-ui/react/merge-props`, `@base-ui/react/use-render`, `lucide-react`

Exports: `Breadcrumb`, `BreadcrumbList`, `BreadcrumbItem`, `BreadcrumbLink`, `BreadcrumbPage`, `BreadcrumbSeparator`, `BreadcrumbEllipsis`

## Behavior & keyboard

- `<nav aria-label=breadcrumb>` with an ordered list; links are normal tab stops; current page item has `aria-current=page`.

## How to read

Each part lists the exact class strings, then **resolved** values for the desktop environment (≥640px so all `sm:` rules apply; also md/lg/xl; fine pointer; hover-capable). Values show `light` / `dark` when they differ. *State* tables only list properties that change versus the part's default. *Mobile* lists properties that differ at 375px / coarse pointer. `src` is the class that wins the cascade. See `../README.md` for conventions.

## Parts

### `Breadcrumb` → `[data-slot=breadcrumb]` · `<nav>` (line 12)

Attributes: `aria-label=breadcrumb`

### `BreadcrumbList` → `[data-slot=breadcrumb-list]` · `<ol>` (line 20)

#### base

```txt
wrap-break-word flex flex-wrap items-center gap-1.5 text-muted-foreground text-sm sm:gap-2.5
```

**element — default**

Key metrics: display `flex` · gap `10px` · color `#686868` / `#818181` (muted-foreground) · font-size `14px` (text-sm) · line-height `20px`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| display | `flex` | `flex` | `flex` |
| flex-wrap | `wrap` | `wrap` | `flex-wrap` |
| align-items | `center` | `center` | `items-center` |
| gap | `10px` | `calc(var(--spacing) * 2.5)` | `sm:gap-2.5` |
| font-size | `14px` | `text-sm` | `text-sm` |
| line-height | `20px` | `var(--tw-leading, var(--text-sm--line-height))` | `text-sm` |
| overflow-wrap | `break-word` | `break-word` | `wrap-break-word` |
| color | `#686868` / `#818181` | `muted-foreground` | `text-muted-foreground` |

**element — mobile differences**: gap `6px`

### `BreadcrumbItem` → `[data-slot=breadcrumb-item]` · `<li>` (line 36)

#### base

```txt
inline-flex items-center gap-1.5
```

**element — default**

Key metrics: display `inline-flex` · gap `6px`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| display | `inline-flex` | `inline-flex` | `inline-flex` |
| align-items | `center` | `center` | `items-center` |
| gap | `6px` | `calc(var(--spacing) * 1.5)` | `gap-1.5` |

### `BreadcrumbLink` → `[data-slot=breadcrumb-link]` · `useRender <a>` (line 49)

#### base

```txt
transition-colors hover:text-foreground
```

**element — default**

Key metrics: transition-property `color, background-color, border-color, outline-color, text-decoration-color, fill, stroke, --tw-gradient-from, --tw-gradient-via, --tw-gradient-to` · transition-duration `150ms` · transition-timing-function `cubic-bezier(0.4, 0, 0.2, 1)`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| transition-property | `color, background-color, border-color, outline-color, text-decoration-color, fill, stroke, --tw-gradient-from, --tw-gradient-via, --tw-gradient-to` | `color, background-color, border-color, outline-color, text-decoration-color, fill, stroke, --tw-gradient-from,` | `transition-colors` |
| transition-timing-function | `cubic-bezier(0.4, 0, 0.2, 1)` | `var(--tw-ease, var(--default-transition-timing-function))` | `transition-colors` |
| transition-duration | `150ms` | `var(--tw-duration, var(--default-transition-duration))` | `transition-colors` |

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| hover | color | `#262626` / `#f5f5f5` | `hover:text-foreground` |

### `BreadcrumbPage` → `[data-slot=breadcrumb-page]` · `<span>` (line 66)

Attributes: `aria-current=page`

#### base

```txt
font-normal text-foreground
```

**element — default**

Key metrics: color `#262626` / `#f5f5f5` (foreground) · font-weight `400` (font-weight-normal)

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| font-weight | `400` | `font-weight-normal` | `font-normal` |
| color | `#262626` / `#f5f5f5` | `foreground` | `text-foreground` |

### `BreadcrumbSeparator` → `[data-slot=breadcrumb-separator]` · `<li>` (line 81)

Attributes: `aria-hidden=true`, `role=presentation`

#### base

```txt
opacity-80 [&>svg]:size-4
```

**element — default**

Key metrics: opacity `80%`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| opacity | `80%` | `80%` | `opacity-80` |

**`>svg` — default**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| width | `16px` | `calc(var(--spacing) * 4)` | `[&>svg]:size-4` |
| height | `16px` | `calc(var(--spacing) * 4)` | `[&>svg]:size-4` |

### `BreadcrumbEllipsis` → `[data-slot=breadcrumb-ellipsis]` · `<span>` (line 98)

Attributes: `aria-hidden=true`, `role=presentation`

### `BreadcrumbEllipsis` → (no data-slot) · `<MoreHorizontal>` (line 105)

#### base

```txt
size-4
```

**element — default**

Key metrics: height `16px` · width `16px`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| width | `16px` | `calc(var(--spacing) * 4)` | `size-4` |
| height | `16px` | `calc(var(--spacing) * 4)` | `size-4` |

### `BreadcrumbEllipsis` → (no data-slot) · `<span>` (line 106)

#### base

```txt
sr-only
```

**element — default**

Key metrics: height `1px` · width `1px` · padding `0` · border-width `0` · border-color `#00000014` / `#ffffff0f` (border)

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| border-width | `0` | `0` | `sr-only` |
| border-style | `solid` | `solid` | `(preflight) *` |
| border-color | `#00000014` / `#ffffff0f` | `border` | `(globals @layer base) * border-border` |
| position | `absolute` | `absolute` | `sr-only` |
| width | `1px` | `1px` | `sr-only` |
| height | `1px` | `1px` | `sr-only` |
| padding | `0` | `0` | `sr-only` |
| margin | `-1px` | `-1px` | `sr-only` |
| overflow | `hidden` | `hidden` | `sr-only` |
| clip-path | `inset(50%)` | `inset(50%)` | `sr-only` |
| white-space | `nowrap` | `nowrap` | `sr-only` |

