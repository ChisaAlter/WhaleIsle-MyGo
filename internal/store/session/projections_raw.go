package session

import (
	"encoding/json"

	"whaleisle/internal/dsh/api"
)

// projectionsRaw restores the raw-JSON form of one session's projection values.
// The pinned 0.2.2 runtime publishes the projection record as a typed struct;
// the decoders in this package still read that record as JSON, so it is
// marshalled once at the boundary instead of threading the generated type
// through every decoder.
func projectionsRaw(values api.SessionListValueItemsItemProjectionsValues) json.RawMessage {
	raw, err := json.Marshal(values)
	if err != nil {
		return nil
	}
	return raw
}
