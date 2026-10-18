package interaction

import (
	"encoding/json"
	"testing"

	"whaleisle/internal/dsh"
)

// The original QuestionComposer derives a draft only from a marked first label.
// Unmarked/later recommendations must not make an untouched answer submittable.
func TestQuestionRecommendedFirstDraftContract(t *testing.T) {
	cases := []struct {
		name               string
		labels             []string
		multi, recommended bool
	}{
		{name: "unmarked", labels: []string{"Fast", "Slow"}},
		{name: "later recommendation", labels: []string{"Fast", "Slow (recommended)"}},
		{name: "english", labels: []string{"Fast (recommended)", "Slow"}, recommended: true},
		{name: "case insensitive", labels: []string{"Fast (ReCoMmEnDeD)"}, recommended: true},
		{name: "chinese", labels: []string{"快速 (推荐)"}, recommended: true},
		{name: "fullwidth chinese", labels: []string{"快速（推荐）"}, recommended: true},
		{name: "fullwidth english", labels: []string{"Fast（recommended）"}, recommended: true},
		{name: "trailing whitespace", labels: []string{"Fast(recommended)\t\n\u00a0\u3000\ufeff"}, recommended: true},
		{name: "not suffix", labels: []string{"Fast (recommended) extra"}},
		{name: "mismatched parentheses", labels: []string{"Fast (recommended）"}},
		{name: "spaces inside marker", labels: []string{"Fast ( recommended )"}},
		{name: "not javascript whitespace", labels: []string{"Fast (recommended)\u0085"}},
		{name: "multi recommended", labels: []string{"Fast (recommended)", "Slow"}, multi: true, recommended: true},
		{name: "multi unmarked", labels: []string{"Fast", "Slow"}, multi: true},
		{name: "no choices"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			q := userQuestionItem{ID: "q", Question: "Pick", MultiSelect: tc.multi}
			for _, label := range tc.labels {
				q.Options = append(q.Options, userQuestionOption{Label: label})
			}
			s := New()
			s.pending = []Pending{{AgentID: "A", EventID: "e", Event: EventUserQuestions}}
			s.waiters["e"] = make(chan dsh.EventResultOutcome, 1)
			s.questionDrafts["e"] = newDraftSession(userQuestionsRequest{Questions: []userQuestionItem{q}})
			if s.QuestionsCanSubmit("e") != tc.recommended {
				t.Fatal("incorrect untouched draft completion")
			}
			if len(s.waiters["e"]) != 0 || s.pending[0].Sending {
				t.Fatal("recommendation submitted without user action")
			}
			ad := s.questionDrafts["e"].answers["q"]
			if ad.skipped {
				t.Fatal("unanswered draft became skipped")
			}
			if tc.recommended {
				if len(ad.selected) != 1 || ad.selected[0] != tc.labels[0] || !*s.QuestionOptionCheckedPtr("e", "q", tc.labels[0]) {
					t.Fatal("recommendation label/binding changed")
				}
				s.ResolveQuestions("e")
				out := <-s.waiters["e"]
				var answer userQuestionsAnswer
				if err := json.Unmarshal(out.Value, &answer); err != nil {
					t.Fatal(err)
				}
				if out.Kind != "result" || len(answer.Answers) != 1 || len(answer.Answers[0].Selected) != 1 || answer.Answers[0].Selected[0] != tc.labels[0] {
					t.Fatal("wire did not preserve complete recommended label")
				}
			} else {
				if len(ad.selected) != 0 || ad.selectWire != "" {
					t.Fatal("unmarked choice was selected")
				}
				s.ResolveQuestions("e")
				if len(s.waiters["e"]) != 0 || s.pending[0].Sending {
					t.Fatal("unanswered draft submitted")
				}
				s.SetQuestionCustom("e", "q", "explicit answer")
				if !s.QuestionsCanSubmit("e") {
					t.Fatal("explicit answer rejected")
				}
			}
		})
	}
}
