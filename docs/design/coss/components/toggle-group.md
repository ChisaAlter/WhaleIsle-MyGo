# Toggle Group

Source: `apps/ui/registry/default/ui/toggle-group.tsx` @ `b2336c45f2b7` · docs: `apps/ui/content/docs/components/toggle-group.mdx`

> Provides a shared state to a series of toggle buttons.

Depends on: `@base-ui/react/toggle`, `@base-ui/react/toggle-group`, `@/ui/separator`, `@/ui/toggle`

Exports: `ToggleGroup`, `ToggleGroupItem`, `ToggleGroupSeparator`

## Behavior & keyboard

- Group of toggles with roving focus (arrow keys), `Enter`/`Space` toggles; single or multiple selection; `data-pressed` on pressed toggles. _(inferred: Base UI primitive behavior; not restated in coss docs)_

## How to read

Each part lists the exact class strings, then **resolved** values for the desktop environment (≥640px so all `sm:` rules apply; also md/lg/xl; fine pointer; hover-capable). Values show `light` / `dark` when they differ. *State* tables only list properties that change versus the part's default. *Mobile* lists properties that differ at 375px / coarse pointer. `src` is the class that wins the cascade. See `../README.md` for conventions.

## Parts

### `ToggleGroup` → `[data-slot=toggle-group]` · `<ToggleGroupPrimitive>` (line 31)

Attributes: `data-size={size}`, `data-variant={variant}`, `orientation={orientation}`

Prop defaults: `variant="default"`, `size="default"`, `orientation="horizontal"`

#### base

```txt
flex w-fit *:focus-visible:z-10 dark:*:[[data-slot=separator]:has(+[data-slot=toggle]:hover)]:before:bg-input/64 dark:*:[[data-slot=separator]:has(+[data-slot=toggle][data-pressed])]:before:bg-input dark:*:[[data-slot=toggle]:hover+[data-slot=separator]]:before:bg-input/64 dark:*:[[data-slot=toggle][data-pressed]+[data-slot=separator]]:before:bg-input
```

**element — default**

Key metrics: display `flex` · width `fit-content`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| display | `flex` | `flex` | `flex` |
| width | `fit-content` | `fit-content` | `w-fit` |

**`> *:focus-visible` — default**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| z-index | `10` | `10` | `*:focus-visible:z-10` |

**`> *:is([data-slot=separator]:has(+[data-slot=toggle]:hover))::before` — default**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| content | — / `""` | `var(--tw-content)` | `dark:*:[[data-slot=separator]:has(+[data-slot=toggle]:hover)]:before:bg-input/64` |
| background-color | — / `#ffffff0d` | `input/64%` | `dark:*:[[data-slot=separator]:has(+[data-slot=toggle]:hover)]:before:bg-input/64` |

**`> *:is([data-slot=separator]:has(+[data-slot=toggle][data-pressed]))::before` — default**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| content | — / `""` | `var(--tw-content)` | `dark:*:[[data-slot=separator]:has(+[data-slot=toggle][data-pressed])]:before:bg-input` |
| background-color | — / `#ffffff14` | `input` | `dark:*:[[data-slot=separator]:has(+[data-slot=toggle][data-pressed])]:before:bg-input` |

**`> *:is([data-slot=toggle]:hover+[data-slot=separator])::before` — default**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| content | — / `""` | `var(--tw-content)` | `dark:*:[[data-slot=toggle]:hover+[data-slot=separator]]:before:bg-input/64` |
| background-color | — / `#ffffff0d` | `input/64%` | `dark:*:[[data-slot=toggle]:hover+[data-slot=separator]]:before:bg-input/64` |

**`> *:is([data-slot=toggle][data-pressed]+[data-slot=separator])::before` — default**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| content | — / `""` | `var(--tw-content)` | `dark:*:[[data-slot=toggle][data-pressed]+[data-slot=separator]]:before:bg-input` |
| background-color | — / `#ffffff14` | `input` | `dark:*:[[data-slot=toggle][data-pressed]+[data-slot=separator]]:before:bg-input` |

