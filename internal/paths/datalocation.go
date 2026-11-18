package paths

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"whaleisle/internal/config"
)

// This file holds the user's chosen data location. It lives outside the data
// directory it points at: a record inside the directory could not say where
// the directory is.
//
// Precedence for the user data directory, highest first:
//  1. WHALEISLE_USER_DATA (tests and isolated runs)
//  2. the location recorded here
//  3. the directory this build defaults to (DefaultUserDataDir)
//
// The record is read once per process and cached: UserDataDir sits on hot
// paths. A location change is applied by rewriting the record and restarting.

const locationFileName = "data-location.json"

// userConfigDir is a seam so tests can point the record at a temp directory.
var userConfigDir = os.UserConfigDir

// ConfigRoot is MyGo's own configuration root: the directory that holds the
// data-location record. It is deliberately not the data directory.
func ConfigRoot() string {
	base, err := userConfigDir()
	if err != nil {
		wd, _ := os.Getwd()
		return filepath.Join(wd, ".whaleisle-config")
	}
	return filepath.Join(base, config.AppDirName)
}

// LocationFile is the record that names the user's chosen data directory.
func LocationFile() string {
	return filepath.Join(ConfigRoot(), locationFileName)
}

// DefaultUserDataDir is the data directory this build uses when neither
// WHALEISLE_USER_DATA nor a recorded location applies. It sits inside
// ConfigRoot, so the record that names another location never travels with a
// copied data directory.
func DefaultUserDataDir() string {
	base, err := userConfigDir()
	if err != nil {
		return DefaultUserDataDirUnder("")
	}
	return DefaultUserDataDirUnder(base)
}

// DefaultUserDataDirUnder is DefaultUserDataDir for an explicit configuration
// directory, the same seam the isolation checks take as userConfigBase. An
// empty base means the platform's configuration directory could not be read.
func DefaultUserDataDirUnder(base string) string {
	if strings.TrimSpace(base) == "" {
		wd, _ := os.Getwd()
		return filepath.Join(wd, "."+strings.ToLower(config.AppDirName), config.DefaultDataDirName)
	}
	return filepath.Join(base, config.AppDirName, config.DefaultDataDirName)
}

type locationRecord struct {
	DataDir string `json:"dataDir"`
	// Previous is the directory this one replaced, so a migration into a new
	// location can offer the old profile as an import source.
	Previous string `json:"previous,omitempty"`
}

var (
	locationMu     sync.Mutex
	locationLoaded bool
	// locationDir is the recorded directory currently in use ("" when the
	// record is absent or unusable). locationRecorded is what the record says,
	// whether or not it can be used, and locationPrevious is the location it
	// replaced; settings shows both to explain a fallback and to offer the old
	// profile as an import source.
	locationDir      string
	locationRecorded string
	locationPrevious string
	locationBroken   bool
	// locationProbed caches the one write probe of the recorded directory
	// that the settings row asks for; the render thread never repeats it.
	locationProbed      bool
	locationNotWritable bool
)

// StoredDataDir returns the data directory recorded by the user, or "" when
// none is recorded or the record cannot be used. An unreadable record, or one
// naming a directory that is not there, falls back to the default with the
// reason logged. The record itself is left alone, so restoring the directory
// restores the location on the next launch.
func StoredDataDir() string {
	locationMu.Lock()
	defer locationMu.Unlock()
	loadLocationLocked()
	return locationDir
}

// RecordedDataDir returns what the record said when this process read it,
// whether or not that directory can be used. Settings explains a fallback from
// it instead of presenting the default directory as if nothing were recorded.
//
// It is cached like UserDataDir: RecordDataDir writes the record for the next
// launch without moving this process, so this keeps returning the location in
// use until the process restarts.
func RecordedDataDir() string {
	locationMu.Lock()
	defer locationMu.Unlock()
	loadLocationLocked()
	return locationRecorded
}

