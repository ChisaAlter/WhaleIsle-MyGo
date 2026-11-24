# sidebar

Source: `apps/ui/registry/default/ui/sidebar.tsx` @ `b2336c45f2b7`

Depends on: `@base-ui/react/merge-props`, `@base-ui/react/use-render`, `lucide-react`, `@/ui/button`, `@/ui/input`, `@/ui/scroll-area`, `@/ui/separator`, `@/ui/sheet`, `@/ui/skeleton`, `@/ui/tooltip`

Exports: `useSidebar`, `SidebarProvider`, `Sidebar`, `SidebarTrigger`, `SidebarRail`, `SidebarInset`, `SidebarInput`, `SidebarHeader`, `SidebarFooter`, `SidebarSeparator`, `SidebarContent`, `SidebarGroup`, `SidebarGroupLabel`, `SidebarGroupAction`, `SidebarGroupContent`, `SidebarMenu`, `SidebarMenuItem`, `SidebarMenuButton`, `SidebarMenuAction`, `SidebarMenuBadge`, `SidebarMenuSkeleton`, `SidebarMenuSub`, `SidebarMenuSubItem`, `SidebarMenuSubButton`

Module constants: `SIDEBAR_COOKIE_NAME = sidebar_state`, `SIDEBAR_WIDTH = 16rem`, `SIDEBAR_WIDTH_MOBILE = 18rem`, `SIDEBAR_WIDTH_ICON = 3rem`, `SIDEBAR_KEYBOARD_SHORTCUT = b`

## Behavior & keyboard

- App sidebar with collapsible states (`offcanvas` / `icon` / `none`), mobile renders as Sheet. Keyboard shortcut `⌘B`/`Ctrl+B` toggles (constant `SIDEBAR_KEYBOARD_SHORTCUT`); state persisted in cookie. _(from source constants)_

## How to read

Each part lists the exact class strings, then **resolved** values for the desktop environment (≥640px so all `sm:` rules apply; also md/lg/xl; fine pointer; hover-capable). Values show `light` / `dark` when they differ. *State* tables only list properties that change versus the part's default. *Mobile* lists properties that differ at 375px / coarse pointer. `src` is the class that wins the cascade. See `../README.md` for conventions.

## Variant definitions (cva)

### `sidebarMenuButtonVariants` (line 35)

Defaults: {"size":"default","variant":"default"}

```txt
base: peer/menu-button flex w-full items-center gap-2 overflow-hidden rounded-lg p-2 text-left text-sm outline-hidden ring-sidebar-ring transition-[width,height,padding] hover:bg-sidebar-accent hover:text-sidebar-accent-foreground focus-visible:ring-2 active:bg-sidebar-accent active:text-sidebar-accent-foreground disabled:pointer-events-none disabled:opacity-50 group-has-data-[sidebar=menu-action]/menu-item:pe-8 aria-disabled:pointer-events-none aria-disabled:opacity-50 data-[active=true]:bg-sidebar-accent data-[active=true]:font-medium data-[active=true]:text-sidebar-accent-foreground data-[state=open]:hover:bg-sidebar-accent data-[state=open]:hover:text-sidebar-accent-foreground group-data-[collapsible=icon]:size-8! group-data-[collapsible=icon]:p-2! [&>span:last-child]:truncate [&>svg:not([class*='size-'])]:size-4 [&>svg]:shrink-0
size.default: h-8 text-sm
size.lg: h-12 text-sm group-data-[collapsible=icon]:p-0!
size.sm: h-7 text-xs
variant.default: hover:bg-sidebar-accent hover:text-sidebar-accent-foreground
variant.outline: bg-background shadow-[0_0_0_1px_hsl(var(--sidebar-border))] hover:bg-sidebar-accent hover:text-sidebar-accent-foreground hover:shadow-[0_0_0_1px_hsl(var(--sidebar-accent))]
```

## Parts

### `useSidebar`

No own classes (re-export / wrapper of the primitive).

### `SidebarProvider` → `[data-slot=sidebar-wrapper]` · `<div>` (line 159)

Attributes: `style={{ "--sidebar-width": SIDEBAR_WIDTH, "--sidebar-width-icon": SIDEBAR_WIDTH_ICON, ...style, } as React.CSSProperties}`

Prop defaults: `defaultOpen=true`

#### base

```txt
group/sidebar-wrapper flex min-h-svh w-full has-data-[variant=inset]:bg-sidebar
```

Classes that produce no CSS (custom markers / group names): `group/sidebar-wrapper`

**element — default**

Key metrics: display `flex` · min-height `100svh` · width `100%`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| display | `flex` | `flex` | `flex` |
| min-height | `100svh` | `100svh` | `min-h-svh` |
| width | `100%` | `100%` | `w-full` |

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| has([data-variant=inset]) | background-color | `#fafafa` / `#111111` | `has-data-[variant=inset]:bg-sidebar` |

### `Sidebar` → `[data-slot=sidebar]` · `<div>` (line 196)

Prop defaults: `side="left"`, `variant="sidebar"`, `collapsible="offcanvas"`

#### base

```txt
flex h-full w-(--sidebar-width) flex-col bg-sidebar text-sidebar-foreground
```

**element — default**

Key metrics: display `flex` · height `100%` · width `var(--sidebar-width)` (runtime var --sidebar-width) · background-color `#fafafa` / `#111111` (sidebar) · color `#727272` / `#a3a3a3` (sidebar-foreground)

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| display | `flex` | `flex` | `flex` |
| height | `100%` | `100%` | `h-full` |
| width | `var(--sidebar-width)` | `runtime var --sidebar-width` | `w-(--sidebar-width)` |
| flex-direction | `column` | `column` | `flex-col` |
| background-color | `#fafafa` / `#111111` | `sidebar` | `bg-sidebar` |
| color | `#727272` / `#a3a3a3` | `sidebar-foreground` | `text-sidebar-foreground` |

### `Sidebar` → `[data-slot=sidebar]` · `<SheetPopup>` (line 212)

Attributes: `data-mobile=true`, `data-sidebar=sidebar`, `side={side}`, `style={{ "--sidebar-width": SIDEBAR_WIDTH_MOBILE, } as React.CSSProperties}`

Prop defaults: `side="left"`, `variant="sidebar"`, `collapsible="offcanvas"`

#### base

```txt
w-(--sidebar-width) bg-sidebar p-0 text-sidebar-foreground [&>button]:hidden
```

**element — default**

Key metrics: width `var(--sidebar-width)` (runtime var --sidebar-width) · padding `0px` · background-color `#fafafa` / `#111111` (sidebar) · color `#727272` / `#a3a3a3` (sidebar-foreground)

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| width | `var(--sidebar-width)` | `runtime var --sidebar-width` | `w-(--sidebar-width)` |
| background-color | `#fafafa` / `#111111` | `sidebar` | `bg-sidebar` |
| padding | `0px` | `calc(var(--spacing) * 0)` | `p-0` |
| color | `#727272` / `#a3a3a3` | `sidebar-foreground` | `text-sidebar-foreground` |

**`>button` — default**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| display | `none` | `none` | `[&>button]:hidden` |

### `Sidebar` → (no data-slot) · `<SheetHeader>` (line 224)

Prop defaults: `side="left"`, `variant="sidebar"`, `collapsible="offcanvas"`

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

### `Sidebar` → (no data-slot) · `<div>` (line 228)

Prop defaults: `side="left"`, `variant="sidebar"`, `collapsible="offcanvas"`

#### base

```txt
flex h-full w-full flex-col
```

**element — default**

Key metrics: display `flex` · height `100%` · width `100%`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| display | `flex` | `flex` | `flex` |
| height | `100%` | `100%` | `h-full` |
| width | `100%` | `100%` | `w-full` |
| flex-direction | `column` | `column` | `flex-col` |

### `Sidebar` → `[data-slot=sidebar]` · `<div>` (line 235)

Attributes: `data-collapsible={state === "collapsed" ? collapsible : ""}`, `data-side={side}`, `data-state={state}`, `data-variant={variant}`

Prop defaults: `side="left"`, `variant="sidebar"`, `collapsible="offcanvas"`

#### base

```txt
group peer hidden text-sidebar-foreground md:block
```

Classes that produce no CSS (custom markers / group names): `group` `peer`

**element — default**

Key metrics: display `block` · color `#727272` / `#a3a3a3` (sidebar-foreground)

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| display | `block` | `block` | `md:block` |
| color | `#727272` / `#a3a3a3` | `sidebar-foreground` | `text-sidebar-foreground` |

**element — mobile differences**: display `none`

### `Sidebar` → `[data-slot=sidebar-gap]` · `<div>` (line 244)

