# Fieldset

Source: `apps/ui/registry/default/ui/fieldset.tsx` @ `b2336c45f2b7` · docs: `apps/ui/content/docs/components/fieldset.mdx` · Base UI: https://base-ui.com/react/components/fieldset#api-reference

> A native fieldset element with a legend.

Depends on: `@base-ui/react/fieldset`

Exports: `Fieldset`, `FieldsetLegend`

## Behavior & keyboard

- Native `<fieldset>`/`<legend>` semantics; disables all descendants when disabled.

## How to read

Each part lists the exact class strings, then **resolved** values for the desktop environment (≥640px so all `sm:` rules apply; also md/lg/xl; fine pointer; hover-capable). Values show `light` / `dark` when they differ. *State* tables only list properties that change versus the part's default. *Mobile* lists properties that differ at 375px / coarse pointer. `src` is the class that wins the cascade. See `../README.md` for conventions.

## Parts

### `Fieldset` → `[data-slot=fieldset]` · `<FieldsetPrimitive.Root>` (line 12)

### `FieldsetLegend` → `[data-slot=fieldset-legend]` · `<FieldsetPrimitive.Legend>` (line 24)

#### base

```txt
font-semibold text-foreground
```

**element — default**

Key metrics: color `#262626` / `#f5f5f5` (foreground) · font-weight `600` (font-weight-semibold)

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| font-weight | `600` | `font-weight-semibold` | `font-semibold` |
| color | `#262626` / `#f5f5f5` | `foreground` | `text-foreground` |

