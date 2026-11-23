# Input

Source: `apps/ui/registry/default/ui/input.tsx` @ `b2336c45f2b7` · docs: `apps/ui/content/docs/components/input.mdx` · Base UI: https://base-ui.com/react/components/input#api-reference

> A native input element.

Depends on: `@base-ui/react/input`

Exports: `Input`

## Behavior & keyboard

- Native `<input>`; focus ring drawn on the wrapping `[data-slot=input-control]` span via `:has(:focus-visible)`.

## How to read

Each part lists the exact class strings, then **resolved** values for the desktop environment (≥640px so all `sm:` rules apply; also md/lg/xl; fine pointer; hover-capable). Values show `light` / `dark` when they differ. *State* tables only list properties that change versus the part's default. *Mobile* lists properties that differ at 375px / coarse pointer. `src` is the class that wins the cascade. See `../README.md` for conventions.

## Parts

### `Input` → `[data-slot=input-control]` · `<span>` (line 36)

Attributes: `data-size={size}`

Prop defaults: `size="default"`, `unstyled=false`, `nativeInput=false`

#### when `!unstyled`

```txt
relative inline-flex w-full rounded-lg border border-input bg-background not-dark:bg-clip-padding text-base shadow-xs/5 ring-ring/24 transition-shadow before:pointer-events-none before:absolute before:inset-0 before:rounded-[calc(var(--radius-lg)-1px)] not-has-disabled:not-has-focus-visible:not-has-aria-invalid:before:shadow-[0_1px_--theme(--color-black/4%)] has-focus-visible:has-aria-invalid:border-destructive/64 has-focus-visible:has-aria-invalid:ring-destructive/16 has-aria-invalid:border-destructive/36 has-focus-visible:border-ring has-autofill:bg-foreground/4 has-disabled:opacity-64 has-[:disabled,:focus-visible,[aria-invalid]]:shadow-none has-focus-visible:ring-[3px] sm:text-sm dark:bg-input/32 dark:has-autofill:bg-foreground/8 dark:has-aria-invalid:ring-destructive/24 dark:not-has-disabled:not-has-focus-visible:not-has-aria-invalid:before:shadow-[0_-1px_--theme(--color-white/6%)]
```

**element — default (changes vs base)**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| border-width | `1px` | `1px` | `border` |
| border-style | `solid` | `var(--tw-border-style)` | `border` |
| border-color | `#0000001a` / `#ffffff14` | `input` | `border-input` |
| position | `relative` | `relative` | `relative` |
| display | `inline-flex` | `inline-flex` | `inline-flex` |
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
| disabled | box-shadow | `none` | `has-[:disabled,:focus-visible,[aria-invalid]]:shadow-none` |
| disabled | opacity | `64%` | `has-disabled:opacity-64` |
| focus-visible | border-color | `#a1a1a1` / `#737373` | `has-focus-visible:border-ring` |
| focus-visible | box-shadow | `0 0 0 3px #a1a1a13d` / `0 0 0 3px #7373733d` | `has-[:disabled,:focus-visible,[aria-invalid]]:shadow-none` |
| invalid | border-color | `#fb2c365c` / `#fb414a5c` | `has-aria-invalid:border-destructive/36` |
| invalid | box-shadow | `none` | `has-[:disabled,:focus-visible,[aria-invalid]]:shadow-none` |
| has(:autofill) | background-color | `#2626260a` / `#f5f5f514` | `has-autofill:bg-foreground/4` |
| focus-visible + invalid | border-color | `#fb2c36a3` / `#fb414aa3` | `has-focus-visible:has-aria-invalid:border-destructive/64` |
| focus-visible + invalid | box-shadow | `0 0 0 3px #fb2c3629` / `0 0 0 3px #fb414a3d` | `has-[:disabled,:focus-visible,[aria-invalid]]:shadow-none` |

**element — mobile differences**: font-size `16px` · line-height `24px`

