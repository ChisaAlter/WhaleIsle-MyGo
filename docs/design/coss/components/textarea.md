# Textarea

Source: `apps/ui/registry/default/ui/textarea.tsx` @ `b2336c45f2b7` · docs: `apps/ui/content/docs/components/textarea.mdx`

> A native textarea element.

Depends on: `@base-ui/react/field`, `@base-ui/react/merge-props`

Exports: `Textarea`

## Behavior & keyboard

- Native `<textarea>` with auto field-sizing; focus ring on the wrapping control via `:has(:focus-visible)`.

## How to read

Each part lists the exact class strings, then **resolved** values for the desktop environment (≥640px so all `sm:` rules apply; also md/lg/xl; fine pointer; hover-capable). Values show `light` / `dark` when they differ. *State* tables only list properties that change versus the part's default. *Mobile* lists properties that differ at 375px / coarse pointer. `src` is the class that wins the cascade. See `../README.md` for conventions.

## Parts

### `Textarea` → `[data-slot=textarea-control]` · `<span>` (line 22)

Attributes: `data-size={size}`

Prop defaults: `size="default"`, `unstyled=false`

#### when `!unstyled`

```txt
relative inline-grid w-full rounded-lg border border-input bg-background not-dark:bg-clip-padding text-base shadow-xs/5 ring-ring/24 transition-shadow before:pointer-events-none before:absolute before:inset-0 before:rounded-[calc(var(--radius-lg)-1px)] has-focus-visible:has-aria-invalid:border-destructive/64 has-focus-visible:has-aria-invalid:ring-destructive/16 has-aria-invalid:border-destructive/36 has-focus-visible:border-ring has-disabled:opacity-64 has-[:disabled,:focus-visible,[aria-invalid]]:shadow-none has-focus-visible:ring-[3px] not-has-disabled:has-not-focus-visible:not-has-aria-invalid:before:shadow-[0_1px_--theme(--color-black/4%)] sm:text-sm dark:bg-input/32 dark:has-aria-invalid:ring-destructive/24 dark:not-has-disabled:has-not-focus-visible:not-has-aria-invalid:before:shadow-[0_-1px_--theme(--color-white/6%)]
```

**element — default (changes vs base)**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| border-width | `1px` | `1px` | `border` |
| border-style | `solid` | `var(--tw-border-style)` | `border` |
| border-color | `#0000001a` / `#ffffff14` | `input` | `border-input` |
| position | `relative` | `relative` | `relative` |
| display | `inline-grid` | `inline-grid` | `inline-grid` |
| width | `100%` | `100%` | `w-full` |
| border-radius | `10px` | `radius` | `rounded-lg` |
| background-color | `#ffffff` / `#ffffff07` | `background` | `bg-background` |
| font-size | `14px` | `text-sm` | `sm:text-sm` |
| line-height | `20px` | `var(--tw-leading, var(--text-sm--line-height))` | `sm:text-sm` |
| box-shadow | `0 1px 2px 0 #0000000d` | `var(--tw-inset-shadow), var(--tw-inset-ring-shadow), var(--tw-ring-offset-shadow), var(--tw-ring-shadow), var(` | `shadow-xs/5` |
| transition-property | `box-shadow` | `box-shadow` | `transition-shadow` |
| transition-timing-function | `cubic-bezier(0.4, 0, 0.2, 1)` | `var(--tw-ease, var(--default-transition-timing-function))` | `transition-shadow` |
| transition-duration | `150ms` | `var(--tw-duration, var(--default-transition-duration))` | `transition-shadow` |
| background-clip | `padding-box` / — | `padding-box` | `not-dark:bg-clip-padding` |

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| focus-visible | border-color | `#a1a1a1` / `#737373` | `has-focus-visible:border-ring` |
| focus-visible | box-shadow | `0 0 0 3px #a1a1a13d` / `0 0 0 3px #7373733d` | `has-[:disabled,:focus-visible,[aria-invalid]]:shadow-none` |
| invalid | border-color | `#fb2c365c` / `#fb414a5c` | `has-aria-invalid:border-destructive/36` |
| invalid | box-shadow | `none` | `has-[:disabled,:focus-visible,[aria-invalid]]:shadow-none` |
| disabled | box-shadow | `none` | `has-[:disabled,:focus-visible,[aria-invalid]]:shadow-none` |
| disabled | opacity | `64%` | `has-disabled:opacity-64` |
| focus-visible + invalid | border-color | `#fb2c36a3` / `#fb414aa3` | `has-focus-visible:has-aria-invalid:border-destructive/64` |
| focus-visible + invalid | box-shadow | `0 0 0 3px #fb2c3629` / `0 0 0 3px #fb414a3d` | `has-[:disabled,:focus-visible,[aria-invalid]]:shadow-none` |

