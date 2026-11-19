package harness

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// HostRecord names the Host process one shell started. The shell writes it when
// the child spawns and removes it when that child exits, so a record that is
// still there on the next launch describes a Host whose shell never got to
// clean up: the detached child outlives a crash, a force quit or a killed lane.
type HostRecord struct {
	Version    int       `json:"version"`
	PID        int       `json:"pid"`
	ShellPID   int       `json:"shellPid"`
	Port       int       `json:"port"`
	Image      string    `json:"image"`
	ShellImage string    `json:"shellImage"`
	StartedAt  time.Time `json:"startedAt"`
}

const hostRecordVersion = 1

// WriteHostRecord records the Host this shell just started.
func WriteHostRecord(path string, record HostRecord) error {
	record.Version = hostRecordVersion
	raw, err := json.Marshal(record)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	return writeFileAtomic(path, raw, 0o600)
}

// ReadHostRecord returns the recorded Host, if the file is there and readable.
func ReadHostRecord(path string) (HostRecord, bool) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return HostRecord{}, false
	}
	var record HostRecord
	if json.Unmarshal(raw, &record) != nil || record.Version != hostRecordVersion || record.PID <= 0 {
		return HostRecord{}, false
	}
	return record, true
}

// RemoveHostRecord drops the record once its Host is confirmed gone. It removes
// the file only when the record still names pid, so a record a newer attempt
// wrote is never deleted by the one that just finished.
func RemoveHostRecord(path string, pid int) {
	record, ok := ReadHostRecord(path)
	if !ok {
		_ = os.Remove(path)
		return
	}
	if pid > 0 && record.PID != pid {
		return
	}
	_ = os.Remove(path)
}

// DetectOrphan reports the Host a previous run left running, and clears records
// that can no longer describe a live process. A record whose shell is still
// answering belongs to a running instance and is left alone; a Host whose
// identity cannot be confirmed is left running rather than guessed at.
func DetectOrphan(path string, self int) (HostRecord, bool) {
	record, ok := ReadHostRecord(path)
	if !ok {
		if _, err := os.Stat(path); err == nil {
			_ = os.Remove(path)
		}
		return HostRecord{}, false
	}
	hostImage, known := processImage(record.PID)
	if !known {
		return HostRecord{}, false
	}
	if hostImage == "" || !sameExecutable(hostImage, record.Image) {
		RemoveHostRecord(path, record.PID)
		return HostRecord{}, false
	}
	if record.ShellPID == self {
		return HostRecord{}, false
	}
	if shellImage, shellKnown := processImage(record.ShellPID); shellKnown && sameExecutable(shellImage, record.ShellImage) {
		return HostRecord{}, false
	}
	return record, true
}

// ReapOrphan ends the process tree of a Host no shell owns any more.
func ReapOrphan(record HostRecord, path string) error {
	if err := reapHostProcess(record.PID); err != nil {
		return err
	}
	RemoveHostRecord(path, record.PID)
	return nil
}

func sameExecutable(a, b string) bool {
	if a == "" || b == "" {
		return false
	}
	return strings.EqualFold(filepath.Clean(a), filepath.Clean(b))
}

// reapOrphan ends a Host a previous shell left running, before this attempt
// picks its port and opens the profile that Host still holds.
func (s *Supervisor) reapOrphan(gen int) {
	if s.opts.RecordPath == "" {
		return
	}
	record, ok := DetectOrphan(s.opts.RecordPath, os.Getpid())
	if !ok {
		return
	}
	s.log(fmt.Sprintf("» an orphaned Host from a previous run is still running (pid %d); ending it", record.PID), gen)
	if err := ReapOrphan(record, s.opts.RecordPath); err != nil {
		s.log("orphan: the previous Host did not stop: "+err.Error(), gen)
	}
}

// recordHost names the child this attempt owns, so the next shell can tell an
// orphan from a stranger if this one never gets to stop it.
func (s *Supervisor) recordHost(pid, port int, image string, gen int) {
	if s.opts.RecordPath == "" {
		return
	}
	shellImage, err := os.Executable()
	if err != nil {
		shellImage = ""
	}
	err = WriteHostRecord(s.opts.RecordPath, HostRecord{
		PID:        pid,
		ShellPID:   os.Getpid(),
		Port:       port,
		Image:      image,
		ShellImage: shellImage,
		StartedAt:  time.Now(),
	})
	if err != nil {
		s.log("host record: "+err.Error(), gen)
	}
}