**`::before` — default (changes vs base)**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| content | `""` | `var(--tw-content)` | `not-has-disabled:not-has-focus-visible:not-has-aria-invalid:before:shadow-[0_1px_--theme(--color-black/4%)]` |
| pointer-events | `none` | `none` | `before:pointer-events-none` |
| position | `absolute` | `absolute` | `before:absolute` |
| inset | `0px` | `calc(var(--spacing) * 0)` | `before:inset-0` |
| border-radius | `9px` | `calc(var(--radius-lg) - 1px)` | `before:rounded-[calc(var(--radius-lg)-1px)]` |
| box-shadow | `0 1px #0000000a` / `0 -1px #ffffff0f` | `var(--tw-inset-shadow), var(--tw-inset-ring-shadow), var(--tw-ring-offset-shadow), var(--tw-ring-shadow), var(` | `not-has-disabled:not-has-focus-visible:not-has-aria-invalid:before:shadow-[0_1px_--theme(--color-black/4%)]` |

**`::before` — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| disabled | box-shadow | `(unset)` | `(removed)` |
| focus-visible | box-shadow | `(unset)` | `(removed)` |
| invalid | box-shadow | `(unset)` | `(removed)` |
| focus-visible + invalid | box-shadow | `(unset)` | `(removed)` |

### `Input` → `[data-slot=input]` · `<input>` (line 48)

Attributes: `size={typeof size === "number" ? size : undefined}`, `style={typeof style === "function" ? undefined : style}`

Prop defaults: `size="default"`, `unstyled=false`, `nativeInput=false`

#### base

```txt
h-8.5 w-full min-w-0 rounded-[inherit] px-[calc(--spacing(3)-1px)] text-foreground leading-8.5 outline-none [transition:background-color_5000000s_ease-in-out_0s] placeholder:text-muted-foreground/72 sm:h-7.5 sm:leading-7.5 autofill:[-webkit-text-fill-color:var(--foreground)]
```

**element — default**

Key metrics: height `30px` · width `100%` · min-width `0px` · padding-inline `11px` · border-radius `inherit` · color `#262626` / `#f5f5f5` (foreground) · line-height `30px`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| height | `30px` | `calc(var(--spacing) * 7.5)` | `sm:h-7.5` |
| width | `100%` | `100%` | `w-full` |
| min-width | `0px` | `calc(var(--spacing) * 0)` | `min-w-0` |
| border-radius | `inherit` | `inherit` | `rounded-[inherit]` |
| padding-inline | `11px` | `calc(calc(var(--spacing) * 3) - 1px)` | `px-[calc(--spacing(3)-1px)]` |
| line-height | `30px` | `calc(var(--spacing) * 7.5)` | `sm:leading-7.5` |
| color | `#262626` / `#f5f5f5` | `foreground` | `text-foreground` |
| outline-style | `none` | `none` | `outline-none` |
| transition | `background-color 5000000000ms ease-in-out 0ms` | `background-color 5000000s ease-in-out 0s` | `[transition:background-color_5000000s_ease-in-out_0s]` |

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| :autofill | -webkit-text-fill-color | `#262626` / `#f5f5f5` | `autofill:[-webkit-text-fill-color:var(--foreground)]` |

**element — mobile differences**: height `34px` · line-height `34px`

**`::placeholder` — default**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| color | `#686868b8` / `#818181b8` | `muted-foreground/72%` | `placeholder:text-muted-foreground/72` |

#### when `size === "sm"`

```txt
h-7.5 px-[calc(--spacing(2.5)-1px)] leading-7.5 sm:h-6.5 sm:leading-6.5
```

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| :autofill | -webkit-text-fill-color | `#262626` / `#f5f5f5` | `autofill:[-webkit-text-fill-color:var(--foreground)]` |

**element — mobile differences**: height `34px` · line-height `34px`

#### when `size === "lg"`

```txt
h-9.5 leading-9.5 sm:h-8.5 sm:leading-8.5
```

