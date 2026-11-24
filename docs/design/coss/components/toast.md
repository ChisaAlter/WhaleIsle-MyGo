# Toast

Source: `apps/ui/registry/default/ui/toast.tsx` @ `b2336c45f2b7` · docs: `apps/ui/content/docs/components/toast.mdx` · Base UI: https://base-ui.com/react/components/toast#api-reference

> A temporary notification that appears on screen to inform users.

Depends on: `@base-ui/react/toast`, `lucide-react`, `@/ui/button`

Exports: `getSwipeDirection`, `upsertReplayClassName`, `Toasts`, `AnchoredToasts`, `ToastProvider`, `AnchoredToastProvider`

## Behavior & keyboard

- Toast region (`F6` focuses the viewport in Base UI), toasts stack and expand on hover/focus; swipe to dismiss (`data-swipe-direction`), close button; `Escape` dismisses focused toast. Stack uses CSS vars `--toast-index`, `--toast-offset-y`, `--toast-height`, `--toast-frontmost-height`, `--toast-swipe-movement-x/y`. _(inferred: Base UI primitive behavior; not restated in coss docs)_

## How to read

Each part lists the exact class strings, then **resolved** values for the desktop environment (≥640px so all `sm:` rules apply; also md/lg/xl; fine pointer; hover-capable). Values show `light` / `dark` when they differ. *State* tables only list properties that change versus the part's default. *Mobile* lists properties that differ at 375px / coarse pointer. `src` is the class that wins the cascade. See `../README.md` for conventions.

## Parts

### `getSwipeDirection`

No own classes (re-export / wrapper of the primitive).

### `upsertReplayClassName`

No own classes (re-export / wrapper of the primitive).

### `Toasts` → `[data-slot=toast-portal]` · `<Toast.Portal>` (line 77)

### `Toasts` → `[data-slot=toast-viewport]` · `<Toast.Viewport>` (line 78)

Attributes: `data-position={position}`

#### base

```txt
fixed z-60 mx-auto flex w-[calc(100%-var(--toast-inset)*2)] max-w-90 [--toast-inset:--spacing(4)] sm:[--toast-inset:--spacing(8)] data-[position*=top]:top-(--toast-inset) data-[position*=bottom]:bottom-(--toast-inset) data-[position*=left]:left-(--toast-inset) data-[position*=right]:right-(--toast-inset) data-[position*=center]:left-1/2 data-[position*=center]:-translate-x-1/2
```

**element — default**

Key metrics: display `flex` · width `calc(100% - 32px * 2)` · max-width `360px` · z-index `60`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| position | `fixed` | `fixed` | `fixed` |
| z-index | `60` | `60` | `z-60` |
| margin-inline | `auto` | `auto` | `mx-auto` |
| display | `flex` | `flex` | `flex` |
| width | `calc(100% - 32px * 2)` | `calc(100% - var(--toast-inset) * 2)` | `w-[calc(100%-var(--toast-inset)*2)]` |
| max-width | `360px` | `calc(var(--spacing) * 90)` | `max-w-90` |

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| [data-position*=top] | top | `32px` | `data-[position*=top]:top-(--toast-inset)` |
| [data-position*=bottom] | bottom | `32px` | `data-[position*=bottom]:bottom-(--toast-inset)` |
| [data-position*=left] | left | `32px` | `data-[position*=left]:left-(--toast-inset)` |
| [data-position*=right] | right | `32px` | `data-[position*=right]:right-(--toast-inset)` |
| [data-position*=center] | left | `50%` | `data-[position*=center]:left-1/2` |
| [data-position*=center] | translate | `-50% 0` | `data-[position*=center]:-translate-x-1/2` |

**element — mobile differences**: width `calc(100% - 16px * 2)`

### `Toasts` → (no data-slot) · `<Toast.Root>` (line 99)

Attributes: `data-position={position}`, `swipeDirection={swipeDirection}`, `toast={toast}`

Dynamic className inputs (not statically resolvable): `upsertReplayClassName(toast)`

#### base

