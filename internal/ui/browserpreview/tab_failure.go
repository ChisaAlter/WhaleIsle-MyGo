package browserpreview

import (
	"log"

	"whaleisle/internal/errcopy"
)

// fail records why an action stopped, in copy the chrome can paint: the
// caller's localized lead plus the cause and next step. The raw error goes to
// the log; Tab.Error never carries it.
func (t *Tab) fail(lead string, err error) {
	if err != nil {
		log.Printf("browser preview: %v", err)
	}
	t.Error = errcopy.Under(lead, err)
}
