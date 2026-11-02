package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"path"
	"strings"

	"whaleisle/internal/dsh"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/i18n"
)

type officePreviewState struct {
	store   *session.Store
	session string
	path    string
	message string
}

func (s *mainState) setOfficePreviewMessage(sessions *session.Store, sid, path, message string) {
	s.mu.Lock()
	s.officePreview = officePreviewState{store: sessions, session: sid, path: path, message: message}
	s.mu.Unlock()
	sessions.SetLastAction(message)
}

func (s *mainState) officePreviewMessage(sessions *session.Store, sid, path string) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	status := s.officePreview
	if status.store != sessions || status.session != sid || status.path != path {
		return ""
	}
	return status.message
}

func officePreviewName(source string) string {
	name := path.Base(strings.ReplaceAll(source, "\\", "/"))
	return strings.TrimSuffix(name, path.Ext(name)) + ".pdf"
}

func officePreviewReady(fonts []string) string {
	message := i18n.T(i18n.KeyCmdOfficePreviewReady)
	if len(fonts) > 0 {
		separator := "、"
		if i18n.Locale() == i18n.LocaleEN {
			separator = ", "
		}
		message += " " + fmt.Sprintf(i18n.T(i18n.KeyCmdOfficePreviewMissingFonts), strings.Join(fonts, separator))
	}
	return message
}

func officePreviewFailure(err error) string {
	unavailable := i18n.T(i18n.KeyCmdOfficePreviewUnavailable)
	var remote *dsh.Error
	if errors.As(err, &remote) {
		switch remote.Code {
		case "gateway/invocation-unavailable", "gateway/service-unavailable":
			return unavailable
		case "gateway/cancelled":
			return i18n.T(i18n.KeyCmdOfficePreviewCancelled)
		case "document-render/failed":
			var details struct {
				Reason string `json:"reason"`
			}
			_ = json.Unmarshal(remote.Details, &details)
			switch details.Reason {
			case "input-too-large", "output-too-large":
				return i18n.T(i18n.KeyCmdOfficePreviewTooLarge)
			case "invalid-document", "unsupported-format":
				return i18n.T(i18n.KeyCmdOfficePreviewInvalid)
			case "timeout":
				return i18n.T(i18n.KeyCmdOfficePreviewTimeout)
			case "unavailable":
				return unavailable
			case "busy":
				return i18n.T(i18n.KeyCmdOfficePreviewBusy)
			case "source-changed":
				return i18n.T(i18n.KeyCmdOfficePreviewSourceChanged)
			default:
				return i18n.T(i18n.KeyCmdOfficePreviewFailed)
			}
		}
	}
	if strings.Contains(err.Error(), "response exceeds bounded wire limit") {
		return i18n.T(i18n.KeyCmdOfficeTransportTooLarge)
	}
	if strings.Contains(err.Error(), "harness not ready") {
		return unavailable
	}
	return fmt.Sprintf(i18n.T(i18n.KeyCmdOfficeRetryFailure), err.Error())
}