Prop defaults: `side="left"`, `variant="sidebar"`, `collapsible="offcanvas"`

#### base

```txt
relative w-(--sidebar-width) bg-transparent transition-[width] duration-200 ease-linear group-data-[collapsible=offcanvas]:w-0 group-data-[side=right]:rotate-180
```

**element — default**

Key metrics: width `var(--sidebar-width)` (runtime var --sidebar-width) · background-color `#00000000` · transition-property `width` · transition-duration `200ms` · transition-timing-function `linear`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| position | `relative` | `relative` | `relative` |
| width | `var(--sidebar-width)` | `runtime var --sidebar-width` | `w-(--sidebar-width)` |
| background-color | `#00000000` | `transparent` | `bg-transparent` |
| transition-property | `width` | `width` | `transition-[width]` |
| transition-timing-function | `linear` | `linear` | `ease-linear` |
| transition-duration | `200ms` | `200ms` | `duration-200` |

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| in(.group[data-collapsible=offcanvas]) | width | `0px` | `group-data-[collapsible=offcanvas]:w-0` |
| in(.group[data-side=right]) | rotate | `180deg` | `group-data-[side=right]:rotate-180` |

#### when `variant === "floating" || variant === "inset"`

```txt
group-data-[collapsible=icon]:w-[calc(var(--sidebar-width-icon)+(--spacing(4)))]
```

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| in(.group[data-collapsible=offcanvas]) | width | `0px` | `group-data-[collapsible=offcanvas]:w-0` |
| in(.group[data-side=right]) | rotate | `180deg` | `group-data-[side=right]:rotate-180` |
| in(.group[data-collapsible=icon]) | width | `calc(var(--sidebar-width-icon) + 16px)` | `group-data-[collapsible=icon]:w-[calc(var(--sidebar-width-icon)+(--spacing(4)))]` |

#### when `!(variant === "floating" || variant === "inset")`

```txt
group-data-[collapsible=icon]:w-(--sidebar-width-icon)
```

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| in(.group[data-collapsible=offcanvas]) | width | `0px` | `group-data-[collapsible=offcanvas]:w-0` |
| in(.group[data-side=right]) | rotate | `180deg` | `group-data-[side=right]:rotate-180` |
| in(.group[data-collapsible=icon]) | width | `var(--sidebar-width-icon)` | `group-data-[collapsible=icon]:w-(--sidebar-width-icon)` |

### `Sidebar` → `[data-slot=sidebar-container]` · `<div>` (line 255)

Prop defaults: `side="left"`, `variant="sidebar"`, `collapsible="offcanvas"`

#### base

```txt
fixed inset-y-0 z-10 hidden h-svh w-(--sidebar-width) transition-[left,right,width] duration-200 ease-linear md:flex
```

**element — default**

Key metrics: display `flex` · height `100svh` · width `var(--sidebar-width)` (runtime var --sidebar-width) · z-index `10` · transition-property `left, right, width` · transition-duration `200ms` · transition-timing-function `linear`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| position | `fixed` | `fixed` | `fixed` |
| inset-block | `0px` | `calc(var(--spacing) * 0)` | `inset-y-0` |
| z-index | `10` | `10` | `z-10` |
| display | `flex` | `flex` | `md:flex` |
| height | `100svh` | `100svh` | `h-svh` |
| width | `var(--sidebar-width)` | `runtime var --sidebar-width` | `w-(--sidebar-width)` |
| transition-property | `left, right, width` | `left,right,width` | `transition-[left,right,width]` |
| transition-timing-function | `linear` | `linear` | `ease-linear` |
| transition-duration | `200ms` | `200ms` | `duration-200` |

**element — mobile differences**: display `none`

#### when `side === "left"`

```txt
left-0 group-data-[collapsible=offcanvas]:left-[calc(var(--sidebar-width)*-1)]
```

**element — default (changes vs base)**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| left | `0px` | `calc(var(--spacing) * 0)` | `left-0` |

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| in(.group[data-collapsible=offcanvas]) | left | `calc(var(--sidebar-width) * -1)` | `group-data-[collapsible=offcanvas]:left-[calc(var(--sidebar-width)*-1)]` |

**element — mobile differences**: display `none`

#### when `!(side === "left")`

```txt
right-0 group-data-[collapsible=offcanvas]:right-[calc(var(--sidebar-width)*-1)]
```

**element — default (changes vs base)**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| right | `0px` | `calc(var(--spacing) * 0)` | `right-0` |

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| in(.group[data-collapsible=offcanvas]) | right | `calc(var(--sidebar-width) * -1)` | `group-data-[collapsible=offcanvas]:right-[calc(var(--sidebar-width)*-1)]` |

**element — mobile differences**: display `none`

#### when `variant === "floating" || variant === "inset"`

```txt
p-2 group-data-[collapsible=icon]:w-[calc(var(--sidebar-width-icon)+(--spacing(4))+2px)]
```

**element — default (changes vs base)**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| padding | `8px` | `calc(var(--spacing) * 2)` | `p-2` |

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| in(.group[data-collapsible=icon]) | width | `calc(var(--sidebar-width-icon) + 16px + 2px)` | `group-data-[collapsible=icon]:w-[calc(var(--sidebar-width-icon)+(--spacing(4))+2px)]` |

**element — mobile differences**: display `none`

#### when `!(variant === "floating" || variant === "inset")`

```txt
group-data-[collapsible=icon]:w-(--sidebar-width-icon) group-data-[side=left]:border-r group-data-[side=right]:border-l
```

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| in(.group[data-collapsible=icon]) | width | `var(--sidebar-width-icon)` | `group-data-[collapsible=icon]:w-(--sidebar-width-icon)` |
| in(.group[data-side=left]) | border-right-style | `solid` | `group-data-[side=left]:border-r` |
| in(.group[data-side=left]) | border-right-width | `1px` | `group-data-[side=left]:border-r` |
| in(.group[data-side=right]) | border-left-style | `solid` | `group-data-[side=right]:border-l` |
| in(.group[data-side=right]) | border-left-width | `1px` | `group-data-[side=right]:border-l` |

**element — mobile differences**: display `none`

### `Sidebar` → `[data-slot=sidebar-inner]` · `<div>` (line 270)

Attributes: `data-sidebar=sidebar`

Prop defaults: `side="left"`, `variant="sidebar"`, `collapsible="offcanvas"`

#### base

```txt
flex h-full w-full flex-col bg-sidebar group-data-[variant=floating]:rounded-lg group-data-[variant=floating]:border group-data-[variant=floating]:border-sidebar-border group-data-[variant=floating]:shadow-sm/5
```

**element — default**

Key metrics: display `flex` · height `100%` · width `100%` · background-color `#fafafa` / `#111111` (sidebar)

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| display | `flex` | `flex` | `flex` |
| height | `100%` | `100%` | `h-full` |
| width | `100%` | `100%` | `w-full` |
| flex-direction | `column` | `column` | `flex-col` |
| background-color | `#fafafa` / `#111111` | `sidebar` | `bg-sidebar` |

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| in(.group[data-variant=floating]) | border-width | `1px` | `group-data-[variant=floating]:border` |
| in(.group[data-variant=floating]) | border-style | `solid` | `group-data-[variant=floating]:border` |
| in(.group[data-variant=floating]) | border-color | `#0000000f` / `#ffffff0d` | `group-data-[variant=floating]:border-sidebar-border` |
| in(.group[data-variant=floating]) | border-radius | `10px` | `group-data-[variant=floating]:rounded-lg` |
| in(.group[data-variant=floating]) | box-shadow | `0 1px 3px 0 #0000000d, 0 1px 2px -1px #0000000d` | `group-data-[variant=floating]:shadow-sm/5` |

### `SidebarTrigger` → `[data-slot=sidebar-trigger]` · `<Button>` (line 290)

Attributes: `data-sidebar=trigger`, `size=icon`, `variant=ghost`

#### base

```txt
size-7
```

**element — default**

Key metrics: height `28px` · width `28px`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| width | `28px` | `calc(var(--spacing) * 7)` | `size-7` |
| height | `28px` | `calc(var(--spacing) * 7)` | `size-7` |

### `SidebarTrigger` → (no data-slot) · `<span>` (line 303)

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

### `SidebarRail` → `[data-slot=sidebar-rail]` · `<button>` (line 315)

Attributes: `aria-label=Toggle Sidebar`, `data-sidebar=rail`, `tabIndex={-1}`, `title=Toggle Sidebar`, `type=button`

#### base

