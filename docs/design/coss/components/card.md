# Card

Source: `apps/ui/registry/default/ui/card.tsx` @ `b2336c45f2b7` · docs: `apps/ui/content/docs/components/card.mdx`

> A content container for grouping related information.

Depends on: `@base-ui/react/merge-props`, `@base-ui/react/use-render`

Exports: `Card`, `CardFrame`, `CardFrameHeader`, `CardFrameTitle`, `CardFrameDescription`, `CardFrameAction`, `CardFrameFooter`, `CardHeader`, `CardTitle`, `CardDescription`, `CardAction`, `CardPanel`, `CardFooter`

## Behavior & keyboard

- Presentational container; no keyboard behavior.

## How to read

Each part lists the exact class strings, then **resolved** values for the desktop environment (≥640px so all `sm:` rules apply; also md/lg/xl; fine pointer; hover-capable). Values show `light` / `dark` when they differ. *State* tables only list properties that change versus the part's default. *Mobile* lists properties that differ at 375px / coarse pointer. `src` is the class that wins the cascade. See `../README.md` for conventions.

## Parts

### `Card` → `[data-slot=card]` · `useRender <div>` (line 13)

#### base

```txt
relative flex flex-col rounded-2xl border bg-card not-dark:bg-clip-padding text-card-foreground shadow-xs/5 before:pointer-events-none before:absolute before:inset-0 before:rounded-[calc(var(--radius-2xl)-1px)] before:shadow-[0_1px_--theme(--color-black/4%)] dark:before:shadow-[0_-1px_--theme(--color-white/6%)]
```

**element — default**

Key metrics: display `flex` · border-width `1px` · border-color `#00000014` / `#ffffff0f` (border) · border-radius `16px` (radius-2xl) · background-color `#ffffff` / `#191919` (card) · color `#262626` / `#f5f5f5` (card-foreground) · box-shadow `0 1px 2px 0 #0000000d`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| border-width | `1px` | `1px` | `border` |
| border-style | `solid` | `var(--tw-border-style)` | `border` |
| border-color | `#00000014` / `#ffffff0f` | `border` | `(globals @layer base) * border-border` |
| position | `relative` | `relative` | `relative` |
| display | `flex` | `flex` | `flex` |
| flex-direction | `column` | `column` | `flex-col` |
| border-radius | `16px` | `radius-2xl` | `rounded-2xl` |
| background-color | `#ffffff` / `#191919` | `card` | `bg-card` |
| color | `#262626` / `#f5f5f5` | `card-foreground` | `text-card-foreground` |
| box-shadow | `0 1px 2px 0 #0000000d` | `var(--tw-inset-shadow), var(--tw-inset-ring-shadow), var(--tw-ring-offset-shadow), var(--tw-ring-shadow), var(` | `shadow-xs/5` |
| background-clip | `padding-box` / — | `padding-box` | `not-dark:bg-clip-padding` |

**`::before` — default**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| content | `""` | `var(--tw-content)` | `before:shadow-[0_1px_--theme(--color-black/4%)]` |
| pointer-events | `none` | `none` | `before:pointer-events-none` |
| position | `absolute` | `absolute` | `before:absolute` |
| inset | `0px` | `calc(var(--spacing) * 0)` | `before:inset-0` |
| border-radius | `15px` | `calc(var(--radius-2xl) - 1px)` | `before:rounded-[calc(var(--radius-2xl)-1px)]` |
| box-shadow | `0 1px #0000000a` / `0 -1px #ffffff0f` | `var(--tw-inset-shadow), var(--tw-inset-ring-shadow), var(--tw-ring-offset-shadow), var(--tw-ring-shadow), var(` | `before:shadow-[0_1px_--theme(--color-black/4%)]` |

### `CardFrame` → `[data-slot=card-frame]` · `useRender <div>` (line 33)

#### base