// PreviousDataDir returns the location this one replaced, or "" when the
// record carries none. Settings offers it as an import source after a change.
func PreviousDataDir() string {
	locationMu.Lock()
	defer locationMu.Unlock()
	loadLocationLocked()
	return locationPrevious
}

// loadLocationLocked reads the record once per process. UserDataDir sits on
// hot paths, so the file is never re-read per call.
func loadLocationLocked() {
	if locationLoaded {
		return
	}
	locationLoaded = true
	record, err := parseLocationRecord()
	if err != nil {
		locationBroken = true
		log.Printf("data location: ignoring %s: %v; using %s", LocationFile(), err, DefaultUserDataDir())
		return
	}
	if record.DataDir == "" {
		return
	}
	locationRecorded = record.DataDir
	locationPrevious = record.Previous
	// The stat blocks on a dead network path exactly as the write probe below
	// does. Both run once, here, and nothing else in this package blocks on
	// the recorded directory again.
	if info, statErr := os.Stat(record.DataDir); statErr != nil || !info.IsDir() {
		log.Printf("data location: recorded directory %s is not usable (%v); using %s", record.DataDir, statErr, DefaultUserDataDir())
		return
	}
	locationDir = record.DataDir
}

// DataLocationIssue is a data-location record that is not in use for a reason
// the user has to see: the record cannot be read, the directory it names is
// gone, or that directory cannot be written. All three mean the process is
// writing somewhere the user did not choose.
type DataLocationIssue struct {
	// Path is the recorded directory, or the record file when the record
	// itself cannot be parsed.
	Path string
	// Unreadable is true when the record document could not be used, so Path
	// names the file rather than a directory.
	Unreadable bool
	// NotWritable is true when the recorded directory exists but rejects a
	// write, so a profile cannot live there any more.
	NotWritable bool
}

// RecordedDataLocationIssue reports a record that is not usable. It reads the
// file and probes the directory as they are now, so startup sees a record or a
// volume changed since the cache loaded. The probe writes one temporary file
// into the recorded directory; a fresh profile is created there anyway.
func RecordedDataLocationIssue() (DataLocationIssue, bool) {
	record, err := parseLocationRecord()
	if err != nil {
		return DataLocationIssue{Path: LocationFile(), Unreadable: true}, true
	}
	dir := record.DataDir
	if dir == "" {
		return DataLocationIssue{}, false
	}
	if info, statErr := os.Stat(dir); statErr != nil || !info.IsDir() {
		return DataLocationIssue{Path: dir}, true
	}
	if err := probeDir(dir); err != nil {
		log.Printf("data location: recorded directory %s rejected a write (%v); using %s", dir, err, DefaultUserDataDir())
		return DataLocationIssue{Path: dir, NotWritable: true}, true
	}
	return DataLocationIssue{}, false
}

// CachedDataLocationIssue is RecordedDataLocationIssue from the process cache,
// for frames that must not re-read the record. The write probe runs once per
// process, on the first frame that asks.
func CachedDataLocationIssue() (DataLocationIssue, bool) {
	locationMu.Lock()
	defer locationMu.Unlock()
	loadLocationLocked()
	if locationBroken {
		return DataLocationIssue{Path: LocationFile(), Unreadable: true}, true
	}
	if locationRecorded == "" {
		return DataLocationIssue{}, false
	}
	if locationDir == "" {
		return DataLocationIssue{Path: locationRecorded}, true
	}
	if !locationProbed {
		locationProbed = true
		locationNotWritable = probeDir(locationRecorded) != nil
	}
	if locationNotWritable {
		return DataLocationIssue{Path: locationRecorded, NotWritable: true}, true
	}
	return DataLocationIssue{}, false
}

