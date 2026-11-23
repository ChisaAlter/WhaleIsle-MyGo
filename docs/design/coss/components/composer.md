# Composer

Source: `apps/ui/registry/default/ui/composer.tsx` @ `b2336c45f2b7` · docs: `apps/ui/content/docs/components/composer.mdx`

> A composer for chat and AI assistant interfaces, with a growing input, persistent actions, and optional mentions or slash commands.

Depends on: `@base-ui/react/button`, `@base-ui/react/merge-props`, `@base-ui/react/popover`, `@base-ui/react/use-render`, `@/ui/scroll-area`

Exports: `ComposerFrame`, `Composer`, `ComposerInput`, `ComposerToken`, `ComposerPopup`, `ComposerList`, `ComposerItem`, `ComposerStatus`, `ComposerActions`, `ComposerActionGroup`, `ComposerSubmit`

Module constants: `composerTokenClassName = wrap-break-word inline cursor-default align-baseline font-normal text-primary data-[variant=default]:font-medium data-[variant=chip]:box-decoration-clone data-[variant=chip]:rounded-md data-[variant=chip]:bg-accent data-[variant=chip]:px-1.5 data-[variant=chip]:py-px data-[variant=chip]:text-accent-foreground [&_[data-slot=composer-token-icon]:empty]:hidden [&_[data-slot=composer-token-icon]>svg]:size-3.5`

## Behavior & keyboard

- Rich chat input (contenteditable-based editor, see `packages/ui/src/lib/composer-*.ts`). `Enter` submits, `Shift+Enter` inserts newline (coss chat docs); token/mention insertion via query popups; Backspace deletes tokens atomically. _(partly inferred from source; verify against composer docs)_

Docs mentions:

- The adaptive layout starts at one visible row; setting `--composer-min-rows` above one keeps the controls below. Keyboard focus follows the same order in both layouts: input, leading actions, then trailing actions.
- ### Keyboard shortcuts
- While suggestions are open, ↑ and ↓ navigate, Enter selects, Escape dismisses, and Tab moves focus normally. Shift+Enter dismisses suggestions and inserts a line break. Otherwise, Enter submits and Shift+Enter inserts a line break in the native contenteditable input or textarea. Set `submitOnEnter="desktop"` to submit only with a fine pointer, or `"never"` to let Enter insert a line break.
- Bottom drawers automatically keep the input above the software keyboard. Place the composer after the scrolling `DrawerPanel` so it stays visible. Set the drawer height in your composition; Drawer handles keyboard and safe-area spacing automatically. Leave the Composer free to shrink in the drawer: its input scrolls in the remaining space while the action row stays visible.
- | `nativeButton` | `boolean` | `true` | Set to `false` only when rendering a non-button element; Base UI supplies keyboard behavior |
- A styled button for a result. Pass `value` to connect selection and keyboard highlighting in its parent `Composer`.
- Item IDs must be unique within each trigger. Set `disabled: true` on an item to skip it during keyboard navigation and prevent selection. `type` defaults to `"mention"`; set `"action"` for slash actions, or `"text"` to insert the item’s `text` (or its `label`) as plain characters instead of a token. Use `kind` for your own categories, such as `"person"` and `"team"`, even when both come from `@`. The selected part retains `kind`. `getComposerText(value)` uses a token’s optional `text` field, or falls back to its trigger and label. Optional `data` is an opaque string preserved through editing, undo/redo, and supported structured copy/paste; the editor never interprets it as HTML. Keep domain validation and authorization in your application.
- Input event handlers run before Composer's handlers. Call `event.preventDefault()` to override a supported keyboard or clipboard action. Base UI's `event.preventBaseUIHandler()` also stops the merged internal handler. Keep application business rules on Composer so they apply to both keyboard and pointer submission.

## How to read

Each part lists the exact class strings, then **resolved** values for the desktop environment (≥640px so all `sm:` rules apply; also md/lg/xl; fine pointer; hover-capable). Values show `light` / `dark` when they differ. *State* tables only list properties that change versus the part's default. *Mobile* lists properties that differ at 375px / coarse pointer. `src` is the class that wins the cascade. See `../README.md` for conventions.

## Parts

### `ComposerFrame` → `[data-slot=composer]` · `useRender <form>` (line 59)

Attributes: `data-layout={layout}`, `data-base-ui-swipe-ignore=`, `data-disabled={editor?.disabled ? "" : undefined}`, `data-readonly={editor?.readOnly ? "" : undefined}`

Prop defaults: `layout="stacked"`

#### base