```txt
absolute z-[calc(9999-var(--toast-index))] h-(--toast-calc-height) w-full select-none rounded-lg border bg-[color-mix(in_srgb,var(--popover),var(--color-black)_calc(1%*max(0,var(--toast-index,0))))] not-dark:bg-clip-padding text-popover-foreground shadow-lg/5 [transition:transform_.5s_cubic-bezier(.22,1,.36,1),opacity_.5s,height_.15s,background-color_.5s] before:pointer-events-none before:absolute before:inset-0 before:rounded-[calc(var(--radius-lg)-1px)] before:shadow-[0_1px_--theme(--color-black/4%)] data-expanded:bg-popover dark:bg-[color-mix(in_srgb,var(--popover),var(--color-black)_calc(6%*max(0,var(--toast-index,0))))] dark:data-expanded:bg-popover dark:before:shadow-[0_-1px_--theme(--color-white/6%)] data-[position*=right]:right-0 data-[position*=right]:left-auto data-[position*=left]:right-auto data-[position*=left]:left-0 data-[position*=center]:right-0 data-[position*=center]:left-0 data-[position*=top]:top-0 data-[position*=top]:bottom-auto data-[position*=top]:origin-[50%_calc(50%-50%*min(var(--toast-index,0),1))] data-[position*=bottom]:top-auto data-[position*=bottom]:bottom-0 data-[position*=bottom]:origin-[50%_calc(50%+50%*min(var(--toast-index,0),1))] after:absolute after:left-0 after:h-[calc(var(--toast-gap)+1px)] after:w-full data-[position*=top]:after:top-full data-[position*=bottom]:after:bottom-full [--toast-calc-height:var(--toast-frontmost-height,var(--toast-height))] [--toast-gap:--spacing(3)] [--toast-peek:--spacing(3)] [--toast-scale:calc(max(0,1-(var(--toast-index)*.1)))] [--toast-shrink:calc(1-var(--toast-scale))] data-[position*=top]:[--toast-calc-offset-y:calc(var(--toast-offset-y)+var(--toast-index)*var(--toast-gap)+var(--toast-swipe-movement-y))] data-[position*=bottom]:[--toast-calc-offset-y:calc(var(--toast-offset-y)*-1+var(--toast-index)*var(--toast-gap)*-1+var(--toast-swipe-movement-y))] data-[position*=top]:transform-[translateX(var(--toast-swipe-movement-x))_translateY(calc(var(--toast-swipe-movement-y)+(var(--toast-index)*var(--toast-peek))+(var(--toast-shrink)*var(--toast-calc-height))))_scale(var(--toast-scale))] data-[position*=bottom]:transform-[translateX(var(--toast-swipe-movement-x))_translateY(calc(var(--toast-swipe-movement-y)-(var(--toast-index)*var(--toast-peek))-(var(--toast-shrink)*var(--toast-calc-height))))_scale(var(--toast-scale))] data-limited:opacity-0 data-expanded:h-(--toast-height) data-position:data-expanded:transform-[translateX(var(--toast-swipe-movement-x))_translateY(var(--toast-calc-offset-y))] motion-safe:data-[position*=top]:data-starting-style:transform-[translateY(calc(-100%-var(--toast-inset)))] motion-safe:data-[position*=bottom]:data-starting-style:transform-[translateY(calc(100%+var(--toast-inset)))] data-ending-style:opacity-0 motion-reduce:data-starting-style:opacity-0 motion-reduce:[transition:opacity_.5s,height_.15s,background-color_.5s] motion-safe:data-[position*=top]:data-ending-style:not-data-limited:not-data-swipe-direction:transform-[translateY(calc(-100%-var(--toast-inset)))] motion-safe:data-[position*=bottom]:data-ending-style:not-data-limited:not-data-swipe-direction:transform-[translateY(calc(100%+var(--toast-inset)))] data-ending-style:data-[swipe-direction=left]:transform-[translateX(calc(var(--toast-swipe-movement-x)-100%-var(--toast-inset)))_translateY(var(--toast-calc-offset-y))] data-ending-style:data-[swipe-direction=right]:transform-[translateX(calc(var(--toast-swipe-movement-x)+100%+var(--toast-inset)))_translateY(var(--toast-calc-offset-y))] data-ending-style:data-[swipe-direction=up]:transform-[translateY(calc(var(--toast-swipe-movement-y)-100%-var(--toast-inset)))] data-ending-style:data-[swipe-direction=down]:transform-[translateY(calc(var(--toast-swipe-movement-y)+100%+var(--toast-inset)))] data-expanded:data-ending-style:data-[swipe-direction=left]:transform-[translateX(calc(var(--toast-swipe-movement-x)-100%-var(--toast-inset)))_translateY(var(--toast-calc-offset-y))] data-expanded:data-ending-style:data-[swipe-direction=right]:transform-[translateX(calc(var(--toast-swipe-movement-x)+100%+var(--toast-inset)))_translateY(var(--toast-calc-offset-y))] data-expanded:data-ending-style:data-[swipe-direction=up]:transform-[translateY(calc(var(--toast-swipe-movement-y)-100%-var(--toast-inset)))] data-expanded:data-ending-style:data-[swipe-direction=down]:transform-[translateY(calc(var(--toast-swipe-movement-y)+100%+var(--toast-inset)))]
```

**element — default**

