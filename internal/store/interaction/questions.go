package interaction

import "strings"

// QuestionRowView is one question line in the interaction card.
type QuestionRowView struct {
	ID           string
	Question     string
	Header       string
	OptionLabels []string
	MultiSelect  bool
}

// QuestionCardView is parsed user-questions/request payload for UI.
type QuestionCardView struct {
	EventID string
	Rows    []QuestionRowView
}

type draftSession struct {
	req     userQuestionsRequest
	answers map[string]*answerDraft
}

type answerDraft struct {
	selected   []string
	custom     string
	selectWire string
	checks     map[string]*bool
}

func newDraftSession(req userQuestionsRequest) *draftSession {
	ds := &draftSession{
		req:     req,
		answers: make(map[string]*answerDraft, len(req.Questions)),
	}
	for _, q := range req.Questions {
		ad := &answerDraft{checks: make(map[string]*bool)}
		if len(q.Options) > 0 && !q.MultiSelect {
			label := q.Options[0].Label
			ad.selected = []string{label}
			ad.selectWire = label
		}
		for _, o := range q.Options {
			v := containsLabel(ad.selected, o.Label)
			ad.checks[o.Label] = &v
		}
		ds.answers[q.ID] = ad
	}
	return ds
}

func (ds *draftSession) cardView(eventID string) QuestionCardView {
	cv := QuestionCardView{EventID: eventID}
	for _, q := range ds.req.Questions {
		row := QuestionRowView{
			ID:          q.ID,
			Question:    q.Question,
			Header:      q.Header,
			MultiSelect: q.MultiSelect,
		}
		for _, o := range q.Options {
			row.OptionLabels = append(row.OptionLabels, o.Label)
		}
		cv.Rows = append(cv.Rows, row)
	}
	return cv
}

func (ds *draftSession) buildAnswer() userQuestionsAnswer {
	var answers []userQuestionsAnswerItem
	for _, q := range ds.req.Questions {
		ad := ds.answers[q.ID]
		if ad == nil {
			continue
		}
		item := userQuestionsAnswerItem{ID: q.ID}
		if len(q.Options) > 0 {
			item.Selected = append([]string(nil), ad.selected...)
		} else if strings.TrimSpace(ad.custom) != "" {
			c := strings.TrimSpace(ad.custom)
			item.Custom = &c
		}
		answers = append(answers, item)
	}
	return userQuestionsAnswer{Answers: answers}
}

func (ds *draftSession) answerValid() bool {
	for _, q := range ds.req.Questions {
		ad := ds.answers[q.ID]
		if ad == nil {
			return false
		}
		if len(q.Options) > 0 {
			if len(ad.selected) == 0 {
				return false
			}
			continue
		}
		if strings.TrimSpace(ad.custom) == "" {
			return false
		}
	}
	return len(ds.req.Questions) > 0
}

func containsLabel(labels []string, label string) bool {
	for _, l := range labels {
		if l == label {
			return true
		}
	}
	return false
}

func appendUnique(labels []string, label string) []string {
	if containsLabel(labels, label) {
		return labels
	}
	return append(labels, label)
}

func removeLabel(labels []string, label string) []string {
	out := labels[:0]
	for _, l := range labels {
		if l != label {
			out = append(out, l)
		}
	}
	return out
}
