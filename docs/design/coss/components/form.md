# Form

Source: `apps/ui/registry/default/ui/form.tsx` @ `b2336c45f2b7` · docs: `apps/ui/content/docs/components/form.mdx` · Base UI: https://base-ui.com/react/components/form#api-reference

> A form wrapper component that simplifies validation and submission.

Depends on: `@base-ui/react/form`

Exports: `Form`

## Behavior & keyboard

- Native `<form>` with Base UI validation integration; errors surface per Field.

## How to read

Each part lists the exact class strings, then **resolved** values for the desktop environment (≥640px so all `sm:` rules apply; also md/lg/xl; fine pointer; hover-capable). Values show `light` / `dark` when they differ. *State* tables only list properties that change versus the part's default. *Mobile* lists properties that differ at 375px / coarse pointer. `src` is the class that wins the cascade. See `../README.md` for conventions.

## Parts

### `Form` → `[data-slot=form]` · `<FormPrimitive>` (line 10)

