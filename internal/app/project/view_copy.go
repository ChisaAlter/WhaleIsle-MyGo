package projectctl

import "whaleisle/internal/errcopy"

// viewError turns a stored failure into the copy a person reads. Flow state
// keeps the raw text: logs read it, and classifiers such as the revert-conflict
// match in assets.go compare against it.
func viewError(raw string) string {
	return errcopy.SentenceText(raw)
}
