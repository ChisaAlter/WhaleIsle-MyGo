# Pagination

Source: `apps/ui/registry/default/ui/pagination.tsx` @ `b2336c45f2b7` · docs: `apps/ui/content/docs/components/pagination.mdx`

> A pagination with page navigation, next and previous links.

Depends on: `@base-ui/react/merge-props`, `@base-ui/react/use-render`, `lucide-react`, `@/ui/button`

Exports: `Pagination`, `PaginationContent`, `PaginationItem`, `PaginationLink`, `PaginationPrevious`, `PaginationNext`, `PaginationEllipsis`

## Behavior & keyboard

- `<nav>` with links/buttons; current page has `aria-current=page`; normal tab order.

## How to read

Each part lists the exact class strings, then **resolved** values for the desktop environment (≥640px so all `sm:` rules apply; also md/lg/xl; fine pointer; hover-capable). Values show `light` / `dark` when they differ. *State* tables only list properties that change versus the part's default. *Mobile* lists properties that differ at 375px / coarse pointer. `src` is the class that wins the cascade. See `../README.md` for conventions.

## Parts

### `Pagination` → `[data-slot=pagination]` · `<nav>` (line 19)

Attributes: `aria-label=pagination`

#### base

```txt
mx-auto flex w-full justify-center
```

**element — default**

Key metrics: display `flex` · width `100%`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| margin-inline | `auto` | `auto` | `mx-auto` |
| display | `flex` | `flex` | `flex` |
| width | `100%` | `100%` | `w-full` |
| justify-content | `center` | `center` | `justify-center` |

### `PaginationContent` → `[data-slot=pagination-content]` · `<ul>` (line 33)

#### base

```txt
flex flex-row items-center gap-1
```

**element — default**

Key metrics: display `flex` · gap `4px`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| display | `flex` | `flex` | `flex` |
| flex-direction | `row` | `row` | `flex-row` |
| align-items | `center` | `center` | `items-center` |
| gap | `4px` | `calc(var(--spacing) * 1)` | `gap-1` |

### `PaginationItem` → `[data-slot=pagination-item]` · `<li>` (line 44)

### `PaginationLink` → `[data-slot=pagination-link]` · `useRender <a>` (line 59)

Attributes: `aria-current={isActive ? ("page" as const) : undefined}`, `data-active={isActive}`

Prop defaults: `size="icon"`

Dynamic className inputs (not statically resolvable): `buttonVariants({ size, variant: isActive ? "outline" : "ghost", })` when `!(render)`

### `PaginationPrevious` → (no data-slot) · `<PaginationLink>` (line 86)

Attributes: `aria-label=Go to previous page`, `size=default`

#### base

```txt
max-sm:aspect-square max-sm:p-0
```

**element — mobile differences**: aspect-ratio `1 / 1` · padding `0px`

### `PaginationPrevious` → (no data-slot) · `<ChevronLeftIcon>` (line 92)

#### base

```txt
sm:-ms-1
```

**element — default**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| margin-inline-start | `-4px` | `calc(var(--spacing) * -1)` | `sm:-ms-1` |

**element — mobile differences**: margin-inline-start `(unset)`

### `PaginationPrevious` → (no data-slot) · `<span>` (line 93)

#### base

```txt
max-sm:hidden
```

**element — mobile differences**: display `none`

### `PaginationNext` → (no data-slot) · `<PaginationLink>` (line 103)

Attributes: `aria-label=Go to next page`, `size=default`

#### base

```txt
max-sm:aspect-square max-sm:p-0
```

**element — mobile differences**: aspect-ratio `1 / 1` · padding `0px`

### `PaginationNext` → (no data-slot) · `<span>` (line 109)

#### base

```txt
max-sm:hidden
```

**element — mobile differences**: display `none`

### `PaginationNext` → (no data-slot) · `<ChevronRightIcon>` (line 110)

#### base

```txt
sm:-me-1
```

**element — default**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| margin-inline-end | `-4px` | `calc(var(--spacing) * -1)` | `sm:-me-1` |

**element — mobile differences**: margin-inline-end `(unset)`

### `PaginationEllipsis` → `[data-slot=pagination-ellipsis]` · `<span>` (line 120)

Attributes: `aria-hidden=true`

#### base

```txt
flex min-w-7 justify-center
```

**element — default**

Key metrics: display `flex` · min-width `28px`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| display | `flex` | `flex` | `flex` |
| min-width | `28px` | `calc(var(--spacing) * 7)` | `min-w-7` |
| justify-content | `center` | `center` | `justify-center` |

### `PaginationEllipsis` → (no data-slot) · `<MoreHorizontalIcon>` (line 126)

#### base

```txt
size-5 sm:size-4
```

**element — default**

Key metrics: height `16px` · width `16px`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| width | `16px` | `calc(var(--spacing) * 4)` | `sm:size-4` |
| height | `16px` | `calc(var(--spacing) * 4)` | `sm:size-4` |

**element — mobile differences**: width `20px` · height `20px`

### `PaginationEllipsis` → (no data-slot) · `<span>` (line 127)

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