```txt
absolute inset-y-0 z-20 hidden w-4 -translate-x-1/2 transition-all ease-linear after:absolute after:inset-y-0 after:left-1/2 after:w-[2px] hover:after:bg-sidebar-border group-data-[side=left]:-right-4 group-data-[side=right]:left-0 sm:flex in-data-[side=left]:cursor-w-resize in-data-[side=right]:cursor-e-resize [[data-side=left][data-state=collapsed]_&]:cursor-e-resize [[data-side=right][data-state=collapsed]_&]:cursor-w-resize group-data-[collapsible=offcanvas]:translate-x-0 hover:group-data-[collapsible=offcanvas]:bg-sidebar group-data-[collapsible=offcanvas]:after:left-full [[data-side=left][data-collapsible=offcanvas]_&]:-right-2 [[data-side=right][data-collapsible=offcanvas]_&]:-left-2
```

**element — default**

Key metrics: display `flex` · width `16px` · z-index `20` · transition-property `all` · transition-duration `150ms` · transition-timing-function `linear`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| position | `absolute` | `absolute` | `absolute` |
| inset-block | `0px` | `calc(var(--spacing) * 0)` | `inset-y-0` |
| z-index | `20` | `20` | `z-20` |
| display | `flex` | `flex` | `sm:flex` |
| width | `16px` | `calc(var(--spacing) * 4)` | `w-4` |
| translate | `-50% 0` | `var(--tw-translate-x) var(--tw-translate-y)` | `-translate-x-1/2` |
| transition-property | `all` | `all` | `transition-all` |
| transition-timing-function | `linear` | `linear` | `ease-linear` |
| transition-duration | `150ms` | `var(--tw-duration, var(--default-transition-duration))` | `transition-all` |

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| in(.group[data-side=left]) | right | `-16px` | `group-data-[side=left]:-right-4` |
| in(.group[data-side=right]) | left | `0px` | `group-data-[side=right]:left-0` |
| in([data-side=left]) | cursor | `w-resize` | `in-data-[side=left]:cursor-w-resize` |
| in([data-side=right]) | cursor | `e-resize` | `in-data-[side=right]:cursor-e-resize` |
| in([data-side=left][data-state=collapsed]) | cursor | `e-resize` | `[[data-side=left][data-state=collapsed]_&]:cursor-e-resize` |
| in([data-side=right][data-state=collapsed]) | cursor | `w-resize` | `[[data-side=right][data-state=collapsed]_&]:cursor-w-resize` |
| in(.group[data-collapsible=offcanvas]) | translate | `0px 0` | `group-data-[collapsible=offcanvas]:translate-x-0` |
| in([data-side=left][data-collapsible=offcanvas]) | right | `-8px` | `[[data-side=left][data-collapsible=offcanvas]_&]:-right-2` |
| in([data-side=right][data-collapsible=offcanvas]) | left | `-8px` | `[[data-side=right][data-collapsible=offcanvas]_&]:-left-2` |
| hover + in(.group[data-collapsible=offcanvas]) | translate | `0px 0` | `group-data-[collapsible=offcanvas]:translate-x-0` |
| hover + in(.group[data-collapsible=offcanvas]) | background-color | `#fafafa` / `#111111` | `hover:group-data-[collapsible=offcanvas]:bg-sidebar` |

**element — mobile differences**: display `none`

**`::after` — default**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| content | `""` | `var(--tw-content)` | `after:w-[2px]` |
| position | `absolute` | `absolute` | `after:absolute` |
| inset-block | `0px` | `calc(var(--spacing) * 0)` | `after:inset-y-0` |
| left | `50%` | `calc(1/2 * 100%)` | `after:left-1/2` |
| width | `2px` | `2px` | `after:w-[2px]` |

**`::after` — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| hover | background-color | `#0000000f` / `#ffffff0d` | `hover:after:bg-sidebar-border` |
| in(.group[data-collapsible=offcanvas]) | left | `100%` | `group-data-[collapsible=offcanvas]:after:left-full` |
| hover + in(.group[data-collapsible=offcanvas]) | left | `100%` | `group-data-[collapsible=offcanvas]:after:left-full` |
| hover + in(.group[data-collapsible=offcanvas]) | background-color | `#0000000f` / `#ffffff0d` | `hover:after:bg-sidebar-border` |

### `SidebarInset` → `[data-slot=sidebar-inset]` · `<main>` (line 342)

#### base

```txt
relative flex w-full flex-1 flex-col bg-background md:peer-data-[variant=inset]:peer-data-[state=collapsed]:ms-2 md:peer-data-[variant=inset]:m-2 md:peer-data-[variant=inset]:ms-0 md:peer-data-[variant=inset]:rounded-xl md:peer-data-[variant=inset]:shadow-sm/5
```

**element — default**

Key metrics: display `flex` · width `100%` · background-color `#ffffff` / `#141414` (background)

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| position | `relative` | `relative` | `relative` |
| display | `flex` | `flex` | `flex` |
| width | `100%` | `100%` | `w-full` |
| flex | `1` | `1` | `flex-1` |
| flex-direction | `column` | `column` | `flex-col` |
| background-color | `#ffffff` / `#141414` | `background` | `bg-background` |

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| peer(.peer[data-variant=inset]) | margin | `8px` | `md:peer-data-[variant=inset]:m-2` |
| peer(.peer[data-variant=inset]) | margin-inline-start | `0px` | `md:peer-data-[variant=inset]:ms-0` |
| peer(.peer[data-variant=inset]) | border-radius | `14px` | `md:peer-data-[variant=inset]:rounded-xl` |
| peer(.peer[data-variant=inset]) | box-shadow | `0 1px 3px 0 #0000000d, 0 1px 2px -1px #0000000d` | `md:peer-data-[variant=inset]:shadow-sm/5` |
| peer(.peer[data-state=collapsed]) + peer(.peer[data-variant=inset]) | margin | `8px` | `md:peer-data-[variant=inset]:m-2` |
| peer(.peer[data-state=collapsed]) + peer(.peer[data-variant=inset]) | margin-inline-start | `8px` | `md:peer-data-[variant=inset]:peer-data-[state=collapsed]:ms-2` |
| peer(.peer[data-state=collapsed]) + peer(.peer[data-variant=inset]) | border-radius | `14px` | `md:peer-data-[variant=inset]:rounded-xl` |
| peer(.peer[data-state=collapsed]) + peer(.peer[data-variant=inset]) | box-shadow | `0 1px 3px 0 #0000000d, 0 1px 2px -1px #0000000d` | `md:peer-data-[variant=inset]:shadow-sm/5` |

### `SidebarInput` → `[data-slot=sidebar-input]` · `<Input>` (line 359)

Attributes: `data-sidebar=input`

#### base

```txt
h-8 w-full bg-background shadow-none
```

**element — default**

Key metrics: height `32px` · width `100%` · background-color `#ffffff` / `#141414` (background) · box-shadow `none`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| height | `32px` | `calc(var(--spacing) * 8)` | `h-8` |
| width | `100%` | `100%` | `w-full` |
| background-color | `#ffffff` / `#141414` | `background` | `bg-background` |
| box-shadow | `none` | `var(--tw-inset-shadow), var(--tw-inset-ring-shadow), var(--tw-ring-offset-shadow), var(--tw-ring-shadow), var(` | `shadow-none` |

### `SidebarHeader` → `[data-slot=sidebar-header]` · `<div>` (line 373)

Attributes: `data-sidebar=header`

#### base

```txt
flex flex-col gap-2 p-2
```

**element — default**

Key metrics: display `flex` · padding `8px` · gap `8px`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| display | `flex` | `flex` | `flex` |
| flex-direction | `column` | `column` | `flex-col` |
| gap | `8px` | `calc(var(--spacing) * 2)` | `gap-2` |
| padding | `8px` | `calc(var(--spacing) * 2)` | `p-2` |

### `SidebarFooter` → `[data-slot=sidebar-footer]` · `<div>` (line 387)

Attributes: `data-sidebar=footer`

#### base

```txt
flex flex-col gap-2 p-2
```

**element — default**

Key metrics: display `flex` · padding `8px` · gap `8px`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| display | `flex` | `flex` | `flex` |
| flex-direction | `column` | `column` | `flex-col` |
| gap | `8px` | `calc(var(--spacing) * 2)` | `gap-2` |
| padding | `8px` | `calc(var(--spacing) * 2)` | `p-2` |

### `SidebarSeparator` → `[data-slot=sidebar-separator]` · `<Separator>` (line 401)

Attributes: `data-sidebar=separator`

#### base

```txt
mx-2 w-auto bg-sidebar-border
```

**element — default**

