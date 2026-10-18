package dirpick

import (
	"testing"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/dsh/api"
	"whaleisle/internal/ui/i18n"
)

func TestDirPickView(t *testing.T) {
	i18n.SetLocale("zh")
	defer i18n.SetLocale(i18n.DefaultLocale)
	name := ""
	tt := ui.NewTester(func(c *ui.Context) {
		View(c, Model{
			Listing: api.DirectoryListing{
				Path: "/home",
				Crumbs: []api.DirectoryListingCrumbsItem{
					{Name: "home", Path: "/home"},
				},
				Entries: []api.DirectoryListingCrumbsItem{
					{Name: "proj", Path: "/home/proj"},
				},
			},
			ThemeID:    "system",
			SystemDark: false,
			NewName:    &name,
			OnNavigate: func(string) {},
			OnPick:     func(string) {},
			OnCreate:   func(string, string) {},
			OnClose:    func() {},
		})
	}, 480, 360)
	for _, label := range []string{
		i18n.T(i18n.KeyDirPickTitle),
		"home",
		"proj",
		i18n.T(i18n.KeyDirPickSelect),
		i18n.T(i18n.KeyDirPickSelectHere),
		i18n.T(i18n.KeyDirPickCreate),
	} {
		if !tt.HasText(label) {
			t.Fatalf("missing %q; texts=%q", label, tt.Texts())
		}
	}
}
