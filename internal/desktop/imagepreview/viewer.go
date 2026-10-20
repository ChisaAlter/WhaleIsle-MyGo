package imagepreview

import (
	"fmt"
	"strings"
	"whaleisle/internal/store/storelocale"
)

// The isolated image reader is deliberately script-free. Native form controls
// retain their checked value in the tab's live WebView and CSS scales only the
// inert image. The scrollport retains both axes when that tab is hidden/shown.
// No document script, application bridge, external URL or form submission is
// needed to change zoom or pan with the scrollbars/trackpad.
func imageViewer(name, source string) string {
	var css, choices, controls strings.Builder
	css.WriteString(`html,body{margin:0;width:100%;height:100%;background:#202124;color:#ddd;font:13px sans-serif}body{display:flex;flex-direction:column}p{padding:8px 12px;margin:0;flex:none}.zoom-choice{position:absolute;width:1px;height:1px;opacity:0}.toolbar{padding:6px 12px;display:flex;gap:8px;align-items:center;flex-wrap:wrap;flex:none}.toolbar label{display:inline-block;border:1px solid #777;border-radius:4px;padding:4px 8px;cursor:pointer}.step{display:none;align-items:center;gap:8px}.presets{display:flex;gap:4px;flex-wrap:wrap}.zoom-choice:focus-visible~.toolbar{outline:2px solid #91baff}main{flex:1;min-width:0;min-height:0;overflow:auto;padding:12px;box-sizing:border-box}img{display:block;max-width:none;height:auto;user-select:none;-webkit-user-drag:none}#zoom-fit:checked~main img{max-width:100%;zoom:1}#zoom-fit:checked~.toolbar .step-fit{display:inline-flex}`)
	choices.WriteString(`<input class="zoom-choice" type="radio" name="image-zoom" id="zoom-fit" aria-label="` + storelocale.T(storelocale.KeyDesktopImagepreviewFitWidth) + `" checked>`)
	// Fit mode remains responsive. Select a fixed preset to enter the 25-percent
	// step ladder; do not mislabel a fixed scale as a step from an unknown fit.
	controls.WriteString(`<span class="step step-fit">` + storelocale.T(storelocale.KeyDesktopImagepreviewFitWidth) + `</span>`)
	for percent := 25; percent <= 400; percent += 25 {
		fmt.Fprintf(&choices, `<input class="zoom-choice" type="radio" name="image-zoom" id="zoom-%d" aria-label="%d%%">`, percent, percent)
		fmt.Fprintf(&css, `#zoom-%d:checked~main img{zoom:%g}#zoom-%d:checked~.toolbar .step-%d{display:inline-flex}`, percent, float64(percent)/100, percent, percent)
		fmt.Fprintf(&controls, `<span class="step step-%d">`, percent)
		if percent > 25 {
			fmt.Fprintf(&controls, `<label for="zoom-%d" title="`+storelocale.T(storelocale.KeyDesktopImagepreviewZoomOut)+`">−</label>`, percent-25)
		} else {
			controls.WriteString(`<span aria-disabled="true">−</span>`)
		}
		fmt.Fprintf(&controls, `<span>%d%%</span>`, percent)
		if percent < 400 {
			fmt.Fprintf(&controls, `<label for="zoom-%d" title="`+storelocale.T(storelocale.KeyDesktopImagepreviewZoomIn)+`">+</label>`, percent+25)
		} else {
			controls.WriteString(`<span aria-disabled="true">+</span>`)
		}
		controls.WriteString(`</span>`)
	}
	var presets strings.Builder
	for _, percent := range []int{25, 50, 100, 150, 200} {
		fmt.Fprintf(&presets, `<label for="zoom-%d">%d%%</label>`, percent, percent)
	}
	return `<style>` + css.String() + `</style><p>` + name + ` · ` + storelocale.T(storelocale.KeyDesktopImagepreviewIsolatedSnapshot) + `</p>` + choices.String() + `<div class="toolbar" role="group" aria-label="` + storelocale.T(storelocale.KeyDesktopImagepreviewImageZoom) + `">` + controls.String() + `<label for="zoom-fit">` + storelocale.T(storelocale.KeyDesktopImagepreviewFitWidth) + `</label><details><summary>` + storelocale.T(storelocale.KeyDesktopImagepreviewZoom) + `</summary><div class="presets">` + presets.String() + `</div></details></div><main tabindex="0" aria-label="` + storelocale.T(storelocale.KeyDesktopImagepreviewImageScrollArea) + `"><img alt="Image codec unavailable in this WebView runtime" draggable="false" referrerpolicy="no-referrer" src="` + source + `"></main>`
}