Key metrics: width `auto` · background-color `#0000000f` / `#ffffff0d` (sidebar-border)

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| margin-inline | `8px` | `calc(var(--spacing) * 2)` | `mx-2` |
| width | `auto` | `auto` | `w-auto` |
| background-color | `#0000000f` / `#ffffff0d` | `sidebar-border` | `bg-sidebar-border` |

### `SidebarContent` → (no data-slot) · `<ScrollArea>` (line 415)

Attributes: `fill=true`, `overscrollContain=true`, `scrollFade=true`

#### base

```txt
min-h-0 flex-1
```

**element — default**

Key metrics: min-height `0px`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| min-height | `0px` | `calc(var(--spacing) * 0)` | `min-h-0` |
| flex | `1` | `1` | `flex-1` |

### `SidebarContent` → `[data-slot=sidebar-content]` · `<div>` (line 416)

Attributes: `data-sidebar=content`

#### base

```txt
flex h-full flex-col gap-2 group-data-[collapsible=icon]:overflow-hidden
```

**element — default**

Key metrics: display `flex` · height `100%` · gap `8px`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| display | `flex` | `flex` | `flex` |
| height | `100%` | `100%` | `h-full` |
| flex-direction | `column` | `column` | `flex-col` |
| gap | `8px` | `calc(var(--spacing) * 2)` | `gap-2` |

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| in(.group[data-collapsible=icon]) | overflow | `hidden` | `group-data-[collapsible=icon]:overflow-hidden` |

### `SidebarGroup` → `[data-slot=sidebar-group]` · `<div>` (line 434)

Attributes: `data-sidebar=group`

#### base

```txt
relative flex w-full min-w-0 flex-col p-2
```

**element — default**

Key metrics: display `flex` · width `100%` · min-width `0px` · padding `8px`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| position | `relative` | `relative` | `relative` |
| display | `flex` | `flex` | `flex` |
| width | `100%` | `100%` | `w-full` |
| min-width | `0px` | `calc(var(--spacing) * 0)` | `min-w-0` |
| flex-direction | `column` | `column` | `flex-col` |
| padding | `8px` | `calc(var(--spacing) * 2)` | `p-2` |

### `SidebarGroupLabel` → `[data-slot=sidebar-group-label]` · `useRender <div>` (line 448)

Attributes: `data-sidebar=group-label`

#### base

```txt
flex h-8 shrink-0 items-center rounded-lg px-2 font-medium text-sidebar-foreground text-xs outline-hidden ring-sidebar-ring transition-[margin,opacity] duration-200 ease-linear focus-visible:ring-2 [&>svg]:size-4 [&>svg]:shrink-0 group-data-[collapsible=icon]:-mt-8 group-data-[collapsible=icon]:opacity-0
```

Rules behind unevaluated conditions (container queries etc.): `@media (forced-colors: active)`

**element — default**

Key metrics: display `flex` · height `32px` · padding-inline `8px` · border-radius `10px` (radius) · color `#727272` / `#a3a3a3` (sidebar-foreground) · font-size `12px` (text-xs) · line-height `16px` · font-weight `500` (font-weight-medium) · transition-property `margin, opacity` · transition-duration `200ms` · transition-timing-function `linear`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| display | `flex` | `flex` | `flex` |
| height | `32px` | `calc(var(--spacing) * 8)` | `h-8` |
| flex-shrink | `0` | `0` | `shrink-0` |
| align-items | `center` | `center` | `items-center` |
| border-radius | `10px` | `radius` | `rounded-lg` |
| padding-inline | `8px` | `calc(var(--spacing) * 2)` | `px-2` |
| font-size | `12px` | `text-xs` | `text-xs` |
| line-height | `16px` | `var(--tw-leading, var(--text-xs--line-height))` | `text-xs` |
| font-weight | `500` | `font-weight-medium` | `font-medium` |
| color | `#727272` / `#a3a3a3` | `sidebar-foreground` | `text-sidebar-foreground` |
| outline-style | `none` | `none` | `outline-hidden` |
| transition-property | `margin, opacity` | `margin,opacity` | `transition-[margin,opacity]` |
| transition-timing-function | `linear` | `linear` | `ease-linear` |
| transition-duration | `200ms` | `200ms` | `duration-200` |

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| focus-visible | box-shadow | `0 0 0 2px #a1a1a1` | `focus-visible:ring-2` |
| in(.group[data-collapsible=icon]) | margin-top | `-32px` | `group-data-[collapsible=icon]:-mt-8` |
| in(.group[data-collapsible=icon]) | opacity | `0%` | `group-data-[collapsible=icon]:opacity-0` |

**`>svg` — default**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| width | `16px` | `calc(var(--spacing) * 4)` | `[&>svg]:size-4` |
| height | `16px` | `calc(var(--spacing) * 4)` | `[&>svg]:size-4` |
| flex-shrink | `0` | `0` | `[&>svg]:shrink-0` |

### `SidebarGroupAction` → `[data-slot=sidebar-group-action]` · `useRender <button>` (line 470)

Attributes: `data-sidebar=group-action`

#### base

```txt
absolute top-3.5 right-3 flex aspect-square w-5 items-center justify-center rounded-lg p-0 text-sidebar-foreground outline-hidden ring-sidebar-ring transition-transform hover:bg-sidebar-accent hover:text-sidebar-accent-foreground focus-visible:ring-2 [&>svg:not([class*='size-'])]:size-4 [&>svg]:shrink-0 after:absolute after:-inset-2 md:after:hidden group-data-[collapsible=icon]:hidden
```

Rules behind unevaluated conditions (container queries etc.): `@media (forced-colors: active)`

**element — default**

Key metrics: display `flex` · width `20px` · padding `0px` · border-radius `10px` (radius) · color `#727272` / `#a3a3a3` (sidebar-foreground) · transition-property `transform, translate, scale, rotate` · transition-duration `150ms` · transition-timing-function `cubic-bezier(0.4, 0, 0.2, 1)`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| position | `absolute` | `absolute` | `absolute` |
| top | `14px` | `calc(var(--spacing) * 3.5)` | `top-3.5` |
| right | `12px` | `calc(var(--spacing) * 3)` | `right-3` |
| display | `flex` | `flex` | `flex` |
| aspect-ratio | `1 / 1` | `1 / 1` | `aspect-square` |
| width | `20px` | `calc(var(--spacing) * 5)` | `w-5` |
| align-items | `center` | `center` | `items-center` |
| justify-content | `center` | `center` | `justify-center` |
| border-radius | `10px` | `radius` | `rounded-lg` |
| padding | `0px` | `calc(var(--spacing) * 0)` | `p-0` |
| color | `#727272` / `#a3a3a3` | `sidebar-foreground` | `text-sidebar-foreground` |
| outline-style | `none` | `none` | `outline-hidden` |
| transition-property | `transform, translate, scale, rotate` | `transform, translate, scale, rotate` | `transition-transform` |
| transition-timing-function | `cubic-bezier(0.4, 0, 0.2, 1)` | `var(--tw-ease, var(--default-transition-timing-function))` | `transition-transform` |
| transition-duration | `150ms` | `var(--tw-duration, var(--default-transition-duration))` | `transition-transform` |

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| hover | color | `#262626` / `#f5f5f5` | `hover:text-sidebar-accent-foreground` |
| hover | background-color | `#0000000a` / `#ffffff0a` | `hover:bg-sidebar-accent` |
| focus-visible | box-shadow | `0 0 0 2px #a1a1a1` | `focus-visible:ring-2` |
| in(.group[data-collapsible=icon]) | display | `none` | `group-data-[collapsible=icon]:hidden` |

**`::after` — default**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| content | `""` | `var(--tw-content)` | `md:after:hidden` |
| position | `absolute` | `absolute` | `after:absolute` |
| inset | `-8px` | `calc(var(--spacing) * -2)` | `after:-inset-2` |
| display | `none` | `none` | `md:after:hidden` |

**`::after` — mobile differences**: display `(unset)`

**`>svg` — default**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| flex-shrink | `0` | `0` | `[&>svg]:shrink-0` |

**`>svg:not([class*='size-'])` — default**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| width | `16px` | `calc(var(--spacing) * 4)` | `[&>svg:not([class*='size-'])]:size-4` |
| height | `16px` | `calc(var(--spacing) * 4)` | `[&>svg:not([class*='size-'])]:size-4` |

### `SidebarGroupContent` → `[data-slot=sidebar-group-content]` · `<div>` (line 494)

Attributes: `data-sidebar=group-content`

#### base

```txt
w-full text-sm
```

**element — default**

Key metrics: width `100%` · font-size `14px` (text-sm) · line-height `20px`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| width | `100%` | `100%` | `w-full` |
| font-size | `14px` | `text-sm` | `text-sm` |
| line-height | `20px` | `var(--tw-leading, var(--text-sm--line-height))` | `text-sm` |

