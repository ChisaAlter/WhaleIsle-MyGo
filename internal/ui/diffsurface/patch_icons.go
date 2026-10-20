package diffsurface

import "github.com/egoist/mygo/ui"

// Fold artwork for a patch file header, from the original 16px regular icon set.
var diffChevronDownIcon = ui.MustParseSVG([]byte(`<svg width="16" height="16" viewBox="0 0 16 16" fill="none" xmlns="http://www.w3.org/2000/svg">
  <path d="M4.5 6.25L8 9.75L11.5 6.25" stroke="currentColor" stroke-width="1.2" stroke-linecap="round" stroke-linejoin="round"/>
</svg>`))

var diffChevronRightIcon = ui.MustParseSVG([]byte(`<svg width="16" height="16" viewBox="0 0 16 16" fill="none" xmlns="http://www.w3.org/2000/svg">
  <path d="M6.25 4.5L9.75 8L6.25 11.5" stroke="currentColor" stroke-width="1.2" stroke-linecap="round" stroke-linejoin="round"/>
</svg>`))
