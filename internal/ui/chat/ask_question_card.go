package chat

import (
	"encoding/json"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

// The installed ask-user transcript card (AskQuestionCard.tsx + CSS): an
// answered card lists each question with its chosen answer lines, a skipped
// question shows the skipped label, and an unanswered card names the verdict
// above the recorded questions. The native presenter
// (toolviews/ask-question-row.tsx) builds it from the call's `questions`
// argument and the result's `answers` batch, pairing them by the stable ids, and
// uses error codes ASK_CANCELLED/ASK_ABORTED for the unanswered verdicts.

// askQuestionCard is the validated, localized transcript card.
type askQuestionCard struct {
	// Kind is "answered" or "unanswered".
	Kind      string
	Questions []askQuestionCardItem
	// SkippedLabel is the answered card's copy for a question without answers.
	SkippedLabel string
	// Verdict is the unanswered card's verdict sentence.
	Verdict string
}

// askQuestionCardItem is one question and the visible answer lines it produced.
type askQuestionCardItem struct {
	ID       string
	Question string
	Answers  []string
}

// askQuestionAnswer is one recorded answer entry, before pairing.
type askQuestionAnswer struct {
	id       string
	selected []string
	custom   *string
}

// askQuestionEntries reads the call's questions; nil when the shape cannot pair
// with answers unambiguously (non-empty array, unique string ids and questions).
func askQuestionEntries(args map[string]json.RawMessage) ([]askQuestionCardItem, bool) {
	raw, exists := args["questions"]
	if !exists {
		return nil, false
	}
	value, ok := toolDetailsDecodeJSON(string(raw))
	if !ok || value.kind != 'a' || len(value.list) == 0 {
		return nil, false
	}
	items := make([]askQuestionCardItem, 0, len(value.list))
	ids := make(map[string]bool, len(value.list))
	for _, entry := range value.list {
		if entry.kind != 'o' {
			return nil, false
		}
		id, okID := entry.get("id")
		question, okQuestion := entry.get("question")
		if !okID || id.kind != 's' || !okQuestion || question.kind != 's' || ids[id.text] {
			return nil, false
		}
		ids[id.text] = true
		items = append(items, askQuestionCardItem{ID: id.text, Question: question.text})
	}
	return items, true
}

// askQuestionAnswerEntries reads the result's recorded answer batch exactly as
// the installed answerEntries does.
func askQuestionAnswerEntries(text string) ([]askQuestionAnswer, bool) {
	value, ok := toolDetailsDecodeJSON(text)
	if !ok || value.kind != 'o' {
		return nil, false
	}
	answers, ok := value.get("answers")
	if !ok || answers.kind != 'a' {
		return nil, false
	}
	entries := make([]askQuestionAnswer, 0, len(answers.list))
	for _, entry := range answers.list {
		if entry.kind != 'o' {
			return nil, false
		}
		id, okID := entry.get("id")
		selected, okSelected := entry.get("selected")
		if !okID || id.kind != 's' || !okSelected || selected.kind != 'a' {
			return nil, false
		}
		lines := make([]string, 0, len(selected.list))
		for _, line := range selected.list {
			if line.kind != 's' {
				return nil, false
			}
			lines = append(lines, line.text)
		}
		answer := askQuestionAnswer{id: id.text, selected: lines}
		if custom, exists := entry.get("custom"); exists {
			if custom.kind != 's' {
				return nil, false
			}
			value := custom.text
			answer.custom = &value
		}
		entries = append(entries, answer)
	}
	return entries, true
}

// askQuestionPairAnswers pairs questions with answers by their echoed ids; nil
// when a count, id or duplicate makes the pairing ambiguous.
func askQuestionPairAnswers(questions []askQuestionCardItem, answers []askQuestionAnswer) ([]askQuestionCardItem, bool) {
	if len(questions) != len(answers) {
		return nil, false
	}
	byID := make(map[string]askQuestionAnswer, len(answers))
	for _, answer := range answers {
		if _, exists := byID[answer.id]; exists {
			return nil, false
		}
		byID[answer.id] = answer
	}
	paired := make([]askQuestionCardItem, 0, len(questions))
	for _, question := range questions {
		answer, exists := byID[question.ID]
		if !exists {
			return nil, false
		}
		lines := append([]string{}, answer.selected...)
		if answer.custom != nil && *answer.custom != "" {
			lines = append(lines, *answer.custom)
		}
		paired = append(paired, askQuestionCardItem{ID: question.ID, Question: question.Question, Answers: lines})
	}
	return paired, true
}

// askQuestionErrorCode reads the recorded tool error code, as the installed row
// reads block.error.code.
func askQuestionErrorCode(result *session.Message) string {
	envelope, ok := terminalModelObject(result.Raw)
	if !ok {
		return ""
	}
	data, ok := terminalModelObject(envelope["data"])
	if !ok {
		return ""
	}
	errorValue, ok := terminalModelObject(data["error"])
	if !ok {
		return ""
	}
	code, _ := terminalModelString(errorValue["code"])
	return code
}

// askQuestionCardFor derives the installed transcript card for one call, or nil
// when the recorded call and result cannot form one (the generic evidence body
// then remains).
func askQuestionCardFor(call *session.ToolCallView) *askQuestionCard {
	if call == nil || call.Result == nil {
		return nil
	}
	result, ok := terminalModelReadResult(call.Result)
	if !ok {
		return nil
	}
	name, args, ok := toolDetailsModelCallArgs(call, result.nested)
	if !ok || name != "ask_user_question" {
		return nil
	}
	questions, ok := askQuestionEntries(args)
	if !ok {
		return nil
	}
	if result.failed {
		verdict := ""
		switch askQuestionErrorCode(call.Result) {
		case "ASK_CANCELLED":
			verdict = i18n.T(i18n.KeyAskQuestionCancelled)
		case "ASK_ABORTED":
			verdict = i18n.T(i18n.KeyAskQuestionInterrupted)
		default:
			return nil
		}
		return &askQuestionCard{Kind: "unanswered", Questions: questions, Verdict: verdict}
	}
	text, ok := terminalModelSingleText(result.content)
	if !ok {
		return nil
	}
	answers, ok := askQuestionAnswerEntries(text)
	if !ok {
		return nil
	}
	paired, ok := askQuestionPairAnswers(questions, answers)
	if !ok {
		return nil
	}
	return &askQuestionCard{Kind: "answered", Questions: paired, SkippedLabel: i18n.T(i18n.KeyAskQuestionSkipped)}
}

// askQuestionCardView paints the installed transcript card: a bordered, capped
// surface whose answered list names each question and its answer lines, and
// whose unanswered form names the verdict above the questions.
func askQuestionCardView(c *ui.Context, card askQuestionCard, st ToolCardStyle, key string) {
	background, borderInk, primary, tertiary := st.Field, st.Border, st.FG, st.Muted
	if st.Colors.BG != "" {
		background, borderInk = chrome.Color(st.Colors.BG), chrome.Color(st.Colors.BorderL1)
		primary, tertiary = chrome.Color(st.Colors.FG), chrome.Color(st.Colors.Muted)
	}
	font := theme.RenderPreferences().Chat.Family
	size := theme.RenderPreferences().Chat.Size
	if size <= 0 {
		size = 14
	}
	line := size + 10
	root := ui.Column(c.Key("ask-card:"+key)).FillWidth().MinWidth(0).Margin(4, 0, 4, 4).
		Border(toolCSSHalfBorder(c), borderInk).Radius(16).Background(background).Clip()
	root.Children(func() {
		ui.Scroll(c.Key("scroll")).FillWidth().MinWidth(0).MaxHeight(360).Children(func() {
			list := ui.Column(c.Key("body")).FillWidth().MinWidth(0).Padding(16, 20).Gap(16)
			list.Children(func() {
				if card.Kind == "unanswered" {
					ui.Text(c, card.Verdict).Font(font).FontSize(size).FixedLineHeight(line).TextColor(primary).
						FillWidth().MinWidth(0).Selectable()
					questions := ui.Column(c.Key("questions")).FillWidth().MinWidth(0).Gap(8).Padding(0, 0, 0, 20)
					questions.Children(func() {
						for _, question := range card.Questions {
							ui.Text(c, question.Question).Font(font).FontSize(size).FixedLineHeight(line).TextColor(tertiary).
								FillWidth().MinWidth(0).Selectable()
						}
					})
					return
				}
				for index, question := range card.Questions {
					item := ui.Column(c.Key(index)).FillWidth().MinWidth(0).Gap(2)
					item.Children(func() {
						ui.Text(c, question.Question).Font(font).FontSize(size).FixedLineHeight(line).TextColor(tertiary).
							FillWidth().MinWidth(0).Selectable()
						if len(question.Answers) == 0 {
							ui.Text(c, card.SkippedLabel).Font(font).FontSize(size).FixedLineHeight(line).TextColor(tertiary).
								FillWidth().MinWidth(0).Selectable()
							return
						}
						for _, answer := range question.Answers {
							ui.Text(c, answer).Font(font).FontSize(size).FixedLineHeight(line).TextColor(primary).
								FillWidth().MinWidth(0).Selectable()
						}
					})
				}
			})
		})
	})
}