```txt
relative flex min-h-0 min-w-0 cursor-text flex-col data-disabled:cursor-default rounded-(--composer-radius) border border-input bg-background not-dark:bg-clip-padding pb-[calc(var(--composer-padding)-1px)] text-foreground shadow-xs/5 ring-ring/24 transition-shadow [--composer-radius:var(--radius-xl)] [--composer-max-rows:4] [--composer-padding:--spacing(2)] before:pointer-events-none before:absolute before:inset-0 before:rounded-[calc(var(--composer-radius)-1px)] not-data-disabled:not-has-[[data-slot=composer-input]:focus-visible]:not-has-aria-invalid:before:shadow-[0_1px_--theme(--color-black/4%)] has-[[data-slot=composer-input]:focus-visible]:has-aria-invalid:border-destructive/64 has-[[data-slot=composer-input]:focus-visible]:has-aria-invalid:ring-destructive/16 has-[[data-slot=composer-input]:focus-visible]:border-ring has-aria-invalid:border-destructive/36 has-[[data-slot=composer-input]:focus-visible]:shadow-none has-aria-invalid:shadow-none has-[[data-slot=composer-input]:focus-visible]:ring-[3px] aria-invalid:border-destructive/36 aria-invalid:shadow-none data-disabled:opacity-64 data-disabled:shadow-none dark:bg-input/32 dark:aria-invalid:ring-destructive/24 dark:has-aria-invalid:ring-destructive/24 dark:not-data-disabled:not-has-[[data-slot=composer-input]:focus-visible]:not-has-aria-invalid:before:shadow-[0_-1px_--theme(--color-white/6%)] md:[--composer-max-rows:8] data-[layout=adaptive]:data-multiline:grid-rows-[minmax(0,1fr)_auto] data-[layout=adaptive]:grid data-[layout=adaptive]:grid-cols-[auto_minmax(0,1fr)_auto] data-[layout=adaptive]:items-center data-[layout=adaptive]:not-data-multiline:px-[calc(--spacing(2)-1px)] data-[layout=adaptive]:not-data-multiline:pt-[calc(var(--composer-padding)-1px)]
```

**element — default**

Key metrics: display `flex` · min-height `0px` · min-width `0px` · padding-bottom `7px` · border-width `1px` · border-color `#0000001a` / `#ffffff14` (input) · border-radius `14px` (runtime var --composer-radius) · background-color `#ffffff` / `#ffffff07` (background) · color `#262626` / `#f5f5f5` (foreground) · box-shadow `0 1px 2px 0 #0000000d` · transition-property `box-shadow` · transition-duration `150ms` · transition-timing-function `cubic-bezier(0.4, 0, 0.2, 1)`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| border-width | `1px` | `1px` | `border` |
| border-style | `solid` | `var(--tw-border-style)` | `border` |
| border-color | `#0000001a` / `#ffffff14` | `input` | `border-input` |
| position | `relative` | `relative` | `relative` |
| display | `flex` | `flex` | `flex` |
| min-height | `0px` | `calc(var(--spacing) * 0)` | `min-h-0` |
| min-width | `0px` | `calc(var(--spacing) * 0)` | `min-w-0` |
| cursor | `text` | `text` | `cursor-text` |
| flex-direction | `column` | `column` | `flex-col` |
| border-radius | `14px` | `runtime var --composer-radius` | `rounded-(--composer-radius)` |
| background-color | `#ffffff` / `#ffffff07` | `background` | `bg-background` |
| padding-bottom | `7px` | `calc(var(--composer-padding) - 1px)` | `pb-[calc(var(--composer-padding)-1px)]` |
| color | `#262626` / `#f5f5f5` | `foreground` | `text-foreground` |
| box-shadow | `0 1px 2px 0 #0000000d` | `var(--tw-inset-shadow), var(--tw-inset-ring-shadow), var(--tw-ring-offset-shadow), var(--tw-ring-shadow), var(` | `shadow-xs/5` |
| transition-property | `box-shadow` | `box-shadow` | `transition-shadow` |
| transition-timing-function | `cubic-bezier(0.4, 0, 0.2, 1)` | `var(--tw-ease, var(--default-transition-timing-function))` | `transition-shadow` |
| transition-duration | `150ms` | `var(--tw-duration, var(--default-transition-duration))` | `transition-shadow` |
| background-clip | `padding-box` / — | `padding-box` | `not-dark:bg-clip-padding` |

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| disabled | cursor | `default` | `data-disabled:cursor-default` |
| disabled | box-shadow | `none` | `data-disabled:shadow-none` |
| disabled | opacity | `64%` | `data-disabled:opacity-64` |
| focus-visible | border-color | `#a1a1a1` / `#737373` | `has-[[data-slot=composer-input]:focus-visible]:border-ring` |
| focus-visible | box-shadow | `0 0 0 3px #a1a1a13d` / `0 0 0 3px #7373733d` | `has-[[data-slot=composer-input]:focus-visible]:ring-[3px]` |
| invalid | border-color | `#fb2c365c` / `#fb414a5c` | `aria-invalid:border-destructive/36` |
| invalid | box-shadow | `none` | `aria-invalid:shadow-none` |
| [data-layout=adaptive] | display | `grid` | `data-[layout=adaptive]:grid` |
| [data-layout=adaptive] | grid-template-columns | `auto minmax(0,1fr) auto` | `data-[layout=adaptive]:grid-cols-[auto_minmax(0,1fr)_auto]` |
| [data-layout=adaptive] | align-items | `center` | `data-[layout=adaptive]:items-center` |
| [data-layout=adaptive] | padding-inline | `7px` | `data-[layout=adaptive]:not-data-multiline:px-[calc(--spacing(2)-1px)]` |
| [data-layout=adaptive] | padding-top | `7px` | `data-[layout=adaptive]:not-data-multiline:pt-[calc(var(--composer-padding)-1px)]` |
| focus-visible + invalid | border-color | `#fb2c36a3` / `#fb414aa3` | `has-[[data-slot=composer-input]:focus-visible]:has-aria-invalid:border-destructive/64` |
| focus-visible + invalid | box-shadow | `0 0 0 3px #fb2c3629` / `0 0 0 3px #fb414a29` | `has-[[data-slot=composer-input]:focus-visible]:ring-[3px]` |
| [data-layout=adaptive] + [data-multiline] | display | `grid` | `data-[layout=adaptive]:grid` |
| [data-layout=adaptive] + [data-multiline] | grid-template-columns | `auto minmax(0,1fr) auto` | `data-[layout=adaptive]:grid-cols-[auto_minmax(0,1fr)_auto]` |
| [data-layout=adaptive] + [data-multiline] | align-items | `center` | `data-[layout=adaptive]:items-center` |
| [data-layout=adaptive] + [data-multiline] | grid-template-rows | `minmax(0,1fr) auto` | `data-[layout=adaptive]:data-multiline:grid-rows-[minmax(0,1fr)_auto]` |

