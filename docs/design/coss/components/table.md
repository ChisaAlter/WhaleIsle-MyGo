# Table

Source: `apps/ui/registry/default/ui/table.tsx` @ `b2336c45f2b7` · docs: `apps/ui/content/docs/components/table.mdx`

> A simple table component for displaying tabular data.

Depends on: `@base-ui/react/merge-props`, `@base-ui/react/use-render`

Exports: `Table`, `TableHeader`, `TableBody`, `TableFooter`, `TableRow`, `TableHead`, `TableCell`, `TableCaption`

## Behavior & keyboard

- Native table semantics; rows may be hover-highlighted; no keyboard behavior.

## How to read

Each part lists the exact class strings, then **resolved** values for the desktop environment (≥640px so all `sm:` rules apply; also md/lg/xl; fine pointer; hover-capable). Values show `light` / `dark` when they differ. *State* tables only list properties that change versus the part's default. *Mobile* lists properties that differ at 375px / coarse pointer. `src` is the class that wins the cascade. See `../README.md` for conventions.

## Parts

### `Table` → `[data-slot=table-container]` · `useRender <div>` (line 21)

Attributes: `data-variant={variant}`

Prop defaults: `variant="default"`

#### base

```txt
relative w-full overflow-x-auto
```

**element — default**

Key metrics: width `100%`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| position | `relative` | `relative` | `relative` |
| width | `100%` | `100%` | `w-full` |
| overflow-x | `auto` | `auto` | `overflow-x-auto` |

### `Table` → `[data-slot=table]` · `<table>` (line 23)

Prop defaults: `variant="default"`

#### base

```txt
w-full caption-bottom in-data-[variant=card]:border-separate in-data-[variant=card]:border-spacing-0 text-sm
```

**element — default**

Key metrics: width `100%` · font-size `14px` (text-sm) · line-height `20px`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| width | `100%` | `100%` | `w-full` |
| caption-side | `bottom` | `bottom` | `caption-bottom` |
| font-size | `14px` | `text-sm` | `text-sm` |
| line-height | `20px` | `var(--tw-leading, var(--text-sm--line-height))` | `text-sm` |

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| in([data-variant=card]) | border-collapse | `separate` | `in-data-[variant=card]:border-separate` |
| in([data-variant=card]) | border-spacing | `0px 0px` | `in-data-[variant=card]:border-spacing-0` |

### `TableHeader` → `[data-slot=table-header]` · `<thead>` (line 49)

#### base

```txt
[&_tr]:border-b
```

**`tr` — default**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| border-bottom-style | `solid` | `var(--tw-border-style)` | `[&_tr]:border-b` |
| border-bottom-width | `1px` | `1px` | `[&_tr]:border-b` |

### `TableBody` → `[data-slot=table-body]` · `<tbody>` (line 62)

#### base

```txt
relative in-data-[variant=card]:rounded-xl in-data-[variant=card]:shadow-xs/5 before:pointer-events-none before:absolute before:inset-px not-in-data-[variant=card]:before:hidden before:rounded-[calc(var(--radius-xl)-1px)] before:shadow-[0_1px_--theme(--color-black/4%)] dark:before:shadow-[0_-1px_--theme(--color-white/8%)] [&_tr:last-child]:border-0 in-data-[variant=card]:*:[tr]:border-0 in-data-[variant=card]:*:[tr]:*:[td]:border-b in-data-[variant=card]:*:[tr]:*:[td]:bg-card in-data-[variant=card]:*:[tr]:first:*:[td]:first:rounded-ss-xl in-data-[variant=card]:*:[tr]:*:[td]:first:border-s in-data-[variant=card]:*:[tr]:first:*:[td]:border-t in-data-[variant=card]:*:[tr]:last:*:[td]:last:rounded-ee-xl in-data-[variant=card]:*:[tr]:*:[td]:last:border-e in-data-[variant=card]:*:[tr]:first:*:[td]:last:rounded-se-xl in-data-[variant=card]:*:[tr]:last:*:[td]:first:rounded-es-xl in-data-[variant=card]:*:[tr]:hover:*:[td]:bg-[color-mix(in_srgb,var(--card),var(--color-black)_2%)] in-data-[variant=card]:*:[tr]:data-[state=selected]:*:[td]:bg-[color-mix(in_srgb,var(--card),var(--color-black)_4%)] dark:in-data-[variant=card]:*:[tr]:data-[state=selected]:*:[td]:bg-[color-mix(in_srgb,var(--card),var(--color-white)_4%)] dark:in-data-[variant=card]:*:[tr]:hover:*:[td]:bg-[color-mix(in_srgb,var(--card),var(--color-white)_2%)]
```

