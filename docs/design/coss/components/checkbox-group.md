# Checkbox Group

Source: `apps/ui/registry/default/ui/checkbox-group.tsx` @ `b2336c45f2b7` · docs: `apps/ui/content/docs/components/checkbox-group.mdx` · Base UI: https://base-ui.com/react/components/checkbox-group#api-reference

> Provides shared state to a series of checkboxes.

Depends on: `@base-ui/react/checkbox-group`

Exports: `CheckboxGroup`

## Behavior & keyboard

- Group of checkboxes sharing state; each checkbox is a tab stop; optional parent checkbox shows `data-indeterminate`. _(inferred: Base UI primitive behavior; not restated in coss docs)_

## How to read

Each part lists the exact class strings, then **resolved** values for the desktop environment (≥640px so all `sm:` rules apply; also md/lg/xl; fine pointer; hover-capable). Values show `light` / `dark` when they differ. *State* tables only list properties that change versus the part's default. *Mobile* lists properties that differ at 375px / coarse pointer. `src` is the class that wins the cascade. See `../README.md` for conventions.

## Parts

### `CheckboxGroup` → (no data-slot) · `<CheckboxGroupPrimitive>` (line 12)

#### base

```txt
flex flex-col items-start gap-3
```

**element — default**

Key metrics: display `flex` · gap `12px`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| display | `flex` | `flex` | `flex` |
| flex-direction | `column` | `column` | `flex-col` |
| align-items | `flex-start` | `flex-start` | `items-start` |
| gap | `12px` | `calc(var(--spacing) * 3)` | `gap-3` |

