# Checkbox

Source: `apps/ui/registry/default/ui/checkbox.tsx` @ `b2336c45f2b7` · docs: `apps/ui/content/docs/components/checkbox.mdx` · Base UI: https://base-ui.com/react/components/checkbox#api-reference

> A control allowing the user to toggle between checked and not checked.

Depends on: `@base-ui/react/checkbox`

Exports: `Checkbox`

## Behavior & keyboard

- `role=checkbox` button; `Space` toggles. States exposed as `data-checked`, `data-unchecked`, `data-indeterminate`, `data-disabled`, `aria-invalid`. _(inferred: Base UI primitive behavior; not restated in coss docs)_

## How to read

Each part lists the exact class strings, then **resolved** values for the desktop environment (≥640px so all `sm:` rules apply; also md/lg/xl; fine pointer; hover-capable). Values show `light` / `dark` when they differ. *State* tables only list properties that change versus the part's default. *Mobile* lists properties that differ at 375px / coarse pointer. `src` is the class that wins the cascade. See `../README.md` for conventions.

## Parts

### `Checkbox` → `[data-slot=checkbox]` · `<CheckboxPrimitive.Root>` (line 12)

#### base

```txt
relative inline-flex size-4.5 shrink-0 items-center justify-center rounded-[.25rem] border border-input bg-background not-dark:bg-clip-padding shadow-xs/5 outline-none ring-ring transition-shadow before:pointer-events-none before:absolute before:inset-0 before:rounded-[3px] not-data-disabled:not-data-checked:not-aria-invalid:before:shadow-[0_1px_--theme(--color-black/4%)] focus-visible:ring-2 focus-visible:ring-offset-1 focus-visible:ring-offset-background aria-invalid:border-destructive/36 focus-visible:aria-invalid:border-destructive/64 focus-visible:aria-invalid:ring-destructive/48 data-disabled:cursor-not-allowed data-disabled:opacity-64 sm:size-4 dark:not-data-checked:bg-input/32 dark:aria-invalid:ring-destructive/24 dark:not-data-disabled:not-data-checked:not-aria-invalid:before:shadow-[0_-1px_--theme(--color-white/6%)] [[data-disabled],[data-checked],[aria-invalid]]:shadow-none
```

**element — default**

Key metrics: display `inline-flex` · height `16px` · width `16px` · border-width `1px` · border-color `#0000001a` / `#ffffff14` (input) · border-radius `4px` · background-color `#ffffff` / `#ffffff07` (background) · box-shadow `0 1px 2px 0 #0000000d` · transition-property `box-shadow` · transition-duration `150ms` · transition-timing-function `cubic-bezier(0.4, 0, 0.2, 1)`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| border-width | `1px` | `1px` | `border` |
| border-style | `solid` | `var(--tw-border-style)` | `border` |
| border-color | `#0000001a` / `#ffffff14` | `input` | `border-input` |
| position | `relative` | `relative` | `relative` |
| display | `inline-flex` | `inline-flex` | `inline-flex` |
| width | `16px` | `calc(var(--spacing) * 4)` | `sm:size-4` |
| height | `16px` | `calc(var(--spacing) * 4)` | `sm:size-4` |
| flex-shrink | `0` | `0` | `shrink-0` |
| align-items | `center` | `center` | `items-center` |
| justify-content | `center` | `center` | `justify-center` |
| border-radius | `4px` | `.25rem` | `rounded-[.25rem]` |
| background-color | `#ffffff` / `#ffffff07` | `background` | `bg-background` |
| box-shadow | `0 1px 2px 0 #0000000d` | `var(--tw-inset-shadow), var(--tw-inset-ring-shadow), var(--tw-ring-offset-shadow), var(--tw-ring-shadow), var(` | `shadow-xs/5` |
| transition-property | `box-shadow` | `box-shadow` | `transition-shadow` |
| transition-timing-function | `cubic-bezier(0.4, 0, 0.2, 1)` | `var(--tw-ease, var(--default-transition-timing-function))` | `transition-shadow` |
| transition-duration | `150ms` | `var(--tw-duration, var(--default-transition-duration))` | `transition-shadow` |
| outline-style | `none` | `none` | `outline-none` |
| background-clip | `padding-box` / — | `padding-box` | `not-dark:bg-clip-padding` |

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| disabled | box-shadow | `none` | `[[data-disabled],[data-checked],[aria-invalid]]:shadow-none` |
| disabled | cursor | `not-allowed` | `data-disabled:cursor-not-allowed` |
| disabled | opacity | `64%` | `data-disabled:opacity-64` |
| checked | background-color | `#ffffff` / `#141414` | `bg-background` |
| checked | box-shadow | `none` | `[[data-disabled],[data-checked],[aria-invalid]]:shadow-none` |
| invalid | border-color | `#fb2c365c` / `#fb414a5c` | `aria-invalid:border-destructive/36` |
| invalid | box-shadow | `none` | `[[data-disabled],[data-checked],[aria-invalid]]:shadow-none` |
| focus-visible | box-shadow | `0 0 0 1px #ffffff, 0 0 0 3px #a1a1a1, 0 1px 2px 0 #0000000d` / `0 0 0 1px #141414, 0 0 0 3px #737373, 0 1px 2px 0 #0000000d` | `focus-visible:ring-2` |
| focus-visible + invalid | border-color | `#fb2c36a3` / `#fb414aa3` | `focus-visible:aria-invalid:border-destructive/64` |
| focus-visible + invalid | box-shadow | `0 0 0 1px #ffffff, 0 0 0 3px #fb2c367a` / `0 0 0 1px #141414, 0 0 0 3px #fb414a3d` | `[[data-disabled],[data-checked],[aria-invalid]]:shadow-none` |

