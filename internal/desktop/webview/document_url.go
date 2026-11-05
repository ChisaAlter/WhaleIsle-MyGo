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
	return documentURL(origin, sessionID, path)
}

// DeclaredDocumentURL retains the Host's session-file address contract for an
// explicitly declared deliverable, including a Host-absolute external path.
// Callers must retain the live declaration grant; this grants no local access.
func DeclaredDocumentURL(origin, sessionID, path string) (string, error) {
	if strings.TrimSpace(sessionID) == "" || strings.TrimSpace(path) == "" || strings.ContainsAny(path, "\x00\r\n") || strings.Contains(path, "://") {
		return "", errors.New("invalid declared document address")
	}
	absolute := strings.HasPrefix(path, "/") || strings.HasPrefix(path, `\`) || len(path) > 2 && path[1] == ':' && (path[2] == '/' || path[2] == '\\')
	if !absolute {
		if err := workspacefs.ValidateRelativePath(path); err != nil {
			return "", err
		}
	}
	return documentURL(origin, sessionID, path)
}

func documentURL(origin, sessionID, path string) (string, error) {
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