```txt
relative flex flex-col rounded-2xl border bg-card not-dark:bg-clip-padding text-card-foreground shadow-xs/5 [--clip-bottom:-1rem] [--clip-top:-1rem] before:pointer-events-none before:absolute before:inset-0 before:rounded-[calc(var(--radius-2xl)-1px)] before:bg-muted/72 before:shadow-[0_1px_--theme(--color-black/4%)] has-data-[slot=table-container]:overflow-hidden *:data-[slot=card]:-m-px *:data-[slot=table-container]:-m-px *:data-[slot=table-container]:w-[calc(100%+2px)] *:not-first:data-[slot=card]:rounded-t-xl *:not-last:data-[slot=card]:rounded-b-xl *:data-[slot=card]:bg-clip-padding *:data-[slot=card]:shadow-none *:data-[slot=card]:before:hidden *:not-first:data-[slot=card]:before:rounded-t-[calc(var(--radius-xl)-1px)] *:not-last:data-[slot=card]:before:rounded-b-[calc(var(--radius-xl)-1px)] dark:before:shadow-[0_-1px_--theme(--color-white/6%)] *:data-[slot=card]:[clip-path:inset(var(--clip-top)_1px_var(--clip-bottom)_1px_round_calc(var(--radius-2xl)-1px))] *:data-[slot=card]:last:[--clip-bottom:1px] *:data-[slot=card]:first:[--clip-top:1px]
```

**element — default**

Key metrics: display `flex` · border-width `1px` · border-color `#00000014` / `#ffffff0f` (border) · border-radius `16px` (radius-2xl) · background-color `#ffffff` / `#191919` (card) · color `#262626` / `#f5f5f5` (card-foreground) · box-shadow `0 1px 2px 0 #0000000d`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| border-width | `1px` | `1px` | `border` |
| border-style | `solid` | `var(--tw-border-style)` | `border` |
| border-color | `#00000014` / `#ffffff0f` | `border` | `(globals @layer base) * border-border` |
| position | `relative` | `relative` | `relative` |
| display | `flex` | `flex` | `flex` |
| flex-direction | `column` | `column` | `flex-col` |
| border-radius | `16px` | `radius-2xl` | `rounded-2xl` |
| background-color | `#ffffff` / `#191919` | `card` | `bg-card` |
| color | `#262626` / `#f5f5f5` | `card-foreground` | `text-card-foreground` |
| box-shadow | `0 1px 2px 0 #0000000d` | `var(--tw-inset-shadow), var(--tw-inset-ring-shadow), var(--tw-ring-offset-shadow), var(--tw-ring-shadow), var(` | `shadow-xs/5` |
| background-clip | `padding-box` / — | `padding-box` | `not-dark:bg-clip-padding` |

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| has([data-slot=table-container]) | overflow | `hidden` | `has-data-[slot=table-container]:overflow-hidden` |

**`::before` — default**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| content | `""` | `var(--tw-content)` | `before:shadow-[0_1px_--theme(--color-black/4%)]` |
| pointer-events | `none` | `none` | `before:pointer-events-none` |
| position | `absolute` | `absolute` | `before:absolute` |
| inset | `0px` | `calc(var(--spacing) * 0)` | `before:inset-0` |
| border-radius | `15px` | `calc(var(--radius-2xl) - 1px)` | `before:rounded-[calc(var(--radius-2xl)-1px)]` |
| background-color | `#00000007` / `#ffffff07` | `muted/72%` | `before:bg-muted/72` |
| box-shadow | `0 1px #0000000a` / `0 -1px #ffffff0f` | `var(--tw-inset-shadow), var(--tw-inset-ring-shadow), var(--tw-ring-offset-shadow), var(--tw-ring-shadow), var(` | `before:shadow-[0_1px_--theme(--color-black/4%)]` |

**`> *:not(*:first-child)[data-slot="card"]` — default**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| border-top-left-radius | `14px` | `calc(var(--radius) + 4px)` | `*:not-first:data-[slot=card]:rounded-t-xl` |
| border-top-right-radius | `14px` | `calc(var(--radius) + 4px)` | `*:not-first:data-[slot=card]:rounded-t-xl` |

