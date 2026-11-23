# OTP Field

Source: `apps/ui/registry/default/ui/otp-field.tsx` @ `b2336c45f2b7` · docs: `apps/ui/content/docs/components/otp-field.mdx` · Base UI: https://base-ui.com/react/components/otp-field

> A segmented input for one-time passwords and verification codes.

Depends on: `@base-ui/react/otp-field`, `@/ui/separator`

Exports: `OTPField`, `OTPFieldInput`, `OTPFieldSeparator`

## Behavior & keyboard

- One input per character; typing advances focus, `Backspace` moves back, arrow keys move between slots, paste fills all slots. _(inferred)_

Docs mentions:

- Set `validationType="none"` with `normalizeValue` when you need to normalize pasted input before it reaches state, or to enforce custom character rules. Use `inputMode` for the virtual keyboard hint, and `onValueInvalid` when you want to react to characters that were rejected after normalization.

## How to read

Each part lists the exact class strings, then **resolved** values for the desktop environment (≥640px so all `sm:` rules apply; also md/lg/xl; fine pointer; hover-capable). Values show `light` / `dark` when they differ. *State* tables only list properties that change versus the part's default. *Mobile* lists properties that differ at 375px / coarse pointer. `src` is the class that wins the cascade. See `../README.md` for conventions.

## Parts

### `OTPField` → `[data-slot=otp-field]` · `<OTPFieldPrimitive.Root>` (line 16)

Attributes: `data-size={size}`

Prop defaults: `size="default"`

#### base

```txt
flex items-center gap-2 has-disabled:opacity-64 has-disabled:**:data-[slot=otp-field-input]:shadow-none has-disabled:**:data-[slot=otp-field-input]:before:shadow-none!
```

**element — default**

Key metrics: display `flex` · gap `8px`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| display | `flex` | `flex` | `flex` |
| align-items | `center` | `center` | `items-center` |
| gap | `8px` | `calc(var(--spacing) * 2)` | `gap-2` |

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| disabled | opacity | `64%` | `has-disabled:opacity-64` |

**`*[data-slot="otp-field-input"]` — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| disabled | box-shadow | `none` | `has-disabled:**:data-[slot=otp-field-input]:shadow-none` |

**`*[data-slot="otp-field-input"]::before` — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| disabled | content | `""` | `has-disabled:**:data-[slot=otp-field-input]:before:shadow-none!` |
| disabled | box-shadow | `none` | `has-disabled:**:data-[slot=otp-field-input]:before:shadow-none!` |

### `OTPFieldInput` → `[data-slot=otp-field-input]` · `<OTPFieldPrimitive.Input>` (line 33)

Attributes: `spellCheck={false}`

#### base

```txt
relative in-[[data-slot=otp-field][data-size=lg]]:size-10 size-9 min-w-0 rounded-lg border border-input bg-background not-dark:bg-clip-padding text-center in-[[data-slot=otp-field][data-size=lg]]:text-lg text-base text-foreground in-[[data-slot=otp-field][data-size=lg]]:leading-10 leading-9 shadow-xs/5 outline-none ring-ring/24 transition-shadow before:pointer-events-none before:absolute before:inset-0 before:rounded-[calc(var(--radius-lg)-1px)] not-focus-visible:not-aria-invalid:before:shadow-[0_1px_--theme(--color-black/4%)] focus-visible:z-10 focus-visible:border-ring focus-visible:shadow-none focus-visible:ring-[3px] focus-visible:ring-ring/24 aria-invalid:border-destructive/36 aria-invalid:shadow-none aria-invalid:focus-visible:border-destructive/64 aria-invalid:focus-visible:ring-destructive/16 sm:in-[[data-slot=otp-field][data-size=lg]]:size-9 sm:size-8 sm:in-[[data-slot=otp-field][data-size=lg]]:text-base sm:text-sm sm:in-[[data-slot=otp-field][data-size=lg]]:leading-9 sm:leading-8 dark:bg-input/32 dark:aria-invalid:focus-visible:ring-destructive/24 dark:not-focus-visible:not-aria-invalid:before:shadow-[0_-1px_--theme(--color-white/6%)]
```

**element — default**

