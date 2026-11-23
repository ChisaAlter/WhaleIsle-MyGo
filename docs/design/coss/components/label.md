# Label

Source: `apps/ui/registry/default/ui/label.tsx` @ `b2336c45f2b7` · docs: `apps/ui/content/docs/components/label.mdx`

> Renders an accessible label associated with controls.

Depends on: `@base-ui/react/merge-props`, `@base-ui/react/use-render`

Exports: `Label`

## Behavior & keyboard

- Native `<label>`; clicking focuses/activates the associated control.

## How to read

Each part lists the exact class strings, then **resolved** values for the desktop environment (≥640px so all `sm:` rules apply; also md/lg/xl; fine pointer; hover-capable). Values show `light` / `dark` when they differ. *State* tables only list properties that change versus the part's default. *Mobile* lists properties that differ at 375px / coarse pointer. `src` is the class that wins the cascade. See `../README.md` for conventions.

## Parts

### `Label` → `[data-slot=label]` · `useRender <label>` (line 13)

#### base

```txt
inline-flex items-center gap-2 font-medium text-base/4.5 text-foreground sm:text-sm/4
```

**element — default**

Key metrics: display `inline-flex` · gap `8px` · color `#262626` / `#f5f5f5` (foreground) · font-size `14px` (text-sm) · line-height `16px` · font-weight `500` (font-weight-medium)

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| display | `inline-flex` | `inline-flex` | `inline-flex` |
| align-items | `center` | `center` | `items-center` |
| gap | `8px` | `calc(var(--spacing) * 2)` | `gap-2` |
| font-size | `14px` | `text-sm` | `sm:text-sm/4` |
| line-height | `16px` | `calc(var(--spacing) * 4)` | `sm:text-sm/4` |
| font-weight | `500` | `font-weight-medium` | `font-medium` |
| color | `#262626` / `#f5f5f5` | `foreground` | `text-foreground` |

**element — mobile differences**: font-size `16px` · line-height `18px`

