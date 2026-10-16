package importer

import (
	"os"
	"path/filepath"
)

// AcquireExternalMaintenance shares import's cross-process lock. It does not
// repair journals: unresolved imports must be recovered by the desktop first.
func AcquireExternalMaintenance(home string) (func(), error) {
	if err := directoryPath(home); err != nil {
		return nil, err
	}
	if err := checkLegacyImportJournal(home); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(home, 0700); err != nil {
		return nil, err
	}
	release, err := acquireImportLock(home)
	if err != nil {
		return nil, err
	}
	if _, err = os.Lstat(filepath.Join(home, transactionDirectory)); !os.IsNotExist(err) {
		release()
		if err != nil {
			return nil, err
		}
		return nil, ErrRecoveryBlocked
	}
	return release, nil
}
