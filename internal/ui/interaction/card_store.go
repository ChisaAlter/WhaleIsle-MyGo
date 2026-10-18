package interaction

import "whaleisle/internal/store/interaction"

// CardStore keeps submission methods virtual: a retained pane can use shared
// drafts without accidentally falling through to the global sender.
type CardStore interface {
	SnapshotForAgent(string) interaction.ViewState
	SyncQuestionFocus(string, string)
	SyncQuestionFocusForAgent(string, string)
	SetAutoAllow(bool)
	ResolveAllow(string)
	ResolveDeny(string)
	Retry(string)
	ApprovePlanReview(string)
	DismissPlanReview(string)
	ReopenPlanReview(string)
	HideQuestion(string)
	ReopenQuestion(string)
	ReviewQuestionHistory(string, string)
	SetQuestionOptionChecked(string, string, string, bool)
	SetQuestionCustom(string, string, string)
	SetQuestionSkipped(string, string, bool)
	AdvanceQuestionChoice(string, string, string)
	ContinueQuestion(string, string, string) (string, string)
	SubmitQuestionBatch(string, string) (string, string)
	SkipQuestionAndContinue(string, string, string) (string, string)
	NavigateQuestion(string, string, int) bool
	TakeQuestionTime(string, string)
}

var _ CardStore = (*interaction.Store)(nil)
var _ CardStore = (*interaction.ScopedStore)(nil)