**`::before` — default**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| content | `""` | `var(--tw-content)` | `not-data-disabled:not-has-[[data-slot=composer-input]:focus-visible]:not-has-aria-invalid:before:shadow-[0_1px_--theme(--color-black/4%)]` |
| pointer-events | `none` | `none` | `before:pointer-events-none` |
| position | `absolute` | `absolute` | `before:absolute` |
| inset | `0px` | `calc(var(--spacing) * 0)` | `before:inset-0` |
| border-radius | `13px` | `calc(var(--composer-radius) - 1px)` | `before:rounded-[calc(var(--composer-radius)-1px)]` |
| box-shadow | `0 1px #0000000a` / `0 -1px #ffffff0f` | `var(--tw-inset-shadow), var(--tw-inset-ring-shadow), var(--tw-ring-offset-shadow), var(--tw-ring-shadow), var(` | `not-data-disabled:not-has-[[data-slot=composer-input]:focus-visible]:not-has-aria-invalid:before:shadow-[0_1px_--theme(--color-black/4%)]` |

**`::before` — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| disabled | box-shadow | `(unset)` | `(removed)` |
| focus-visible | box-shadow | `(unset)` | `(removed)` |
| invalid | box-shadow | `(unset)` | `(removed)` |
| focus-visible + invalid | box-shadow | `(unset)` | `(removed)` |

### `Composer`

No own classes (re-export / wrapper of the primitive).

### `ComposerInput` → `[data-slot=composer-input]` · `useRender <div>` (line 162)

Attributes: `data-empty={!textarea && editor?.isEmpty && !editor.isComposing ? "" : undefined}`, `data-placeholder={placeholder}`, `aria-placeholder={editor ? placeholder : undefined}`

#### base

```txt
field-sizing-content wrap-anywhere relative block min-h-[calc(var(--composer-min-rows,1)*1lh+--spacing(2))] w-full min-w-0 resize-none whitespace-pre-wrap border-0 bg-transparent in-[[data-layout=adaptive]:not([data-multiline])]:px-1 px-[calc(--spacing(3)-1px)] py-1 outline-none placeholder:text-muted-foreground/72 data-empty:before:pointer-events-none data-empty:before:absolute data-empty:before:text-muted-foreground/72 data-empty:before:content-[attr(data-placeholder)] md:in-data-[layout=adaptive]:min-h-[calc(var(--composer-min-rows,1)*1lh+--spacing(2))] md:min-h-[calc(var(--composer-min-rows,2)*1lh+--spacing(2))]
```

**element — default**

