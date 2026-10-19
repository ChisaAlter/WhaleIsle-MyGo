package market

import (
	"context"
	"encoding/base64"
	"testing"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/ui/svgimage"
)

func TestPluginImageBrowserSVGRequiresLiveOwner(t *testing.T) {
	data := []byte(`<svg width="1in" height="2in"><animate attributeName="opacity" values="0;1" dur="1s"/></svg>`)
	var decoded readmeDecodedImage
	loadPluginImageData(&decoded, data, func() {})
	if decoded.svgSeat == nil || decoded.width != 96 || decoded.height != 192 || decoded.bitmap != nil || decoded.svg != nil {
		t.Fatal("animated SVG became a partial static image or lost intrinsic hint")
	}
	defer decoded.svgSeat.Close()
	source := "data:image/svg+xml;base64," + base64.StdEncoding.EncodeToString(data)
	for _, owner := range []*svgimage.Owner{nil, {ID: "revoked", Context: context.Background(), Window: &mygo.Window{}, Valid: func() bool { return false }}} {
		ui.NewTester(func(c *ui.Context) {
			if PluginImage(c, source, 640, 0, owner) {
				t.Fatal("unowned artwork acquired a browser seat")
			}
		}, 800, 600)
	}
}