**element — mobile differences**: width `18px` · height `18px`

**`::before` — default**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| content | `""` | `var(--tw-content)` | `not-data-disabled:not-data-checked:not-aria-invalid:before:shadow-[0_1px_--theme(--color-black/4%)]` |
| pointer-events | `none` | `none` | `before:pointer-events-none` |
| position | `absolute` | `absolute` | `before:absolute` |
| inset | `0px` | `calc(var(--spacing) * 0)` | `before:inset-0` |
| border-radius | `3px` | `3px` | `before:rounded-[3px]` |
| box-shadow | `0 1px #0000000a` / `0 -1px #ffffff0f` | `var(--tw-inset-shadow), var(--tw-inset-ring-shadow), var(--tw-ring-offset-shadow), var(--tw-ring-shadow), var(` | `not-data-disabled:not-data-checked:not-aria-invalid:before:shadow-[0_1px_--theme(--color-black/4%)]` |

**`::before` — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| disabled | box-shadow | `(unset)` | `(removed)` |
| checked | box-shadow | `(unset)` | `(removed)` |
| invalid | box-shadow | `(unset)` | `(removed)` |
| focus-visible + invalid | box-shadow | `(unset)` | `(removed)` |

### `Checkbox` → `[data-slot=checkbox-indicator]` · `<CheckboxPrimitive.Indicator>` (line 20)

#### base

```txt
absolute -inset-px flex items-center justify-center rounded-[.25rem] text-primary-foreground data-unchecked:hidden data-checked:bg-primary data-indeterminate:text-foreground
```

**element — default**

Key metrics: display `flex` · border-radius `4px` · color `#fafafa` / `#262626` (primary-foreground)

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| position | `absolute` | `absolute` | `absolute` |
| inset | `-1px` | `-1px` | `-inset-px` |
| display | `flex` | `flex` | `flex` |
| align-items | `center` | `center` | `items-center` |
| justify-content | `center` | `center` | `justify-center` |
| border-radius | `4px` | `.25rem` | `rounded-[.25rem]` |
| color | `#fafafa` / `#262626` | `primary-foreground` | `text-primary-foreground` |

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| unchecked | display | `none` | `data-unchecked:hidden` |
| checked | background-color | `#262626` / `#f5f5f5` | `data-checked:bg-primary` |
| indeterminate | color | `#262626` / `#f5f5f5` | `data-indeterminate:text-foreground` |

### `Checkbox` → (no data-slot) · `<svg>` (line 29)

Attributes: `aria-hidden=true`, `fill=none`, `height=24`, `stroke=currentColor`, `strokeLinecap=round`, `strokeLinejoin=round`, `strokeWidth=3`, `viewBox=0 0 24 24`, `width=24`, `xmlns=http://www.w3.org/2000/svg`

#### base

```txt
size-3.5 sm:size-3
```

**element — default**

Key metrics: height `12px` · width `12px`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| width | `12px` | `calc(var(--spacing) * 3)` | `sm:size-3` |
| height | `12px` | `calc(var(--spacing) * 3)` | `sm:size-3` |

**element — mobile differences**: width `14px` · height `14px`

### `Checkbox` → (no data-slot) · `<svg>` (line 45)

Attributes: `aria-hidden=true`, `fill=none`, `height=24`, `stroke=currentColor`, `strokeLinecap=round`, `strokeLinejoin=round`, `strokeWidth=3`, `viewBox=0 0 24 24`, `width=24`, `xmlns=http://www.w3.org/2000/svg`

#### base

```txt
size-3.5 sm:size-3
```

**element — default**

Key metrics: height `12px` · width `12px`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| width | `12px` | `calc(var(--spacing) * 3)` | `sm:size-3` |
| height | `12px` | `calc(var(--spacing) * 3)` | `sm:size-3` |

**element — mobile differences**: width `14px` · height `14px`

