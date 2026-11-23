package settings

import (
	"errors"

	"whaleisle/internal/store/storelocale"
)

// writeRefusal is a settings write the Host did not store. Error is the
// sentence a page shows: the reader's language, and the next step. Raw keeps
// the Host's own words, which belong in the log and a collapsed details block,
// never in that sentence.
type writeRefusal struct{ raw string }

func (e *writeRefusal) Error() string {
	return storelocale.T(storelocale.KeyStoreSettingsWriteNotStored)
}

// Raw is the unlocalized text behind the refusal.
func (e *writeRefusal) Raw() string { return e.raw }

// writeRefusalDetail is the raw text behind a refusal, "" for any other error.
func writeRefusalDetail(err error) string {
	if err == nil {
		return ""
	}
	var refusal *writeRefusal
	if errors.As(err, &refusal) {
		return refusal.raw
	}
	return ""
}