Key metrics: display `block` · min-height `calc(2 * 1lh + 8px)` · width `100%` · min-width `0px` · padding-inline `11px` · padding-block `4px` · border-width `0px` · border-color `#00000014` / `#ffffff0f` (border) · background-color `#00000000`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| border-width | `0px` | `0px` | `border-0` |
| border-style | `solid` | `var(--tw-border-style)` | `border-0` |
| border-color | `#00000014` / `#ffffff0f` | `border` | `(globals @layer base) * border-border` |
| position | `relative` | `relative` | `relative` |
| display | `block` | `block` | `block` |
| field-sizing | `content` | `content` | `field-sizing-content` |
| min-height | `calc(2 * 1lh + 8px)` | `calc(var(--composer-min-rows,2) * 1lh + calc(var(--spacing) * 2))` | `md:min-h-[calc(var(--composer-min-rows,2)*1lh+--spacing(2))]` |
| width | `100%` | `100%` | `w-full` |
| min-width | `0px` | `calc(var(--spacing) * 0)` | `min-w-0` |
| resize | `none` | `none` | `resize-none` |
| background-color | `#00000000` | `transparent` | `bg-transparent` |
| padding-inline | `11px` | `calc(calc(var(--spacing) * 3) - 1px)` | `px-[calc(--spacing(3)-1px)]` |
| padding-block | `4px` | `calc(var(--spacing) * 1)` | `py-1` |
| overflow-wrap | `anywhere` | `anywhere` | `wrap-anywhere` |
| white-space | `pre-wrap` | `pre-wrap` | `whitespace-pre-wrap` |
| outline-style | `none` | `none` | `outline-none` |

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| in(:where(*:is([data-layout=adaptive]:not([data-multiline])))) | padding-inline | `4px` | `in-[[data-layout=adaptive]:not([data-multiline])]:px-1` |
| in([data-layout=adaptive]) | min-height | `calc(1 * 1lh + 8px)` | `md:in-data-[layout=adaptive]:min-h-[calc(var(--composer-min-rows,1)*1lh+--spacing(2))]` |

**element — mobile differences**: min-height `calc(1 * 1lh + 8px)`

**`::before` — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| [data-empty] | content | `attr(data-placeholder)` | `data-empty:before:content-[attr(data-placeholder)]` |
| [data-empty] | pointer-events | `none` | `data-empty:before:pointer-events-none` |
| [data-empty] | position | `absolute` | `data-empty:before:absolute` |
| [data-empty] | color | `#686868b8` / `#818181b8` | `data-empty:before:text-muted-foreground/72` |

**`::placeholder` — default**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| color | `#686868b8` / `#818181b8` | `muted-foreground/72%` | `placeholder:text-muted-foreground/72` |

### `ComposerInput` → `[data-slot=composer-scroll-area]` · `<ScrollArea>` (line 203)

Attributes: `overscrollContain=true`, `scrollFade=true`

#### base

```txt
in-[[data-layout=adaptive][data-multiline]]:col-span-full in-[[data-layout=adaptive][data-multiline]]:col-start-1 in-data-[layout=adaptive]:col-start-2 in-data-[layout=adaptive]:row-start-1 flex h-auto! min-h-0 min-w-0 flex-col in-[[data-layout=adaptive][data-multiline]]:self-stretch in-[[data-layout=adaptive]:not([data-multiline])]:rounded-none rounded-t-[inherit] text-base [--composer-scroll-padding:var(--composer-padding)] in-[[data-layout=adaptive]:not([data-multiline])]:[--composer-scroll-padding:0px] **:data-[slot=scroll-area-viewport]:h-auto! **:data-[slot=scroll-area-viewport]:max-h-[min(var(--composer-max-height,calc(var(--composer-max-rows,8)*var(--composer-line-height,1lh)+var(--composer-input-padding,--spacing(2))+var(--composer-scroll-padding))),50dvh)] **:data-[slot=scroll-area-viewport]:min-h-0 **:data-[slot=scroll-area-content]:pt-[max(0px,calc(var(--composer-scroll-padding)-1px))] **:data-[slot=scroll-area-viewport]:pb-0! md:text-sm
```

**element — default**

Key metrics: display `flex` · height `auto` · min-height `0px` · min-width `0px` · font-size `14px` (text-sm) · line-height `20px`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| display | `flex` | `flex` | `flex` |
| min-height | `0px` | `calc(var(--spacing) * 0)` | `min-h-0` |
| min-width | `0px` | `calc(var(--spacing) * 0)` | `min-w-0` |
| flex-direction | `column` | `column` | `flex-col` |
| border-top-left-radius | `inherit` | `inherit` | `rounded-t-[inherit]` |
| border-top-right-radius | `inherit` | `inherit` | `rounded-t-[inherit]` |
| font-size | `14px` | `text-sm` | `md:text-sm` |
| line-height | `20px` | `var(--tw-leading, var(--text-sm--line-height))` | `md:text-sm` |
| height | `auto` | `auto` | `h-auto!` |

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| in(:where(*:is([data-layout=adaptive][data-multiline]))) | grid-column | `1 / -1` | `in-[[data-layout=adaptive][data-multiline]]:col-span-full` |
| in(:where(*:is([data-layout=adaptive][data-multiline]))) | grid-column-start | `1` | `in-[[data-layout=adaptive][data-multiline]]:col-start-1` |
| in(:where(*:is([data-layout=adaptive][data-multiline]))) | align-self | `stretch` | `in-[[data-layout=adaptive][data-multiline]]:self-stretch` |
| in([data-layout=adaptive]) | grid-column-start | `2` | `in-data-[layout=adaptive]:col-start-2` |
| in([data-layout=adaptive]) | grid-row-start | `1` | `in-data-[layout=adaptive]:row-start-1` |
| in(:where(*:is([data-layout=adaptive]:not([data-multiline])))) | border-radius | `0` | `in-[[data-layout=adaptive]:not([data-multiline])]:rounded-none` |

