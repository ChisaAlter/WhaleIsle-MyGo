package remote

import (
	"image"
	"image/color"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/desktop/phoneremote"
	"whaleisle/internal/ui/i18n"
)

// pairingQRBitmap builds a QR-shaped module grid. The panel only needs an
// image to draw; using a synthetic square keeps this test independent of the
// runtime QR encoder while still proving the bitmap reaches the surface.
func pairingQRBitmap(modules, scale int) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, modules*scale, modules*scale))
	white := color.RGBA{R: 255, G: 255, B: 255, A: 255}
	black := color.RGBA{R: 17, G: 17, B: 19, A: 255}
	for y := 0; y < img.Bounds().Dy(); y++ {
		for x := 0; x < img.Bounds().Dx(); x++ {
			img.Set(x, y, white)
		}
	}
	// Finder squares in three corners plus a checker body: dark enough that the
	// drawn area is unmistakable in the frame diff below.
	draw := func(originX, originY, size int) {
		for y := originY; y < originY+size; y++ {
			for x := originX; x < originX+size; x++ {
				col, row := (x-originX)/scale, (y-originY)/scale
				ring := col == 0 || row == 0 || col == size/scale-1 || row == size/scale-1
				core := col >= 2 && row >= 2 && col <= size/scale-3 && row <= size/scale-3
				if ring || core {
					img.Set(x, y, black)
				}
			}
		}
	}
	draw(scale, scale, scale*7)
	draw((modules-8)*scale, scale, scale*7)
	draw(scale, (modules-8)*scale, scale*7)
	for row := 0; row < modules; row++ {
		for col := 0; col < modules; col++ {
			if (row*7+col*3)%5 != 0 {
				continue
			}
			for y := row * scale; y < (row+1)*scale; y++ {
				for x := col * scale; x < (col+1)*scale; x++ {
					img.Set(x, y, black)
				}
			}
		}
	}
	return img
}

func phonePanelModel(runtime phoneremote.RuntimeState, onCopy func()) Model {
	return Model{
		GatewayProtocol:   "chisacode-v2",
		RemoteEnabled:     true,
		RemoteMode:        "lan",
		RemotePort:        6767,
		RemoteBindAddress: "0.0.0.0",
		PhoneRuntime:      runtime,
		OnCopyConnectURL:  onCopy,
		OnOpenConnectURL:  func() {},
		OnRefreshPhone:    func() {},
	}
}

func renderPhonePanel(t *testing.T, m Model) *ui.Tester {
	t.Helper()
	return ui.NewTester(func(c *ui.Context) {
		ui.Column(c).FillWidth().MinWidth(0).Padding(16).Gap(10).Children(func() {
			viewPhoneSection(c, m, c.Theme().Text, c.Theme().TextMuted, c.Theme().Surface,
				c.Theme().Border, c.Theme().Accent)
		})
	}, 760, 1500)
}

func differingPixels(a, b *image.RGBA) int {
	if a == nil || b == nil {
		return 0
	}
	bounds := a.Bounds().Intersect(b.Bounds())
	count := 0
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			if a.RGBAAt(x, y) != b.RGBAAt(x, y) {
				count++
			}
		}
	}
	return count
}

// TestPhonePanelDrawsPairingQRNotPlaceholder proves the panel renders the live
// pairing QR bitmap and the copy/open entry, and that the QR-less state falls
// back to the link-only copy instead of claiming pairing is unavailable.
func TestPhonePanelDrawsPairingQRNotPlaceholder(t *testing.T) {
	previous := i18n.Locale()
	i18n.SetLocale(i18n.LocaleZH)
	defer i18n.SetLocale(previous)

	copied := 0
	live := phoneremote.RuntimeState{
		Status:        "ready",
		Listening:     true,
		PairingURL:    "http://192.168.53.103:3180/#offer=c3ludGhldGljLW9ubHk",
		PairingImage:  pairingQRBitmap(25, 12),
		ExpiresAt:     time.Now().Add(9 * time.Minute),
		DevicesLoaded: true,
		BindAddresses: []string{"192.168.53.103"},
		DeviceAction:  phoneremote.DeviceActionState{},
		Devices:       nil,
	}
	withQR := renderPhonePanel(t, phonePanelModel(live, func() { copied++ }))
	saveSurfaceSnapshot(t, withQR, "surface-remote-phone-qr.png")
	if !withQR.HasText(i18n.T(i18n.KeyMiscRemoteCopyPairingLink)) {
		t.Fatalf("panel does not offer the pairing-copy entry: %v", withQR.Texts())
	}
	if !withQR.HasText(i18n.T(i18n.KeyMiscRemoteOpenPairingLink)) {
		t.Fatal("panel does not offer the pairing-open entry")
	}
	if !withQR.HasText(i18n.T(i18n.KeyMiscRemoteScanPrivatelyToPairYourPhone)) {
		t.Fatal("panel does not caption the QR as private pairing")
	}
	if withQR.HasText(i18n.T(i18n.KeyMiscRemoteQrUnavailableCopyTheTemporaryLinkP)) {
		t.Fatal("panel claims the QR is unavailable while a bitmap is present")
	}
	if err := withQR.Click(i18n.T(i18n.KeyMiscRemoteCopyPairingLink)); err != nil {
		t.Fatalf("pairing-copy entry is not clickable: %v", err)
	}
	if copied != 1 {
		t.Fatalf("pairing-copy entry did not reach the host: copied=%d", copied)
	}

	// Same panel without a bitmap: the layout must change by roughly the drawn
	// 320x320 QR area, and the copy must explain the link-only fallback.
	withoutImage := live
	withoutImage.PairingImage = nil
	noQR := renderPhonePanel(t, phonePanelModel(withoutImage, func() { copied++ }))
	if !noQR.HasText(i18n.T(i18n.KeyMiscRemoteQrUnavailableCopyTheTemporaryLinkP)) {
		t.Fatal("QR-less panel does not state that the link is the fallback")
	}
	if diff := differingPixels(withQR.Image(), noQR.Image()); diff < 20000 {
		t.Fatalf("panel frames differ by only %d pixels; the QR bitmap was not drawn", diff)
	}

	// An expired offer must withdraw the pairing entries instead of reusing the
	// stale bitmap.
	expired := live
	expired.ExpiresAt = time.Now().Add(-time.Minute)
	stale := renderPhonePanel(t, phonePanelModel(expired, func() { copied++ }))
	if stale.HasText(i18n.T(i18n.KeyMiscRemoteCopyPairingLink)) {
		t.Fatal("expired offer still offers a pairing copy")
	}
	if stale.HasText(i18n.T(i18n.KeyMiscRemoteScanPrivatelyToPairYourPhone)) {
		t.Fatal("expired offer still advertises a scannable QR")
	}
}
