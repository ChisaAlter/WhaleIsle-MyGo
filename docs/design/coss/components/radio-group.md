# Radio Group

Source: `apps/ui/registry/default/ui/radio-group.tsx` @ `b2336c45f2b7` · docs: `apps/ui/content/docs/components/radio-group.mdx` · Base UI: https://base-ui.com/react/components/radio#api-reference

> A set of checkable buttons where no more than one of the buttons can be checked at a time.

Depends on: `@base-ui/react/radio`, `@base-ui/react/radio-group`

Exports: `RadioGroup`, `Radio`

## Behavior & keyboard

- Single tab stop for the group; `ArrowUp/Down/Left/Right` move selection and focus; `Space` selects focused radio. State `data-checked`. _(inferred: Base UI primitive behavior; not restated in coss docs)_

## How to read

Each part lists the exact class strings, then **resolved** values for the desktop environment (≥640px so all `sm:` rules apply; also md/lg/xl; fine pointer; hover-capable). Values show `light` / `dark` when they differ. *State* tables only list properties that change versus the part's default. *Mobile* lists properties that differ at 375px / coarse pointer. `src` is the class that wins the cascade. See `../README.md` for conventions.

## Parts

### `RadioGroup` → `[data-slot=radio-group]` · `<RadioGroupPrimitive>` (line 13)

#### base

```txt
flex flex-col gap-3
```

**element — default**

Key metrics: display `flex` · gap `12px`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| display | `flex` | `flex` | `flex` |
| flex-direction | `column` | `column` | `flex-col` |
| gap | `12px` | `calc(var(--spacing) * 3)` | `gap-3` |

### `Radio` → `[data-slot=radio]` · `<RadioPrimitive.Root>` (line 26)

#### base

```txt
relative inline-flex size-4.5 shrink-0 items-center justify-center rounded-full border border-input bg-background not-dark:bg-clip-padding shadow-xs/5 outline-none transition-shadow before:pointer-events-none before:absolute before:inset-0 before:rounded-full not-data-disabled:not-data-checked:not-aria-invalid:before:shadow-[0_1px_--theme(--color-black/4%)] focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-1 focus-visible:ring-offset-background aria-invalid:border-destructive/36 focus-visible:aria-invalid:border-destructive/64 focus-visible:aria-invalid:ring-destructive/48 data-disabled:cursor-not-allowed data-disabled:opacity-64 sm:size-4 dark:not-data-checked:bg-input/32 dark:aria-invalid:ring-destructive/24 dark:not-data-disabled:not-data-checked:not-aria-invalid:before:shadow-[0_-1px_--theme(--color-white/6%)] [[data-disabled],[data-checked],[aria-invalid]]:shadow-none
```

**element — default**

Key metrics: display `inline-flex` · height `16px` · width `16px` · border-width `1px` · border-color `#0000001a` / `#ffffff14` (input) · border-radius `9999px` · background-color `#ffffff` / `#ffffff07` (background) · box-shadow `0 1px 2px 0 #0000000d` · transition-property `box-shadow` · transition-duration `150ms` · transition-timing-function `cubic-bezier(0.4, 0, 0.2, 1)`

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
| border-radius | `9999px` | `calc(infinity * 1px)` | `rounded-full` |
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
| border-radius | `9999px` | `calc(infinity * 1px)` | `before:rounded-full` |
| box-shadow | `0 1px #0000000a` / `0 -1px #ffffff0f` | `var(--tw-inset-shadow), var(--tw-inset-ring-shadow), var(--tw-ring-offset-shadow), var(--tw-ring-shadow), var(` | `not-data-disabled:not-data-checked:not-aria-invalid:before:shadow-[0_1px_--theme(--color-black/4%)]` |

**`::before` — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| disabled | box-shadow | `(unset)` | `(removed)` |
| checked | box-shadow | `(unset)` | `(removed)` |
| invalid | box-shadow | `(unset)` | `(removed)` |
| focus-visible + invalid | box-shadow | `(unset)` | `(removed)` |

### `Radio` → `[data-slot=radio-indicator]` · `<RadioPrimitive.Indicator>` (line 34)

#### base

```txt
absolute -inset-px flex size-4.5 items-center justify-center rounded-full before:size-2 before:rounded-full before:bg-primary-foreground data-unchecked:hidden data-checked:bg-primary sm:size-4 sm:before:size-1.5
```

**element — default**

Key metrics: display `flex` · height `16px` · width `16px` · border-radius `9999px`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| position | `absolute` | `absolute` | `absolute` |
| inset | `-1px` | `-1px` | `-inset-px` |
| display | `flex` | `flex` | `flex` |
| width | `16px` | `calc(var(--spacing) * 4)` | `sm:size-4` |
| height | `16px` | `calc(var(--spacing) * 4)` | `sm:size-4` |
| align-items | `center` | `center` | `items-center` |
| justify-content | `center` | `center` | `justify-center` |
| border-radius | `9999px` | `calc(infinity * 1px)` | `rounded-full` |

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| unchecked | display | `none` | `data-unchecked:hidden` |
| checked | background-color | `#262626` / `#f5f5f5` | `data-checked:bg-primary` |

**element — mobile differences**: width `18px` · height `18px`

**`::before` — default**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| content | `""` | `var(--tw-content)` | `sm:before:size-1.5` |
| width | `6px` | `calc(var(--spacing) * 1.5)` | `sm:before:size-1.5` |
| height | `6px` | `calc(var(--spacing) * 1.5)` | `sm:before:size-1.5` |
| border-radius | `9999px` | `calc(infinity * 1px)` | `before:rounded-full` |
| background-color | `#fafafa` / `#262626` | `primary-foreground` | `before:bg-primary-foreground` |

**`::before` — mobile differences**: width `8px` · height `8px`