**element — default**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| position | `relative` | `relative` | `relative` |

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| in([data-variant=card]) | border-radius | `14px` | `in-data-[variant=card]:rounded-xl` |
| in([data-variant=card]) | box-shadow | `0 1px 2px 0 #0000000d` | `in-data-[variant=card]:shadow-xs/5` |

**`::before` — default**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| content | `""` | `var(--tw-content)` | `not-in-data-[variant=card]:before:hidden` |
| pointer-events | `none` | `none` | `before:pointer-events-none` |
| position | `absolute` | `absolute` | `before:absolute` |
| inset | `1px` | `1px` | `before:inset-px` |
| border-radius | `13px` | `calc(var(--radius-xl) - 1px)` | `before:rounded-[calc(var(--radius-xl)-1px)]` |
| box-shadow | `0 1px #0000000a` / `0 -1px #ffffff14` | `var(--tw-inset-shadow), var(--tw-inset-ring-shadow), var(--tw-ring-offset-shadow), var(--tw-ring-shadow), var(` | `before:shadow-[0_1px_--theme(--color-black/4%)]` |
| display | `none` | `none` | `not-in-data-[variant=card]:before:hidden` |

**`::before` — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| in([data-variant=card]) | display | `(unset)` | `(removed)` |

**`> *:is(tr)` — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| in([data-variant=card]) | border-style | `solid` | `in-data-[variant=card]:*:[tr]:border-0` |
| in([data-variant=card]) | border-width | `0px` | `in-data-[variant=card]:*:[tr]:border-0` |

**`> *:is(tr):first-child:is(& > *):is(td)` — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| in([data-variant=card]) | border-top-style | `solid` | `in-data-[variant=card]:*:[tr]:first:*:[td]:border-t` |
| in([data-variant=card]) | border-top-width | `1px` | `in-data-[variant=card]:*:[tr]:first:*:[td]:border-t` |

**`> *:is(tr):first-child:is(& > *):is(td):first-child` — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| in([data-variant=card]) | border-start-start-radius | `14px` | `in-data-[variant=card]:*:[tr]:first:*:[td]:first:rounded-ss-xl` |

**`> *:is(tr):first-child:is(& > *):is(td):last-child` — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| in([data-variant=card]) | border-start-end-radius | `14px` | `in-data-[variant=card]:*:[tr]:first:*:[td]:last:rounded-se-xl` |

**`> *:is(tr):hover:is(& > *):is(td)` — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| in([data-variant=card]) | background-color | `#fafafa` / `#1d1d1d` | `in-data-[variant=card]:*:[tr]:hover:*:[td]:bg-[color-mix(in_srgb,var(--card),var(--color-black)_2%)]` |

**`> *:is(tr):is(& > *):is(td)` — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| in([data-variant=card]) | border-bottom-style | `solid` | `in-data-[variant=card]:*:[tr]:*:[td]:border-b` |
| in([data-variant=card]) | border-bottom-width | `1px` | `in-data-[variant=card]:*:[tr]:*:[td]:border-b` |
| in([data-variant=card]) | background-color | `#ffffff` / `#191919` | `in-data-[variant=card]:*:[tr]:*:[td]:bg-card` |

**`> *:is(tr):is(& > *):is(td):first-child` — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| in([data-variant=card]) | border-inline-start-style | `solid` | `in-data-[variant=card]:*:[tr]:*:[td]:first:border-s` |
| in([data-variant=card]) | border-inline-start-width | `1px` | `in-data-[variant=card]:*:[tr]:*:[td]:first:border-s` |

**`> *:is(tr):is(& > *):is(td):last-child` — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| in([data-variant=card]) | border-inline-end-style | `solid` | `in-data-[variant=card]:*:[tr]:*:[td]:last:border-e` |
| in([data-variant=card]) | border-inline-end-width | `1px` | `in-data-[variant=card]:*:[tr]:*:[td]:last:border-e` |

**`> *:is(tr):last-child:is(& > *):is(td):first-child` — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| in([data-variant=card]) | border-end-start-radius | `14px` | `in-data-[variant=card]:*:[tr]:last:*:[td]:first:rounded-es-xl` |