**`> *:not(*:first-child)[data-slot="card"]::before` — default**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| content | `""` | `var(--tw-content)` | `*:not-first:data-[slot=card]:before:rounded-t-[calc(var(--radius-xl)-1px)]` |
| border-top-left-radius | `13px` | `calc(var(--radius-xl) - 1px)` | `*:not-first:data-[slot=card]:before:rounded-t-[calc(var(--radius-xl)-1px)]` |
| border-top-right-radius | `13px` | `calc(var(--radius-xl) - 1px)` | `*:not-first:data-[slot=card]:before:rounded-t-[calc(var(--radius-xl)-1px)]` |

**`> *:not(*:last-child)[data-slot="card"]` — default**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| border-bottom-right-radius | `14px` | `calc(var(--radius) + 4px)` | `*:not-last:data-[slot=card]:rounded-b-xl` |
| border-bottom-left-radius | `14px` | `calc(var(--radius) + 4px)` | `*:not-last:data-[slot=card]:rounded-b-xl` |

**`> *:not(*:last-child)[data-slot="card"]::before` — default**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| content | `""` | `var(--tw-content)` | `*:not-last:data-[slot=card]:before:rounded-b-[calc(var(--radius-xl)-1px)]` |
| border-bottom-right-radius | `13px` | `calc(var(--radius-xl) - 1px)` | `*:not-last:data-[slot=card]:before:rounded-b-[calc(var(--radius-xl)-1px)]` |
| border-bottom-left-radius | `13px` | `calc(var(--radius-xl) - 1px)` | `*:not-last:data-[slot=card]:before:rounded-b-[calc(var(--radius-xl)-1px)]` |

**`> *[data-slot="card"]` — default**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| margin | `-1px` | `-1px` | `*:data-[slot=card]:-m-px` |
| background-clip | `padding-box` | `padding-box` | `*:data-[slot=card]:bg-clip-padding` |
| box-shadow | `none` | `var(--tw-inset-shadow), var(--tw-inset-ring-shadow), var(--tw-ring-offset-shadow), var(--tw-ring-shadow), var(` | `*:data-[slot=card]:shadow-none` |
| clip-path | `inset(-16px 1px -16px 1px round 15px)` | `inset(var(--clip-top) 1px var(--clip-bottom) 1px round calc(var(--radius-2xl) - 1px))` | `*:data-[slot=card]:[clip-path:inset(var(--clip-top)_1px_var(--clip-bottom)_1px_round_calc(var(--radius-2xl)-1px))]` |

**`> *[data-slot="card"]::before` — default**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| content | `""` | `var(--tw-content)` | `*:data-[slot=card]:before:hidden` |
| display | `none` | `none` | `*:data-[slot=card]:before:hidden` |

**`> *[data-slot="table-container"]` — default**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| margin | `-1px` | `-1px` | `*:data-[slot=table-container]:-m-px` |
| width | `calc(100% + 2px)` | `calc(100% + 2px)` | `*:data-[slot=table-container]:w-[calc(100%+2px)]` |

### `CardFrameHeader` → `[data-slot=card-frame-header]` · `useRender <div>` (line 53)

#### base

```txt
relative flex grid auto-rows-min grid-rows-[auto_auto] flex-col items-start gap-x-4 px-6 py-4 has-data-[slot=card-frame-action]:grid-cols-[1fr_auto]
```

**element — default**

