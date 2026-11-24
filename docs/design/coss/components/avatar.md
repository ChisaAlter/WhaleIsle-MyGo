# Avatar

Source: `apps/ui/registry/default/ui/avatar.tsx` @ `b2336c45f2b7` · docs: `apps/ui/content/docs/components/avatar.mdx` · Base UI: https://base-ui.com/react/components/avatar#api-reference

> An image element with a fallback for representing the user.

Depends on: `@base-ui/react/avatar`

Exports: `Avatar`, `AvatarImage`, `AvatarFallback`, `AvatarGroup`, `AvatarGroupCount`

Module constants: `AVATAR_GROUP_CUT = url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' viewBox='0 0 2 2'%3E%3Ccircle cx='1' cy='1' r='1'/%3E%3C/svg%3E")`

## Behavior & keyboard

- Image with fallback; fallback renders after optional delay when image fails/loads slowly. No keyboard behavior.

## How to read

Each part lists the exact class strings, then **resolved** values for the desktop environment (≥640px so all `sm:` rules apply; also md/lg/xl; fine pointer; hover-capable). Values show `light` / `dark` when they differ. *State* tables only list properties that change versus the part's default. *Mobile* lists properties that differ at 375px / coarse pointer. `src` is the class that wins the cascade. See `../README.md` for conventions.

## Parts

### `Avatar` → `[data-slot=avatar]` · `<AvatarPrimitive.Root>` (line 15)

#### base

```txt
relative isolate inline-flex size-(--avatar-size,--spacing(8)) shrink-0 select-none items-center justify-center overflow-hidden rounded-full align-middle font-medium text-[calc(var(--avatar-size,--spacing(8))*0.375)]/none
```

**element — default**

Key metrics: display `inline-flex` · height `32px` · width `32px` · border-radius `9999px` · font-size `12px` · line-height `12px` · font-weight `500` (font-weight-medium)

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| position | `relative` | `relative` | `relative` |
| isolation | `isolate` | `isolate` | `isolate` |
| display | `inline-flex` | `inline-flex` | `inline-flex` |
| width | `32px` | `var(--avatar-size,calc(var(--spacing) * 8))` | `size-(--avatar-size,--spacing(8))` |
| height | `32px` | `var(--avatar-size,calc(var(--spacing) * 8))` | `size-(--avatar-size,--spacing(8))` |
| flex-shrink | `0` | `0` | `shrink-0` |
| align-items | `center` | `center` | `items-center` |
| justify-content | `center` | `center` | `justify-center` |
| overflow | `hidden` | `hidden` | `overflow-hidden` |
| border-radius | `9999px` | `calc(infinity * 1px)` | `rounded-full` |
| vertical-align | `middle` | `middle` | `align-middle` |
| font-size | `12px` | `calc(var(--avatar-size,calc(var(--spacing) * 8)) * 0.375)` | `text-[calc(var(--avatar-size,--spacing(8))*0.375)]/none` |
| line-height | `12px` | `1` | `text-[calc(var(--avatar-size,--spacing(8))*0.375)]/none` |
| font-weight | `500` | `font-weight-medium` | `font-medium` |
| -webkit-user-select | `none` | `none` | `select-none` |
| user-select | `none` | `none` | `select-none` |

### `AvatarImage` → `[data-slot=avatar-image]` · `<AvatarPrimitive.Image>` (line 31)

#### base

```txt
absolute inset-0 z-10 size-full rounded-[inherit] object-cover data-error:invisible data-loading:invisible
```

**element — default**

Key metrics: height `100%` · width `100%` · border-radius `inherit` · z-index `10`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| position | `absolute` | `absolute` | `absolute` |
| inset | `0px` | `calc(var(--spacing) * 0)` | `inset-0` |
| z-index | `10` | `10` | `z-10` |
| width | `100%` | `100%` | `size-full` |
| height | `100%` | `100%` | `size-full` |
| border-radius | `inherit` | `inherit` | `rounded-[inherit]` |
| object-fit | `cover` | `cover` | `object-cover` |

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| [data-error] | visibility | `hidden` | `data-error:invisible` |
| [data-loading] | visibility | `hidden` | `data-loading:invisible` |

### `AvatarFallback` → `[data-slot=avatar-fallback]` · `<AvatarPrimitive.Fallback>` (line 47)

#### base

```txt
absolute inset-0 flex size-full items-center justify-center rounded-[inherit] bg-border
```

**element — default**

Key metrics: display `flex` · height `100%` · width `100%` · border-radius `inherit` · background-color `#00000014` / `#ffffff0f` (border)

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| position | `absolute` | `absolute` | `absolute` |
| inset | `0px` | `calc(var(--spacing) * 0)` | `inset-0` |
| display | `flex` | `flex` | `flex` |
| width | `100%` | `100%` | `size-full` |
| height | `100%` | `100%` | `size-full` |
| align-items | `center` | `center` | `items-center` |
| justify-content | `center` | `center` | `justify-center` |
| border-radius | `inherit` | `inherit` | `rounded-[inherit]` |
| background-color | `#00000014` / `#ffffff0f` | `border` | `bg-border` |

### `AvatarGroup` → `[data-slot=avatar-group]` · `<div>` (line 69)

Attributes: `data-topmost={topmost}`, `style={{ "--avatar-group-cut-image": AVATAR_GROUP_CUT, ...style, } as React.CSSProperties}`

Prop defaults: `topmost="first"`

#### base