// parseLocationRecord reads and validates the record's shape: an empty record
// for a missing file, an error for a document this package cannot use.
func parseLocationRecord() (locationRecord, error) {
	raw, err := os.ReadFile(LocationFile())
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return locationRecord{}, nil
		}
		return locationRecord{}, err
	}
	var rec locationRecord
	if err := json.Unmarshal(raw, &rec); err != nil {
		return locationRecord{}, fmt.Errorf("not a data-location record: %w", err)
	}
	dir := strings.TrimSpace(rec.DataDir)
	if dir == "" {
		return locationRecord{}, errors.New("the record names no directory")
	}
	if !filepath.IsAbs(dir) {
		return locationRecord{}, fmt.Errorf("the record names a relative directory %q", dir)
	}
	rec.DataDir = filepath.Clean(dir)
	// The previous location is informational: an unusable one only means the
	// import offer does not appear.
	if previous := strings.TrimSpace(rec.Previous); filepath.IsAbs(previous) && filepath.Clean(previous) != rec.DataDir {
		rec.Previous = filepath.Clean(previous)
	} else {
		rec.Previous = ""
	}
	return rec, nil
}

// SetStoredDataDir records dir as the user data directory and drops the cached
// record, so this process sees the new location immediately. Tests use it; the
// running application uses RecordDataDir instead, because it keeps using the
// directory it opened with until it relaunches.
func SetStoredDataDir(dir string) error {
	if err := RecordDataDir(dir, ""); err != nil {
		return err
	}
	InvalidateDataLocation()
	return nil
}

// RecordDataDir writes dir as the user data directory for the next launch
// without changing what this process already resolves. A location change takes
// effect on relaunch, so the cache must keep naming the directory in use.
// previous names the directory being replaced, for the migration offer.
func RecordDataDir(dir, previous string) error {
	abs, err := filepath.Abs(strings.TrimSpace(dir))
	if err != nil {
		return err
	}
	abs = filepath.Clean(abs)
	// This package owns the invariant, not its callers: a record inside the
	// directory it names cannot say where that directory is.
	if root, ok := canonicalOrEmpty(ConfigRoot()); ok && insideOrSame(abs, root) {
		return fmt.Errorf("data location %q would contain the record in %s", abs, root)
	}
	// The record must never name a directory that is not there: the next
	// launch would fall back to the default instead of opening this one.
	if err := os.MkdirAll(abs, 0o755); err != nil {
		return err
	}
	rec := locationRecord{DataDir: abs}
	if old := strings.TrimSpace(previous); old != "" {
		if oldAbs, err := filepath.Abs(old); err == nil {
			oldAbs = filepath.Clean(oldAbs)
			if !strings.EqualFold(oldAbs, abs) {
				rec.Previous = oldAbs
			}
		}
	}
	if err := os.MkdirAll(ConfigRoot(), 0o755); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(rec, "", "  ")
	if err != nil {
		return err
	}
	raw = append(raw, '\n')
	tmpFile, err := os.CreateTemp(ConfigRoot(), "."+locationFileName+".*.tmp")
	if err != nil {
		return err
	}
	tmp := tmpFile.Name()
	defer os.Remove(tmp)
	if _, err := tmpFile.Write(raw); err != nil {
		tmpFile.Close()
		return err
	}
	if err := tmpFile.Sync(); err != nil {
		tmpFile.Close()
		return err
	}
	if err := tmpFile.Close(); err != nil {
		return err
	}
	return os.Rename(tmp, LocationFile())
}

// ClearStoredDataDir removes the recorded location and drops the cache, so
// this process returns to the default immediately. Tests use it; the running
// application uses ClearRecordedDataDir.
func ClearStoredDataDir() error {
	if err := ClearRecordedDataDir(); err != nil {
		return err
	}
	InvalidateDataLocation()
	return nil
}

// ClearRecordedDataDir removes the recorded location without changing what
// this process already resolves; the default applies on the next launch.
func ClearRecordedDataDir() error {
	err := os.Remove(LocationFile())
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// InvalidateDataLocation drops the cached record so the next read sees the
// file as it is now. Tests call it after changing the seam.
func InvalidateDataLocation() {
	locationMu.Lock()
	locationLoaded = false
	locationDir = ""
	locationRecorded = ""
	locationPrevious = ""
	locationBroken = false
	locationProbed = false
	locationNotWritable = false
	locationMu.Unlock()
}
