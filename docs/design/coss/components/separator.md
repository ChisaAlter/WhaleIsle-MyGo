# Separator

Source: `apps/ui/registry/default/ui/separator.tsx` @ `b2336c45f2b7` · docs: `apps/ui/content/docs/components/separator.mdx` · Base UI: https://base-ui.com/react/components/separator#api-reference

> A separator element accessible to screen readers.

Depends on: `@base-ui/react/separator`

Exports: `Separator`

## Behavior & keyboard

- `role=separator`; not focusable.

## How to read

Each part lists the exact class strings, then **resolved** values for the desktop environment (≥640px so all `sm:` rules apply; also md/lg/xl; fine pointer; hover-capable). Values show `light` / `dark` when they differ. *State* tables only list properties that change versus the part's default. *Mobile* lists properties that differ at 375px / coarse pointer. `src` is the class that wins the cascade. See `../README.md` for conventions.

## Parts

### `Separator` → `[data-slot=separator]` · `<SeparatorPrimitive>` (line 11)

Attributes: `orientation={orientation}`

Prop defaults: `orientation="horizontal"`

#### base

```txt
shrink-0 bg-border data-[orientation=horizontal]:h-px data-[orientation=horizontal]:w-full data-[orientation=vertical]:w-px data-[orientation=vertical]:not-[[class^='h-']]:not-[[class*='_h-']]:self-stretch
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
| [data-orientation=horizontal] | height | `1px` | `data-[orientation=horizontal]:h-px` |
| [data-orientation=horizontal] | width | `100%` | `data-[orientation=horizontal]:w-full` |
| [data-orientation=vertical] | width | `1px` | `data-[orientation=vertical]:w-px` |
| [data-orientation=vertical] | align-self | `stretch` | `data-[orientation=vertical]:not-[[class^='h-']]:not-[[class*='_h-']]:self-stretch` |