**element — mobile differences**: font-size `16px` · line-height `24px`

**`*[data-slot="scroll-area-content"]` — default**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| padding-top | `max(0px, 7px)` | `max(0px, calc(var(--composer-scroll-padding) - 1px))` | `**:data-[slot=scroll-area-content]:pt-[max(0px,calc(var(--composer-scroll-padding)-1px))]` |

**`*[data-slot="scroll-area-viewport"]` — default**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| max-height | `min(calc(8 * 1lh + 8px + 8px), 50dvh)` | `min(var(--composer-max-height,calc(var(--composer-max-rows,8) * var(--composer-line-height,1lh) + var(--compos` | `**:data-[slot=scroll-area-viewport]:max-h-[min(var(--composer-max-height,calc(var(--composer-max-rows,8)*var(--composer-line-height,1lh)+var(--composer-input-padding,--spacing(2))+var(--composer-scroll-padding))),50dvh)]` |
| min-height | `0px` | `calc(var(--spacing) * 0)` | `**:data-[slot=scroll-area-viewport]:min-h-0` |
| height | `auto` | `auto` | `**:data-[slot=scroll-area-viewport]:h-auto!` |
| padding-bottom | `0px` | `calc(var(--spacing) * 0)` | `**:data-[slot=scroll-area-viewport]:pb-0!` |

**`*[data-slot="scroll-area-viewport"]` — mobile differences**: max-height `min(calc(4 * 1lh + 8px + 8px), 50dvh)`

### `ComposerToken` → `[data-slot=composer-token]` · `<span>` (line 251)

Attributes: `contentEditable={false}`, `data-kind={token.kind}`, `data-type={token.type}`, `data-variant={variant}`

Prop defaults: `variant="default"`

#### base

```txt
wrap-break-word inline cursor-default align-baseline font-normal text-primary data-[variant=default]:font-medium data-[variant=chip]:box-decoration-clone data-[variant=chip]:rounded-md data-[variant=chip]:bg-accent data-[variant=chip]:px-1.5 data-[variant=chip]:py-px data-[variant=chip]:text-accent-foreground [&_[data-slot=composer-token-icon]:empty]:hidden [&_[data-slot=composer-token-icon]>svg]:size-3.5
```

**element — default**

Key metrics: display `inline` · color `#262626` / `#f5f5f5` (primary) · font-weight `400` (font-weight-normal)

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| display | `inline` | `inline` | `inline` |
| cursor | `default` | `default` | `cursor-default` |
| vertical-align | `baseline` | `baseline` | `align-baseline` |
| font-weight | `400` | `font-weight-normal` | `font-normal` |
| overflow-wrap | `break-word` | `break-word` | `wrap-break-word` |
| color | `#262626` / `#f5f5f5` | `primary` | `text-primary` |

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| [data-variant=default] | font-weight | `500` | `data-[variant=default]:font-medium` |
| [data-variant=chip] | border-radius | `8px` | `data-[variant=chip]:rounded-md` |
| [data-variant=chip] | background-color | `#0000000a` / `#ffffff0a` | `data-[variant=chip]:bg-accent` |
| [data-variant=chip] | -webkit-box-decoration-break | `clone` | `data-[variant=chip]:box-decoration-clone` |
| [data-variant=chip] | box-decoration-break | `clone` | `data-[variant=chip]:box-decoration-clone` |
| [data-variant=chip] | padding-inline | `6px` | `data-[variant=chip]:px-1.5` |
| [data-variant=chip] | padding-block | `1px` | `data-[variant=chip]:py-px` |

**`[data-slot=composer-token-icon]:empty` — default**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| display | `none` | `none` | `[&_[data-slot=composer-token-icon]:empty]:hidden` |

**`[data-slot=composer-token-icon]>svg` — default**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| width | `14px` | `calc(var(--spacing) * 3.5)` | `[&_[data-slot=composer-token-icon]>svg]:size-3.5` |
| height | `14px` | `calc(var(--spacing) * 3.5)` | `[&_[data-slot=composer-token-icon]>svg]:size-3.5` |

### `ComposerToken` → `[data-slot=composer-token-icon]` · `<span>` (line 261)

Attributes: `aria-hidden=true`

Prop defaults: `variant="default"`

#### base

```txt
inline whitespace-nowrap opacity-80 after:content-['\2060\00a0'] [&>svg]:inline-block [&>svg]:align-[-0.125em]
```

**element — default**