**element — default (changes vs base)**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| height | `34px` | `calc(var(--spacing) * 8.5)` | `sm:h-8.5` |
| line-height | `34px` | `calc(var(--spacing) * 8.5)` | `sm:leading-8.5` |

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| :autofill | -webkit-text-fill-color | `#262626` / `#f5f5f5` | `autofill:[-webkit-text-fill-color:var(--foreground)]` |

**element — mobile differences**: height `38px` · line-height `38px`

#### when `props.type === "search"`

```txt
[&::-webkit-search-cancel-button]:appearance-none [&::-webkit-search-decoration]:appearance-none [&::-webkit-search-results-button]:appearance-none [&::-webkit-search-results-decoration]:appearance-none
```

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| :autofill | -webkit-text-fill-color | `#262626` / `#f5f5f5` | `autofill:[-webkit-text-fill-color:var(--foreground)]` |

**element — mobile differences**: height `34px` · line-height `34px`

**`::-webkit-search-cancel-button` — default (changes vs base)**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| appearance | `none` | `none` | `[&::-webkit-search-cancel-button]:appearance-none` |

**`::-webkit-search-decoration` — default (changes vs base)**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| appearance | `none` | `none` | `[&::-webkit-search-decoration]:appearance-none` |

**`::-webkit-search-results-button` — default (changes vs base)**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| appearance | `none` | `none` | `[&::-webkit-search-results-button]:appearance-none` |

**`::-webkit-search-results-decoration` — default (changes vs base)**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| appearance | `none` | `none` | `[&::-webkit-search-results-decoration]:appearance-none` |

#### when `props.type === "file"`

```txt
text-muted-foreground file:me-3 file:bg-transparent file:font-medium file:text-foreground file:text-sm
```

**element — default (changes vs base)**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| color | `#686868` / `#818181` | `muted-foreground` | `text-muted-foreground` |

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| :autofill | -webkit-text-fill-color | `#262626` / `#f5f5f5` | `autofill:[-webkit-text-fill-color:var(--foreground)]` |

**element — mobile differences**: height `34px` · line-height `34px`

**`::file-selector-button` — default (changes vs base)**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| margin-inline-end | `12px` | `calc(var(--spacing) * 3)` | `file:me-3` |
| background-color | `#00000000` | `transparent` | `file:bg-transparent` |
| font-size | `14px` | `text-sm` | `file:text-sm` |
| line-height | `20px` | `var(--tw-leading, var(--text-sm--line-height))` | `file:text-sm` |
| font-weight | `500` | `font-weight-medium` | `file:font-medium` |
| color | `#262626` / `#f5f5f5` | `foreground` | `file:text-foreground` |

### `Input` → `[data-slot=input]` · `<InputPrimitive>` (line 56)

Attributes: `size={typeof size === "number" ? size : undefined}`, `style={style}`

Prop defaults: `size="default"`, `unstyled=false`, `nativeInput=false`

#### base

```txt
h-8.5 w-full min-w-0 rounded-[inherit] px-[calc(--spacing(3)-1px)] text-foreground leading-8.5 outline-none [transition:background-color_5000000s_ease-in-out_0s] placeholder:text-muted-foreground/72 sm:h-7.5 sm:leading-7.5 autofill:[-webkit-text-fill-color:var(--foreground)]
```

**element — default**

Key metrics: height `30px` · width `100%` · min-width `0px` · padding-inline `11px` · border-radius `inherit` · color `#262626` / `#f5f5f5` (foreground) · line-height `30px`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| height | `30px` | `calc(var(--spacing) * 7.5)` | `sm:h-7.5` |
| width | `100%` | `100%` | `w-full` |
| min-width | `0px` | `calc(var(--spacing) * 0)` | `min-w-0` |
| border-radius | `inherit` | `inherit` | `rounded-[inherit]` |
| padding-inline | `11px` | `calc(calc(var(--spacing) * 3) - 1px)` | `px-[calc(--spacing(3)-1px)]` |
| line-height | `30px` | `calc(var(--spacing) * 7.5)` | `sm:leading-7.5` |
| color | `#262626` / `#f5f5f5` | `foreground` | `text-foreground` |
| outline-style | `none` | `none` | `outline-none` |
| transition | `background-color 5000000000ms ease-in-out 0ms` | `background-color 5000000s ease-in-out 0s` | `[transition:background-color_5000000s_ease-in-out_0s]` |

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| :autofill | -webkit-text-fill-color | `#262626` / `#f5f5f5` | `autofill:[-webkit-text-fill-color:var(--foreground)]` |

