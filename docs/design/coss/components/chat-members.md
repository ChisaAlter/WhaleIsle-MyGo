# chat-members

Source: `apps/ui/registry/default/chat/chat-members.tsx` @ `b2336c45f2b7` · docs: `apps/ui/content/docs/(root)/chat.mdx`

Depends on: —

Exports: `ChatMemberList`, `ChatMemberSection`

## Behavior & keyboard

_No interactive keyboard behavior beyond native focus order (inferred: presentational component)._

## How to read

Each part lists the exact class strings, then **resolved** values for the desktop environment (≥640px so all `sm:` rules apply; also md/lg/xl; fine pointer; hover-capable). Values show `light` / `dark` when they differ. *State* tables only list properties that change versus the part's default. *Mobile* lists properties that differ at 375px / coarse pointer. `src` is the class that wins the cascade. See `../README.md` for conventions.

## Parts

### `ChatMemberList` → `[data-slot=chat-member-list]` · `<ul>` (line 25)

#### base

```txt
m-0 flex list-none flex-col gap-2 p-0
```

**element — default**

Key metrics: display `flex` · padding `0px` · gap `8px`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| margin | `0px` | `calc(var(--spacing) * 0)` | `m-0` |
| display | `flex` | `flex` | `flex` |
| list-style-type | `none` | `none` | `list-none` |
| flex-direction | `column` | `column` | `flex-col` |
| gap | `8px` | `calc(var(--spacing) * 2)` | `gap-2` |
| padding | `0px` | `calc(var(--spacing) * 0)` | `p-0` |

### `ChatMemberList` → (no data-slot) · `<li>` (line 31)

#### base

```txt
flex min-w-0 items-center gap-3
```

**element — default**

Key metrics: display `flex` · min-width `0px` · gap `12px`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| display | `flex` | `flex` | `flex` |
| min-width | `0px` | `calc(var(--spacing) * 0)` | `min-w-0` |
| align-items | `center` | `center` | `items-center` |
| gap | `12px` | `calc(var(--spacing) * 3)` | `gap-3` |

### `ChatMemberList` → (no data-slot) · `<div>` (line 33)

#### base

```txt
flex min-w-0 flex-1 flex-wrap items-center gap-x-1.5 gap-y-1
```

**element — default**

Key metrics: display `flex` · min-width `0px` · column-gap `6px` · row-gap `4px`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| display | `flex` | `flex` | `flex` |
| min-width | `0px` | `calc(var(--spacing) * 0)` | `min-w-0` |
| flex | `1` | `1` | `flex-1` |
| flex-wrap | `wrap` | `wrap` | `flex-wrap` |
| align-items | `center` | `center` | `items-center` |
| column-gap | `6px` | `calc(var(--spacing) * 1.5)` | `gap-x-1.5` |
| row-gap | `4px` | `calc(var(--spacing) * 1)` | `gap-y-1` |

### `ChatMemberList` → (no data-slot) · `<span>` (line 34)

#### base

```txt
truncate text-sm
```

**element — default**

Key metrics: font-size `14px` (text-sm) · line-height `20px`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| overflow | `hidden` | `hidden` | `truncate` |
| text-overflow | `ellipsis` | `ellipsis` | `truncate` |
| white-space | `nowrap` | `nowrap` | `truncate` |
| font-size | `14px` | `text-sm` | `text-sm` |
| line-height | `20px` | `var(--tw-leading, var(--text-sm--line-height))` | `text-sm` |

#### when `member.dimmed`

```txt
text-muted-foreground
```

**element — default (changes vs base)**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| color | `#686868` / `#818181` | `muted-foreground` | `text-muted-foreground` |

#### when `!(member.dimmed)`

```txt
font-medium
```

**element — default (changes vs base)**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| font-weight | `500` | `font-weight-medium` | `font-medium` |

### `ChatMemberSection` → (no data-slot) · `<section>` (line 64)

Attributes: `aria-label={label}`

#### base

```txt
flex flex-col gap-4
```

**element — default**

Key metrics: display `flex` · gap `16px`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| display | `flex` | `flex` | `flex` |
| flex-direction | `column` | `column` | `flex-col` |
| gap | `16px` | `calc(var(--spacing) * 4)` | `gap-4` |

### `ChatMemberSection` → (no data-slot) · `<h3>` (line 66)

#### base

```txt
m-0 flex items-center gap-2 font-medium text-muted-foreground text-xs
```

**element — default**

Key metrics: display `flex` · gap `8px` · color `#686868` / `#818181` (muted-foreground) · font-size `12px` (text-xs) · line-height `16px` · font-weight `500` (font-weight-medium)

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| margin | `0px` | `calc(var(--spacing) * 0)` | `m-0` |
| display | `flex` | `flex` | `flex` |
| align-items | `center` | `center` | `items-center` |
| gap | `8px` | `calc(var(--spacing) * 2)` | `gap-2` |
| font-size | `12px` | `text-xs` | `text-xs` |
| line-height | `16px` | `var(--tw-leading, var(--text-xs--line-height))` | `text-xs` |
| font-weight | `500` | `font-weight-medium` | `font-medium` |
| color | `#686868` / `#818181` | `muted-foreground` | `text-muted-foreground` |