Key metrics: height `var(--toast-height)` (runtime var --toast-calc-height) · width `100%` · border-width `1px` · border-color `#00000014` / `#ffffff0f` (border) · border-radius `10px` (radius) · background-color `color-mix(in srgb,#fff,#000 calc(1% * max(0, 0)))` / `color-mix(in srgb,#1d1d1d,#000 calc(6% * max(0, 0)))` · color `#262626` / `#f5f5f5` (popover-foreground) · box-shadow `0 10px 15px -3px #0000000d, 0 4px 6px -4px #0000000d` · z-index `calc(9999 - var(--toast-index))`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| border-width | `1px` | `1px` | `border` |
| border-style | `solid` | `var(--tw-border-style)` | `border` |
| border-color | `#00000014` / `#ffffff0f` | `border` | `(globals @layer base) * border-border` |
| position | `absolute` | `absolute` | `absolute` |
| z-index | `calc(9999 - var(--toast-index))` | `calc(9999 - var(--toast-index))` | `z-[calc(9999-var(--toast-index))]` |
| height | `var(--toast-height)` | `runtime var --toast-calc-height` | `h-(--toast-calc-height)` |
| width | `100%` | `100%` | `w-full` |
| border-radius | `10px` | `radius` | `rounded-lg` |
| background-color | `color-mix(in srgb,#fff,#000 calc(1% * max(0, 0)))` / `color-mix(in srgb,#1d1d1d,#000 calc(6% * max(0, 0)))` | `color-mix(in srgb,var(--popover),var(--color-black) calc(1% * max(0, var(--toast-index,0))))` | `bg-[color-mix(in_srgb,var(--popover),var(--color-black)_calc(1%*max(0,var(--toast-index,0))))]` |
| color | `#262626` / `#f5f5f5` | `popover-foreground` | `text-popover-foreground` |
| box-shadow | `0 10px 15px -3px #0000000d, 0 4px 6px -4px #0000000d` | `var(--tw-inset-shadow), var(--tw-inset-ring-shadow), var(--tw-ring-offset-shadow), var(--tw-ring-shadow), var(` | `shadow-lg/5` |
| -webkit-user-select | `none` | `none` | `select-none` |
| user-select | `none` | `none` | `select-none` |
| transition | `transform 500ms cubic-bezier(.22,1,.36,1), opacity 500ms, height 150ms, background-color 500ms` | `transform .5s cubic-bezier(.22,1,.36,1),opacity .5s,height .15s,background-color .5s` | `[transition:transform_.5s_cubic-bezier(.22,1,.36,1),opacity_.5s,height_.15s,background-color_.5s]` |
| background-clip | `padding-box` / — | `padding-box` | `not-dark:bg-clip-padding` |

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| open | background-color | `#ffffff` / `#1d1d1d` | `data-expanded:bg-popover` |
| [data-position*=right] | right | `0px` | `data-[position*=right]:right-0` |
| [data-position*=right] | left | `auto` | `data-[position*=right]:left-auto` |
| [data-position*=left] | right | `auto` | `data-[position*=left]:right-auto` |
| [data-position*=left] | left | `0px` | `data-[position*=left]:left-0` |
| [data-position*=center] | right | `0px` | `data-[position*=center]:right-0` |
| [data-position*=center] | left | `0px` | `data-[position*=center]:left-0` |
| [data-position*=top] | top | `0px` | `data-[position*=top]:top-0` |
| [data-position*=top] | bottom | `auto` | `data-[position*=top]:bottom-auto` |
| [data-position*=top] | transform-origin | `50% calc(50% - 50% * min(0, 1))` | `data-[position*=top]:origin-[50%_calc(50%-50%*min(var(--toast-index,0),1))]` |
| [data-position*=top] | transform | `translateX(var(--toast-swipe-movement-x)) translateY(calc(var(--toast-swipe-movement-y) + (var(--toast-index) * 12px) + (calc(1 - calc(max(0, 1 - (var(--toast-index) * .1)))) * var(--toast-height)))) scale(calc(max(0, 1 - (var(--toast-index) * .1))))` | `data-[position*=top]:transform-[translateX(var(--toast-swipe-movement-x))_translateY(calc(var(--toast-swipe-movement-y)+(var(--toast-index)*var(--toast-peek))+(var(--toast-shrink)*var(--toast-calc-height))))_scale(var(--toast-scale))]` |
| [data-position*=bottom] | top | `auto` | `data-[position*=bottom]:top-auto` |
| [data-position*=bottom] | bottom | `0px` | `data-[position*=bottom]:bottom-0` |
| [data-position*=bottom] | transform-origin | `50% calc(50% + 50% * min(0, 1))` | `data-[position*=bottom]:origin-[50%_calc(50%+50%*min(var(--toast-index,0),1))]` |
| [data-position*=bottom] | transform | `translateX(var(--toast-swipe-movement-x)) translateY(calc(var(--toast-swipe-movement-y) - (var(--toast-index) * 12px) - (calc(1 - calc(max(0, 1 - (var(--toast-index) * .1)))) * var(--toast-height)))) scale(calc(max(0, 1 - (var(--toast-index) * .1))))` | `data-[position*=bottom]:transform-[translateX(var(--toast-swipe-movement-x))_translateY(calc(var(--toast-swipe-movement-y)-(var(--toast-index)*var(--toast-peek))-(var(--toast-shrink)*var(--toast-calc-height))))_scale(var(--toast-scale))]` |
| [data-limited] | opacity | `0%` | `data-limited:opacity-0` |
| ending-style (exit to) | opacity | `0%` | `data-ending-style:opacity-0` |
| [data-position] + open | background-color | `#ffffff` / `#1d1d1d` | `data-expanded:bg-popover` |
| [data-position] + open | transform | `translateX(var(--toast-swipe-movement-x)) translateY(var(--toast-calc-offset-y))` | `data-position:data-expanded:transform-[translateX(var(--toast-swipe-movement-x))_translateY(var(--toast-calc-offset-y))]` |
| [data-position*=top] + starting-style (enter from) | top | `0px` | `data-[position*=top]:top-0` |
| [data-position*=top] + starting-style (enter from) | bottom | `auto` | `data-[position*=top]:bottom-auto` |
| [data-position*=top] + starting-style (enter from) | transform-origin | `50% calc(50% - 50% * min(0, 1))` | `data-[position*=top]:origin-[50%_calc(50%-50%*min(var(--toast-index,0),1))]` |
| [data-position*=top] + starting-style (enter from) | transform | `translateY(calc(-100% - 32px))` | `motion-safe:data-[position*=top]:data-starting-style:transform-[translateY(calc(-100%-var(--toast-inset)))]` |
| [data-position*=bottom] + starting-style (enter from) | top | `auto` | `data-[position*=bottom]:top-auto` |
| [data-position*=bottom] + starting-style (enter from) | bottom | `0px` | `data-[position*=bottom]:bottom-0` |
| [data-position*=bottom] + starting-style (enter from) | transform-origin | `50% calc(50% + 50% * min(0, 1))` | `data-[position*=bottom]:origin-[50%_calc(50%+50%*min(var(--toast-index,0),1))]` |
| [data-position*=bottom] + starting-style (enter from) | transform | `translateY(calc(100% + 32px))` | `motion-safe:data-[position*=bottom]:data-starting-style:transform-[translateY(calc(100%+var(--toast-inset)))]` |
| [data-position*=top] + ending-style (exit to) | opacity | `0%` | `data-ending-style:opacity-0` |
| [data-position*=top] + ending-style (exit to) | top | `0px` | `data-[position*=top]:top-0` |
| [data-position*=top] + ending-style (exit to) | bottom | `auto` | `data-[position*=top]:bottom-auto` |
| [data-position*=top] + ending-style (exit to) | transform-origin | `50% calc(50% - 50% * min(0, 1))` | `data-[position*=top]:origin-[50%_calc(50%-50%*min(var(--toast-index,0),1))]` |
| [data-position*=top] + ending-style (exit to) | transform | `translateY(calc(-100% - 32px))` | `motion-safe:data-[position*=top]:data-ending-style:not-data-limited:not-data-swipe-direction:transform-[translateY(calc(-100%-var(--toast-inset)))]` |
| [data-position*=bottom] + ending-style (exit to) | opacity | `0%` | `data-ending-style:opacity-0` |
| [data-position*=bottom] + ending-style (exit to) | top | `auto` | `data-[position*=bottom]:top-auto` |
| [data-position*=bottom] + ending-style (exit to) | bottom | `0px` | `data-[position*=bottom]:bottom-0` |
| [data-position*=bottom] + ending-style (exit to) | transform-origin | `50% calc(50% + 50% * min(0, 1))` | `data-[position*=bottom]:origin-[50%_calc(50%+50%*min(var(--toast-index,0),1))]` |
| [data-position*=bottom] + ending-style (exit to) | transform | `translateY(calc(100% + 32px))` | `motion-safe:data-[position*=bottom]:data-ending-style:not-data-limited:not-data-swipe-direction:transform-[translateY(calc(100%+var(--toast-inset)))]` |
| [data-swipe-direction=left] + ending-style (exit to) | opacity | `0%` | `data-ending-style:opacity-0` |
| [data-swipe-direction=left] + ending-style (exit to) | transform | `translateX(calc(var(--toast-swipe-movement-x) - 100% - 32px)) translateY(var(--toast-calc-offset-y))` | `data-ending-style:data-[swipe-direction=left]:transform-[translateX(calc(var(--toast-swipe-movement-x)-100%-var(--toast-inset)))_translateY(var(--toast-calc-offset-y))]` |
| [data-swipe-direction=right] + ending-style (exit to) | opacity | `0%` | `data-ending-style:opacity-0` |
| [data-swipe-direction=right] + ending-style (exit to) | transform | `translateX(calc(var(--toast-swipe-movement-x) + 100% + 32px)) translateY(var(--toast-calc-offset-y))` | `data-ending-style:data-[swipe-direction=right]:transform-[translateX(calc(var(--toast-swipe-movement-x)+100%+var(--toast-inset)))_translateY(var(--toast-calc-offset-y))]` |
| [data-swipe-direction=up] + ending-style (exit to) | opacity | `0%` | `data-ending-style:opacity-0` |
| [data-swipe-direction=up] + ending-style (exit to) | transform | `translateY(calc(var(--toast-swipe-movement-y) - 100% - 32px))` | `data-ending-style:data-[swipe-direction=up]:transform-[translateY(calc(var(--toast-swipe-movement-y)-100%-var(--toast-inset)))]` |
| [data-swipe-direction=down] + ending-style (exit to) | opacity | `0%` | `data-ending-style:opacity-0` |
| [data-swipe-direction=down] + ending-style (exit to) | transform | `translateY(calc(var(--toast-swipe-movement-y) + 100% + 32px))` | `data-ending-style:data-[swipe-direction=down]:transform-[translateY(calc(var(--toast-swipe-movement-y)+100%+var(--toast-inset)))]` |
| [data-swipe-direction=left] + ending-style (exit to) + open | background-color | `#ffffff` / `#1d1d1d` | `data-expanded:bg-popover` |
| [data-swipe-direction=left] + ending-style (exit to) + open | opacity | `0%` | `data-ending-style:opacity-0` |
| [data-swipe-direction=left] + ending-style (exit to) + open | transform | `translateX(calc(var(--toast-swipe-movement-x) - 100% - 32px)) translateY(var(--toast-calc-offset-y))` | `data-expanded:data-ending-style:data-[swipe-direction=left]:transform-[translateX(calc(var(--toast-swipe-movement-x)-100%-var(--toast-inset)))_translateY(var(--toast-calc-offset-y))]` |
| [data-swipe-direction=right] + ending-style (exit to) + open | background-color | `#ffffff` / `#1d1d1d` | `data-expanded:bg-popover` |
| [data-swipe-direction=right] + ending-style (exit to) + open | opacity | `0%` | `data-ending-style:opacity-0` |
| [data-swipe-direction=right] + ending-style (exit to) + open | transform | `translateX(calc(var(--toast-swipe-movement-x) + 100% + 32px)) translateY(var(--toast-calc-offset-y))` | `data-expanded:data-ending-style:data-[swipe-direction=right]:transform-[translateX(calc(var(--toast-swipe-movement-x)+100%+var(--toast-inset)))_translateY(var(--toast-calc-offset-y))]` |
| [data-swipe-direction=up] + ending-style (exit to) + open | background-color | `#ffffff` / `#1d1d1d` | `data-expanded:bg-popover` |
| [data-swipe-direction=up] + ending-style (exit to) + open | opacity | `0%` | `data-ending-style:opacity-0` |
| [data-swipe-direction=up] + ending-style (exit to) + open | transform | `translateY(calc(var(--toast-swipe-movement-y) - 100% - 32px))` | `data-expanded:data-ending-style:data-[swipe-direction=up]:transform-[translateY(calc(var(--toast-swipe-movement-y)-100%-var(--toast-inset)))]` |
| [data-swipe-direction=down] + ending-style (exit to) + open | background-color | `#ffffff` / `#1d1d1d` | `data-expanded:bg-popover` |
| [data-swipe-direction=down] + ending-style (exit to) + open | opacity | `0%` | `data-ending-style:opacity-0` |
| [data-swipe-direction=down] + ending-style (exit to) + open | transform | `translateY(calc(var(--toast-swipe-movement-y) + 100% + 32px))` | `data-expanded:data-ending-style:data-[swipe-direction=down]:transform-[translateY(calc(var(--toast-swipe-movement-y)+100%+var(--toast-inset)))]` |