Key metrics: display `inline` · opacity `80%`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| display | `inline` | `inline` | `inline` |
| white-space | `nowrap` | `nowrap` | `whitespace-nowrap` |
| opacity | `80%` | `80%` | `opacity-80` |

**`::after` — default**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| content | `'\2060\00a0'` | `var(--tw-content)` | `after:content-['\2060\00a0']` |

**`>svg` — default**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| display | `inline-block` | `inline-block` | `[&>svg]:inline-block` |
| vertical-align | `-0.125em` | `-0.125em` | `[&>svg]:align-[-0.125em]` |

### `ComposerToken` → `[data-slot=composer-token-label]` · `<span>` (line 269)

Prop defaults: `variant="default"`

### `ComposerPopup` → `[data-slot=composer-positioner]` · `<PopoverPrimitive.Positioner>` (line 292)

Attributes: `anchor={anchor ?? anchorRef ?? undefined}`, `side={side}`, `align={align}`, `sideOffset={sideOffset}`, `alignOffset={alignOffset}`

Prop defaults: `side="top"`, `align="start"`, `sideOffset=4`

Dynamic className inputs (not statically resolvable): `positionerClassName`

#### base

```txt
z-50 w-(--anchor-width)
```

**element — default**

Key metrics: width `var(--anchor-width)` (runtime var --anchor-width) · z-index `50`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| z-index | `50` | `50` | `z-50` |
| width | `var(--anchor-width)` | `runtime var --anchor-width` | `w-(--anchor-width)` |

### `ComposerPopup` → `[data-slot=composer-popup]` · `<PopoverPrimitive.Popup>` (line 301)

Attributes: `role={editor ? "presentation" : undefined}`, `initialFocus={editor ? false : undefined}`, `finalFocus={editor ? false : undefined}`

Prop defaults: `side="top"`, `align="start"`, `sideOffset=4`

#### base

```txt
relative w-(--anchor-width) max-w-(--available-width) rounded-lg border bg-popover not-dark:bg-clip-padding text-popover-foreground shadow-lg/5 outline-none transition-opacity before:pointer-events-none before:absolute before:inset-0 before:rounded-[calc(var(--radius-lg)-1px)] before:shadow-[0_1px_--theme(--color-black/4%)] data-ending-style:invisible data-starting-style:opacity-0 dark:before:shadow-[0_-1px_--theme(--color-white/6%)]
```

**element — default**

Key metrics: width `var(--anchor-width)` (runtime var --anchor-width) · max-width `var(--available-width)` (runtime var --available-width) · border-width `1px` · border-color `#00000014` / `#ffffff0f` (border) · border-radius `10px` (radius) · background-color `#ffffff` / `#1d1d1d` (popover) · color `#262626` / `#f5f5f5` (popover-foreground) · box-shadow `0 10px 15px -3px #0000000d, 0 4px 6px -4px #0000000d` · transition-property `opacity` · transition-duration `150ms` · transition-timing-function `cubic-bezier(0.4, 0, 0.2, 1)`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| border-width | `1px` | `1px` | `border` |
| border-style | `solid` | `var(--tw-border-style)` | `border` |
| border-color | `#00000014` / `#ffffff0f` | `border` | `(globals @layer base) * border-border` |
| position | `relative` | `relative` | `relative` |
| width | `var(--anchor-width)` | `runtime var --anchor-width` | `w-(--anchor-width)` |
| max-width | `var(--available-width)` | `runtime var --available-width` | `max-w-(--available-width)` |
| border-radius | `10px` | `radius` | `rounded-lg` |
| background-color | `#ffffff` / `#1d1d1d` | `popover` | `bg-popover` |
| color | `#262626` / `#f5f5f5` | `popover-foreground` | `text-popover-foreground` |
| box-shadow | `0 10px 15px -3px #0000000d, 0 4px 6px -4px #0000000d` | `var(--tw-inset-shadow), var(--tw-inset-ring-shadow), var(--tw-ring-offset-shadow), var(--tw-ring-shadow), var(` | `shadow-lg/5` |
| transition-property | `opacity` | `opacity` | `transition-opacity` |
| transition-timing-function | `cubic-bezier(0.4, 0, 0.2, 1)` | `var(--tw-ease, var(--default-transition-timing-function))` | `transition-opacity` |
| transition-duration | `150ms` | `var(--tw-duration, var(--default-transition-duration))` | `transition-opacity` |
| outline-style | `none` | `none` | `outline-none` |
| background-clip | `padding-box` / — | `padding-box` | `not-dark:bg-clip-padding` |

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| ending-style (exit to) | visibility | `hidden` | `data-ending-style:invisible` |
| starting-style (enter from) | opacity | `0%` | `data-starting-style:opacity-0` |

