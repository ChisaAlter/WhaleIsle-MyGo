package custompanel

import (
	"whaleisle/internal/errcopy"
	panel "whaleisle/internal/store/custompanel"
	"whaleisle/internal/ui/i18n"
)

// Failure is a refused panel action worded for a person: the sentence the panel
// shows, and the store's own words behind 详情. The store names a problem and
// never writes the sentence, because that one has to be translated.
type Failure struct {
	Sentence string
	Detail   string
}

// problemKeys names the registered copy for each problem. The map holds keys,
// not sentences: a sentence read at package init would freeze the locale.
var problemKeys = map[panel.Problem]string{
	panel.ProblemJSON:            i18n.KeyCmdPanelJSONInvalid,
	panel.ProblemTooLarge:        i18n.KeyCmdPanelTooLarge,
	panel.ProblemTrailingContent: i18n.KeyCmdPanelTrailingContent,
	panel.ProblemMissingTitle:    i18n.KeyCmdPanelTitleRequired,
	panel.ProblemSize:            i18n.KeyCmdPanelSizeOutOfRange,
	panel.ProblemWidgetCount:     i18n.KeyCmdPanelWidgetCount,
	panel.ProblemWidgetID:        i18n.KeyCmdPanelWidgetID,
	panel.ProblemWidgetKind:      i18n.KeyCmdPanelWidgetKind,
	panel.ProblemFontSize:        i18n.KeyCmdPanelFontSize,
	panel.ProblemProgressRange:   i18n.KeyCmdPanelProgressRange,
	panel.ProblemSelectOptions:   i18n.KeyCmdPanelSelectOptions,
	panel.ProblemLibraryVersion:  i18n.KeyCmdPanelLibraryVersion,
	panel.ProblemScope:           i18n.KeyCmdPanelScopeMissing,
	panel.ProblemAlreadyAdded:    i18n.KeyCmdPanelAlreadyAdded,
	panel.ProblemTemplateMissing: i18n.KeyCmdPanelTemplateMissing,
	panel.ProblemInstanceMissing: i18n.KeyCmdPanelInstanceMissing,
}

// Word turns an error the panel store reported into the pair the page draws. A
// problem the store named becomes its registered sentence; anything else keeps
// errcopy's classification and files the error's own text behind 详情.
func Word(err error) Failure {
	if err == nil {
		return Failure{}
	}
	if key, ok := problemKeys[panel.ProblemOf(err)]; ok {
		return Failure{Sentence: i18n.T(key), Detail: err.Error()}
	}
	return Failure{Sentence: errcopy.Sentence(err), Detail: err.Error()}
}

// Words is a failure the view already worded itself, with nothing to disclose.
func Words(sentence string) Failure { return Failure{Sentence: sentence} }