**`::after` — default**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| content | `""` | `var(--tw-content)` | `after:w-full` |
| position | `absolute` | `absolute` | `after:absolute` |
| left | `0px` | `calc(var(--spacing) * 0)` | `after:left-0` |
| height | `13px` | `calc(var(--toast-gap) + 1px)` | `after:h-[calc(var(--toast-gap)+1px)]` |
| width | `100%` | `100%` | `after:w-full` |

**`::after` — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| [data-position*=top] | top | `100%` | `data-[position*=top]:after:top-full` |
| [data-position*=bottom] | bottom | `100%` | `data-[position*=bottom]:after:bottom-full` |
| [data-position*=top] + starting-style (enter from) | top | `100%` | `data-[position*=top]:after:top-full` |
| [data-position*=bottom] + starting-style (enter from) | bottom | `100%` | `data-[position*=bottom]:after:bottom-full` |
| [data-position*=top] + ending-style (exit to) | top | `100%` | `data-[position*=top]:after:top-full` |
| [data-position*=bottom] + ending-style (exit to) | bottom | `100%` | `data-[position*=bottom]:after:bottom-full` |

**`::before` — default**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| content | `""` | `var(--tw-content)` | `before:shadow-[0_1px_--theme(--color-black/4%)]` |
| pointer-events | `none` | `none` | `before:pointer-events-none` |
| position | `absolute` | `absolute` | `before:absolute` |
| inset | `0px` | `calc(var(--spacing) * 0)` | `before:inset-0` |
| border-radius | `9px` | `calc(var(--radius-lg) - 1px)` | `before:rounded-[calc(var(--radius-lg)-1px)]` |
| box-shadow | `0 1px #0000000a` / `0 -1px #ffffff0f` | `var(--tw-inset-shadow), var(--tw-inset-ring-shadow), var(--tw-ring-offset-shadow), var(--tw-ring-shadow), var(` | `before:shadow-[0_1px_--theme(--color-black/4%)]` |

