# Toolbar

Source: `apps/ui/registry/default/ui/toolbar.tsx` @ `b2336c45f2b7` · docs: `apps/ui/content/docs/components/toolbar.mdx` · Base UI: https://base-ui.com/react/components/toolbar#api-reference

> A container for grouping a set of buttons and controls.

Depends on: `@base-ui/react/toolbar`

Exports: `Toolbar`, `ToolbarButton`, `ToolbarLink`, `ToolbarInput`, `ToolbarGroup`, `ToolbarSeparator`

## Behavior & keyboard

- `role=toolbar` with roving focus: `ArrowLeft/Right` move between items, `Home`/`End`. _(inferred: Base UI primitive behavior; not restated in coss docs)_

## How to read

Each part lists the exact class strings, then **resolved** values for the desktop environment (≥640px so all `sm:` rules apply; also md/lg/xl; fine pointer; hover-capable). Values show `light` / `dark` when they differ. *State* tables only list properties that change versus the part's default. *Mobile* lists properties that differ at 375px / coarse pointer. `src` is the class that wins the cascade. See `../README.md` for conventions.

## Parts

### `Toolbar` → `[data-slot=toolbar]` · `<ToolbarPrimitive.Root>` (line 12)

#### base

```txt
relative flex gap-2 rounded-xl border bg-card not-dark:bg-clip-padding p-1 text-card-foreground
```

**element — default**

Key metrics: display `flex` · padding `4px` · gap `8px` · border-width `1px` · border-color `#00000014` / `#ffffff0f` (border) · border-radius `14px` · background-color `#ffffff` / `#191919` (card) · color `#262626` / `#f5f5f5` (card-foreground)

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| border-width | `1px` | `1px` | `border` |
| border-style | `solid` | `var(--tw-border-style)` | `border` |
| border-color | `#00000014` / `#ffffff0f` | `border` | `(globals @layer base) * border-border` |
| position | `relative` | `relative` | `relative` |
| display | `flex` | `flex` | `flex` |
| gap | `8px` | `calc(var(--spacing) * 2)` | `gap-2` |
| border-radius | `14px` | `calc(var(--radius) + 4px)` | `rounded-xl` |
| background-color | `#ffffff` / `#191919` | `card` | `bg-card` |
| padding | `4px` | `calc(var(--spacing) * 1)` | `p-1` |
| color | `#262626` / `#f5f5f5` | `card-foreground` | `text-card-foreground` |
| background-clip | `padding-box` / — | `padding-box` | `not-dark:bg-clip-padding` |

### `ToolbarButton` → `[data-slot=toolbar-button]` · `<ToolbarPrimitive.Button>` (line 28)

### `ToolbarLink` → `[data-slot=toolbar-link]` · `<ToolbarPrimitive.Link>` (line 41)

### `ToolbarInput` → `[data-slot=toolbar-input]` · `<ToolbarPrimitive.Input>` (line 54)

### `ToolbarGroup` → `[data-slot=toolbar-group]` · `<ToolbarPrimitive.Group>` (line 67)

#### base

```txt
flex items-center gap-1
```

**element — default**

Key metrics: display `flex` · gap `4px`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| display | `flex` | `flex` | `flex` |
| align-items | `center` | `center` | `items-center` |
| gap | `4px` | `calc(var(--spacing) * 1)` | `gap-1` |

### `ToolbarSeparator` → `[data-slot=toolbar-separator]` · `<ToolbarPrimitive.Separator>` (line 80)

#### base

```txt
shrink-0 bg-border data-[orientation=horizontal]:my-0.5 data-[orientation=vertical]:my-1.5 data-[orientation=horizontal]:h-px data-[orientation=horizontal]:w-full data-[orientation=vertical]:w-px data-[orientation=vertical]:not-[[class^='h-']]:not-[[class*='_h-']]:self-stretch
```

**element — default**

Key metrics: background-color `#00000014` / `#ffffff0f` (border)

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| flex-shrink | `0` | `0` | `shrink-0` |
| background-color | `#00000014` / `#ffffff0f` | `border` | `bg-border` |

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| [data-orientation=horizontal] | margin-block | `2px` | `data-[orientation=horizontal]:my-0.5` |
| [data-orientation=horizontal] | height | `1px` | `data-[orientation=horizontal]:h-px` |
| [data-orientation=horizontal] | width | `100%` | `data-[orientation=horizontal]:w-full` |
| [data-orientation=vertical] | margin-block | `6px` | `data-[orientation=vertical]:my-1.5` |
| [data-orientation=vertical] | width | `1px` | `data-[orientation=vertical]:w-px` |
| [data-orientation=vertical] | align-self | `stretch` | `data-[orientation=vertical]:not-[[class^='h-']]:not-[[class*='_h-']]:self-stretch` |

