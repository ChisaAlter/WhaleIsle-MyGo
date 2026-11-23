# Collapsible

Source: `apps/ui/registry/default/ui/collapsible.tsx` @ `b2336c45f2b7` · docs: `apps/ui/content/docs/components/collapsible.mdx` · Base UI: https://base-ui.com/react/components/collapsible#api-reference

> A collapsible panel controlled by a button trigger.

Depends on: `@base-ui/react/collapsible`

Exports: `Collapsible`, `CollapsibleTrigger`, `CollapsiblePanel`

## Behavior & keyboard

- Trigger button toggles panel with `Enter`/`Space`; panel height animates via `--collapsible-panel-height` and `data-starting-style`/`data-ending-style`. _(inferred: Base UI primitive behavior; not restated in coss docs)_

## How to read

Each part lists the exact class strings, then **resolved** values for the desktop environment (≥640px so all `sm:` rules apply; also md/lg/xl; fine pointer; hover-capable). Values show `light` / `dark` when they differ. *State* tables only list properties that change versus the part's default. *Mobile* lists properties that differ at 375px / coarse pointer. `src` is the class that wins the cascade. See `../README.md` for conventions.

## Parts

### `Collapsible` → `[data-slot=collapsible]` · `<CollapsiblePrimitive.Root>` (line 10)

### `CollapsibleTrigger` → `[data-slot=collapsible-trigger]` · `<CollapsiblePrimitive.Trigger>` (line 18)

### `CollapsiblePanel` → `[data-slot=collapsible-panel]` · `<CollapsiblePrimitive.Panel>` (line 31)

#### base

```txt
h-(--collapsible-panel-height) overflow-hidden transition-[height] duration-200 data-ending-style:h-0 data-starting-style:h-0
```

**element — default**

Key metrics: height `var(--collapsible-panel-height)` (runtime var --collapsible-panel-height) · transition-property `height` · transition-duration `200ms` · transition-timing-function `cubic-bezier(0.4, 0, 0.2, 1)`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| height | `var(--collapsible-panel-height)` | `runtime var --collapsible-panel-height` | `h-(--collapsible-panel-height)` |
| overflow | `hidden` | `hidden` | `overflow-hidden` |
| transition-property | `height` | `height` | `transition-[height]` |
| transition-timing-function | `cubic-bezier(0.4, 0, 0.2, 1)` | `var(--tw-ease, var(--default-transition-timing-function))` | `transition-[height]` |
| transition-duration | `200ms` | `200ms` | `duration-200` |

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| ending-style (exit to) | height | `0px` | `data-ending-style:h-0` |
| starting-style (enter from) | height | `0px` | `data-starting-style:h-0` |