### `Toasts` → (no data-slot) · `<Toast.Content>` (line 151)

#### base

```txt
pointer-events-auto flex items-center justify-between gap-1.5 overflow-hidden px-3.5 py-3 text-sm transition-opacity duration-250 data-behind:not-data-expanded:pointer-events-none data-behind:opacity-0 data-expanded:opacity-100
```

**element — default**

Key metrics: display `flex` · padding-inline `14px` · padding-block `12px` · gap `6px` · font-size `14px` (text-sm) · line-height `20px` · transition-property `opacity` · transition-duration `250ms` · transition-timing-function `cubic-bezier(0.4, 0, 0.2, 1)`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| pointer-events | `auto` | `auto` | `pointer-events-auto` |
| display | `flex` | `flex` | `flex` |
| align-items | `center` | `center` | `items-center` |
| justify-content | `space-between` | `space-between` | `justify-between` |
| gap | `6px` | `calc(var(--spacing) * 1.5)` | `gap-1.5` |
| overflow | `hidden` | `hidden` | `overflow-hidden` |
| padding-inline | `14px` | `calc(var(--spacing) * 3.5)` | `px-3.5` |
| padding-block | `12px` | `calc(var(--spacing) * 3)` | `py-3` |
| font-size | `14px` | `text-sm` | `text-sm` |
| line-height | `20px` | `var(--tw-leading, var(--text-sm--line-height))` | `text-sm` |
| transition-property | `opacity` | `opacity` | `transition-opacity` |
| transition-timing-function | `cubic-bezier(0.4, 0, 0.2, 1)` | `var(--tw-ease, var(--default-transition-timing-function))` | `transition-opacity` |
| transition-duration | `250ms` | `250ms` | `duration-250` |

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| [data-behind] | pointer-events | `none` | `data-behind:not-data-expanded:pointer-events-none` |
| [data-behind] | opacity | `0%` | `data-behind:opacity-0` |
| open | opacity | `100%` | `data-expanded:opacity-100` |

### `Toasts` → (no data-slot) · `<div>` (line 152)

#### base

```txt
flex gap-2
```