### `SidebarMenu` → `[data-slot=sidebar-menu]` · `<ul>` (line 508)

Attributes: `data-sidebar=menu`

#### base

```txt
flex w-full min-w-0 flex-col gap-1
```

**element — default**

Key metrics: display `flex` · width `100%` · min-width `0px` · gap `4px`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| display | `flex` | `flex` | `flex` |
| width | `100%` | `100%` | `w-full` |
| min-width | `0px` | `calc(var(--spacing) * 0)` | `min-w-0` |
| flex-direction | `column` | `column` | `flex-col` |
| gap | `4px` | `calc(var(--spacing) * 1)` | `gap-1` |

### `SidebarMenuItem` → `[data-slot=sidebar-menu-item]` · `<li>` (line 522)

Attributes: `data-sidebar=menu-item`

#### base

```txt
group/menu-item relative
```

Classes that produce no CSS (custom markers / group names): `group/menu-item`

**element — default**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| position | `relative` | `relative` | `relative` |

### `SidebarMenuButton` → `[data-slot=sidebar-menu-button]` · `useRender <button>` (line 545)

Attributes: `data-active={isActive}`, `data-sidebar=menu-button`, `data-size={size}`

Prop defaults: `isActive=false`, `variant="default"`, `size="default"`

#### base (cva defaults: size=default, variant=default)

```txt
peer/menu-button flex w-full items-center gap-2 overflow-hidden rounded-lg p-2 text-left text-sm outline-hidden ring-sidebar-ring transition-[width,height,padding] hover:bg-sidebar-accent hover:text-sidebar-accent-foreground focus-visible:ring-2 active:bg-sidebar-accent active:text-sidebar-accent-foreground disabled:pointer-events-none disabled:opacity-50 group-has-data-[sidebar=menu-action]/menu-item:pe-8 aria-disabled:pointer-events-none aria-disabled:opacity-50 data-[active=true]:bg-sidebar-accent data-[active=true]:font-medium data-[active=true]:text-sidebar-accent-foreground data-[state=open]:hover:bg-sidebar-accent data-[state=open]:hover:text-sidebar-accent-foreground group-data-[collapsible=icon]:size-8! group-data-[collapsible=icon]:p-2! [&>span:last-child]:truncate [&>svg:not([class*='size-'])]:size-4 [&>svg]:shrink-0 h-8 text-sm hover:bg-sidebar-accent hover:text-sidebar-accent-foreground
```

Classes that produce no CSS (custom markers / group names): `peer/menu-button`

Rules behind unevaluated conditions (container queries etc.): `@media (forced-colors: active)`

**element — default**

Key metrics: display `flex` · height `32px` · width `100%` · padding `8px` · gap `8px` · border-radius `10px` (radius) · font-size `14px` (text-sm) · line-height `20px` · transition-property `width, height, padding` · transition-duration `150ms` · transition-timing-function `cubic-bezier(0.4, 0, 0.2, 1)`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| display | `flex` | `flex` | `flex` |
| height | `32px` | `calc(var(--spacing) * 8)` | `h-8` |
| width | `100%` | `100%` | `w-full` |
| align-items | `center` | `center` | `items-center` |
| gap | `8px` | `calc(var(--spacing) * 2)` | `gap-2` |
| overflow | `hidden` | `hidden` | `overflow-hidden` |
| border-radius | `10px` | `radius` | `rounded-lg` |
| padding | `8px` | `calc(var(--spacing) * 2)` | `p-2` |
| text-align | `left` | `left` | `text-left` |
| font-size | `14px` | `text-sm` | `text-sm` |
| line-height | `20px` | `var(--tw-leading, var(--text-sm--line-height))` | `text-sm` |
| outline-style | `none` | `none` | `outline-hidden` |
| transition-property | `width, height, padding` | `width,height,padding` | `transition-[width,height,padding]` |
| transition-timing-function | `cubic-bezier(0.4, 0, 0.2, 1)` | `var(--tw-ease, var(--default-transition-timing-function))` | `transition-[width,height,padding]` |
| transition-duration | `150ms` | `var(--tw-duration, var(--default-transition-duration))` | `transition-[width,height,padding]` |

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| hover | background-color | `#0000000a` / `#ffffff0a` | `hover:bg-sidebar-accent` |
| hover | color | `#262626` / `#f5f5f5` | `hover:text-sidebar-accent-foreground` |
| focus-visible | box-shadow | `0 0 0 2px #a1a1a1` | `focus-visible:ring-2` |
| pressed | background-color | `#0000000a` / `#ffffff0a` | `active:bg-sidebar-accent` |
| pressed | color | `#262626` / `#f5f5f5` | `active:text-sidebar-accent-foreground` |
| disabled | pointer-events | `none` | `aria-disabled:pointer-events-none` |
| disabled | opacity | `50%` | `aria-disabled:opacity-50` |
| in(.group\/menu-item:has(*[data-sidebar=menu-action])) | padding-inline-end | `32px` | `group-has-data-[sidebar=menu-action]/menu-item:pe-8` |
| selected | background-color | `#0000000a` / `#ffffff0a` | `data-[active=true]:bg-sidebar-accent` |
| selected | font-weight | `500` | `data-[active=true]:font-medium` |
| selected | color | `#262626` / `#f5f5f5` | `data-[active=true]:text-sidebar-accent-foreground` |
| in(.group[data-collapsible=icon]) | width | `32px` | `group-data-[collapsible=icon]:size-8!` |
| hover + open | background-color | `#0000000a` / `#ffffff0a` | `data-[state=open]:hover:bg-sidebar-accent` |
| hover + open | color | `#262626` / `#f5f5f5` | `data-[state=open]:hover:text-sidebar-accent-foreground` |

**`>span:last-child` — default**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| overflow | `hidden` | `hidden` | `[&>span:last-child]:truncate` |
| text-overflow | `ellipsis` | `ellipsis` | `[&>span:last-child]:truncate` |
| white-space | `nowrap` | `nowrap` | `[&>span:last-child]:truncate` |

**`>svg` — default**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| flex-shrink | `0` | `0` | `[&>svg]:shrink-0` |

**`>svg:not([class*='size-'])` — default**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| width | `16px` | `calc(var(--spacing) * 4)` | `[&>svg:not([class*='size-'])]:size-4` |
| height | `16px` | `calc(var(--spacing) * 4)` | `[&>svg:not([class*='size-'])]:size-4` |

#### sidebarMenuButtonVariants: size=lg

```txt
h-12 text-sm group-data-[collapsible=icon]:p-0!
```

Classes that produce no CSS (custom markers / group names): `peer/menu-button`

Rules behind unevaluated conditions (container queries etc.): `@media (forced-colors: active)`

**element — default (changes vs base)**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| height | `48px` | `calc(var(--spacing) * 12)` | `h-12` |

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| hover | background-color | `#0000000a` / `#ffffff0a` | `hover:bg-sidebar-accent` |
| hover | color | `#262626` / `#f5f5f5` | `hover:text-sidebar-accent-foreground` |
| focus-visible | box-shadow | `0 0 0 2px #a1a1a1` | `focus-visible:ring-2` |
| pressed | background-color | `#0000000a` / `#ffffff0a` | `active:bg-sidebar-accent` |
| pressed | color | `#262626` / `#f5f5f5` | `active:text-sidebar-accent-foreground` |
| disabled | pointer-events | `none` | `aria-disabled:pointer-events-none` |
| disabled | opacity | `50%` | `aria-disabled:opacity-50` |
| in(.group\/menu-item:has(*[data-sidebar=menu-action])) | padding-inline-end | `32px` | `group-has-data-[sidebar=menu-action]/menu-item:pe-8` |
| selected | background-color | `#0000000a` / `#ffffff0a` | `data-[active=true]:bg-sidebar-accent` |
| selected | font-weight | `500` | `data-[active=true]:font-medium` |
| selected | color | `#262626` / `#f5f5f5` | `data-[active=true]:text-sidebar-accent-foreground` |
| in(.group[data-collapsible=icon]) | height | `32px` | `group-data-[collapsible=icon]:size-8!` |
| in(.group[data-collapsible=icon]) | width | `32px` | `group-data-[collapsible=icon]:size-8!` |
| hover + open | background-color | `#0000000a` / `#ffffff0a` | `data-[state=open]:hover:bg-sidebar-accent` |
| hover + open | color | `#262626` / `#f5f5f5` | `data-[state=open]:hover:text-sidebar-accent-foreground` |

#### sidebarMenuButtonVariants: size=sm

```txt
h-7 text-xs
```