Key metrics: height `32px` · width `32px` · min-width `0px` · border-width `1px` · border-color `#0000001a` / `#ffffff14` (input) · border-radius `10px` (radius) · background-color `#ffffff` / `#ffffff07` (background) · color `#262626` / `#f5f5f5` (foreground) · font-size `14px` (text-sm) · line-height `32px` · box-shadow `0 1px 2px 0 #0000000d` · transition-property `box-shadow` · transition-duration `150ms` · transition-timing-function `cubic-bezier(0.4, 0, 0.2, 1)`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| border-width | `1px` | `1px` | `border` |
| border-style | `solid` | `var(--tw-border-style)` | `border` |
| border-color | `#0000001a` / `#ffffff14` | `input` | `border-input` |
| position | `relative` | `relative` | `relative` |
| width | `32px` | `calc(var(--spacing) * 8)` | `sm:size-8` |
| height | `32px` | `calc(var(--spacing) * 8)` | `sm:size-8` |
| min-width | `0px` | `calc(var(--spacing) * 0)` | `min-w-0` |
| border-radius | `10px` | `radius` | `rounded-lg` |
| background-color | `#ffffff` / `#ffffff07` | `background` | `bg-background` |
| text-align | `center` | `center` | `text-center` |
| font-size | `14px` | `text-sm` | `sm:text-sm` |
| line-height | `32px` | `calc(var(--spacing) * 8)` | `sm:leading-8` |
| color | `#262626` / `#f5f5f5` | `foreground` | `text-foreground` |
| box-shadow | `0 1px 2px 0 #0000000d` | `var(--tw-inset-shadow), var(--tw-inset-ring-shadow), var(--tw-ring-offset-shadow), var(--tw-ring-shadow), var(` | `shadow-xs/5` |
| transition-property | `box-shadow` | `box-shadow` | `transition-shadow` |
| transition-timing-function | `cubic-bezier(0.4, 0, 0.2, 1)` | `var(--tw-ease, var(--default-transition-timing-function))` | `transition-shadow` |
| transition-duration | `150ms` | `var(--tw-duration, var(--default-transition-duration))` | `transition-shadow` |
| outline-style | `none` | `none` | `outline-none` |
| background-clip | `padding-box` / — | `padding-box` | `not-dark:bg-clip-padding` |

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| in(:where(*:is([data-slot=otp-field][data-size=lg]))) | width | `36px` | `sm:in-[[data-slot=otp-field][data-size=lg]]:size-9` |
| in(:where(*:is([data-slot=otp-field][data-size=lg]))) | height | `36px` | `sm:in-[[data-slot=otp-field][data-size=lg]]:size-9` |
| in(:where(*:is([data-slot=otp-field][data-size=lg]))) | font-size | `16px` | `sm:in-[[data-slot=otp-field][data-size=lg]]:text-base` |
| in(:where(*:is([data-slot=otp-field][data-size=lg]))) | line-height | `36px` | `sm:in-[[data-slot=otp-field][data-size=lg]]:leading-9` |
| focus-visible | border-color | `#a1a1a1` / `#737373` | `focus-visible:border-ring` |
| focus-visible | box-shadow | `0 0 0 3px #a1a1a13d` / `0 0 0 3px #7373733d` | `focus-visible:ring-[3px]` |
| focus-visible | z-index | `10` | `focus-visible:z-10` |
| invalid | border-color | `#fb2c365c` / `#fb414a5c` | `aria-invalid:border-destructive/36` |
| invalid | box-shadow | `none` | `aria-invalid:shadow-none` |
| focus-visible + invalid | border-color | `#fb2c36a3` / `#fb414aa3` | `aria-invalid:focus-visible:border-destructive/64` |
| focus-visible + invalid | box-shadow | `0 0 0 3px #fb2c3629` / `0 0 0 3px #fb414a3d` | `aria-invalid:shadow-none` |
| focus-visible + invalid | z-index | `10` | `focus-visible:z-10` |

**element — mobile differences**: width `36px` · height `36px` · font-size `16px` · line-height `36px`

**`::before` — default**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| content | `""` | `var(--tw-content)` | `not-focus-visible:not-aria-invalid:before:shadow-[0_1px_--theme(--color-black/4%)]` |
| pointer-events | `none` | `none` | `before:pointer-events-none` |
| position | `absolute` | `absolute` | `before:absolute` |
| inset | `0px` | `calc(var(--spacing) * 0)` | `before:inset-0` |
| border-radius | `9px` | `calc(var(--radius-lg) - 1px)` | `before:rounded-[calc(var(--radius-lg)-1px)]` |
| box-shadow | `0 1px #0000000a` / `0 -1px #ffffff0f` | `var(--tw-inset-shadow), var(--tw-inset-ring-shadow), var(--tw-ring-offset-shadow), var(--tw-ring-shadow), var(` | `not-focus-visible:not-aria-invalid:before:shadow-[0_1px_--theme(--color-black/4%)]` |

**`::before` — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| focus-visible | box-shadow | `(unset)` | `(removed)` |
| invalid | box-shadow | `(unset)` | `(removed)` |
| focus-visible + invalid | box-shadow | `(unset)` | `(removed)` |

### `OTPFieldSeparator` → (no data-slot) · `<Separator>` (line 52)

Attributes: `orientation=horizontal`

#### base

```txt
rounded-full bg-input data-[orientation=horizontal]:h-0.5 data-[orientation=horizontal]:w-3
```

**element — default**

Key metrics: border-radius `9999px` · background-color `#0000001a` / `#ffffff14` (input)

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| border-radius | `9999px` | `calc(infinity * 1px)` | `rounded-full` |
| background-color | `#0000001a` / `#ffffff14` | `input` | `bg-input` |

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| [data-orientation=horizontal] | height | `2px` | `data-[orientation=horizontal]:h-0.5` |
| [data-orientation=horizontal] | width | `12px` | `data-[orientation=horizontal]:w-3` |

