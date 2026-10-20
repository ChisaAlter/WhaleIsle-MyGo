package files

import "github.com/egoist/mygo/ui"

// Tab-strip artwork, scaled from the original 16px regular icon set: the close
// control and the trailing affordance that returns the tree to a listing.
var tabCloseIcon = ui.MustParseSVG([]byte(`<svg width="16" height="16" viewBox="0 0 16 16" fill="none" xmlns="http://www.w3.org/2000/svg">
  <path d="M3.5 3.5L12.5 12.5" stroke="currentColor" stroke-width="1.2" stroke-linecap="round"/>
  <path d="M12.5 3.5L3.5 12.5" stroke="currentColor" stroke-width="1.2" stroke-linecap="round"/>
</svg>`))

var tabPlusIcon = ui.MustParseSVG([]byte(`<svg width="16" height="16" viewBox="0 0 16 16" fill="none" xmlns="http://www.w3.org/2000/svg">
  <path d="M8 2.5V13.5" stroke="currentColor" stroke-width="1.2" stroke-linecap="round"/>
  <path d="M2.5 8H13.5" stroke="currentColor" stroke-width="1.2" stroke-linecap="round"/>
</svg>`))