**`::before` — default**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| content | `""` | `var(--tw-content)` | `before:shadow-[0_1px_--theme(--color-black/4%)]` |
| pointer-events | `none` | `none` | `before:pointer-events-none` |
| position | `absolute` | `absolute` | `before:absolute` |
| inset | `0px` | `calc(var(--spacing) * 0)` | `before:inset-0` |
| border-radius | `9px` | `calc(var(--radius-lg) - 1px)` | `before:rounded-[calc(var(--radius-lg)-1px)]` |
| box-shadow | `0 1px #0000000a` / `0 -1px #ffffff0f` | `var(--tw-inset-shadow), var(--tw-inset-ring-shadow), var(--tw-ring-offset-shadow), var(--tw-ring-shadow), var(` | `before:shadow-[0_1px_--theme(--color-black/4%)]` |

### `ComposerList` → `[data-slot=composer-list]` · `useRender <div>` (line 329)

Attributes: `id={suggestions?.listId}`, `role={suggestions ? "listbox" : undefined}`, `aria-label={suggestions ? "Suggestions" : undefined}`

#### base

```txt
not-empty:scroll-py-1 not-empty:p-1 in-data-has-overflow-y:pe-3
```

**element — default**

Key metrics: padding `4px`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| scroll-padding-block | `4px` | `calc(var(--spacing) * 1)` | `not-empty:scroll-py-1` |
| padding | `4px` | `calc(var(--spacing) * 1)` | `not-empty:p-1` |

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| :empty | scroll-padding-block | `(unset)` | `(removed)` |
| :empty | padding | `(unset)` | `(removed)` |
| in([data-has-overflow-y]) | padding-inline-end | `12px` | `in-data-has-overflow-y:pe-3` |

### `ComposerList` → (no data-slot) · `<ScrollArea>` (line 352)

Attributes: `overscrollContain=true`, `scrollbarGutter=true`, `scrollFade=true`

#### base

```txt
h-auto! **:data-[slot=scroll-area-viewport]:h-auto! **:data-[slot=scroll-area-viewport]:max-h-[min(var(--available-height),16rem)]
```

**element — default**

Key metrics: height `auto`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| height | `auto` | `auto` | `h-auto!` |

**`*[data-slot="scroll-area-viewport"]` — default**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| max-height | `min(var(--available-height), 256px)` | `min(var(--available-height), 16rem)` | `**:data-[slot=scroll-area-viewport]:max-h-[min(var(--available-height),16rem)]` |
| height | `auto` | `auto` | `**:data-[slot=scroll-area-viewport]:h-auto!` |

### `ComposerItem` → `[data-slot=composer-item]` · `useRender <button>` (line 382)

Attributes: `type={"button" as const}`

#### base

```txt
flex min-h-8 w-full cursor-default select-none items-center gap-2 rounded-sm px-2 py-1 text-start text-base outline-none data-disabled:pointer-events-none data-highlighted:bg-accent data-highlighted:text-accent-foreground data-disabled:opacity-64 sm:min-h-7 sm:text-sm
```

**element — default**

Key metrics: display `flex` · min-height `28px` · width `100%` · padding-inline `8px` · padding-block `4px` · gap `8px` · border-radius `6px` · font-size `14px` (text-sm) · line-height `20px`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| display | `flex` | `flex` | `flex` |
| min-height | `28px` | `calc(var(--spacing) * 7)` | `sm:min-h-7` |
| width | `100%` | `100%` | `w-full` |
| cursor | `default` | `default` | `cursor-default` |
| align-items | `center` | `center` | `items-center` |
| gap | `8px` | `calc(var(--spacing) * 2)` | `gap-2` |
| border-radius | `6px` | `calc(var(--radius) - 4px)` | `rounded-sm` |
| padding-inline | `8px` | `calc(var(--spacing) * 2)` | `px-2` |
| padding-block | `4px` | `calc(var(--spacing) * 1)` | `py-1` |
| text-align | `start` | `start` | `text-start` |
| font-size | `14px` | `text-sm` | `sm:text-sm` |
| line-height | `20px` | `var(--tw-leading, var(--text-sm--line-height))` | `sm:text-sm` |
| outline-style | `none` | `none` | `outline-none` |
| -webkit-user-select | `none` | `none` | `select-none` |
| user-select | `none` | `none` | `select-none` |

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| disabled | pointer-events | `none` | `data-disabled:pointer-events-none` |
| disabled | opacity | `64%` | `data-disabled:opacity-64` |
| highlighted | background-color | `#0000000a` / `#ffffff0a` | `data-highlighted:bg-accent` |
| highlighted | color | `#262626` / `#f5f5f5` | `data-highlighted:text-accent-foreground` |

**element — mobile differences**: min-height `32px` · font-size `16px` · line-height `24px`

### `ComposerStatus` → `[data-slot=composer-status]` · `useRender <div>` (line 407)

Attributes: `role=status`

#### base

```txt
px-3 py-2 font-medium text-muted-foreground text-xs empty:m-0 empty:p-0
```

**element — default**

