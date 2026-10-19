package market

import (
	"bytes"
	"os"
	"testing"
	"testing/synctest"
	"time"

	"github.com/egoist/mygo/ui"
)

func TestMarketReadmeAVIFSequenceNativePaintAndHide(t *testing.T) {
	data, err := os.ReadFile("../../desktop/imagepreview/testdata/test.avifs")
	if err != nil {
		t.Fatal(err)
	}
	cache := readmeImageCache(t)
	const source = "https://example.invalid/motion.avif"
	if err := os.WriteFile(cache.FilePath(source), data, 0600); err != nil {
		t.Fatal(err)
	}
	synctest.Test(t, func(t *testing.T) {
		visible := true
		tt := ui.NewTester(func(c *ui.Context) {
			if visible {
				marketReadme(c, Model{}, "![AVIF animation]("+source+")", c.Theme().Text, c.Theme().TextMuted, c.Theme().Surface, c.Theme().Accent)
			}
		}, 300, 300)
		synctest.Wait()
		tt.Frame()
		r, ok := tt.Find("README image " + source)
		if !ok || r.W != 300 || r.H != 216 || tt.HasText("AVIF animation") {
			t.Fatalf("AVIF README missing native image: %+v", r)
		}
		before := append([]byte(nil), tt.Image().Pix...)
		time.Sleep(90 * time.Millisecond)
		tt.Frame()
		if bytes.Equal(before, tt.Image().Pix) {
			t.Fatal("README AVIF remains first frame")
		}
		visible = false
		tt.Frame()
		time.Sleep(time.Second)
		synctest.Wait()
		tt.Frame()
		if _, ok := tt.Find("README image " + source); ok {
			t.Fatal("hidden README retained AVIF image seat")
		}
	})
}