**element — default**

Key metrics: display `flex` · gap `8px`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| display | `flex` | `flex` | `flex` |
| gap | `8px` | `calc(var(--spacing) * 2)` | `gap-2` |

### `Toasts` → `[data-slot=toast-icon]` · `<div>` (line 154)

#### base

```txt
[&>svg]:h-lh [&>svg]:w-4 [&_svg]:pointer-events-none [&_svg]:shrink-0
```

**`>svg` — default**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| height | `1lh` | `1lh` | `[&>svg]:h-lh` |
| width | `16px` | `calc(var(--spacing) * 4)` | `[&>svg]:w-4` |

**`svg` — default**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| pointer-events | `none` | `none` | `[&_svg]:pointer-events-none` |
| flex-shrink | `0` | `0` | `[&_svg]:shrink-0` |

### `Toasts` → (no data-slot) · `<Icon>` (line 158)

#### base

```txt
in-data-[type=loading]:animate-spin in-data-[type=error]:text-destructive in-data-[type=info]:text-info in-data-[type=success]:text-success in-data-[type=warning]:text-warning in-data-[type=loading]:opacity-80
```

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| in([data-type=loading]) | animation | `spin 1000ms linear infinite` | `in-data-[type=loading]:animate-spin` |
| in([data-type=loading]) | opacity | `80%` | `in-data-[type=loading]:opacity-80` |
| in([data-type=error]) | color | `#fb2c36` / `#fb414a` | `in-data-[type=error]:text-destructive` |
| in([data-type=info]) | color | `#2b7fff` | `in-data-[type=info]:text-info` |
| in([data-type=success]) | color | `#00bc7d` | `in-data-[type=success]:text-success` |
| in([data-type=warning]) | color | `#fe9a00` | `in-data-[type=warning]:text-warning` |

### `Toasts` → (no data-slot) · `<div>` (line 162)

#### base

```txt
flex flex-col gap-0.5
```

**element — default**

Key metrics: display `flex` · gap `2px`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| display | `flex` | `flex` | `flex` |
| flex-direction | `column` | `column` | `flex-col` |
| gap | `2px` | `calc(var(--spacing) * 0.5)` | `gap-0.5` |

### `Toasts` → `[data-slot=toast-title]` · `<Toast.Title>` (line 163)

#### base

```txt
font-medium
```

**element — default**

Key metrics: font-weight `500` (font-weight-medium)

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| font-weight | `500` | `font-weight-medium` | `font-medium` |

### `Toasts` → `[data-slot=toast-description]` · `<Toast.Description>` (line 167)

#### base

```txt
text-muted-foreground
```

**element — default**

Key metrics: color `#686868` / `#818181` (muted-foreground)

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| color | `#686868` / `#818181` | `muted-foreground` | `text-muted-foreground` |

### `Toasts` → `[data-slot=toast-action]` · `<Toast.Action>` (line 174)

Dynamic className inputs (not statically resolvable): `buttonVariants({ size: "xs" })`

### `AnchoredToasts` → `[data-slot=toast-portal-anchored]` · `<Toast.Portal>` (line 198)

### `AnchoredToasts` → `[data-slot=toast-viewport-anchored]` · `<Toast.Viewport>` (line 199)

#### base

```txt
outline-none
```

**element — default**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| outline-style | `none` | `none` | `outline-none` |

### `AnchoredToasts` → `[data-slot=toast-positioner]` · `<Toast.Positioner>` (line 216)

Attributes: `sideOffset={positionerProps.sideOffset ?? 4}`, `toast={toast}`

#### base

```txt
z-60 max-w-[min(--spacing(64),var(--available-width))]
```

**element — default**

Key metrics: max-width `min(256px, var(--available-width))` · z-index `60`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| z-index | `60` | `60` | `z-60` |
| max-width | `min(256px, var(--available-width))` | `min(calc(var(--spacing) * 64), var(--available-width))` | `max-w-[min(--spacing(64),var(--available-width))]` |

### `AnchoredToasts` → `[data-slot=toast-popup]` · `<Toast.Root>` (line 223)

Attributes: `toast={toast}`

Dynamic className inputs (not statically resolvable): `upsertReplayClassName(toast)`

#### base

```txt
relative text-balance border bg-popover not-dark:bg-clip-padding text-popover-foreground text-xs transition-[scale,opacity] before:pointer-events-none before:absolute before:inset-0 before:shadow-[0_1px_--theme(--color-black/4%)] data-ending-style:opacity-0 data-starting-style:opacity-0 motion-safe:data-ending-style:scale-98 motion-safe:data-starting-style:scale-98 dark:before:shadow-[0_-1px_--theme(--color-white/6%)]
```

**element — default**