**element — mobile differences**: font-size `16px` · line-height `24px`

**`::before` — default (changes vs base)**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| content | `""` | `var(--tw-content)` | `before:rounded-[calc(var(--radius-lg)-1px)]` |
| pointer-events | `none` | `none` | `before:pointer-events-none` |
| position | `absolute` | `absolute` | `before:absolute` |
| inset | `0px` | `calc(var(--spacing) * 0)` | `before:inset-0` |
| border-radius | `9px` | `calc(var(--radius-lg) - 1px)` | `before:rounded-[calc(var(--radius-lg)-1px)]` |

**`::before` — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| has(:not(:focus-visible)) | box-shadow | `0 1px #0000000a` / `0 -1px #ffffff0f` | `not-has-disabled:has-not-focus-visible:not-has-aria-invalid:before:shadow-[0_1px_--theme(--color-black/4%)]` |

### `Textarea` → `[data-slot=textarea]` · `<textarea>` (line 41)

Prop defaults: `size="default"`, `unstyled=false`

#### base

```txt
field-sizing-content min-h-17.5 w-full rounded-[inherit] px-[calc(--spacing(3)-1px)] py-[calc(--spacing(1.5)-1px)] text-foreground outline-none placeholder:text-muted-foreground/72 max-sm:min-h-20.5
```

**element — default**

Key metrics: min-height `70px` · width `100%` · padding-inline `11px` · padding-block `5px` · border-radius `inherit` · color `#262626` / `#f5f5f5` (foreground)

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| field-sizing | `content` | `content` | `field-sizing-content` |
| min-height | `70px` | `calc(var(--spacing) * 17.5)` | `min-h-17.5` |
| width | `100%` | `100%` | `w-full` |
| border-radius | `inherit` | `inherit` | `rounded-[inherit]` |
| padding-inline | `11px` | `calc(calc(var(--spacing) * 3) - 1px)` | `px-[calc(--spacing(3)-1px)]` |
| padding-block | `5px` | `calc(calc(var(--spacing) * 1.5) - 1px)` | `py-[calc(--spacing(1.5)-1px)]` |
| color | `#262626` / `#f5f5f5` | `foreground` | `text-foreground` |
| outline-style | `none` | `none` | `outline-none` |

**element — mobile differences**: min-height `82px`

**`::placeholder` — default**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| color | `#686868b8` / `#818181b8` | `muted-foreground/72%` | `placeholder:text-muted-foreground/72` |

#### when `size === "sm"`

```txt
min-h-16.5 px-[calc(--spacing(2.5)-1px)] py-[calc(--spacing(1)-1px)] max-sm:min-h-19.5
```

**element — mobile differences**: min-height `82px`

#### when `size === "lg"`

```txt
min-h-18.5 py-[calc(--spacing(2)-1px)] max-sm:min-h-21.5
```

**element — default (changes vs base)**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| min-height | `74px` | `calc(var(--spacing) * 18.5)` | `min-h-18.5` |
| padding-block | `7px` | `calc(calc(var(--spacing) * 2) - 1px)` | `py-[calc(--spacing(2)-1px)]` |

**element — mobile differences**: min-height `86px`

