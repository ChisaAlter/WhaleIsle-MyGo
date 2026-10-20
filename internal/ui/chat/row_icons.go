package chat

import "github.com/egoist/mygo/ui"

// Row artwork for conversation disclosures, mirroring ui-primitives' think and
// data glyphs at the shared 16-unit viewBox.
var (
	disclosureIconThink = ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 16 16" fill="none" stroke-width="1">
    <path d="M8 1.6C11.2 1.6 13.6 4 13.6 7C13.6 8.7 12.9 9.9 11.8 10.8V13.2C11.8 13.6 11.5 13.9 11.1 13.9H8.9C8.5 13.9 8.2 13.6 8.2 13.2V12.3H6.1C5.7 12.3 5.4 12 5.4 11.6V10.4C4 9.5 2.4 8.6 2.4 7C2.4 4 4.8 1.6 8 1.6Z" stroke="currentColor" />
    <path d="M6 6.2C6 5.3 6.9 4.7 7.7 5C8.5 5.3 8.7 6.3 8.1 6.8C7.6 7.2 7.6 7.7 7.9 8.1" stroke="currentColor" />
  </svg>`))
	disclosureIconData = ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 16 16" fill="none" stroke-width="1">
    <rect x="1.8" y="9.2" width="2.6" height="5" rx=".6" stroke="currentColor" />
    <rect x="6.7" y="5.4" width="2.6" height="8.8" rx=".6" stroke="currentColor" />
    <rect x="11.6" y="1.8" width="2.6" height="12.4" rx=".6" stroke="currentColor" />
  </svg>`))
)

// rowOpenState is the per-row disclosure state used where no shared map owns
// it. It lives on the row element so list re-renders keep the choice.
func rowOpenState(row ui.Element, key string, initial bool) *bool {
	return ui.Local(row, "row-open:"+key, func() bool { return initial })
}