Key metrics: border-width `1px` · border-color `#00000014` / `#ffffff0f` (border) · background-color `#ffffff` / `#1d1d1d` (popover) · color `#262626` / `#f5f5f5` (popover-foreground) · font-size `12px` (text-xs) · line-height `16px` · transition-property `scale, opacity` · transition-duration `150ms` · transition-timing-function `cubic-bezier(0.4, 0, 0.2, 1)`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| border-width | `1px` | `1px` | `border` |
| border-style | `solid` | `var(--tw-border-style)` | `border` |
| border-color | `#00000014` / `#ffffff0f` | `border` | `(globals @layer base) * border-border` |
| position | `relative` | `relative` | `relative` |
| background-color | `#ffffff` / `#1d1d1d` | `popover` | `bg-popover` |
| font-size | `12px` | `text-xs` | `text-xs` |
| line-height | `16px` | `var(--tw-leading, var(--text-xs--line-height))` | `text-xs` |
| text-wrap | `balance` | `balance` | `text-balance` |
| color | `#262626` / `#f5f5f5` | `popover-foreground` | `text-popover-foreground` |
| transition-property | `scale, opacity` | `scale,opacity` | `transition-[scale,opacity]` |
| transition-timing-function | `cubic-bezier(0.4, 0, 0.2, 1)` | `var(--tw-ease, var(--default-transition-timing-function))` | `transition-[scale,opacity]` |
| transition-duration | `150ms` | `var(--tw-duration, var(--default-transition-duration))` | `transition-[scale,opacity]` |
| background-clip | `padding-box` / — | `padding-box` | `not-dark:bg-clip-padding` |

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| ending-style (exit to) | opacity | `0%` | `data-ending-style:opacity-0` |
| ending-style (exit to) | scale | `98% 98%` | `motion-safe:data-ending-style:scale-98` |
| starting-style (enter from) | opacity | `0%` | `data-starting-style:opacity-0` |
| starting-style (enter from) | scale | `98% 98%` | `motion-safe:data-starting-style:scale-98` |

**`::before` — default**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| content | `""` | `var(--tw-content)` | `before:shadow-[0_1px_--theme(--color-black/4%)]` |
| pointer-events | `none` | `none` | `before:pointer-events-none` |
| position | `absolute` | `absolute` | `before:absolute` |
| inset | `0px` | `calc(var(--spacing) * 0)` | `before:inset-0` |
| box-shadow | `0 1px #0000000a` / `0 -1px #ffffff0f` | `var(--tw-inset-shadow), var(--tw-inset-ring-shadow), var(--tw-ring-offset-shadow), var(--tw-ring-shadow), var(` | `before:shadow-[0_1px_--theme(--color-black/4%)]` |

#### when `tooltipStyle`

```txt
rounded-md shadow-md/5 before:rounded-[calc(var(--radius-md)-1px)]
```

**element — default (changes vs base)**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| border-radius | `8px` | `calc(var(--radius) - 2px)` | `rounded-md` |
| box-shadow | `0 4px 6px -1px #0000000d, 0 2px 4px -2px #0000000d` | `var(--tw-inset-shadow), var(--tw-inset-ring-shadow), var(--tw-ring-offset-shadow), var(--tw-ring-shadow), var(` | `shadow-md/5` |

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| ending-style (exit to) | opacity | `0%` | `data-ending-style:opacity-0` |
| ending-style (exit to) | scale | `98% 98%` | `motion-safe:data-ending-style:scale-98` |
| starting-style (enter from) | opacity | `0%` | `data-starting-style:opacity-0` |
| starting-style (enter from) | scale | `98% 98%` | `motion-safe:data-starting-style:scale-98` |

**`::before` — default (changes vs base)**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| border-radius | `7px` | `calc(var(--radius-md) - 1px)` | `before:rounded-[calc(var(--radius-md)-1px)]` |

#### when `!(tooltipStyle)`

```txt
rounded-lg shadow-lg/5 before:rounded-[calc(var(--radius-lg)-1px)]
```

**element — default (changes vs base)**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| border-radius | `10px` | `radius` | `rounded-lg` |
| box-shadow | `0 10px 15px -3px #0000000d, 0 4px 6px -4px #0000000d` | `var(--tw-inset-shadow), var(--tw-inset-ring-shadow), var(--tw-ring-offset-shadow), var(--tw-ring-shadow), var(` | `shadow-lg/5` |

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| ending-style (exit to) | opacity | `0%` | `data-ending-style:opacity-0` |
| ending-style (exit to) | scale | `98% 98%` | `motion-safe:data-ending-style:scale-98` |
| starting-style (enter from) | opacity | `0%` | `data-starting-style:opacity-0` |
| starting-style (enter from) | scale | `98% 98%` | `motion-safe:data-starting-style:scale-98` |

**`::before` — default (changes vs base)**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| border-radius | `9px` | `calc(var(--radius-lg) - 1px)` | `before:rounded-[calc(var(--radius-lg)-1px)]` |

### `AnchoredToasts` → (no data-slot) · `<Toast.Content>` (line 236)

#### base

```txt
pointer-events-auto px-2 py-1
```

**element — default**

Key metrics: padding-inline `8px` · padding-block `4px`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| pointer-events | `auto` | `auto` | `pointer-events-auto` |
| padding-inline | `8px` | `calc(var(--spacing) * 2)` | `px-2` |
| padding-block | `4px` | `calc(var(--spacing) * 1)` | `py-1` |

### `AnchoredToasts` → `[data-slot=toast-title]` · `<Toast.Title>` (line 237)

### `AnchoredToasts` → (no data-slot) · `<Toast.Content>` (line 240)

#### base

```txt
pointer-events-auto flex items-center justify-between gap-1.5 overflow-hidden px-3.5 py-3 text-sm
```

**element — default**