```txt
flex w-fit items-center [--avatar-group-size:var(--avatar-size,--spacing(8))] [--avatar-group-gap:var(--avatar-gap,calc(var(--avatar-group-size)/16))] [--avatar-group-overlap:var(--avatar-overlap,calc(var(--avatar-group-size)*0.3))] [--avatar-group-cut-end:calc(var(--avatar-group-size)*1.5-var(--avatar-group-overlap))] [--avatar-group-cut-radius:calc(var(--avatar-group-size)/2+var(--avatar-group-gap))] [--avatar-group-cut-start:calc(var(--avatar-group-overlap)-var(--avatar-group-size)/2)] [--avatar-group-mask-position:0_0,calc(var(--avatar-group-cut-x)-var(--avatar-group-cut-radius))_50%] [--avatar-group-mask-size:100%_100%,calc(var(--avatar-group-cut-radius)*2)_calc(var(--avatar-group-cut-radius)*2)] [--avatar-group-mask:none] supports-[mask-composite:subtract]:[--avatar-group-mask:var(--avatar-group-cut-image),var(--avatar-group-cut-image)] [--avatar-group-nudge:var(--avatar-nudge,calc((var(--avatar-group-overlap)+var(--avatar-group-gap))*0.3))] *:not-first:-ms-(--avatar-group-overlap)
```

**element — default**

Key metrics: display `flex` · width `fit-content`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| display | `flex` | `flex` | `flex` |
| width | `fit-content` | `fit-content` | `w-fit` |
| align-items | `center` | `center` | `items-center` |

**`> *:not(*:first-child)` — default**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| margin-inline-start | `-9.6px` | `calc(var(--avatar-group-overlap) * -1)` | `*:not-first:-ms-(--avatar-group-overlap)` |

#### when `topmost === "first"`

```txt
*:not-first:mask-(--avatar-group-mask) *:not-first:mask-subtract *:not-first:mask-position-(--avatar-group-mask-position) *:not-first:mask-no-repeat *:not-first:mask-size-(--avatar-group-mask-size) [--avatar-group-cut-x:var(--avatar-group-cut-start)] *:not-first:*:data-[slot=avatar-fallback]:ps-[calc(var(--avatar-group-nudge)*2)] supports-[mask-composite:subtract]:*:not-first:rounded-none rtl:[--avatar-group-cut-x:var(--avatar-group-cut-end)]
```

**`> *:not(*:first-child)` — default (changes vs base)**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| mask-image | `var(--avatar-group-cut-image), var(--avatar-group-cut-image)` | `runtime var --avatar-group-mask` | `*:not-first:mask-(--avatar-group-mask)` |
| mask-composite | `subtract` | `subtract` | `*:not-first:mask-subtract` |
| mask-size | `100% 100%, 36px 36px` | `runtime var --avatar-group-mask-size` | `*:not-first:mask-size-(--avatar-group-mask-size)` |
| mask-position | `0 0, calc(var(--avatar-group-cut-x) - 18px) 50%` | `runtime var --avatar-group-mask-position` | `*:not-first:mask-position-(--avatar-group-mask-position)` |
| mask-repeat | `no-repeat` | `no-repeat` | `*:not-first:mask-no-repeat` |
| border-radius | `0` | `0` | `supports-[mask-composite:subtract]:*:not-first:rounded-none` |

**`> *:not(*:first-child):is(& > *)[data-slot="avatar-fallback"]` — default (changes vs base)**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| padding-inline-start | `6.96px` | `calc(var(--avatar-group-nudge) * 2)` | `*:not-first:*:data-[slot=avatar-fallback]:ps-[calc(var(--avatar-group-nudge)*2)]` |

#### when `!(topmost === "first")`

```txt
*:not-last:mask-(--avatar-group-mask) *:not-last:mask-subtract *:not-last:mask-position-(--avatar-group-mask-position) *:not-last:mask-no-repeat *:not-last:mask-size-(--avatar-group-mask-size) [--avatar-group-cut-x:var(--avatar-group-cut-end)] *:not-last:*:data-[slot=avatar-fallback]:pe-[calc(var(--avatar-group-nudge)*2)] supports-[mask-composite:subtract]:*:not-last:rounded-none rtl:[--avatar-group-cut-x:var(--avatar-group-cut-start)]
```

**`> *:not(*:last-child)` — default (changes vs base)**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| mask-image | `var(--avatar-group-cut-image), var(--avatar-group-cut-image)` | `runtime var --avatar-group-mask` | `*:not-last:mask-(--avatar-group-mask)` |
| mask-composite | `subtract` | `subtract` | `*:not-last:mask-subtract` |
| mask-size | `100% 100%, 36px 36px` | `runtime var --avatar-group-mask-size` | `*:not-last:mask-size-(--avatar-group-mask-size)` |
| mask-position | `0 0, calc(var(--avatar-group-cut-x) - 18px) 50%` | `runtime var --avatar-group-mask-position` | `*:not-last:mask-position-(--avatar-group-mask-position)` |
| mask-repeat | `no-repeat` | `no-repeat` | `*:not-last:mask-no-repeat` |
| border-radius | `0` | `0` | `supports-[mask-composite:subtract]:*:not-last:rounded-none` |

**`> *:not(*:last-child):is(& > *)[data-slot="avatar-fallback"]` — default (changes vs base)**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| padding-inline-end | `6.96px` | `calc(var(--avatar-group-nudge) * 2)` | `*:not-last:*:data-[slot=avatar-fallback]:pe-[calc(var(--avatar-group-nudge)*2)]` |

### `AvatarGroupCount` → `[data-slot=avatar-group-count]` · `<Avatar>` (line 108)

#### base

```txt
text-muted-foreground
```

**element — default**

Key metrics: color `#686868` / `#818181` (muted-foreground)

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| color | `#686868` / `#818181` | `muted-foreground` | `text-muted-foreground` |