**`> *:is(tr):last-child:is(& > *):is(td):last-child` — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| in([data-variant=card]) | border-end-end-radius | `14px` | `in-data-[variant=card]:*:[tr]:last:*:[td]:last:rounded-ee-xl` |

**`> *:is(tr)[data-state="selected"]:is(& > *):is(td)` — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| in([data-variant=card]) | background-color | `#f5f5f5` / `#222222` | `in-data-[variant=card]:*:[tr]:data-[state=selected]:*:[td]:bg-[color-mix(in_srgb,var(--card),var(--color-black)_4%)]` |

**`tr:last-child` — default**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| border-style | `solid` | `var(--tw-border-style)` | `[&_tr:last-child]:border-0` |
| border-width | `0px` | `0px` | `[&_tr:last-child]:border-0` |

### `TableFooter` → `[data-slot=table-footer]` · `<tfoot>` (line 78)

#### base

```txt
border-t in-data-[variant=card]:border-none bg-transparent not-in-data-[variant=card]:bg-[color-mix(in_srgb,var(--card),var(--color-black)_2%)] font-medium dark:not-in-data-[variant=card]:bg-[color-mix(in_srgb,var(--card),var(--color-white)_2%)] [&>tr]:last:border-b-0
```

**element — default**

Key metrics: border-top-width `1px` · background-color `#fafafa` / `#1d1d1d` · font-weight `500` (font-weight-medium)

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| border-top-style | `solid` | `var(--tw-border-style)` | `border-t` |
| border-top-width | `1px` | `1px` | `border-t` |
| background-color | `#fafafa` / `#1d1d1d` | `color-mix(in srgb,var(--card),var(--color-black) 2%)` | `not-in-data-[variant=card]:bg-[color-mix(in_srgb,var(--card),var(--color-black)_2%)]` |
| font-weight | `500` | `font-weight-medium` | `font-medium` |

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| in([data-variant=card]) | border-top-style | `none` | `border-t` |
| in([data-variant=card]) | background-color | `#00000000` | `bg-transparent` |
| in([data-variant=card]) | border-style | `none` | `in-data-[variant=card]:border-none` |

**`>tr:last-child` — default**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| border-bottom-style | `solid` | `var(--tw-border-style)` | `[&>tr]:last:border-b-0` |
| border-bottom-width | `0px` | `0px` | `[&>tr]:last:border-b-0` |

### `TableRow` → `[data-slot=table-row]` · `<tr>` (line 94)

#### base

```txt
relative border-b not-in-data-[variant=card]:hover:bg-[color-mix(in_srgb,var(--background),var(--color-black)_2%)] not-in-data-[variant=card]:data-[state=selected]:bg-[color-mix(in_srgb,var(--background),var(--color-black)_4%)] dark:not-in-data-[variant=card]:data-[state=selected]:bg-[color-mix(in_srgb,var(--background),var(--color-white)_4%)] dark:not-in-data-[variant=card]:hover:bg-[color-mix(in_srgb,var(--background),var(--color-white)_2%)]
```

**element — default**

Key metrics: border-bottom-width `1px`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| position | `relative` | `relative` | `relative` |
| border-bottom-style | `solid` | `var(--tw-border-style)` | `border-b` |
| border-bottom-width | `1px` | `1px` | `border-b` |

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| hover | background-color | `#fafafa` / `#191919` | `not-in-data-[variant=card]:hover:bg-[color-mix(in_srgb,var(--background),var(--color-black)_2%)]` |
| selected | background-color | `#f5f5f5` / `#1d1d1d` | `not-in-data-[variant=card]:data-[state=selected]:bg-[color-mix(in_srgb,var(--background),var(--color-black)_4%)]` |

### `TableHead` → `[data-slot=table-head]` · `<th>` (line 110)

#### base

```txt
h-10 whitespace-nowrap px-2.5 text-left align-middle font-medium text-muted-foreground leading-none has-[[role=checkbox]]:w-px last:has-[[role=checkbox]]:ps-0 first:has-[[role=checkbox]]:pe-0
```

**element — default**

