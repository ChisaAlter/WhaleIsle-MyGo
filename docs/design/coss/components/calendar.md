# Calendar

Source: `apps/ui/registry/default/ui/calendar.tsx` @ `b2336c45f2b7` · docs: `apps/ui/content/docs/components/calendar.mdx` · Base UI: https://daypicker.dev/

> A date picker component with range and multi-select support.

Depends on: `lucide-react`

Exports: `Calendar`

Module constants: `buttonClassNames = relative flex size-(--cell-size) text-base sm:text-sm items-center justify-center rounded-lg text-foreground not-in-data-selected:hover:bg-accent disabled:pointer-events-none disabled:opacity-64 [&_svg:not([class*='opacity-'])]:opacity-80 [&_svg:not([class*='size-'])]:size-4.5 sm:[&_svg:not([class*='size-'])]:size-4 [&_svg]:pointer-events-none [&_svg]:shrink-0`

## Behavior & keyboard

- Built on react-day-picker. Arrow keys move focus between days, `PageUp`/`PageDown` change month, `Shift+PageUp/PageDown` change year, `Home`/`End` start/end of week, `Enter`/`Space` select. _(inferred: react-day-picker behavior)_

## How to read

Each part lists the exact class strings, then **resolved** values for the desktop environment (≥640px so all `sm:` rules apply; also md/lg/xl; fine pointer; hover-capable). Values show `light` / `dark` when they differ. *State* tables only list properties that change versus the part's default. *Mobile* lists properties that differ at 375px / coarse pointer. `src` is the class that wins the cascade. See `../README.md` for conventions.

## Parts

### `Calendar` → (no data-slot) · `<ChevronLeftIcon>` (line 84)

Attributes: `aria-hidden=true`

Prop defaults: `showOutsideDays=true`, `mode="single"`

#### base

```txt
rtl:rotate-180
```

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| rtl | rotate | `180deg` | `rtl:rotate-180` |

### `Calendar` → (no data-slot) · `<ChevronRightIcon>` (line 94)

Attributes: `aria-hidden=true`

Prop defaults: `showOutsideDays=true`, `mode="single"`

#### base

```txt
rtl:rotate-180
```

**element — states**

| state | property | value (light / dark) | src |
|---|---|---|---|
| rtl | rotate | `180deg` | `rtl:rotate-180` |

### `Calendar` → `[data-slot=calendar]` · `props object` (line 117)

Attributes: `classNames={mergedClassNames}`, `components={mergedComponents}`, `formatters={{ formatMonthDropdown: (month: Date, dateLib: DateLib) => dateLib.format(month, "LLL"), } as React.ComponentProps<typeo`

Prop defaults: `showOutsideDays=true`, `mode="single"`

#### base

```txt
w-fit [--cell-size:--spacing(10)] sm:[--cell-size:--spacing(9)]
```

**element — default**

Key metrics: width `fit-content`

| property | value (light / dark) | token / expr | src |
|---|---|---|---|
| width | `fit-content` | `fit-content` | `w-fit` |

## Other class strings in this file

Class strings not bound directly to a JSX `className` (lookup tables, maps, helpers). Resolved per class (desktop, light / dark, no state):

- line 13: context `buttonClassNames = "relative flex size-(--cell-size) text-base sm:text-sm items-center justify-center rounded-lg text-foreground not-in-data`

  ```txt
  relative flex size-(--cell-size) text-base sm:text-sm items-center justify-center rounded-lg text-foreground not-in-data-selected:hover:bg-accent disabled:pointer-events-none disabled:opacity-64 [&_svg:not([class*='opacity-'])]:opacity-80 [&_svg:not([class*='size-'])]:size-4.5 sm:[&_svg:not([class*='size-'])]:size-4 [&_svg]:pointer-events-none [&_svg]:shrink-0
  ```

  Resolved: position: `relative` · display: `flex` · width: `36px` · height: `36px` · align-items: `center` · justify-content: `center` · border-radius: `10px` · font-size: `14px` · line-height: `20px` · color: `#262626` / `#f5f5f5` · [hover] background-color: `#0000000a` / `#ffffff0a` · [disabled] pointer-events: `none` · [disabled] opacity: `64%` · svg pointer-events: `none` · svg flex-shrink: `0` · svg:not([class*='opacity-']) opacity: `80%` · svg:not([class*='size-']) width: `16px` · svg:not([class*='size-']) height: `16px`

