package update

// Port of WhaleIsle@90465b67 src/main/update-journal.js. The update evidence
// journal keeps JSONL milestones outside the installation directory so update
// decisions correlate across restarts. Only whitelisted fields reach disk: no
// raw diagnostics, free text or download URLs are persisted.

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// JournalSchemaVersion is the schema of every journal line.
const JournalSchemaVersion = 1

// Update phases the journal retains (electron-updater state names).
const (
	JournalPhaseChecking     = "checking"
	JournalPhaseAvailable    = "available"
	JournalPhaseNotAvailable = "not-available"
	JournalPhaseDownloading  = "downloading"
	JournalPhaseDownloaded   = "downloaded"
	JournalPhaseError        = "error"
)

// Journal actions are the fixed milestones the application may append.
const (
	JournalActionStarted           = "started"
	JournalActionWorkspaceReady    = "workspace-ready"
	JournalActionWorkspaceFailed   = "workspace-failed"
	JournalActionCheckRequested    = "check-requested"
	JournalActionDownloadRequested = "download-requested"
	JournalActionInstallConfirmed  = "install-confirmed"
	JournalActionQuitRequested     = "quit-requested"
)

// journalActions is the closed set of accepted milestone names.
var journalActions = map[string]bool{
	JournalActionStarted:           true,
	JournalActionWorkspaceReady:    true,
	JournalActionWorkspaceFailed:   true,
	JournalActionCheckRequested:    true,
	JournalActionDownloadRequested: true,
	JournalActionInstallConfirmed:  true,
	JournalActionQuitRequested:     true,
}

// journalErrorCodes is the closed set of error classes kept on disk. A message
// without any of these codes is recorded as unclassified.
var journalErrorCodes = []string{
	"ETIMEDOUT",
	"ENOSPC",
	"ERR_INTERNET_DISCONNECTED",
	"ERR_CONNECTION_RESET",
	"ERR_CONNECTION_CLOSED",
	"ERR_NAME_NOT_RESOLVED",
	"ERR_UPDATER_INVALID_SIGNATURE",
	"ERR_UPDATER_CHECKSUM_MISMATCH",
}

// JournalErrorUnclassified marks an error whose class is not in the whitelist.
const JournalErrorUnclassified = "UNCLASSIFIED"

// JournalState is one update state. Percent is only retained while the phase is
// downloading; FailedOperation and Message only while it is an error.
type JournalState struct {
	Phase           string
	Version         string
	Percent         *float64
	FailedOperation string
	Message         string
}

// Journal is a process-owned JSONL evidence file; each append flushes before
// returning, so a crash cannot hide the milestone already reached.
type Journal struct {
	path     string
	version  string
	sequence int

	mu            sync.Mutex
	previousState string
}

// journalRecord is the on-disk line. Field order matches the original journal.
type journalRecord struct {
	SchemaVersion   int    `json:"schemaVersion"`
	Sequence        int    `json:"sequence"`
	Time            string `json:"time"`
	PID             int    `json:"pid"`
	Version         string `json:"version"`
	Event           string `json:"event"`
	Phase           string `json:"phase,omitempty"`
	TargetVersion   string `json:"targetVersion,omitempty"`
	Percent         *int   `json:"percent,omitempty"`
	FailedOperation string `json:"failedOperation,omitempty"`
	ErrorCode       string `json:"errorCode,omitempty"`
}

// NewJournal creates the journal in directory. directory must be absolute so
// the evidence never lands in a working directory that can be replaced by the
// update itself.
func NewJournal(directory, version string) (*Journal, error) {
	directory = filepath.Clean(directory)
	if directory == "" || !filepath.IsAbs(directory) {
		return nil, errors.New("update journal: directory must be absolute")
	}
	if err := os.MkdirAll(directory, 0o755); err != nil {
		return nil, err
	}
	var raw [8]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return nil, err
	}
	path := filepath.Join(directory, fmt.Sprintf("%d-%s.jsonl", time.Now().UnixMilli(), hex.EncodeToString(raw[:])))
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return nil, err
	}
	if err := f.Close(); err != nil {
		return nil, err
	}
	j := &Journal{path: path, version: version}
	if err := j.Action(JournalActionStarted); err != nil {
		return nil, err
	}
	return j, nil
}

// Path is the evidence file this process appends to.
func (j *Journal) Path() string {
	if j == nil {
		return ""
	}
	return j.path
}

// Action appends one fixed milestone. Unknown actions are rejected so a caller
// cannot turn the evidence file into free text.
func (j *Journal) Action(action string) error {
	if j == nil {
		return errors.New("update journal: nil journal")
	}
	if !journalActions[action] {
		return fmt.Errorf("update journal: unknown action %q", action)
	}
	return j.append(journalRecord{Event: action})
}

// State retains one state change. A state that encodes to the previous line is
// skipped, so integer download progress is recorded at most once per percent.
func (j *Journal) State(state JournalState) error {
	if j == nil {
		return errors.New("update journal: nil journal")
	}
	fields := journalStateFields(state)
	encoded, err := json.Marshal(fields)
	if err != nil {
		return err
	}
	j.mu.Lock()
	if string(encoded) == j.previousState {
		j.mu.Unlock()
		return nil
	}
	j.mu.Unlock()
	record := journalRecord{Event: "state"}
	record.Phase = fields.Phase
	record.TargetVersion = fields.TargetVersion
	record.Percent = fields.Percent
	record.FailedOperation = fields.FailedOperation
	record.ErrorCode = fields.ErrorCode
	if err := j.append(record); err != nil {
		return err
	}
	j.mu.Lock()
	j.previousState = string(encoded)
	j.mu.Unlock()
	return nil
}

// journalStateFields is the whitelist one state is reduced to.
func journalStateFields(state JournalState) journalRecord {
	out := journalRecord{Phase: state.Phase}
	if state.Version != "" {
		out.TargetVersion = state.Version
	}
	if state.Phase == JournalPhaseDownloading && state.Percent != nil {
		percent := int(*state.Percent)
		out.Percent = &percent
	}
	if state.Phase == JournalPhaseError {
		out.FailedOperation = state.FailedOperation
		out.ErrorCode = JournalErrorUnclassified
		for _, code := range journalErrorCodes {
			if strings.Contains(state.Message, code) {
				out.ErrorCode = code
				break
			}
		}
	}
	return out
}

func (j *Journal) append(fields journalRecord) error {
	j.mu.Lock()
	record := fields
	record.SchemaVersion = JournalSchemaVersion
	record.Sequence = j.sequence
	record.Time = time.Now().UTC().Format(time.RFC3339Nano)
	record.PID = os.Getpid()
	record.Version = j.version
	line, err := json.Marshal(record)
	j.mu.Unlock()
	if err != nil {
		return err
	}
	line = append(line, '\n')
	f, err := os.OpenFile(j.path, os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return err
	}
	if _, err := f.Write(line); err != nil {
		f.Close()
		return err
	}
	if err := f.Sync(); err != nil {
		f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	j.mu.Lock()
	j.sequence++
	j.mu.Unlock()
	return nil
}