**element — mobile differences**: height `34px` · line-height `34px`

**`::placeholder` — default**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| color | `#686868b8` / `#818181b8` | `muted-foreground/72%` | `placeholder:text-muted-foreground/72` |

#### when `size === "sm"`

```txt
h-7.5 px-[calc(--spacing(2.5)-1px)] leading-7.5 sm:h-6.5 sm:leading-6.5
```

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| :autofill | -webkit-text-fill-color | `#262626` / `#f5f5f5` | `autofill:[-webkit-text-fill-color:var(--foreground)]` |

**element — mobile differences**: height `34px` · line-height `34px`

#### when `size === "lg"`

```txt
h-9.5 leading-9.5 sm:h-8.5 sm:leading-8.5
```

**element — default (changes vs base)**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| height | `34px` | `calc(var(--spacing) * 8.5)` | `sm:h-8.5` |
| line-height | `34px` | `calc(var(--spacing) * 8.5)` | `sm:leading-8.5` |

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| :autofill | -webkit-text-fill-color | `#262626` / `#f5f5f5` | `autofill:[-webkit-text-fill-color:var(--foreground)]` |

**element — mobile differences**: height `38px` · line-height `38px`

#### when `props.type === "search"`

```txt
[&::-webkit-search-cancel-button]:appearance-none [&::-webkit-search-decoration]:appearance-none [&::-webkit-search-results-button]:appearance-none [&::-webkit-search-results-decoration]:appearance-none
```

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| :autofill | -webkit-text-fill-color | `#262626` / `#f5f5f5` | `autofill:[-webkit-text-fill-color:var(--foreground)]` |

**element — mobile differences**: height `34px` · line-height `34px`

**`::-webkit-search-cancel-button` — default (changes vs base)**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| appearance | `none` | `none` | `[&::-webkit-search-cancel-button]:appearance-none` |

**`::-webkit-search-decoration` — default (changes vs base)**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| appearance | `none` | `none` | `[&::-webkit-search-decoration]:appearance-none` |

**`::-webkit-search-results-button` — default (changes vs base)**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| appearance | `none` | `none` | `[&::-webkit-search-results-button]:appearance-none` |

**`::-webkit-search-results-decoration` — default (changes vs base)**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| appearance | `none` | `none` | `[&::-webkit-search-results-decoration]:appearance-none` |

#### when `props.type === "file"`

```txt
text-muted-foreground file:me-3 file:bg-transparent file:font-medium file:text-foreground file:text-sm
```

**element — default (changes vs base)**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| color | `#686868` / `#818181` | `muted-foreground` | `text-muted-foreground` |

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| :autofill | -webkit-text-fill-color | `#262626` / `#f5f5f5` | `autofill:[-webkit-text-fill-color:var(--foreground)]` |

**element — mobile differences**: height `34px` · line-height `34px`

**`::file-selector-button` — default (changes vs base)**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| margin-inline-end | `12px` | `calc(var(--spacing) * 3)` | `file:me-3` |
| background-color | `#00000000` | `transparent` | `file:bg-transparent` |
| font-size | `14px` | `text-sm` | `file:text-sm` |
| line-height | `20px` | `var(--tw-leading, var(--text-sm--line-height))` | `file:text-sm` |
| font-weight | `500` | `font-weight-medium` | `file:font-medium` |
| color | `#262626` / `#f5f5f5` | `foreground` | `file:text-foreground` |

