# Accordion

Source: `apps/ui/registry/default/ui/accordion.tsx` @ `b2336c45f2b7` · docs: `apps/ui/content/docs/components/accordion.mdx` · Base UI: https://base-ui.com/react/components/accordion#api-reference

> A set of collapsible panels with headings and content.

Depends on: `@base-ui/react/accordion`, `lucide-react`

Exports: `Accordion`, `AccordionItem`, `AccordionTrigger`, `AccordionPanel`

## Behavior & keyboard

- Trigger is a `<button>` inside a heading; `Enter`/`Space` toggles the panel; `ArrowDown`/`ArrowUp` move focus between triggers, `Home`/`End` jump to first/last.
- Panel height animates using `--accordion-panel-height` (set by Base UI) with `data-starting-style`/`data-ending-style` → height 0.
- `data-open` on item/trigger when expanded; `data-disabled` when disabled. _(inferred: Base UI primitive behavior; not restated in coss docs)_

## How to read

Each part lists the exact class strings, then **resolved** values for the desktop environment (≥640px so all `sm:` rules apply; also md/lg/xl; fine pointer; hover-capable). Values show `light` / `dark` when they differ. *State* tables only list properties that change versus the part's default. *Mobile* lists properties that differ at 375px / coarse pointer. `src` is the class that wins the cascade. See `../README.md` for conventions.

## Parts

### `Accordion` → `[data-slot=accordion]` · `<AccordionPrimitive.Root>` (line 11)

### `AccordionItem` → `[data-slot=accordion-item]` · `<AccordionPrimitive.Item>` (line 19)

#### base

```txt
border-b last:border-b-0
```

**element — default**

Key metrics: border-bottom-width `1px`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| border-bottom-style | `solid` | `var(--tw-border-style)` | `border-b` |
| border-bottom-width | `1px` | `1px` | `border-b` |

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| :last-child | border-bottom-width | `0px` | `last:border-b-0` |

### `AccordionTrigger` → (no data-slot) · `<AccordionPrimitive.Header>` (line 33)

#### base

```txt
flex
```

**element — default**

Key metrics: display `flex`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| display | `flex` | `flex` | `flex` |

### `AccordionTrigger` → `[data-slot=accordion-trigger]` · `<AccordionPrimitive.Trigger>` (line 34)

#### base

```txt
flex flex-1 cursor-pointer items-start justify-between gap-4 rounded-md py-4 text-left font-medium text-sm outline-none transition-all focus-visible:ring-[3px] focus-visible:ring-ring disabled:pointer-events-none disabled:opacity-64 data-panel-open:*:data-[slot=accordion-indicator]:rotate-180
```

**element — default**

Key metrics: display `flex` · padding-block `16px` · gap `16px` · border-radius `8px` · font-size `14px` (text-sm) · line-height `20px` · font-weight `500` (font-weight-medium) · transition-property `all` · transition-duration `150ms` · transition-timing-function `cubic-bezier(0.4, 0, 0.2, 1)`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| display | `flex` | `flex` | `flex` |
| flex | `1` | `1` | `flex-1` |
| cursor | `pointer` | `pointer` | `cursor-pointer` |
| align-items | `flex-start` | `flex-start` | `items-start` |
| justify-content | `space-between` | `space-between` | `justify-between` |
| gap | `16px` | `calc(var(--spacing) * 4)` | `gap-4` |
| border-radius | `8px` | `calc(var(--radius) - 2px)` | `rounded-md` |
| padding-block | `16px` | `calc(var(--spacing) * 4)` | `py-4` |
| text-align | `left` | `left` | `text-left` |
| font-size | `14px` | `text-sm` | `text-sm` |
| line-height | `20px` | `var(--tw-leading, var(--text-sm--line-height))` | `text-sm` |
| font-weight | `500` | `font-weight-medium` | `font-medium` |
| transition-property | `all` | `all` | `transition-all` |
| transition-timing-function | `cubic-bezier(0.4, 0, 0.2, 1)` | `var(--tw-ease, var(--default-transition-timing-function))` | `transition-all` |
| transition-duration | `150ms` | `var(--tw-duration, var(--default-transition-duration))` | `transition-all` |
| outline-style | `none` | `none` | `outline-none` |

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| focus-visible | box-shadow | `0 0 0 3px #a1a1a1` / `0 0 0 3px #737373` | `focus-visible:ring-[3px]` |
| disabled | pointer-events | `none` | `disabled:pointer-events-none` |
| disabled | opacity | `64%` | `disabled:opacity-64` |