#### when `orientation === "horizontal"`

```txt
*:pointer-coarse:after:min-w-auto
```

**`> *::after` — mobile differences**: content `""` · min-width `auto`

#### when `!(orientation === "horizontal")`

```txt
*:pointer-coarse:after:min-h-auto
```

**`> *::after` — mobile differences**: content `""` · min-height `auto`

#### when `variant === "default"`

```txt
gap-0.5
```

**element — default (changes vs base)**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| gap | `2px` | `calc(var(--spacing) * 0.5)` | `gap-0.5` |

#### when `!(variant === "default") && orientation === "horizontal"`

```txt
*:not-first:rounded-s-none *:not-last:rounded-e-none *:not-first:border-s-0 *:not-last:border-e-0 *:not-first:not-data-[slot=separator]:before:-start-[0.5px] *:not-last:not-data-[slot=separator]:before:-end-[0.5px] *:not-first:before:rounded-s-none *:not-last:before:rounded-e-none
```

**`> *:not(*:first-child)` — default (changes vs base)**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| border-start-start-radius | `0` | `0` | `*:not-first:rounded-s-none` |
| border-end-start-radius | `0` | `0` | `*:not-first:rounded-s-none` |
| border-inline-start-style | `solid` | `var(--tw-border-style)` | `*:not-first:border-s-0` |
| border-inline-start-width | `0px` | `0px` | `*:not-first:border-s-0` |

**`> *:not(*:first-child)::before` — default (changes vs base)**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| content | `""` | `var(--tw-content)` | `*:not-first:before:rounded-s-none` |
| border-start-start-radius | `0` | `0` | `*:not-first:before:rounded-s-none` |
| border-end-start-radius | `0` | `0` | `*:not-first:before:rounded-s-none` |

**`> *:not(*:first-child):not(*[data-slot="separator"])::before` — default (changes vs base)**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| content | `""` | `var(--tw-content)` | `*:not-first:not-data-[slot=separator]:before:-start-[0.5px]` |
| inset-inline-start | `-0.5px` | `calc(0.5px * -1)` | `*:not-first:not-data-[slot=separator]:before:-start-[0.5px]` |

**`> *:not(*:last-child)` — default (changes vs base)**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| border-start-end-radius | `0` | `0` | `*:not-last:rounded-e-none` |
| border-end-end-radius | `0` | `0` | `*:not-last:rounded-e-none` |
| border-inline-end-style | `solid` | `var(--tw-border-style)` | `*:not-last:border-e-0` |
| border-inline-end-width | `0px` | `0px` | `*:not-last:border-e-0` |

**`> *:not(*:last-child)::before` — default (changes vs base)**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| content | `""` | `var(--tw-content)` | `*:not-last:before:rounded-e-none` |
| border-start-end-radius | `0` | `0` | `*:not-last:before:rounded-e-none` |
| border-end-end-radius | `0` | `0` | `*:not-last:before:rounded-e-none` |

**`> *:not(*:last-child):not(*[data-slot="separator"])::before` — default (changes vs base)**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| content | `""` | `var(--tw-content)` | `*:not-last:not-data-[slot=separator]:before:-end-[0.5px]` |
| inset-inline-end | `-0.5px` | `calc(0.5px * -1)` | `*:not-last:not-data-[slot=separator]:before:-end-[0.5px]` |

#### when `!(variant === "default") && !(orientation === "horizontal")`

```txt
flex-col *:not-first:rounded-t-none *:not-last:rounded-b-none *:not-first:border-t-0 *:not-last:border-b-0 *:not-first:not-data-[slot=separator]:before:-top-[0.5px] *:not-last:not-data-[slot=separator]:before:-bottom-[0.5px] *:not-first:before:rounded-t-none *:not-last:before:rounded-b-none *:data-[slot=toggle]:not-last:before:hidden dark:*:last:before:hidden dark:*:first:before:block
```

**element — default (changes vs base)**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| flex-direction | `column` | `column` | `flex-col` |

