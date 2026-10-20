package interaction

import "testing"

// The Host marks its recommended option by suffixing the label; the row keeps
// that label for the wire answer and exposes the flag for the badge.
func TestRecommendedOptionFlagsAndDisplayLabel(t *testing.T) {
	req := userQuestionsRequest{Questions: []userQuestionItem{{
		ID:       "q",
		Question: "Pick one",
		Options: []userQuestionOption{
			{Label: "Fast (recommended)"},
			{Label: "Slow"},
		},
	}}}
	cv := newDraftSession(req).cardView("evt")
	if len(cv.Rows) != 1 {
		t.Fatalf("rows = %d", len(cv.Rows))
	}
	row := cv.Rows[0]
	if len(row.Recommended) != 2 || !row.Recommended[0] || row.Recommended[1] {
		t.Fatalf("recommended flags = %v", row.Recommended)
	}
	if got := RecommendedQuestionLabel(row.OptionLabels[0]); got != "Fast" {
		t.Fatalf("display label = %q", got)
	}
	if got := RecommendedQuestionLabel(row.OptionLabels[1]); got != "Slow" {
		t.Fatalf("plain label changed: %q", got)
	}
	if got := RecommendedQuestionLabel("（推荐）"); got != "（推荐）" {
		t.Fatalf("a label that is only the suffix must not become empty: %q", got)
	}
}
