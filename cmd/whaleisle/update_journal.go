package main

import (
	"log"
	"path/filepath"
	"strings"
	"sync"

	desktopupdate "whaleisle/internal/desktop/update"
	"whaleisle/internal/paths"
)

// updateJournalDirectory keeps update evidence outside the installation
// directory, which the installer replaces (port of
// reference/electron/src/main/update-journal.js).
const updateJournalDirectory = "update-journal"

// updateJournalBook owns the per-run journal. Recording is best effort: a
// journal that cannot be created must never block a check, download or install,
// it only leaves the evidence absent (logged once).
type updateJournalBook struct {
	mu      sync.Mutex
	journal *desktopupdate.Journal
	failed  bool
}

// record appends one milestone or state, creating the journal on first use.
func (b *updateJournalBook) record(version, action string, state *desktopupdate.JournalState) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.failed {
		return
	}
	if b.journal == nil {
		dir := filepath.Join(paths.UserDataDir(), updateJournalDirectory)
		journal, err := desktopupdate.NewJournal(dir, version)
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
		// A failed append keeps the journal path but stops repeating the error;
		// the update flow itself is unaffected.
		b.failed = true
		log.Printf("update journal: %v", err)
	}
}

// recordUpdateAction appends one fixed update milestone.
func (s *mainState) recordUpdateAction(action string) {
	if s == nil {
		return
	}
	s.updateJournal.record(buildVersion, strings.TrimSpace(action), nil)
}

// recordUpdateState appends one whitelisted update state.
func (s *mainState) recordUpdateState(state desktopupdate.JournalState) {
	if s == nil {
		return
	}
	s.updateJournal.record(buildVersion, "", &state)
}

// recordUpdateCheckResult keeps the settled check as one state line: available,
// not-available or a classified error, never the raw feed URL.
func (s *mainState) recordUpdateCheckResult(res desktopupdate.Result, err error) {
	switch {
	case err != nil:
		s.recordUpdateState(desktopupdate.JournalState{
			Phase:           desktopupdate.JournalPhaseError,
			FailedOperation: "check",
			Message:         err.Error(),
		})
	case res.Status == desktopupdate.StatusUpdateAvailable:
		s.recordUpdateState(desktopupdate.JournalState{Phase: desktopupdate.JournalPhaseAvailable, Version: res.Version})
	case res.Status == desktopupdate.StatusNotConfigured:
		s.recordUpdateState(desktopupdate.JournalState{Phase: desktopupdate.JournalPhaseNotAvailable})
	default:
		s.recordUpdateState(desktopupdate.JournalState{Phase: desktopupdate.JournalPhaseNotAvailable, Version: res.Version})
	}
}