Classes that produce no CSS (custom markers / group names): `peer/menu-button`

Rules behind unevaluated conditions (container queries etc.): `@media (forced-colors: active)`

**element — default (changes vs base)**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| height | `28px` | `calc(var(--spacing) * 7)` | `h-7` |
| font-size | `12px` | `text-xs` | `text-xs` |
| line-height | `16px` | `var(--tw-leading, var(--text-xs--line-height))` | `text-xs` |

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| hover | background-color | `#0000000a` / `#ffffff0a` | `hover:bg-sidebar-accent` |
| hover | color | `#262626` / `#f5f5f5` | `hover:text-sidebar-accent-foreground` |
| focus-visible | box-shadow | `0 0 0 2px #a1a1a1` | `focus-visible:ring-2` |
| pressed | background-color | `#0000000a` / `#ffffff0a` | `active:bg-sidebar-accent` |
| pressed | color | `#262626` / `#f5f5f5` | `active:text-sidebar-accent-foreground` |
| disabled | pointer-events | `none` | `aria-disabled:pointer-events-none` |
| disabled | opacity | `50%` | `aria-disabled:opacity-50` |
| in(.group\/menu-item:has(*[data-sidebar=menu-action])) | padding-inline-end | `32px` | `group-has-data-[sidebar=menu-action]/menu-item:pe-8` |
| selected | background-color | `#0000000a` / `#ffffff0a` | `data-[active=true]:bg-sidebar-accent` |
| selected | font-weight | `500` | `data-[active=true]:font-medium` |
| selected | color | `#262626` / `#f5f5f5` | `data-[active=true]:text-sidebar-accent-foreground` |
| in(.group[data-collapsible=icon]) | height | `32px` | `group-data-[collapsible=icon]:size-8!` |
| in(.group[data-collapsible=icon]) | width | `32px` | `group-data-[collapsible=icon]:size-8!` |
| hover + open | background-color | `#0000000a` / `#ffffff0a` | `data-[state=open]:hover:bg-sidebar-accent` |
| hover + open | color | `#262626` / `#f5f5f5` | `data-[state=open]:hover:text-sidebar-accent-foreground` |

#### sidebarMenuButtonVariants: variant=outline

```txt
bg-background shadow-[0_0_0_1px_hsl(var(--sidebar-border))] hover:bg-sidebar-accent hover:text-sidebar-accent-foreground hover:shadow-[0_0_0_1px_hsl(var(--sidebar-accent))]
```

Classes that produce no CSS (custom markers / group names): `peer/menu-button`

Rules behind unevaluated conditions (container queries etc.): `@media (forced-colors: active)`

**element — default (changes vs base)**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| background-color | `#ffffff` / `#141414` | `background` | `bg-background` |
| box-shadow | `0 0 0 1px hsl(#0000000f)` / `0 0 0 1px hsl(#ffffff0d)` | `var(--tw-inset-shadow), var(--tw-inset-ring-shadow), var(--tw-ring-offset-shadow), var(--tw-ring-shadow), var(` | `shadow-[0_0_0_1px_hsl(var(--sidebar-border))]` |

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| hover | background-color | `#0000000a` / `#ffffff0a` | `hover:bg-sidebar-accent` |
| hover | box-shadow | `0 0 0 1px hsl(#0000000a)` / `0 0 0 1px hsl(#ffffff0a)` | `hover:shadow-[0_0_0_1px_hsl(var(--sidebar-accent))]` |
| hover | color | `#262626` / `#f5f5f5` | `hover:text-sidebar-accent-foreground` |
| focus-visible | box-shadow | `0 0 0 2px #a1a1a1, 0 0 0 1px hsl(#0000000f)` / `0 0 0 2px #a1a1a1, 0 0 0 1px hsl(#ffffff0d)` | `focus-visible:ring-2` |
| pressed | background-color | `#0000000a` / `#ffffff0a` | `active:bg-sidebar-accent` |
| pressed | box-shadow | `0 0 0 1px hsl(#0000000a)` / `0 0 0 1px hsl(#ffffff0a)` | `hover:shadow-[0_0_0_1px_hsl(var(--sidebar-accent))]` |
| pressed | color | `#262626` / `#f5f5f5` | `active:text-sidebar-accent-foreground` |
| disabled | pointer-events | `none` | `aria-disabled:pointer-events-none` |
| disabled | opacity | `50%` | `aria-disabled:opacity-50` |
| in(.group\/menu-item:has(*[data-sidebar=menu-action])) | padding-inline-end | `32px` | `group-has-data-[sidebar=menu-action]/menu-item:pe-8` |
| selected | background-color | `#0000000a` / `#ffffff0a` | `data-[active=true]:bg-sidebar-accent` |
| selected | font-weight | `500` | `data-[active=true]:font-medium` |
| selected | color | `#262626` / `#f5f5f5` | `data-[active=true]:text-sidebar-accent-foreground` |
| in(.group[data-collapsible=icon]) | width | `32px` | `group-data-[collapsible=icon]:size-8!` |
| hover + open | background-color | `#0000000a` / `#ffffff0a` | `data-[state=open]:hover:bg-sidebar-accent` |
| hover + open | box-shadow | `0 0 0 1px hsl(#0000000a)` / `0 0 0 1px hsl(#ffffff0a)` | `hover:shadow-[0_0_0_1px_hsl(var(--sidebar-accent))]` |
| hover + open | color | `#262626` / `#f5f5f5` | `data-[state=open]:hover:text-sidebar-accent-foreground` |

### `SidebarMenuAction` → `[data-slot=sidebar-menu-action]` · `useRender <button>` (line 594)

Attributes: `data-sidebar=menu-action`

Prop defaults: `showOnHover=false`

#### base

```txt
absolute top-1.5 right-1 flex aspect-square w-5 items-center justify-center rounded-lg p-0 text-sidebar-foreground outline-hidden ring-sidebar-ring transition-transform hover:bg-sidebar-accent hover:text-sidebar-accent-foreground focus-visible:ring-2 peer-hover/menu-button:text-sidebar-accent-foreground [&>svg:not([class*='size-'])]:size-4 [&>svg]:shrink-0 after:absolute after:-inset-2 md:after:hidden peer-data-[size=sm]/menu-button:top-1 peer-data-[size=default]/menu-button:top-1.5 peer-data-[size=lg]/menu-button:top-2.5 group-data-[collapsible=icon]:hidden
```

Rules behind unevaluated conditions (container queries etc.): `@media (forced-colors: active)`

**element — default**

Key metrics: display `flex` · width `20px` · padding `0px` · border-radius `10px` (radius) · color `#727272` / `#a3a3a3` (sidebar-foreground) · transition-property `transform, translate, scale, rotate` · transition-duration `150ms` · transition-timing-function `cubic-bezier(0.4, 0, 0.2, 1)`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| position | `absolute` | `absolute` | `absolute` |
| top | `6px` | `calc(var(--spacing) * 1.5)` | `top-1.5` |
| right | `4px` | `calc(var(--spacing) * 1)` | `right-1` |
| display | `flex` | `flex` | `flex` |
| aspect-ratio | `1 / 1` | `1 / 1` | `aspect-square` |
| width | `20px` | `calc(var(--spacing) * 5)` | `w-5` |
| align-items | `center` | `center` | `items-center` |
| justify-content | `center` | `center` | `justify-center` |
| border-radius | `10px` | `radius` | `rounded-lg` |
| padding | `0px` | `calc(var(--spacing) * 0)` | `p-0` |
| color | `#727272` / `#a3a3a3` | `sidebar-foreground` | `text-sidebar-foreground` |
| outline-style | `none` | `none` | `outline-hidden` |
| transition-property | `transform, translate, scale, rotate` | `transform, translate, scale, rotate` | `transition-transform` |
| transition-timing-function | `cubic-bezier(0.4, 0, 0.2, 1)` | `var(--tw-ease, var(--default-transition-timing-function))` | `transition-transform` |
| transition-duration | `150ms` | `var(--tw-duration, var(--default-transition-duration))` | `transition-transform` |

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| hover | color | `#262626` / `#f5f5f5` | `hover:text-sidebar-accent-foreground` |
| hover | background-color | `#0000000a` / `#ffffff0a` | `hover:bg-sidebar-accent` |
| focus-visible | box-shadow | `0 0 0 2px #a1a1a1` | `focus-visible:ring-2` |
| peer(.peer\/menu-button:hover) | color | `#262626` / `#f5f5f5` | `peer-hover/menu-button:text-sidebar-accent-foreground` |
| peer(.peer\/menu-button[data-size=sm]) | top | `4px` | `peer-data-[size=sm]/menu-button:top-1` |
| peer(.peer\/menu-button[data-size=lg]) | top | `10px` | `peer-data-[size=lg]/menu-button:top-2.5` |
| in(.group[data-collapsible=icon]) | display | `none` | `group-data-[collapsible=icon]:hidden` |

