package settings

import (
	"fmt"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/ui/i18n"
)

func TestAbbreviatePathMiddleKeepsBothEnds(t *testing.T) {
	long := `C:\Users\someone\AppData\Roaming\WhaleIsle-MyGo\data`
	short := abbreviatePathMiddle(long, 24)
	if len([]rune(short)) > 24 {
		t.Fatalf("abbreviated path is %d runes: %q", len([]rune(short)), short)
	}
	if !strings.HasPrefix(short, `C:\`) || !strings.HasSuffix(short, `MyGo\data`) {
		t.Fatalf("abbreviation lost an end: %q", short)
	}
	if !strings.Contains(short, "…") {
		t.Fatalf("abbreviation has no ellipsis: %q", short)
	}
	if got := abbreviatePathMiddle(`C:\short`, 60); got != `C:\short` {
		t.Fatalf("short path changed: %q", got)
	}
}

func dataLocationFixture(changes *int, opens *int) Model {
	return Model{
		DesktopActive: true,
		Desktop: &DesktopAccess{
			DataLocation: &DataLocationAccess{
				Current: `C:\Users\someone\AppData\Roaming\WhaleIsle-MyGo\data`,
				Default: `C:\Users\someone\AppData\Roaming\WhaleIsle-MyGo\data`,
				Err:     i18n.T(i18n.KeyDataLocationRefusedNested),
				OnChange: func() {
					*changes++
				},
				OnOpen: func() {
					*opens++
				},
			},
		},
	}
}

func TestDataLocationRowRendersPathAndRefusal(t *testing.T) {
	previous := i18n.Locale()
	i18n.SetLocale(i18n.LocaleZH)
	t.Cleanup(func() { i18n.SetLocale(previous) })

	changes, opens := 0, 0
	m := dataLocationFixture(&changes, &opens)
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Padding(24).Children(func() {
			dataLocationRow(c, m.Desktop.DataLocation, c.Theme().Text, c.Theme().TextMuted)
		})
	}, 900, 420)

	for _, want := range []string{
		i18n.T(i18n.KeyDataLocationTitle),
		i18n.T(i18n.KeyDataLocationHint),
		i18n.T(i18n.KeyDataLocationChange),
		i18n.T(i18n.KeyDataLocationOpen),
		i18n.T(i18n.KeyDataLocationRefusedNested),
	} {
		if !tt.HasText(want) {
			t.Fatalf("data-location row is missing %q\n%v", want, tt.Texts())
		}
	}
	// The path is shortened in the middle but keeps both ends readable.
	if !tt.HasText(i18n.T(i18n.KeyDataLocationCurrent) + abbreviatePathMiddle(m.Desktop.DataLocation.Current, 60)) {
		t.Fatalf("data-location row did not render the abbreviated path: %v", tt.Texts())
	}
	// 恢复默认 stays hidden while the current directory already is the default.
	if tt.HasText(i18n.T(i18n.KeyDataLocationReset)) {
		t.Fatalf("restore-default was offered for the default directory: %v", tt.Texts())
	}
	if err := tt.Click(i18n.T(i18n.KeyDataLocationChange)); err != nil {
		t.Fatalf("change action unreachable: %v", err)
	}
	if err := tt.Click(i18n.T(i18n.KeyDataLocationOpen)); err != nil {
		t.Fatalf("open action unreachable: %v", err)
	}
	if changes != 1 || opens != 1 {
		t.Fatalf("actions ran %d/%d times", changes, opens)
	}
}

func TestDataLocationRowShowsUnusableRecordAndRestore(t *testing.T) {
	previous := i18n.Locale()
	i18n.SetLocale(i18n.LocaleEN)
	t.Cleanup(func() { i18n.SetLocale(previous) })

	m := Model{
		DesktopActive: true,
		Desktop: &DesktopAccess{
			DataLocation: &DataLocationAccess{
				Current:  `C:\Users\someone\AppData\Roaming\WhaleIsle-MyGo\data`,
				Default:  `C:\Users\someone\AppData\Roaming\WhaleIsle-MyGo\data`,
				Recorded: `D:\gone\data`,
				OnReset:  func() {},
			},
		},
	}
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Padding(24).Children(func() {
			dataLocationRow(c, m.Desktop.DataLocation, c.Theme().Text, c.Theme().TextMuted)
		})
	}, 900, 420)
	if !tt.HasText(fmt.Sprintf(i18n.T(i18n.KeyDataLocationUnavailable), `D:\gone\data`)) {
		t.Fatalf("unusable recorded location is not shown: %v", tt.Texts())
	}
	if !tt.HasText(i18n.T(i18n.KeyDataLocationReset)) {
		t.Fatalf("restore-default is missing for a recorded location: %v", tt.Texts())
	}
}

func TestDataLocationRowOffersLegacyImport(t *testing.T) {
	previous := i18n.Locale()
	i18n.SetLocale(i18n.LocaleEN)
	t.Cleanup(func() { i18n.SetLocale(previous) })

	imports := 0
	m := Model{
		DesktopActive: true,
		Desktop: &DesktopAccess{
			DataLocation: &DataLocationAccess{
				Current:         `C:\data`,
				Default:         `C:\data`,
				ImportAvailable: true,
				OnImport:        func() { imports++ },
			},
		},
	}
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Padding(24).Children(func() {
			dataLocationRow(c, m.Desktop.DataLocation, c.Theme().Text, c.Theme().TextMuted)
		})
	}, 900, 420)
	if err := tt.Click(i18n.T(i18n.KeyDataLocationImport)); err != nil {
		t.Fatalf("import action unreachable: %v", err)
	}
	if imports != 1 {
		t.Fatalf("import ran %d times", imports)
	}
}

func TestDataLocationRowReadOnlyHidesEditing(t *testing.T) {
	previous := i18n.Locale()
	i18n.SetLocale(i18n.LocaleEN)
	t.Cleanup(func() { i18n.SetLocale(previous) })

	changes := 0
	m := Model{
		DesktopActive: true,
		Desktop: &DesktopAccess{
			DataLocation: &DataLocationAccess{
				Current:        `C:\data`,
				Default:        `C:\data`,
				ReadOnly:       true,
				ReadOnlyReason: i18n.T(i18n.KeyDataLocationReadOnly),
				OnChange:       func() { changes++ },
			},
		},
	}
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Padding(24).Children(func() {
			dataLocationRow(c, m.Desktop.DataLocation, c.Theme().Text, c.Theme().TextMuted)
		})
	}, 900, 420)
	if !tt.HasText(i18n.T(i18n.KeyDataLocationReadOnly)) {
		t.Fatalf("read-only reason is missing: %v", tt.Texts())
	}
	// 打开目录 still works: the profile exists, only recording a location is
	// refused.
	if !tt.HasText(i18n.T(i18n.KeyDataLocationOpen)) {
		t.Fatalf("open action was hidden in a read-only run: %v", tt.Texts())
	}
	if tt.HasText(i18n.T(i18n.KeyDataLocationChange)) || tt.HasText(i18n.T(i18n.KeyDataLocationReset)) {
		t.Fatalf("editing actions are offered in a read-only run: %v", tt.Texts())
	}
	if changes != 0 {
		t.Fatalf("read-only row ran the change action %d times", changes)
	}
}

func TestDataLocationRowNamesTheBrokenRecordFile(t *testing.T) {
	previous := i18n.Locale()
	i18n.SetLocale(i18n.LocaleEN)
	t.Cleanup(func() { i18n.SetLocale(previous) })

	recordFile := `C:\Users\someone\AppData\Roaming\WhaleIsle-MyGo\data-location.json`
	m := Model{
		DesktopActive: true,
		Desktop: &DesktopAccess{
			DataLocation: &DataLocationAccess{
				Current:      `C:\Users\someone\AppData\Roaming\WhaleIsle-MyGo\data`,
				Default:      `C:\Users\someone\AppData\Roaming\WhaleIsle-MyGo\data`,
				Recorded:     recordFile,
				RecordBroken: true,
				OnReset:      func() {},
			},
		},
	}
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Padding(24).Children(func() {
			dataLocationRow(c, m.Desktop.DataLocation, c.Theme().Text, c.Theme().TextMuted)
		})
	}, 900, 420)
	// The broken sentence names the record file, and it must not read as if
	// the default location were that file.
	want := fmt.Sprintf(i18n.T(i18n.KeyDataLocationRecordBroken), recordFile)
	if !tt.HasText(want) {
		t.Fatalf("broken record is not named: %v", tt.Texts())
	}
	if tt.HasText(fmt.Sprintf(i18n.T(i18n.KeyDataLocationUnavailable), recordFile)) {
		t.Fatalf("broken record used the missing-directory sentence: %v", tt.Texts())
	}
}
