package chrome

import (
	"testing"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/ui/i18n"
)

func TestGitCommitStripHiddenWhenClosed(t *testing.T) {
	i18n.SetLocale("en")
	defer i18n.SetLocale(i18n.DefaultLocale)

	msg := "draft"
	tt := ui.NewTester(func(c *ui.Context) {
		DrawGitCommitStrip(c, DefaultTitleBarColors(), false, GitCommitStrip{
			Message:     &msg,
			Placeholder: i18n.T(i18n.KeyGitCommitPlaceholder),
			CommitLabel: i18n.T(i18n.KeyGitCommit),
		})
	}, 600, 80)

	if tt.HasText(i18n.T(i18n.KeyGitCommit)) {
		t.Fatalf("commit strip should be hidden; texts = %q", tt.Texts())
	}
}

func TestGitCommitStripOpen(t *testing.T) {
	i18n.SetLocale("en")
	defer i18n.SetLocale(i18n.DefaultLocale)

	msg := ""
	tt := ui.NewTester(func(c *ui.Context) {
		DrawGitCommitStrip(c, DefaultTitleBarColors(), true, GitCommitStrip{
			Message:     &msg,
			Dirty:       true,
			Placeholder: i18n.T(i18n.KeyGitCommitPlaceholder),
			CommitLabel: i18n.T(i18n.KeyGitCommit),
			CancelLabel: i18n.T(i18n.KeyGitCommitCancel),
			OnCommit:    func() {},
			OnCancel:    func() {},
		})
	}, 600, 80)

	if !tt.HasText(i18n.T(i18n.KeyGitCommit)) {
		t.Fatalf("missing commit button; texts = %q", tt.Texts())
	}
	if !tt.HasText(i18n.T(i18n.KeyGitCommitCancel)) {
		t.Fatalf("missing cancel button; texts = %q", tt.Texts())
	}
}
