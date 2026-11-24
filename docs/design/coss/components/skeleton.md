# Skeleton

Source: `apps/ui/registry/default/ui/skeleton.tsx` @ `b2336c45f2b7` · docs: `apps/ui/content/docs/components/skeleton.mdx`

> A loading state skeleton for your components.

Depends on: —

Exports: `Skeleton`

## Behavior & keyboard

- Presentational shimmer; no behavior. Animation `skeleton 2s -1s infinite linear` (background-position sweep).

## How to read

Each part lists the exact class strings, then **resolved** values for the desktop environment (≥640px so all `sm:` rules apply; also md/lg/xl; fine pointer; hover-capable). Values show `light` / `dark` when they differ. *State* tables only list properties that change versus the part's default. *Mobile* lists properties that differ at 375px / coarse pointer. `src` is the class that wins the cascade. See `../README.md` for conventions.

## Parts

### `Skeleton` → `[data-slot=skeleton]` · `<div>` (line 9)

#### base

```txt
animate-skeleton rounded-sm [--skeleton-highlight:--alpha(var(--color-white)/64%)] [background:linear-gradient(120deg,transparent_40%,var(--skeleton-highlight),transparent_60%)_var(--color-muted)_0_0/200%_100%_fixed] dark:[--skeleton-highlight:--alpha(var(--color-white)/4%)]
```

**element — default**

Key metrics: border-radius `6px` · animation `skeleton 2000ms -1000ms infinite linear`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| animation | `skeleton 2000ms -1000ms infinite linear` | `skeleton 2s -1s infinite linear` | `animate-skeleton` |
| border-radius | `6px` | `calc(var(--radius) - 4px)` | `rounded-sm` |
| background | `linear-gradient(120deg,transparent 40%,#ffffffa3,transparent 60%) #0000000a 0 0/200% 100% fixed` / `linear-gradient(120deg,transparent 40%,#ffffff0a,transparent 60%) #ffffff0a 0 0/200% 100% fixed` | `linear-gradient(120deg,transparent 40%,var(--skeleton-highlight),transparent 60%) var(--color-muted) 0 0/200% ` | `[background:linear-gradient(120deg,transparent_40%,var(--skeleton-highlight),transparent_60%)_var(--color-muted)_0_0/200%_100%_fixed]` |

