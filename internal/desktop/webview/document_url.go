package webview

import (
	"errors"
	"net/url"
	"strings"
	"whaleisle/internal/desktop/workspacefs"
)

// DocumentURL is presentation navigation, not a file capability. The original
// resource owner resolves this Session path through its authenticated Host.
func DocumentURL(origin, sessionID, path string) (string, error) {
	if strings.TrimSpace(sessionID) == "" {
		return "", errors.New("document viewer requires a Session")
	}
	if err := workspacefs.ValidateRelativePath(path); err != nil {
		return "", err
	}
	base := normalizeSlotBase(origin)
	parsed, err := url.Parse(base)
	if err != nil || base == "" {
		return "", errors.New("document viewer requires a Host")
	}
	query := parsed.Query()
	query.Set("desktopDocumentSession", sessionID)
	query.Set("desktopDocumentPath", strings.ReplaceAll(path, "\\", "/"))
	parsed.RawQuery = query.Encode()
	return parsed.String(), nil
}
