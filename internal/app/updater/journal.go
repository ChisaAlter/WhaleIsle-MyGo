package updater

import (
	"log"
	"path/filepath"
	"strings"
	"sync"

	desktopupdate "whaleisle/internal/desktop/update"
)

// JournalDirectory stores update evidence outside the installation directory
// so upgrades do not erase diagnostic milestones.
const JournalDirectory = "update-journal"

type journalBook struct {
	mu      sync.Mutex
	journal *desktopupdate.Journal
	failed  bool
}

func (b *journalBook) record(dir, version, action string, state *desktopupdate.JournalState) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.failed {
		return
	}
	if b.journal == nil {
		journal, err := desktopupdate.NewJournal(filepath.Join(dir, JournalDirectory), version)
		if err != nil {
			b.failed = true
			log.Printf("update journal: %v", err)
			return
		}
		b.journal = journal
	}
	var err error
	if state != nil {
		err = b.journal.State(*state)
	} else if action != "" {
		err = b.journal.Action(action)
	}
	if err != nil {
		b.failed = true
		log.Printf("update journal: %v", err)
	}
}

// RecordAction appends one whitelisted update milestone to the per-run journal.
func (c *Center) RecordAction(action string) {
	if c == nil {
		return
	}
	c.journal.record(c.userDataDir(), c.env().BuildVersion, strings.TrimSpace(action), nil)
}

// RecordState appends one whitelisted update phase transition to the journal.
func (c *Center) RecordState(state desktopupdate.JournalState) {
	if c == nil {
		return
	}
	c.journal.record(c.userDataDir(), c.env().BuildVersion, "", &state)
}

// RecordCheckResult writes the settled check state without exposing raw feed URLs.
func (c *Center) RecordCheckResult(res desktopupdate.Result, err error) {
	switch {
	case err != nil:
		c.RecordState(desktopupdate.JournalState{
			Phase:           desktopupdate.JournalPhaseError,
			FailedOperation: "check",
			Message:         err.Error(),
		})
	case res.Status == desktopupdate.StatusUpdateAvailable:
		c.RecordState(desktopupdate.JournalState{Phase: desktopupdate.JournalPhaseAvailable, Version: res.Version})
	case res.Status == desktopupdate.StatusNotConfigured:
		c.RecordState(desktopupdate.JournalState{Phase: desktopupdate.JournalPhaseNotAvailable})
	default:
		c.RecordState(desktopupdate.JournalState{Phase: desktopupdate.JournalPhaseNotAvailable, Version: res.Version})
	}
}

// JournalFailed reports whether best-effort journal initialization or appending failed.
func (c *Center) JournalFailed() bool {
	if c == nil {
		return false
	}
	c.journal.mu.Lock()
	defer c.journal.mu.Unlock()
	return c.journal.failed
}
