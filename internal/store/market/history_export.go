package market

import (
	"fmt"
	"strings"
	"time"
)

// ExportHistory exports all retained receipts, independent of the current UI
// filter. Only the existing safe persisted fields are included, not raw RPC or
// installer output, credentials, favorites, or local storage paths.
func (s *Store) ExportHistory() []byte {
	var out strings.Builder
	for _, row := range s.Personal().History {
		fmt.Fprintf(&out, "%s %s %s\n%s\n", row.At.UTC().Format(time.RFC3339), row.Action, row.Plugin, row.Result)
		if row.Attempt != "" {
			fmt.Fprintf(&out, "attempt: %s\n", row.Attempt)
		}
		if row.Error != "" {
			fmt.Fprintln(&out, row.Error)
		}
		out.WriteByte('\n')
	}
	return []byte(out.String())
}
