package interaction

import (
	"regexp"
	"strings"
	"time"
)

// QuestionRowView is one question line in the interaction card.
type QuestionRowView struct {
	Selected           []string
	Custom             string
	ID                 string
	Question           string
	Detail             string
	OptionDescriptions []string
	Header             string
	OptionLabels       []string
	MultiSelect        bool
	Skipped            bool
}

// QuestionCardView is parsed user-questions/request payload for UI.
type QuestionCardView struct {
	Wait             *QuestionWaitView
	Hidden           bool
	PlanReview       *PlanReview
	PlanReviewHidden bool
	EventID          string
	Rows             []QuestionRowView
}

type draftSession struct {
	wait             questionWaitState
	hidden           bool
	planReviewHidden bool
	req              userQuestionsRequest
	answers          map[string]*answerDraft
}

type answerDraft struct {
	skipped    bool
	selected   []string
	custom     string
	selectWire string
	checks     map[string]*bool
}

// Matches QuestionComposer.parseRecommendedLabel's suffix and ECMAScript whitespace.
// Keep the complete label in the draft and wire answer; this is not confirmation.
var recommendedQuestionSuffix = regexp.MustCompile(`(?i)(?:\((?:recommended|推荐)\)|（(?:recommended|推荐)）)[\t\n\v\f\r \x{00a0}\x{1680}\x{2000}-\x{200a}\x{2028}\x{2029}\x{202f}\x{205f}\x{3000}\x{feff}]*$`)

func newDraftSession(req userQuestionsRequest) *draftSession {
	ds := &draftSession{
		req:     req,
		answers: make(map[string]*answerDraft, len(req.Questions)),
	}
	for _, q := range req.Questions {
		ad := &answerDraft{checks: make(map[string]*bool)}
		if len(q.Options) > 0 && planReviewOf(req.Questions) == nil && recommendedQuestionSuffix.MatchString(q.Options[0].Label) {
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
	cv := QuestionCardView{Wait: ds.wait.view(time.Now()), Hidden: ds.hidden, EventID: eventID, PlanReview: planReviewOf(ds.req.Questions), PlanReviewHidden: ds.planReviewHidden}
	for _, q := range ds.req.Questions {
		row := QuestionRowView{
			ID:          q.ID,
			Question:    q.Question,
			Header:      q.Header,
			Detail:      q.Detail,
			MultiSelect: q.MultiSelect,
		}
		if ad := ds.answers[q.ID]; ad != nil {
			row.Skipped = ad.skipped
			row.Selected = append([]string(nil), ad.selected...)
			row.Custom = ad.custom
		}
		for _, o := range q.Options {
			row.OptionLabels = append(row.OptionLabels, o.Label)
			row.OptionDescriptions = append(row.OptionDescriptions, o.Description)
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
		item := userQuestionsAnswerItem{ID: q.ID, Selected: []string{}}
		if ad.skipped {
			answers = append(answers, item)
			continue
		}
		if len(q.Options) > 0 {
			item.Selected = append([]string{}, ad.selected...)
		}
		if strings.TrimSpace(ad.custom) != "" {
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
		if ad.skipped {
			continue
		}
		if len(q.Options) > 0 {
			if len(ad.selected) == 0 && strings.TrimSpace(ad.custom) == "" {
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

// SetQuestionSkipped explicitly completes one generic question without an answer.
// Undoing a skip leaves it unanswered; selecting or typing an answer also clears it.
func (s *Store) SetQuestionSkipped(eventID, questionID string, skipped bool) {
	s.mu.Lock()
	p, ds := s.findPending(eventID), s.questionDrafts[eventID]
	if p == nil || p.Event != EventUserQuestions || p.Sending || ds == nil || planReviewOf(ds.req.Questions) != nil {
		s.mu.Unlock()
		return
	}
	ad := ds.answers[questionID]
	if ad == nil {
		s.mu.Unlock()
		return
	}
	s.engageQuestionLocked(eventID)
	ad.skipped = skipped
	if skipped {
		ad.selected = nil
		ad.custom, ad.selectWire = "", ""
		for _, ptr := range ad.checks {
			if ptr != nil {
				*ptr = false
			}
		}
	}
	s.mu.Unlock()
	s.fire()
}