Key metrics: display `flex` · padding-inline `14px` · padding-block `12px` · gap `6px` · font-size `14px` (text-sm) · line-height `20px`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| pointer-events | `auto` | `auto` | `pointer-events-auto` |
| display | `flex` | `flex` | `flex` |
| align-items | `center` | `center` | `items-center` |
| justify-content | `space-between` | `space-between` | `justify-between` |
| gap | `6px` | `calc(var(--spacing) * 1.5)` | `gap-1.5` |
| overflow | `hidden` | `hidden` | `overflow-hidden` |
| padding-inline | `14px` | `calc(var(--spacing) * 3.5)` | `px-3.5` |
| padding-block | `12px` | `calc(var(--spacing) * 3)` | `py-3` |
| font-size | `14px` | `text-sm` | `text-sm` |
| line-height | `20px` | `var(--tw-leading, var(--text-sm--line-height))` | `text-sm` |

### `AnchoredToasts` → (no data-slot) · `<div>` (line 241)

#### base

```txt
flex gap-2
```

**element — default**

Key metrics: display `flex` · gap `8px`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| display | `flex` | `flex` | `flex` |
| gap | `8px` | `calc(var(--spacing) * 2)` | `gap-2` |

### `AnchoredToasts` → `[data-slot=toast-icon]` · `<div>` (line 243)

#### base

```txt
[&>svg]:h-lh [&>svg]:w-4 [&_svg]:pointer-events-none [&_svg]:shrink-0
```

**`>svg` — default**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| height | `1lh` | `1lh` | `[&>svg]:h-lh` |
| width | `16px` | `calc(var(--spacing) * 4)` | `[&>svg]:w-4` |

**`svg` — default**

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| pointer-events | `none` | `none` | `[&_svg]:pointer-events-none` |
| flex-shrink | `0` | `0` | `[&_svg]:shrink-0` |

### `AnchoredToasts` → (no data-slot) · `<Icon>` (line 247)

#### base

```txt
in-data-[type=loading]:animate-spin in-data-[type=error]:text-destructive in-data-[type=info]:text-info in-data-[type=success]:text-success in-data-[type=warning]:text-warning in-data-[type=loading]:opacity-80
```

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| in([data-type=loading]) | animation | `spin 1000ms linear infinite` | `in-data-[type=loading]:animate-spin` |
| in([data-type=loading]) | opacity | `80%` | `in-data-[type=loading]:opacity-80` |
| in([data-type=error]) | color | `#fb2c36` / `#fb414a` | `in-data-[type=error]:text-destructive` |
| in([data-type=info]) | color | `#2b7fff` | `in-data-[type=info]:text-info` |
| in([data-type=success]) | color | `#00bc7d` | `in-data-[type=success]:text-success` |
| in([data-type=warning]) | color | `#fe9a00` | `in-data-[type=warning]:text-warning` |

### `AnchoredToasts` → (no data-slot) · `<div>` (line 251)

#### base

```txt
flex flex-col gap-0.5
```

**element — default**

Key metrics: display `flex` · gap `2px`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| display | `flex` | `flex` | `flex` |
| flex-direction | `column` | `column` | `flex-col` |
| gap | `2px` | `calc(var(--spacing) * 0.5)` | `gap-0.5` |

### `AnchoredToasts` → `[data-slot=toast-title]` · `<Toast.Title>` (line 252)

#### base

```txt
font-medium
```

**element — default**

Key metrics: font-weight `500` (font-weight-medium)

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| font-weight | `500` | `font-weight-medium` | `font-medium` |

### `AnchoredToasts` → `[data-slot=toast-description]` · `<Toast.Description>` (line 256)

#### base

```txt
text-muted-foreground
```

**element — default**

Key metrics: color `#686868` / `#818181` (muted-foreground)

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| color | `#686868` / `#818181` | `muted-foreground` | `text-muted-foreground` |

### `AnchoredToasts` → `[data-slot=toast-action]` · `<Toast.Action>` (line 263)

Dynamic className inputs (not statically resolvable): `buttonVariants({ size: "xs" })`

### `ToastProvider`

No own classes (re-export / wrapper of the primitive); prop defaults: `position="bottom-right"`.

### `AnchoredToastProvider`

No own classes (re-export / wrapper of the primitive).

## Other class strings in this file

Class strings not bound directly to a JSX `className` (lookup tables, maps, helpers). Resolved per class (desktop, light / dark, no state):

- line 58: context `isEven ? "animate-toast-error-even motion-reduce:animate-none" : "animate-toast-error-odd motion-reduce:animate-none"`

  ```txt
  animate-toast-error-even motion-reduce:animate-none
  ```

  Resolved: animation: `toast-error-even 280ms cubic-bezier(0.5, 1, 0.89, 1)`

- line 59: context `isEven ? "animate-toast-error-even motion-reduce:animate-none" : "animate-toast-error-odd motion-reduce:animate-none"`

  ```txt
  animate-toast-error-odd motion-reduce:animate-none
  ```

  Resolved: animation: `toast-error-odd 280ms cubic-bezier(0.5, 1, 0.89, 1)`

- line 62: context `isEven ? "animate-toast-success-even motion-reduce:animate-none" : "animate-toast-success-odd motion-reduce:animate-none"`

  ```txt
  animate-toast-success-even motion-reduce:animate-none
  ```

  Resolved: animation: `toast-success-even 320ms cubic-bezier(0.5, 1, 0.89, 1)`

- line 63: context `isEven ? "animate-toast-success-even motion-reduce:animate-none" : "animate-toast-success-odd motion-reduce:animate-none"`

  ```txt
  animate-toast-success-odd motion-reduce:animate-none
  ```

  Resolved: animation: `toast-success-odd 320ms cubic-bezier(0.5, 1, 0.89, 1)`

