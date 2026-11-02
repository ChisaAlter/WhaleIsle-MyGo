package openui

import (
	"errors"
	"net/url"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

// IsOwnedViewer compares the actual document with this build's embedded runtime.
// A renderer's self-reported runtime name is never sufficient to enable actions.
// Callers must separately disable actions for every explicit legacy override.
func IsOwnedViewer(viewer string) bool {
	return viewer != "" && (viewer == embeddedViewer || IsWhaleViewer(viewer))
}

// ValidateNativeAction checks the bounded typed data, not a user gesture proof.
// UserGesture is meaningful only after the trusted shell's exact source check,
// owned-document admission, session replay claim and current-owner UI check.
func ValidateNativeAction(e Event) error {
	if !e.UserGesture || e.Sequence == 0 || e.Sequence > 9007199254740991 ||
		e.RequestID != "native-action-"+strconv.FormatUint(e.Sequence, 10) {
		return errors.New("invalid OpenUI native action identity or gesture")
	}
	switch e.Type {
	case "clipboard":
		if len(e.Text) > MaxEventBytes || !utf8.ValidString(e.Text) || strings.ContainsRune(e.Text, '\x00') || e.URL != "" {
			return errors.New("invalid OpenUI clipboard text")
		}
	case "download":
		if e.Parts < 1 || e.Parts > 3000 || e.Text != "" || e.URL != "" {
			return errors.New("invalid Whale UI export identity")
		}
		return ValidateDownload(e)
	case "open-url":
		if len(e.URL) == 0 || len(e.URL) > MaxEventBytes || !utf8.ValidString(e.URL) || e.Text != "" ||
			strings.TrimSpace(e.URL) != e.URL || strings.ContainsFunc(e.URL, unicode.IsControl) {
			return errors.New("invalid OpenUI external URL")
		}
		u, err := url.Parse(e.URL)
		if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Hostname() == "" || u.User != nil || u.Opaque != "" {
			return errors.New("OpenUI links require an absolute HTTP(S) URL without credentials")
		}
	default:
		return errors.New("unsupported OpenUI native action")
	}
	return nil
}
