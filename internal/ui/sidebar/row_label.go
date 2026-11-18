package sidebar

import (
	"strings"

	"whaleisle/internal/errcopy"
)

// sessionRowLabel is the text a Session row shows. A title that is the Host's
// provider failure text names a route and an environment variable, which is not
// copy and not a title: the row speaks the same cause sentence as the failure
// row, and the raw text stays in the retained event.
func sessionRowLabel(title string) string {
	trimmed := strings.TrimSpace(title)
	if !errcopy.ProviderEnvelope(trimmed) {
		return title
	}
	if sentence := errcopy.SentenceText(trimmed); sentence != "" {
		return sentence
	}
	return title
}