Key metrics: display `grid` · padding-inline `24px` · padding-block `16px` · column-gap `16px`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| position | `relative` | `relative` | `relative` |
| display | `grid` | `grid` | `grid` |
| grid-auto-rows | `min-content` | `min-content` | `auto-rows-min` |
| grid-template-rows | `auto auto` | `auto auto` | `grid-rows-[auto_auto]` |
| flex-direction | `column` | `column` | `flex-col` |
| align-items | `flex-start` | `flex-start` | `items-start` |
| column-gap | `16px` | `calc(var(--spacing) * 4)` | `gap-x-4` |
| padding-inline | `24px` | `calc(var(--spacing) * 6)` | `px-6` |
| padding-block | `16px` | `calc(var(--spacing) * 4)` | `py-4` |

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| has([data-slot=card-frame-action]) | grid-template-columns | `1fr auto` | `has-data-[slot=card-frame-action]:grid-cols-[1fr_auto]` |

### `CardFrameTitle` → `[data-slot=card-frame-title]` · `useRender <div>` (line 73)

#### base

```txt
self-center font-semibold text-sm
```

**element — default**

Key metrics: font-size `14px` (text-sm) · line-height `20px` · font-weight `600` (font-weight-semibold)

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| align-self | `center` | `center` | `self-center` |
| font-size | `14px` | `text-sm` | `text-sm` |
| line-height | `20px` | `var(--tw-leading, var(--text-sm--line-height))` | `text-sm` |
| font-weight | `600` | `font-weight-semibold` | `font-semibold` |

### `CardFrameDescription` → `[data-slot=card-frame-description]` · `useRender <div>` (line 90)

#### base

```txt
self-center text-muted-foreground text-sm
```

**element — default**

Key metrics: color `#686868` / `#818181` (muted-foreground) · font-size `14px` (text-sm) · line-height `20px`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| align-self | `center` | `center` | `self-center` |
| font-size | `14px` | `text-sm` | `text-sm` |
| line-height | `20px` | `var(--tw-leading, var(--text-sm--line-height))` | `text-sm` |
| color | `#686868` / `#818181` | `muted-foreground` | `text-muted-foreground` |

### `CardFrameAction` → `[data-slot=card-frame-action]` · `useRender <div>` (line 107)

#### base

```txt
col-start-2 nth-3:row-span-2 nth-3:row-start-1 inline-flex self-center justify-self-end
```

**element — default**

Key metrics: display `inline-flex`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| grid-column-start | `2` | `2` | `col-start-2` |
| display | `inline-flex` | `inline-flex` | `inline-flex` |
| align-self | `center` | `center` | `self-center` |
| justify-self | `flex-end` | `flex-end` | `justify-self-end` |

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| :nth-child(3) | grid-row | `span 2 / span 2` | `nth-3:row-span-2` |
| :nth-child(3) | grid-row-start | `1` | `nth-3:row-start-1` |

### `CardFrameFooter` → `[data-slot=card-frame-footer]` · `useRender <div>` (line 127)

#### base

```txt
px-6 py-4
```

**element — default**

Key metrics: padding-inline `24px` · padding-block `16px`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| padding-inline | `24px` | `calc(var(--spacing) * 6)` | `px-6` |
| padding-block | `16px` | `calc(var(--spacing) * 4)` | `py-4` |

### `CardHeader` → `[data-slot=card-header]` · `useRender <div>` (line 144)

#### base

```txt
grid auto-rows-min grid-rows-[auto_auto] items-start gap-1.5 p-6 in-[[data-slot=card]:has(>[data-slot=card-panel])]:pb-4 has-data-[slot=card-action]:grid-cols-[1fr_auto]
```

**element — default**

Key metrics: display `grid` · padding `24px` · gap `6px`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| display | `grid` | `grid` | `grid` |
| grid-auto-rows | `min-content` | `min-content` | `auto-rows-min` |
| grid-template-rows | `auto auto` | `auto auto` | `grid-rows-[auto_auto]` |
| align-items | `flex-start` | `flex-start` | `items-start` |
| gap | `6px` | `calc(var(--spacing) * 1.5)` | `gap-1.5` |
| padding | `24px` | `calc(var(--spacing) * 6)` | `p-6` |

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| in(:where(*:is([data-slot=card]:has(>[data-slot=card-panel])))) | padding-bottom | `16px` | `in-[[data-slot=card]:has(>[data-slot=card-panel])]:pb-4` |
| has([data-slot=card-action]) | grid-template-columns | `1fr auto` | `has-data-[slot=card-action]:grid-cols-[1fr_auto]` |

