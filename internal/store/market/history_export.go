package market

import (
	"fmt"
	"strings"
	"time"
)

// ExportHistory exports all retained receipts, independent of the current UI
// filter. Diagnostics are bounded and redacted on storage and again on export.
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
		if row.Diagnostic != "" {
			fmt.Fprintln(&out, RedactOperationLog(row.Diagnostic, 2000))
		}
		if row.Log != "" {
			fmt.Fprintln(&out, RedactOperationLog(row.Log, 16000))
		}
		out.WriteByte('\n')
	}
	return []byte(out.String())
}
