package importer

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

func readRecoveryFile(path string, max int64) ([]byte, error) {
	if err := privatePath(path); err != nil {
		return nil, err
	}
	f, err := os.Open(path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, max+1))
	if int64(len(data)) > max {
		return nil, fmt.Errorf("recovery journal exceeds size limit: %s", path)
	}
	return data, err
}

// An Electron import can be interrupted before a user changes desktop shells.
// Never start the native Host over its unresolved transaction, or interpret
// Electron payload-side journals as native instructions. Its original journal
// and all copies remain untouched for the original recovery flow/manual repair.
func checkLegacyImportJournal(home string) error {
	path := filepath.Join(filepath.Dir(home), "import-journal.json")
	raw, err := readRecoveryFile(path, 8<<20)
	if err != nil {
		return fmt.Errorf("%w: %s: %v", ErrRecoveryBlocked, path, err)
	}
	if raw == nil {
		return nil
	}
	var record map[string]json.RawMessage
	if json.Unmarshal(raw, &record) != nil || record == nil {
		return fmt.Errorf("%w: invalid legacy journal %s", ErrRecoveryBlocked, path)
	}
	var phase, dest string
	if json.Unmarshal(record["phase"], &phase) != nil || json.Unmarshal(record["destHome"], &dest) != nil || strings.TrimSpace(dest) == "" {
		return fmt.Errorf("%w: invalid legacy journal %s", ErrRecoveryBlocked, path)
	}
	switch phase {
	case "copying", "blocked", "recovered", "done":
	default:
		return fmt.Errorf("%w: invalid legacy journal %s", ErrRecoveryBlocked, path)
	}
	if ids, exists := record["txnIds"]; exists {
		var values []string
		if json.Unmarshal(ids, &values) != nil || values == nil {
			return fmt.Errorf("%w: invalid legacy inventory %s", ErrRecoveryBlocked, path)
		}
		for _, id := range values {
			if id == "" {
				return fmt.Errorf("%w: invalid legacy inventory %s", ErrRecoveryBlocked, path)
			}
		}
	}
	if destinations, exists := record["txnDests"]; exists {
		var values map[string]string
		if json.Unmarshal(destinations, &values) != nil || values == nil {
			return fmt.Errorf("%w: invalid legacy inventory %s", ErrRecoveryBlocked, path)
		}
		for _, value := range values {
			if value == "" {
				return fmt.Errorf("%w: invalid legacy inventory %s", ErrRecoveryBlocked, path)
			}
		}
	}
	if phase == "done" || phase == "recovered" {
		return nil
	}
	left, leftErr := filepath.Abs(home)
	right, rightErr := filepath.Abs(dest)
	if leftErr != nil || rightErr != nil {
		return fmt.Errorf("%w: invalid legacy destination %s", ErrRecoveryBlocked, path)
	}
	same := left == right
	if runtime.GOOS == "windows" {
		same = strings.EqualFold(left, right)
	}
	if !same {
		return nil
	}
	return fmt.Errorf("%w: Electron import is unfinished at %s; open the original desktop recovery flow or repair the preserved transaction before starting", ErrRecoveryBlocked, path)
}

// CheckImportReady is read-only apart from taking the existing writer lock.
// Restart must not run recovery while a Host may still be writing its data.
func CheckImportReady(home string) error {
	if err := directoryPath(home); err != nil {
		return err
	}
	if err := checkLegacyImportJournal(home); err != nil {
		return err
	}
	if _, err := os.Lstat(filepath.Join(home, transactionDirectory)); err == nil {
		return fmt.Errorf("%w: restart after stopping the Host to recover %s", ErrRecoveryBlocked, filepath.Join(home, transactionDirectory))
	} else if !os.IsNotExist(err) {
		return err
	}
	if _, err := os.Lstat(filepath.Join(home, ".mygo-import.lock")); os.IsNotExist(err) {
		return nil
	} else if err != nil {
		return err
	}
	release, err := acquireImportLock(home)
	if err != nil {
		return err
	}
	release()
	return nil
}
