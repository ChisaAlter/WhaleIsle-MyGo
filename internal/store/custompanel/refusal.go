package custompanel

import (
	"errors"
	"fmt"
)

// Problem is what is wrong with a panel definition. The store names the
// problem; the sentence a person reads is copy in the interface language, so it
// belongs to the surface that speaks it, not here.
type Problem string

const (
	ProblemJSON            Problem = "json"
	ProblemTooLarge        Problem = "too_large"
	ProblemTrailingContent Problem = "trailing_content"
	ProblemMissingTitle    Problem = "missing_title"
	ProblemSize            Problem = "size"
	ProblemWidgetCount     Problem = "widget_count"
	ProblemWidgetID        Problem = "widget_id"
	ProblemWidgetKind      Problem = "widget_kind"
	ProblemFontSize        Problem = "font_size"
	ProblemProgressRange   Problem = "progress_range"
	ProblemSelectOptions   Problem = "select_options"
	ProblemLibraryVersion  Problem = "library_version"
	ProblemScope           Problem = "scope"
	ProblemAlreadyAdded    Problem = "already_added"
	ProblemTemplateMissing Problem = "template_missing"
	ProblemInstanceMissing Problem = "instance_missing"
)

// Refusal is why the store rejected a panel. Error() is the store's own
// wording — evidence for 详情 and the log — and never the sentence a person
// reads, because that one has to be translated.
type Refusal struct {
	Problem Problem
	Detail  string
}

func (r Refusal) Error() string { return r.Detail }

func refuse(problem Problem, format string, args ...any) error {
	return Refusal{Problem: problem, Detail: fmt.Sprintf(format, args...)}
}

// ProblemOf reports which problem err names, or "" when it is not a refusal.
func ProblemOf(err error) Problem {
	var refusal Refusal
	if errors.As(err, &refusal) {
		return refusal.Problem
	}
	return ""
}
