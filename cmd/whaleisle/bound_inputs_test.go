package main

import (
	"context"
	"testing"

	"github.com/egoist/mygo/ui"

	desktopgit "whaleisle/internal/desktop/git"
	"whaleisle/internal/ui/i18n"
)

// tabFocus advances the focus chain n steps; inputs carry no findable label,
// so tests reach them through focus order instead of tt.Click.
func tabFocus(tt *ui.Tester, n int) {
	for i := 0; i < n; i++ {
		tt.Key(0, ui.KeyTab)
	}
}

// Editing any reviewed PR field must drop the prepared plan. The bound
// values used to be compared before applyInputs flushed the edits, so the
// plan survived stale.
func TestPhonePRReviewFieldEditsClearPlan(t *testing.T) {
	newReview := func() *phonePRReview {
		return &phonePRReview{ctx: context.Background(), root: "root", plan: &desktopgit.PullRequestPlan{}}
	}
	mk := func(q *phonePRReview) *ui.Tester {
		return ui.NewTester(func(c *ui.Context) { q.view(c) }, 400, 700)
	}
	t.Run("title", func(t *testing.T) {
		q := newReview()
		tt := mk(q)
		// Focus order: title input, body area, draft checkbox, buttons.
		tabFocus(tt, 1)
		tt.Type("x")
		if q.plan != nil {
			t.Fatal("title edit did not clear the prepared plan")
		}
	})
	t.Run("body", func(t *testing.T) {
		q := newReview()
		tt := mk(q)
		tabFocus(tt, 2)
		tt.Type("x")
		if q.plan != nil {
			t.Fatal("body edit did not clear the prepared plan")
		}
	})
	t.Run("draft", func(t *testing.T) {
		q := newReview()
		tt := mk(q)
		if err := tt.Click(i18n.T(i18n.KeyCmdCreateDraftPr)); err != nil {
			t.Fatalf("draft checkbox not reachable: %v", err)
		}
		if q.plan != nil {
			t.Fatal("draft toggle did not clear the prepared plan")
		}
	})
}
