package usage

import (
	"fmt"
	"image/png"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/ui/i18n"
)

// The installed header opens a three-item export menu and the billing modal.
func TestUsageHeaderExportMenuAndBillingSettings(t *testing.T) {
	old := i18n.Locale()
	i18n.SetLocale(i18n.LocaleEN)
	defer i18n.SetLocale(old)
	formats := []string{}
	s := usageFixture(t, "empty")
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Scroll(c).Fill().Children(func() {
			View(c, Model{Store: s, Embedded: true, OnExportStats: func(format string) { formats = append(formats, format) }})
		})
	}, 680, 1000)
	for _, item := range []string{"Export JSON", "Export daily CSV", "Export model CSV"} {
		if err := tt.Click("Export"); err != nil {
			t.Fatalf("export action is missing: %v", err)
		}
		if got := tt.Menu(); !reflect.DeepEqual(got, []string{"Export JSON", "Export daily CSV", "Export model CSV"}) {
			t.Fatalf("wrong export menu: %v", got)
		}
		if err := tt.ChooseMenuItem(item); err != nil {
			t.Fatal(err)
		}
	}
	if !reflect.DeepEqual(formats, []string{"json", "days", "models"}) {
		t.Fatalf("wrong export dispatch: %v", formats)
	}
	if err := tt.Click("Settings"); err != nil {
		t.Fatal(err)
	}
	for deadline := time.Now().Add(time.Second); (s.Billing().Busy || s.BillingModels().Loading) && time.Now().Before(deadline); {
		time.Sleep(time.Millisecond)
	}
	tt.Frame()
	tt.SetTime(time.Now().Add(time.Second))
	text := strings.Join(tt.Texts(), "\n")
	if !strings.Contains(text, "Billing settings") || !tt.HasText("Close") || !tt.HasText("Off-peak prices") || !tt.HasText("Peak prices") {
		t.Fatalf("Settings did not open billing: %s", text)
	}
	if dir := os.Getenv("SECONDARY_SNAPSHOT_DIR"); dir != "" {
		for _, width := range []int{900, 400} {
			tt.SetSize(width, 900)
			for _, period := range []string{"Off-peak prices", "Peak prices"} {
				bottom := float32(-1)
				for _, label := range []string{"Input (cache hit)", "Input (cache miss)", "Output"} {
					r, ok := tt.Find(period + " · " + label)
					if !ok || r.H != 36 {
						t.Fatalf("missing price input at width %d: %s %+v", width, label, r)
					}
					if bottom >= 0 && r.Y+r.H != bottom {
						t.Fatalf("price input bottoms differ at width %d: %s %+v, previous %g", width, label, r, bottom)
					}
					bottom = r.Y + r.H
				}
			}
			f, err := os.Create(filepath.Join(dir, fmt.Sprintf("billing-modal-%d.png", width)))
			if err != nil {
				t.Fatal(err)
			}
			if err = png.Encode(f, tt.Image()); err != nil {
				t.Fatal(err)
			}
			if err = f.Close(); err != nil {
				t.Fatal(err)
			}
		}
	}
	if err := tt.SetValue("Off-peak prices · Input (cache hit)", "0.05"); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click("Save"); err != nil {
		t.Fatal(err)
	}
	for deadline := time.Now().Add(time.Second); s.Billing().Busy && time.Now().Before(deadline); {
		time.Sleep(time.Millisecond)
	}
	tt.Frame()
	if tt.HasText("Billing settings") || s.Billing().Err != "" || s.Billing().Dirty {
		t.Fatal("successful Save did not close billing", s.Billing())
	}
}