**`> *[data-slot="accordion-indicator"]` — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| open | rotate | `180deg` | `data-panel-open:*:data-[slot=accordion-indicator]:rotate-180` |

### `AccordionTrigger` → `[data-slot=accordion-indicator]` · `<ChevronDownIcon>` (line 43)

#### base

```txt
pointer-events-none size-4 shrink-0 translate-y-0.5 opacity-80 transition-transform duration-200 ease-in-out
```

**element — default**

Key metrics: height `16px` · width `16px` · opacity `80%` · transition-property `transform, translate, scale, rotate` · transition-duration `200ms` · transition-timing-function `cubic-bezier(0.4, 0, 0.2, 1)` (ease-in-out)

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| pointer-events | `none` | `none` | `pointer-events-none` |
| width | `16px` | `calc(var(--spacing) * 4)` | `size-4` |
| height | `16px` | `calc(var(--spacing) * 4)` | `size-4` |
| flex-shrink | `0` | `0` | `shrink-0` |
| translate | `0 2px` | `var(--tw-translate-x) var(--tw-translate-y)` | `translate-y-0.5` |
| opacity | `80%` | `80%` | `opacity-80` |
| transition-property | `transform, translate, scale, rotate` | `transform, translate, scale, rotate` | `transition-transform` |
| transition-timing-function | `cubic-bezier(0.4, 0, 0.2, 1)` | `ease-in-out` | `ease-in-out` |
| transition-duration | `200ms` | `200ms` | `duration-200` |

### `AccordionPanel` → `[data-slot=accordion-panel]` · `<AccordionPrimitive.Panel>` (line 58)

#### base

```txt
h-(--accordion-panel-height) overflow-hidden text-muted-foreground text-sm transition-[height] duration-200 ease-in-out data-ending-style:h-0 data-starting-style:h-0
```

**element — default**

Key metrics: height `var(--accordion-panel-height)` (runtime var --accordion-panel-height) · color `#686868` / `#818181` (muted-foreground) · font-size `14px` (text-sm) · line-height `20px` · transition-property `height` · transition-duration `200ms` · transition-timing-function `cubic-bezier(0.4, 0, 0.2, 1)` (ease-in-out)

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| height | `var(--accordion-panel-height)` | `runtime var --accordion-panel-height` | `h-(--accordion-panel-height)` |
| overflow | `hidden` | `hidden` | `overflow-hidden` |
| font-size | `14px` | `text-sm` | `text-sm` |
| line-height | `20px` | `var(--tw-leading, var(--text-sm--line-height))` | `text-sm` |
| color | `#686868` / `#818181` | `muted-foreground` | `text-muted-foreground` |
| transition-property | `height` | `height` | `transition-[height]` |
| transition-timing-function | `cubic-bezier(0.4, 0, 0.2, 1)` | `ease-in-out` | `ease-in-out` |
| transition-duration | `200ms` | `200ms` | `duration-200` |

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| ending-style (exit to) | height | `0px` | `data-ending-style:h-0` |
| starting-style (enter from) | height | `0px` | `data-starting-style:h-0` |

### `AccordionPanel` → (no data-slot) · `<div>` (line 63)

#### base

```txt
pt-0 pb-4
```

**element — default**

Key metrics: padding-top `0px` · padding-bottom `16px`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| padding-top | `0px` | `calc(var(--spacing) * 0)` | `pt-0` |
| padding-bottom | `16px` | `calc(var(--spacing) * 4)` | `pb-4` |