Key metrics: height `40px` · padding-inline `10px` · color `#686868` / `#818181` (muted-foreground) · line-height `1` · font-weight `500` (font-weight-medium)

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| height | `40px` | `calc(var(--spacing) * 10)` | `h-10` |
| padding-inline | `10px` | `calc(var(--spacing) * 2.5)` | `px-2.5` |
| text-align | `left` | `left` | `text-left` |
| vertical-align | `middle` | `middle` | `align-middle` |
| line-height | `1` | `1` | `leading-none` |
| font-weight | `500` | `font-weight-medium` | `font-medium` |
| white-space | `nowrap` | `nowrap` | `whitespace-nowrap` |
| color | `#686868` / `#818181` | `muted-foreground` | `text-muted-foreground` |

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| has([role=checkbox]) | width | `1px` | `has-[[role=checkbox]]:w-px` |
| :last-child + has([role=checkbox]) | width | `1px` | `has-[[role=checkbox]]:w-px` |
| :last-child + has([role=checkbox]) | padding-inline-start | `0px` | `last:has-[[role=checkbox]]:ps-0` |
| :first-child + has([role=checkbox]) | width | `1px` | `has-[[role=checkbox]]:w-px` |
| :first-child + has([role=checkbox]) | padding-inline-end | `0px` | `first:has-[[role=checkbox]]:pe-0` |

### `TableCell` → `[data-slot=table-cell]` · `<td>` (line 126)

#### base

```txt
whitespace-nowrap bg-clip-padding p-2.5 in-data-[slot=table-footer]:py-3.5 align-middle leading-none in-data-[variant=card]:first:ps-[calc(--spacing(2.5)-1px)] in-data-[variant=card]:last:pe-[calc(--spacing(2.5)-1px)] has-[[role=checkbox]]:w-px last:has-[[role=checkbox]]:ps-0 first:has-[[role=checkbox]]:pe-0
```

**element — default**

Key metrics: padding `10px` · line-height `1`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| background-clip | `padding-box` | `padding-box` | `bg-clip-padding` |
| padding | `10px` | `calc(var(--spacing) * 2.5)` | `p-2.5` |
| vertical-align | `middle` | `middle` | `align-middle` |
| line-height | `1` | `1` | `leading-none` |
| white-space | `nowrap` | `nowrap` | `whitespace-nowrap` |

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| in([data-slot=table-footer]) | padding-block | `14px` | `in-data-[slot=table-footer]:py-3.5` |
| has([role=checkbox]) | width | `1px` | `has-[[role=checkbox]]:w-px` |
| :first-child + in([data-variant=card]) | padding-inline-start | `9px` | `in-data-[variant=card]:first:ps-[calc(--spacing(2.5)-1px)]` |
| :last-child + in([data-variant=card]) | padding-inline-end | `9px` | `in-data-[variant=card]:last:pe-[calc(--spacing(2.5)-1px)]` |
| :last-child + has([role=checkbox]) | width | `1px` | `has-[[role=checkbox]]:w-px` |
| :last-child + has([role=checkbox]) | padding-inline-start | `0px` | `last:has-[[role=checkbox]]:ps-0` |
| :first-child + has([role=checkbox]) | width | `1px` | `has-[[role=checkbox]]:w-px` |
| :first-child + has([role=checkbox]) | padding-inline-end | `0px` | `first:has-[[role=checkbox]]:pe-0` |

### `TableCaption` → `[data-slot=table-caption]` · `<caption>` (line 142)

#### base

```txt
in-data-[variant=card]:my-4 mt-4 text-muted-foreground text-sm
```

**element — default**

Key metrics: color `#686868` / `#818181` (muted-foreground) · font-size `14px` (text-sm) · line-height `20px`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| margin-top | `16px` | `calc(var(--spacing) * 4)` | `mt-4` |
| font-size | `14px` | `text-sm` | `text-sm` |
| line-height | `20px` | `var(--tw-leading, var(--text-sm--line-height))` | `text-sm` |
| color | `#686868` / `#818181` | `muted-foreground` | `text-muted-foreground` |

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| in([data-variant=card]) | margin-block | `16px` | `in-data-[variant=card]:my-4` |

## Other class strings in this file

Class strings not bound directly to a JSX `className` (lookup tables, maps, helpers). Resolved per class (desktop, light / dark, no state):

- line 99: context `data-slot="table-row"`

  ```txt
  table-row
  ```

  Resolved: display: `table-row`

- line 131: context `data-slot="table-cell"`

  ```txt
  table-cell
  ```

  Resolved: display: `table-cell`

- line 147: context `data-slot="table-caption"`

  ```txt
  table-caption
  ```

  Resolved: display: `table-caption`

