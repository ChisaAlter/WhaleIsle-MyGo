package theme

// The scroll bar's thumb colors, sampled from the harness's own tokens
// (ui-theme/src/styles/design-platform.css, `--dsw-alias-scrollbar-*`). The
// tokens resolve to entries of the static neutral scale, so the shell pins the
// resolved colors rather than the token names:
//
//	light  --scrollbar-bg-l1/-l2:    --dsw-static-neutral-200  rgb(229, 229, 229)
//	       --scrollbar-hover-l1/-l2: --dsw-static-neutral-300  rgb(212, 212, 212)
//	dark   --scrollbar-bg-l1:        --dsw-static-neutral-700  rgb(60, 60, 61)
//	       --scrollbar-hover-l1:     --dsw-static-neutral-600  rgb(84, 85, 87)
//	       --scrollbar-bg-l2:        --dsw-static-neutral-600  rgb(84, 85, 87)
//	       --scrollbar-hover-l2:     --dsw-static-neutral-550  rgb(101, 103, 107)
//
// The l2 pair is what the reference rebinds on a raised surface — a menu, a
// modal, a raised settings pane — so its thumbs keep their contrast against a
// lighter (dark scheme) or darker (light scheme) fill instead of the page's base.
// The toolkit paints one pair for the whole window (ui.Theme.Scrollbar), so the
// shell wires the l1 pair today; the l2 pair is carried here for the surface
// pass that can paint it per element, and the difference is only visible in the
// dark scheme, where l1 and l2 differ.
const (
	// ScrollbarRestLight is the light scheme's thumb, --dsw-static-neutral-200.
	ScrollbarRestLight = "rgb(229, 229, 229)"
	// ScrollbarHoverLight is the light scheme's thumb under the pointer,
	// --dsw-static-neutral-300.
	ScrollbarHoverLight = "rgb(212, 212, 212)"
	// ScrollbarRestDark is the dark scheme's level-1 thumb,
	// --dsw-static-neutral-700.
	ScrollbarRestDark = "rgb(60, 60, 61)"
	// ScrollbarHoverDark is the dark scheme's level-1 thumb under the pointer,
	// --dsw-static-neutral-600.
	ScrollbarHoverDark = "rgb(84, 85, 87)"
	// ScrollbarRestDarkRaised is the dark scheme's level-2 thumb,
	// --dsw-static-neutral-600.
	ScrollbarRestDarkRaised = "rgb(84, 85, 87)"
	// ScrollbarHoverDarkRaised is the dark scheme's level-2 thumb under the
	// pointer, --dsw-static-neutral-550.
	ScrollbarHoverDarkRaised = "rgb(101, 103, 107)"
)

// ScrollbarColors returns the thumb color a scroll container paints, and the one
// it paints while the pointer is over it, for the scheme: the l1 pair, which is
// what a page's own background uses. A raised surface takes the l2 pair (see
// ScrollbarRestDarkRaised); the light half resolves both levels to the same two
// colors.
func ScrollbarColors(dark bool) (rest, hover string) {
	if dark {
		return ScrollbarRestDark, ScrollbarHoverDark
	}
	return ScrollbarRestLight, ScrollbarHoverLight
}