### `CardTitle` → `[data-slot=card-title]` · `useRender <div>` (line 164)

#### base

```txt
font-heading font-semibold text-lg leading-none
```

**element — default**

Key metrics: font-size `18px` (text-lg) · line-height `18px` · font-weight `600` (font-weight-semibold)

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| font-family | `ui-sans-serif, system-ui, sans-serif, 'Apple Color Emoji', 'Segoe UI Emoji', 'Segoe UI Symbol', 'Noto Color Emoji'` | `font-sans` | `font-heading` |
| font-size | `18px` | `text-lg` | `text-lg` |
| line-height | `18px` | `1` | `leading-none` |
| font-weight | `600` | `font-weight-semibold` | `font-semibold` |

### `CardDescription` → `[data-slot=card-description]` · `useRender <div>` (line 181)

#### base

```txt
text-muted-foreground text-sm
```

**element — default**

Key metrics: color `#686868` / `#818181` (muted-foreground) · font-size `14px` (text-sm) · line-height `20px`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| font-size | `14px` | `text-sm` | `text-sm` |
| line-height | `20px` | `var(--tw-leading, var(--text-sm--line-height))` | `text-sm` |
| color | `#686868` / `#818181` | `muted-foreground` | `text-muted-foreground` |

### `CardAction` → `[data-slot=card-action]` · `useRender <div>` (line 198)

#### base

```txt
col-start-2 row-span-2 row-start-1 inline-flex self-start justify-self-end
```

**element — default**

Key metrics: display `inline-flex`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| grid-column-start | `2` | `2` | `col-start-2` |
| grid-row | `span 2 / span 2` | `span 2 / span 2` | `row-span-2` |
| grid-row-start | `1` | `1` | `row-start-1` |
| display | `inline-flex` | `inline-flex` | `inline-flex` |
| align-self | `flex-start` | `flex-start` | `self-start` |
| justify-self | `flex-end` | `flex-end` | `justify-self-end` |

### `CardPanel` → `[data-slot=card-panel]` · `useRender <div>` (line 218)

#### base

```txt
flex-1 p-6 in-[[data-slot=card]:has(>[data-slot=card-header]:not(.border-b))]:pt-0 in-[[data-slot=card]:has(>[data-slot=card-footer]:not(.border-t))]:pb-0
```

**element — default**

Key metrics: padding `24px`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| flex | `1` | `1` | `flex-1` |
| padding | `24px` | `calc(var(--spacing) * 6)` | `p-6` |

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| in(:where(*:is([data-slot=card]:has(>[data-slot=card-header]:not(.border-b))))) | padding-top | `0px` | `in-[[data-slot=card]:has(>[data-slot=card-header]:not(.border-b))]:pt-0` |
| in(:where(*:is([data-slot=card]:has(>[data-slot=card-footer]:not(.border-t))))) | padding-bottom | `0px` | `in-[[data-slot=card]:has(>[data-slot=card-footer]:not(.border-t))]:pb-0` |

### `CardFooter` → `[data-slot=card-footer]` · `useRender <div>` (line 238)

#### base

```txt
flex items-center p-6 in-[[data-slot=card]:has(>[data-slot=card-panel])]:pt-4
```

**element — default**

Key metrics: display `flex` · padding `24px`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| display | `flex` | `flex` | `flex` |
| align-items | `center` | `center` | `items-center` |
| padding | `24px` | `calc(var(--spacing) * 6)` | `p-6` |

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| in(:where(*:is([data-slot=card]:has(>[data-slot=card-panel])))) | padding-top | `16px` | `in-[[data-slot=card]:has(>[data-slot=card-panel])]:pt-4` |