- line 27: context `caption_label: "text-base sm:text-sm font-medium flex items-center gap-2 h-full"`

  ```txt
  text-base sm:text-sm font-medium flex items-center gap-2 h-full
  ```

  Resolved: display: `flex` · height: `100%` · align-items: `center` · gap: `8px` · font-size: `14px` · line-height: `20px` · font-weight: `500`

- line 28: context `day: "size-(--cell-size) text-sm py-px"`

  ```txt
  size-(--cell-size) text-sm py-px
  ```

  Resolved: width: `36px` · height: `36px` · padding-block: `1px` · font-size: `14px` · line-height: `20px`

- line 31: context `cn( buttonClassNames, "in-data-disabled:pointer-events-none in-[.range-middle]:rounded-none in-[.range-end:not(.range-start)]:rounded-s-none`

  ```txt
  in-data-disabled:pointer-events-none in-[.range-middle]:rounded-none in-[.range-end:not(.range-start)]:rounded-s-none in-[.range-start:not(.range-end)]:rounded-e-none in-[.range-middle]:in-data-selected:bg-accent in-data-selected:bg-primary in-[.range-middle]:in-data-selected:text-foreground in-data-disabled:text-muted-foreground/72 in-data-outside:text-muted-foreground/72 in-data-selected:in-data-outside:text-primary-foreground in-data-selected:text-primary-foreground in-data-disabled:line-through outline-none in-[[data-selected]:not(.range-middle)]:transition-[border-radius,box-shadow] focus-visible:z-1 focus-visible:ring-[3px] focus-visible:ring-ring/50
  ```

  Resolved: outline-style: `none` · [in([data-disabled])] pointer-events: `none` · [in([data-disabled])] color: `#686868b8` / `#818181b8` · [in([data-disabled])] text-decoration-line: `line-through` · [in(:where(*:is(.range-middle)))] border-radius: `0` · [in(:where(*:is(.range-end:not(.range-start))))] border-start-start-radius: `0` · [in(:where(*:is(.range-end:not(.range-start))))] border-end-start-radius: `0` · [in(:where(*:is(.range-start:not(.range-end))))] border-start-end-radius: `0` · [in(:where(*:is(.range-start:not(.range-end))))] border-end-end-radius: `0` · [in([data-selected])] background-color: `#262626` / `#f5f5f5` · [in([data-selected])] color: `#fafafa` / `#262626` · [in([data-outside])] color: `#686868b8` / `#818181b8` · [in(:where(*:is([data-selected]:not(.range-middle))))] transition-property: `border-radius, box-shadow` · [in(:where(*:is([data-selected]:not(.range-middle))))] transition-timing-function: `cubic-bezier(0.4, 0, 0.2, 1)` · [in(:where(*:is([data-selected]:not(.range-middle))))] transition-duration: `150ms` · [focus-visible] z-index: `1` · [focus-visible] box-shadow: `0 0 0 3px #a1a1a180` / `0 0 0 3px #73737380` · [in(:where(*:is(.range-middle))) + in([data-selected])] background-color: `#0000000a` / `#ffffff0a` · [in(:where(*:is(.range-middle))) + in([data-selected])] color: `#262626` / `#f5f5f5` · [in(:where(*:is(.range-middle))) + in([data-selected])] border-radius: `0` · [in([data-outside]) + in([data-selected])] color: `#fafafa` / `#262626` · [in([data-outside]) + in([data-selected])] background-color: `#262626` / `#f5f5f5`

- line 33: context `dropdown: "absolute bg-popover inset-0 opacity-0"`

  ```txt
  absolute bg-popover inset-0 opacity-0
  ```

  Resolved: position: `absolute` · inset: `0px` · background-color: `#ffffff` / `#1d1d1d` · opacity: `0%`

- line 35: context `dropdown_root: "relative has-focus:border-ring has-focus:ring-ring/50 has-focus:ring-[3px] border border-input shadow-xs/5 rounded-lg px-[ca`

  ```txt
  relative has-focus:border-ring has-focus:ring-ring/50 has-focus:ring-[3px] border border-input shadow-xs/5 rounded-lg px-[calc(--spacing(3)-1px)] h-9 sm:h-8 [&_svg:not([class*='opacity-'])]:opacity-80 [&_svg:not([class*='size-'])]:size-4.5 sm:[&_svg:not([class*='size-'])]:size-4 [&_svg]:pointer-events-none [&_svg]:-me-1
  ```

  Resolved: border-width: `1px` · border-style: `solid` · border-color: `#0000001a` / `#ffffff14` · position: `relative` · height: `32px` · border-radius: `10px` · padding-inline: `11px` · box-shadow: `0 1px 2px 0 #0000000d` · [has(:focus)] border-color: `#a1a1a1` / `#737373` · [has(:focus)] box-shadow: `0 0 0 3px #a1a1a180, 0 1px 2px 0 #0000000d` / `0 0 0 3px #73737380, 0 1px 2px 0 #0000000d` · svg pointer-events: `none` · svg margin-inline-end: `-4px` · svg:not([class*='opacity-']) opacity: `80%` · svg:not([class*='size-']) width: `16px` · svg:not([class*='size-']) height: `16px`

