# Spinner

Source: `apps/ui/registry/default/ui/spinner.tsx` @ `b2336c45f2b7` · docs: `apps/ui/content/docs/components/spinner.mdx`

> An indicator that can be used to show a loading state.

Depends on: `lucide-react`

Exports: `Spinner`

## Behavior & keyboard

- Presentational rotating icon (`animate-spin`, 1s linear infinite). `role=status`/`aria-label=Loading`.

## How to read

Each part lists the exact class strings, then **resolved** values for the desktop environment (≥640px so all `sm:` rules apply; also md/lg/xl; fine pointer; hover-capable). Values show `light` / `dark` when they differ. *State* tables only list properties that change versus the part's default. *Mobile* lists properties that differ at 375px / coarse pointer. `src` is the class that wins the cascade. See `../README.md` for conventions.

## Parts

### `Spinner` → (no data-slot) · `<Loader2Icon>` (line 10)

Attributes: `aria-label=Loading`, `role=status`

#### base

```txt
animate-spin
```

**element — default**

Key metrics: animation `spin 1000ms linear infinite` (animate-spin)

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| animation | `spin 1000ms linear infinite` | `animate-spin` | `animate-spin` |