Key metrics: padding-inline `12px` · padding-block `8px` · color `#686868` / `#818181` (muted-foreground) · font-size `12px` (text-xs) · line-height `16px` · font-weight `500` (font-weight-medium)

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| padding-inline | `12px` | `calc(var(--spacing) * 3)` | `px-3` |
| padding-block | `8px` | `calc(var(--spacing) * 2)` | `py-2` |
| font-size | `12px` | `text-xs` | `text-xs` |
| line-height | `16px` | `var(--tw-leading, var(--text-xs--line-height))` | `text-xs` |
| font-weight | `500` | `font-weight-medium` | `font-medium` |
| color | `#686868` / `#818181` | `muted-foreground` | `text-muted-foreground` |

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| :empty | margin | `0px` | `empty:m-0` |
| :empty | padding | `0px` | `empty:p-0` |

### `ComposerActions` → `[data-slot=composer-actions]` · `<div>` (line 428)

#### base

```txt
in-data-[layout=adaptive]:col-span-full in-data-[layout=adaptive]:row-start-2 mx-[calc(--spacing(2)-1px)] flex in-[[data-layout=adaptive]:not([data-multiline])]:contents shrink-0 items-center justify-between gap-2
```

**element — default**

Key metrics: display `flex` · gap `8px`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| margin-inline | `7px` | `calc(calc(var(--spacing) * 2) - 1px)` | `mx-[calc(--spacing(2)-1px)]` |
| display | `flex` | `flex` | `flex` |
| flex-shrink | `0` | `0` | `shrink-0` |
| align-items | `center` | `center` | `items-center` |
| justify-content | `space-between` | `space-between` | `justify-between` |
| gap | `8px` | `calc(var(--spacing) * 2)` | `gap-2` |

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| in([data-layout=adaptive]) | grid-column | `1 / -1` | `in-data-[layout=adaptive]:col-span-full` |
| in([data-layout=adaptive]) | grid-row-start | `2` | `in-data-[layout=adaptive]:row-start-2` |
| in(:where(*:is([data-layout=adaptive]:not([data-multiline])))) | display | `contents` | `in-[[data-layout=adaptive]:not([data-multiline])]:contents` |

### `ComposerActionGroup` → `[data-slot=composer-action-group]` · `<div>` (line 445)

Attributes: `data-align={align}`

Prop defaults: `align="start"`

#### base

```txt
in-[[data-layout=adaptive]:not([data-multiline])]:row-start-1 flex min-w-0 flex-wrap items-center gap-1
```

**element — default**

Key metrics: display `flex` · min-width `0px` · gap `4px`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| display | `flex` | `flex` | `flex` |
| min-width | `0px` | `calc(var(--spacing) * 0)` | `min-w-0` |
| flex-wrap | `wrap` | `wrap` | `flex-wrap` |
| align-items | `center` | `center` | `items-center` |
| gap | `4px` | `calc(var(--spacing) * 1)` | `gap-1` |

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| in(:where(*:is([data-layout=adaptive]:not([data-multiline])))) | grid-row-start | `1` | `in-[[data-layout=adaptive]:not([data-multiline])]:row-start-1` |

#### when `align === "start"`

```txt
in-[[data-layout=adaptive]:not([data-multiline])]:col-start-1 in-[[data-layout=adaptive]:not([data-multiline])]:me-2
```

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| in(:where(*:is([data-layout=adaptive]:not([data-multiline])))) | grid-column-start | `1` | `in-[[data-layout=adaptive]:not([data-multiline])]:col-start-1` |
| in(:where(*:is([data-layout=adaptive]:not([data-multiline])))) | grid-row-start | `1` | `in-[[data-layout=adaptive]:not([data-multiline])]:row-start-1` |
| in(:where(*:is([data-layout=adaptive]:not([data-multiline])))) | margin-inline-end | `8px` | `in-[[data-layout=adaptive]:not([data-multiline])]:me-2` |

#### when `!(align === "start")`

```txt
in-[[data-layout=adaptive]:not([data-multiline])]:col-start-3 in-[[data-layout=adaptive]:not([data-multiline])]:ms-2 ms-auto justify-end
```

**element — default (changes vs base)**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| margin-inline-start | `auto` | `auto` | `ms-auto` |
| justify-content | `flex-end` | `flex-end` | `justify-end` |

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| in(:where(*:is([data-layout=adaptive]:not([data-multiline])))) | margin-inline-start | `8px` | `in-[[data-layout=adaptive]:not([data-multiline])]:ms-2` |
| in(:where(*:is([data-layout=adaptive]:not([data-multiline])))) | grid-column-start | `3` | `in-[[data-layout=adaptive]:not([data-multiline])]:col-start-3` |
| in(:where(*:is([data-layout=adaptive]:not([data-multiline])))) | grid-row-start | `1` | `in-[[data-layout=adaptive]:not([data-multiline])]:row-start-1` |

### `ComposerSubmit`

No own classes (re-export / wrapper of the primitive).

