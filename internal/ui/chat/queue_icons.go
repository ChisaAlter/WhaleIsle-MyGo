package chat

import "github.com/egoist/mygo/ui"

// queueIconSteer is the "send now" arrow the dock uses for steering, matching
// the Original queue dock's icon-only actions.
var queueIconSteer = ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.3" stroke-linecap="round" stroke-linejoin="round">
    <path d="M4.4 11.6 11.6 4.4M6.2 4.4h5.4v5.4"/>
  </svg>`))