**`::after` — default**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| content | `""` | `var(--tw-content)` | `md:after:hidden` |
| position | `absolute` | `absolute` | `after:absolute` |
| inset | `-8px` | `calc(var(--spacing) * -2)` | `after:-inset-2` |
| display | `none` | `none` | `md:after:hidden` |

**`::after` — mobile differences**: display `(unset)`

**`>svg` — default**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| flex-shrink | `0` | `0` | `[&>svg]:shrink-0` |

**`>svg:not([class*='size-'])` — default**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| width | `16px` | `calc(var(--spacing) * 4)` | `[&>svg:not([class*='size-'])]:size-4` |
| height | `16px` | `calc(var(--spacing) * 4)` | `[&>svg:not([class*='size-'])]:size-4` |

#### when `showOnHover`

```txt
group-focus-within/menu-item:opacity-100 group-hover/menu-item:opacity-100 data-[state=open]:opacity-100 peer-data-[active=true]/menu-button:text-sidebar-accent-foreground md:opacity-0
```

Rules behind unevaluated conditions (container queries etc.): `@media (forced-colors: active)`

**element — default (changes vs base)**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| opacity | `0%` | `0%` | `md:opacity-0` |

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| hover | color | `#262626` / `#f5f5f5` | `hover:text-sidebar-accent-foreground` |
| hover | background-color | `#0000000a` / `#ffffff0a` | `hover:bg-sidebar-accent` |
| focus-visible | box-shadow | `0 0 0 2px #a1a1a1` | `focus-visible:ring-2` |
| peer(.peer\/menu-button:hover) | color | `#262626` / `#f5f5f5` | `peer-hover/menu-button:text-sidebar-accent-foreground` |
| peer(.peer\/menu-button[data-size=sm]) | top | `4px` | `peer-data-[size=sm]/menu-button:top-1` |
| peer(.peer\/menu-button[data-size=lg]) | top | `10px` | `peer-data-[size=lg]/menu-button:top-2.5` |
| in(.group[data-collapsible=icon]) | display | `none` | `group-data-[collapsible=icon]:hidden` |
| in(.group\/menu-item:focus-within) | opacity | `100%` | `group-focus-within/menu-item:opacity-100` |
| in(.group\/menu-item:hover) | opacity | `100%` | `group-hover/menu-item:opacity-100` |
| open | opacity | `100%` | `data-[state=open]:opacity-100` |
| peer(.peer\/menu-button[data-active=true]) | color | `#262626` / `#f5f5f5` | `peer-data-[active=true]/menu-button:text-sidebar-accent-foreground` |

**element — mobile differences**: opacity `(unset)`

**`::after` — mobile differences**: display `(unset)`

### `SidebarMenuBadge` → `[data-slot=sidebar-menu-badge]` · `<div>` (line 623)

Attributes: `data-sidebar=menu-badge`

#### base

```txt
pointer-events-none absolute right-1 flex h-5 min-w-5 select-none items-center justify-center rounded-lg px-1 font-medium text-sidebar-foreground text-xs tabular-nums peer-hover/menu-button:text-sidebar-accent-foreground peer-data-[active=true]/menu-button:text-sidebar-accent-foreground peer-data-[size=sm]/menu-button:top-1 peer-data-[size=default]/menu-button:top-1.5 peer-data-[size=lg]/menu-button:top-2.5 group-data-[collapsible=icon]:hidden
```

**element — default**

Key metrics: display `flex` · height `20px` · min-width `20px` · padding-inline `4px` · border-radius `10px` (radius) · color `#727272` / `#a3a3a3` (sidebar-foreground) · font-size `12px` (text-xs) · line-height `16px` · font-weight `500` (font-weight-medium)

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| pointer-events | `none` | `none` | `pointer-events-none` |
| position | `absolute` | `absolute` | `absolute` |
| right | `4px` | `calc(var(--spacing) * 1)` | `right-1` |
| display | `flex` | `flex` | `flex` |
| height | `20px` | `calc(var(--spacing) * 5)` | `h-5` |
| min-width | `20px` | `calc(var(--spacing) * 5)` | `min-w-5` |
| align-items | `center` | `center` | `items-center` |
| justify-content | `center` | `center` | `justify-center` |
| border-radius | `10px` | `radius` | `rounded-lg` |
| padding-inline | `4px` | `calc(var(--spacing) * 1)` | `px-1` |
| font-size | `12px` | `text-xs` | `text-xs` |
| line-height | `16px` | `var(--tw-leading, var(--text-xs--line-height))` | `text-xs` |
| font-weight | `500` | `font-weight-medium` | `font-medium` |
| color | `#727272` / `#a3a3a3` | `sidebar-foreground` | `text-sidebar-foreground` |
| font-variant-numeric | `tabular-nums` | `var(--tw-ordinal,) var(--tw-slashed-zero,) var(--tw-numeric-figure,) var(--tw-numeric-spacing,) var(--tw-numer` | `tabular-nums` |
| -webkit-user-select | `none` | `none` | `select-none` |
| user-select | `none` | `none` | `select-none` |

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| peer(.peer\/menu-button:hover) | color | `#262626` / `#f5f5f5` | `peer-hover/menu-button:text-sidebar-accent-foreground` |
| peer(.peer\/menu-button[data-active=true]) | color | `#262626` / `#f5f5f5` | `peer-data-[active=true]/menu-button:text-sidebar-accent-foreground` |
| peer(.peer\/menu-button[data-size=sm]) | top | `4px` | `peer-data-[size=sm]/menu-button:top-1` |
| peer(.peer\/menu-button[data-size=default]) | top | `6px` | `peer-data-[size=default]/menu-button:top-1.5` |
| peer(.peer\/menu-button[data-size=lg]) | top | `10px` | `peer-data-[size=lg]/menu-button:top-2.5` |
| in(.group[data-collapsible=icon]) | display | `none` | `group-data-[collapsible=icon]:hidden` |

### `SidebarMenuSkeleton` → `[data-slot=sidebar-menu-skeleton]` · `<div>` (line 653)

Attributes: `data-sidebar=menu-skeleton`

Prop defaults: `showIcon=false`

#### base

```txt
flex h-8 items-center gap-2 rounded-lg px-2
```

**element — default**

Key metrics: display `flex` · height `32px` · padding-inline `8px` · gap `8px` · border-radius `10px` (radius)

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| display | `flex` | `flex` | `flex` |
| height | `32px` | `calc(var(--spacing) * 8)` | `h-8` |
| align-items | `center` | `center` | `items-center` |
| gap | `8px` | `calc(var(--spacing) * 2)` | `gap-2` |
| border-radius | `10px` | `radius` | `rounded-lg` |
| padding-inline | `8px` | `calc(var(--spacing) * 2)` | `px-2` |

### `SidebarMenuSkeleton` → (no data-slot) · `<Skeleton>` (line 660)

Attributes: `data-sidebar=menu-skeleton-icon`

Prop defaults: `showIcon=false`

#### base

```txt
size-4 rounded-lg
```

**element — default**

Key metrics: height `16px` · width `16px` · border-radius `10px` (radius)

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| width | `16px` | `calc(var(--spacing) * 4)` | `size-4` |
| height | `16px` | `calc(var(--spacing) * 4)` | `size-4` |
| border-radius | `10px` | `radius` | `rounded-lg` |

### `SidebarMenuSkeleton` → (no data-slot) · `<Skeleton>` (line 665)

Attributes: `data-sidebar=menu-skeleton-text`, `style={{ "--skeleton-width": width, } as React.CSSProperties}`

Prop defaults: `showIcon=false`

#### base

```txt
h-4 max-w-(--skeleton-width) flex-1
```

**element — default**

Key metrics: height `16px` · max-width `var(--skeleton-width)` (runtime var --skeleton-width)

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| height | `16px` | `calc(var(--spacing) * 4)` | `h-4` |
| max-width | `var(--skeleton-width)` | `runtime var --skeleton-width` | `max-w-(--skeleton-width)` |
| flex | `1` | `1` | `flex-1` |

### `SidebarMenuSub` → `[data-slot=sidebar-menu-sub]` · `<ul>` (line 683)

Attributes: `data-sidebar=menu-sub`

#### base

```txt
mx-3.5 flex min-w-0 translate-x-px flex-col gap-1 border-sidebar-border border-l px-2.5 py-0.5 group-data-[collapsible=icon]:hidden
```

**element — default**