**`> *:first-child::before` — default (changes vs base)**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| content | — / `""` | `var(--tw-content)` | `dark:*:first:before:block` |
| display | — / `block` | `block` | `dark:*:first:before:block` |

**`> *:last-child::before` — default (changes vs base)**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| content | — / `""` | `var(--tw-content)` | `dark:*:last:before:hidden` |
| display | — / `none` | `none` | `dark:*:last:before:hidden` |

**`> *:not(*:first-child)` — default (changes vs base)**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| border-top-left-radius | `0` | `0` | `*:not-first:rounded-t-none` |
| border-top-right-radius | `0` | `0` | `*:not-first:rounded-t-none` |
| border-top-style | `solid` | `var(--tw-border-style)` | `*:not-first:border-t-0` |
| border-top-width | `0px` | `0px` | `*:not-first:border-t-0` |

**`> *:not(*:first-child)::before` — default (changes vs base)**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| content | `""` | `var(--tw-content)` | `*:not-first:before:rounded-t-none` |
| border-top-left-radius | `0` | `0` | `*:not-first:before:rounded-t-none` |
| border-top-right-radius | `0` | `0` | `*:not-first:before:rounded-t-none` |

**`> *:not(*:first-child):not(*[data-slot="separator"])::before` — default (changes vs base)**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| content | `""` | `var(--tw-content)` | `*:not-first:not-data-[slot=separator]:before:-top-[0.5px]` |
| top | `-0.5px` | `calc(0.5px * -1)` | `*:not-first:not-data-[slot=separator]:before:-top-[0.5px]` |

**`> *:not(*:last-child)` — default (changes vs base)**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| border-bottom-right-radius | `0` | `0` | `*:not-last:rounded-b-none` |
| border-bottom-left-radius | `0` | `0` | `*:not-last:rounded-b-none` |
| border-bottom-style | `solid` | `var(--tw-border-style)` | `*:not-last:border-b-0` |
| border-bottom-width | `0px` | `0px` | `*:not-last:border-b-0` |

**`> *:not(*:last-child)::before` — default (changes vs base)**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| content | `""` | `var(--tw-content)` | `*:not-last:before:rounded-b-none` |
| border-bottom-right-radius | `0` | `0` | `*:not-last:before:rounded-b-none` |
| border-bottom-left-radius | `0` | `0` | `*:not-last:before:rounded-b-none` |

**`> *:not(*:last-child):not(*[data-slot="separator"])::before` — default (changes vs base)**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| content | `""` | `var(--tw-content)` | `*:not-last:not-data-[slot=separator]:before:-bottom-[0.5px]` |
| bottom | `-0.5px` | `calc(0.5px * -1)` | `*:not-last:not-data-[slot=separator]:before:-bottom-[0.5px]` |

**`> *[data-slot="toggle"]:not(*:last-child)::before` — default (changes vs base)**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| content | `""` | `var(--tw-content)` | `*:data-[slot=toggle]:not-last:before:hidden` |
| display | `none` | `none` | `*:data-[slot=toggle]:not-last:before:hidden` |

### `ToggleGroupItem`

No own classes (re-export / wrapper of the primitive).

### `ToggleGroupSeparator` → (no data-slot) · `<Separator>` (line 92)

Attributes: `orientation={orientation}`

Prop defaults: `orientation="vertical"`

#### base

```txt
pointer-events-none relative bg-input before:absolute before:inset-0 dark:before:bg-input/32
```

**element — default**

Key metrics: background-color `#0000001a` / `#ffffff14` (input)

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| pointer-events | `none` | `none` | `pointer-events-none` |
| position | `relative` | `relative` | `relative` |
| background-color | `#0000001a` / `#ffffff14` | `input` | `bg-input` |

**`::before` — default**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| content | `""` | `var(--tw-content)` | `before:inset-0` |
| position | `absolute` | `absolute` | `before:absolute` |
| inset | `0px` | `calc(var(--spacing) * 0)` | `before:inset-0` |
| background-color | — / `#ffffff07` | `input/32%` | `dark:before:bg-input/32` |