- line 37: context `dropdowns: "w-full flex items-center text-base sm:text-sm justify-center h-(--cell-size) gap-1.5 *:[span]:font-medium"`

  ```txt
  w-full flex items-center text-base sm:text-sm justify-center h-(--cell-size) gap-1.5 *:[span]:font-medium
  ```

  Resolved: display: `flex` · height: `36px` · width: `100%` · align-items: `center` · justify-content: `center` · gap: `6px` · font-size: `14px` · line-height: `20px` · > *:is(span) font-weight: `500`

- line 39: context `month: "w-full"`

  ```txt
  w-full
  ```

  Resolved: width: `100%`

- line 41: context `month_caption: "relative mx-(--cell-size) px-1 mb-1 flex h-(--cell-size) items-center justify-center z-2"`

  ```txt
  relative mx-(--cell-size) px-1 mb-1 flex h-(--cell-size) items-center justify-center z-2
  ```

  Resolved: position: `relative` · z-index: `2` · margin-inline: `36px` · margin-bottom: `4px` · display: `flex` · height: `36px` · align-items: `center` · justify-content: `center` · padding-inline: `4px`

- line 42: context `months: "relative flex flex-col sm:flex-row gap-2"`

  ```txt
  relative flex flex-col sm:flex-row gap-2
  ```

  Resolved: position: `relative` · display: `flex` · flex-direction: `row` · gap: `8px`

- line 43: context `nav: "absolute top-0 flex w-full justify-between z-1"`

  ```txt
  absolute top-0 flex w-full justify-between z-1
  ```

  Resolved: position: `absolute` · top: `0px` · z-index: `1` · display: `flex` · width: `100%` · justify-content: `space-between`

- line 45: context `outside: "text-muted-foreground data-selected:bg-accent/50 data-selected:text-muted-foreground"`

  ```txt
  text-muted-foreground data-selected:bg-accent/50 data-selected:text-muted-foreground
  ```

  Resolved: color: `#686868` / `#818181` · [selected] background-color: `#00000005` / `#ffffff05`

- line 50: context `today: "*:after:pointer-events-none *:after:absolute *:after:bottom-1 *:after:start-1/2 *:after:z-1 *:after:size-[3px] *:after:-translate-x-`

  ```txt
  *:after:pointer-events-none *:after:absolute *:after:bottom-1 *:after:start-1/2 *:after:z-1 *:after:size-[3px] *:after:-translate-x-1/2 *:after:rounded-full *:after:bg-primary [&[data-selected]:not(.range-middle)>*]:after:bg-background [&[data-disabled]>*]:after:bg-foreground/30
  ```

  Resolved: [selected] *::after content: `""` · [selected] *::after background-color: `#ffffff` / `#141414` · [disabled] *::after content: `""` · [disabled] *::after background-color: `#2626264d` / `#f5f5f54d` · > *::after content: `""` · > *::after pointer-events: `none` · > *::after position: `absolute` · > *::after inset-inline-start: `50%` · > *::after bottom: `4px` · > *::after z-index: `1` · > *::after width: `3px` · > *::after height: `3px` · > *::after translate: `-50% 0` · > *::after border-radius: `9999px` · > *::after background-color: `#262626` / `#f5f5f5`

- line 52: context `week_number: "size-(--cell-size) p-0 text-xs font-medium text-muted-foreground/72"`

  ```txt
  size-(--cell-size) p-0 text-xs font-medium text-muted-foreground/72
  ```

  Resolved: width: `36px` · height: `36px` · padding: `0px` · font-size: `12px` · line-height: `16px` · font-weight: `500` · color: `#686868b8` / `#818181b8`

- line 54: context `weekday: "size-(--cell-size) p-0 text-xs font-medium text-muted-foreground/72"`

  ```txt
  size-(--cell-size) p-0 text-xs font-medium text-muted-foreground/72
  ```

  Resolved: width: `36px` · height: `36px` · padding: `0px` · font-size: `12px` · line-height: `16px` · font-weight: `500` · color: `#686868b8` / `#818181b8`