Key metrics: display `flex` · min-width `0px` · padding-inline `10px` · padding-block `2px` · gap `4px` · border-color `#0000000f` / `#ffffff0d` (sidebar-border)

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| border-color | `#0000000f` / `#ffffff0d` | `sidebar-border` | `border-sidebar-border` |
| margin-inline | `14px` | `calc(var(--spacing) * 3.5)` | `mx-3.5` |
| display | `flex` | `flex` | `flex` |
| min-width | `0px` | `calc(var(--spacing) * 0)` | `min-w-0` |
| translate | `1px 0` | `var(--tw-translate-x) var(--tw-translate-y)` | `translate-x-px` |
| flex-direction | `column` | `column` | `flex-col` |
| gap | `4px` | `calc(var(--spacing) * 1)` | `gap-1` |
| border-left-style | `solid` | `var(--tw-border-style)` | `border-l` |
| border-left-width | `1px` | `1px` | `border-l` |
| padding-inline | `10px` | `calc(var(--spacing) * 2.5)` | `px-2.5` |
| padding-block | `2px` | `calc(var(--spacing) * 0.5)` | `py-0.5` |

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| in(.group[data-collapsible=icon]) | display | `none` | `group-data-[collapsible=icon]:hidden` |

### `SidebarMenuSubItem` → `[data-slot=sidebar-menu-sub-item]` · `<li>` (line 701)

Attributes: `data-sidebar=menu-sub-item`

#### base

```txt
group/menu-sub-item relative
```

Classes that produce no CSS (custom markers / group names): `group/menu-sub-item`

**element — default**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| position | `relative` | `relative` | `relative` |

### `SidebarMenuSubButton` → `[data-slot=sidebar-menu-sub-button]` · `useRender <a>` (line 720)

Attributes: `data-active={isActive}`, `data-sidebar=menu-sub-button`, `data-size={size}`

Prop defaults: `size="md"`, `isActive=false`

#### base

```txt
flex h-8 min-w-0 -translate-x-px items-center gap-2 overflow-hidden rounded-lg px-2 text-sidebar-foreground outline-hidden ring-sidebar-ring hover:bg-sidebar-accent hover:text-sidebar-accent-foreground focus-visible:ring-2 active:bg-sidebar-accent active:text-sidebar-accent-foreground disabled:pointer-events-none disabled:opacity-50 aria-disabled:pointer-events-none aria-disabled:opacity-50 sm:h-7 [&>span:last-child]:truncate [&>svg:not([class*='size-'])]:size-4 [&>svg]:shrink-0 [&>svg]:text-sidebar-accent-foreground data-[active=true]:bg-sidebar-accent data-[active=true]:text-sidebar-accent-foreground group-data-[collapsible=icon]:hidden
```

Rules behind unevaluated conditions (container queries etc.): `@media (forced-colors: active)`

**element — default**

Key metrics: display `flex` · height `28px` · min-width `0px` · padding-inline `8px` · gap `8px` · border-radius `10px` (radius) · color `#727272` / `#a3a3a3` (sidebar-foreground)

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| display | `flex` | `flex` | `flex` |
| height | `28px` | `calc(var(--spacing) * 7)` | `sm:h-7` |
| min-width | `0px` | `calc(var(--spacing) * 0)` | `min-w-0` |
| translate | `-1px 0` | `var(--tw-translate-x) var(--tw-translate-y)` | `-translate-x-px` |
| align-items | `center` | `center` | `items-center` |
| gap | `8px` | `calc(var(--spacing) * 2)` | `gap-2` |
| overflow | `hidden` | `hidden` | `overflow-hidden` |
| border-radius | `10px` | `radius` | `rounded-lg` |
| padding-inline | `8px` | `calc(var(--spacing) * 2)` | `px-2` |
| color | `#727272` / `#a3a3a3` | `sidebar-foreground` | `text-sidebar-foreground` |
| outline-style | `none` | `none` | `outline-hidden` |

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| hover | color | `#262626` / `#f5f5f5` | `hover:text-sidebar-accent-foreground` |
| hover | background-color | `#0000000a` / `#ffffff0a` | `hover:bg-sidebar-accent` |
| focus-visible | box-shadow | `0 0 0 2px #a1a1a1` | `focus-visible:ring-2` |
| pressed | color | `#262626` / `#f5f5f5` | `active:text-sidebar-accent-foreground` |
| pressed | background-color | `#0000000a` / `#ffffff0a` | `active:bg-sidebar-accent` |
| disabled | pointer-events | `none` | `aria-disabled:pointer-events-none` |
| disabled | opacity | `50%` | `aria-disabled:opacity-50` |
| selected | color | `#262626` / `#f5f5f5` | `data-[active=true]:text-sidebar-accent-foreground` |
| selected | background-color | `#0000000a` / `#ffffff0a` | `data-[active=true]:bg-sidebar-accent` |
| in(.group[data-collapsible=icon]) | display | `none` | `group-data-[collapsible=icon]:hidden` |

**element — mobile differences**: height `32px`

**`>span:last-child` — default**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| overflow | `hidden` | `hidden` | `[&>span:last-child]:truncate` |
| text-overflow | `ellipsis` | `ellipsis` | `[&>span:last-child]:truncate` |
| white-space | `nowrap` | `nowrap` | `[&>span:last-child]:truncate` |

**`>svg` — default**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| flex-shrink | `0` | `0` | `[&>svg]:shrink-0` |
| color | `#262626` / `#f5f5f5` | `sidebar-accent-foreground` | `[&>svg]:text-sidebar-accent-foreground` |

**`>svg:not([class*='size-'])` — default**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| width | `16px` | `calc(var(--spacing) * 4)` | `[&>svg:not([class*='size-'])]:size-4` |
| height | `16px` | `calc(var(--spacing) * 4)` | `[&>svg:not([class*='size-'])]:size-4` |

#### when `size === "sm"`

```txt
text-xs
```

Rules behind unevaluated conditions (container queries etc.): `@media (forced-colors: active)`

**element — default (changes vs base)**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| font-size | `12px` | `text-xs` | `text-xs` |
| line-height | `16px` | `var(--tw-leading, var(--text-xs--line-height))` | `text-xs` |

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| hover | color | `#262626` / `#f5f5f5` | `hover:text-sidebar-accent-foreground` |
| hover | background-color | `#0000000a` / `#ffffff0a` | `hover:bg-sidebar-accent` |
| focus-visible | box-shadow | `0 0 0 2px #a1a1a1` | `focus-visible:ring-2` |
| pressed | color | `#262626` / `#f5f5f5` | `active:text-sidebar-accent-foreground` |
| pressed | background-color | `#0000000a` / `#ffffff0a` | `active:bg-sidebar-accent` |
| disabled | pointer-events | `none` | `aria-disabled:pointer-events-none` |
| disabled | opacity | `50%` | `aria-disabled:opacity-50` |
| selected | color | `#262626` / `#f5f5f5` | `data-[active=true]:text-sidebar-accent-foreground` |
| selected | background-color | `#0000000a` / `#ffffff0a` | `data-[active=true]:bg-sidebar-accent` |
| in(.group[data-collapsible=icon]) | display | `none` | `group-data-[collapsible=icon]:hidden` |

**element — mobile differences**: height `32px`

#### when `size === "md"`

```txt
text-sm
```

Rules behind unevaluated conditions (container queries etc.): `@media (forced-colors: active)`

**element — default (changes vs base)**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| font-size | `14px` | `text-sm` | `text-sm` |
| line-height | `20px` | `var(--tw-leading, var(--text-sm--line-height))` | `text-sm` |

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| hover | color | `#262626` / `#f5f5f5` | `hover:text-sidebar-accent-foreground` |
| hover | background-color | `#0000000a` / `#ffffff0a` | `hover:bg-sidebar-accent` |
| focus-visible | box-shadow | `0 0 0 2px #a1a1a1` | `focus-visible:ring-2` |
| pressed | color | `#262626` / `#f5f5f5` | `active:text-sidebar-accent-foreground` |
| pressed | background-color | `#0000000a` / `#ffffff0a` | `active:bg-sidebar-accent` |
| disabled | pointer-events | `none` | `aria-disabled:pointer-events-none` |
| disabled | opacity | `50%` | `aria-disabled:opacity-50` |
| selected | color | `#262626` / `#f5f5f5` | `data-[active=true]:text-sidebar-accent-foreground` |
| selected | background-color | `#0000000a` / `#ffffff0a` | `data-[active=true]:bg-sidebar-accent` |
| in(.group[data-collapsible=icon]) | display | `none` | `group-data-[collapsible=icon]:hidden` |

**element — mobile differences**: height `32px`

