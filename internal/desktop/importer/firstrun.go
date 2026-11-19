package importer

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"whaleisle/internal/config"
	"whaleisle/internal/credentials"
)

// This file is the one-time migration into MyGo's own data directory. It
// reuses the import catalog for everything under dsh-home and copies the two
// profile files the catalog does not cover: config.json as published bytes and
// credentials.json through the credentials package, which re-seals the secrets
// with the destination profile's own key instead of copying an envelope the
// destination cannot decrypt.
//
// Nothing here moves or deletes source data, nothing overwrites an existing
// destination file, and running it twice copies only what is still missing.

// FirstRunMarkerName is the record of the offer's outcome inside the
// destination data directory. It is deliberately not one of the artifacts the
// emptiness check looks at, so a skipped offer still leaves an empty profile.
const FirstRunMarkerName = "first-run-import.json"

// FirstRun outcomes recorded in the marker.
const (
	FirstRunDone    = "done"
	FirstRunSkipped = "skipped"
	// FirstRunDismissed marks an offer closed with the window frame rather than
	// answered: the first-run dialog does not return, and the same import stays
	// available from the data-location settings.
	FirstRunDismissed = "dismissed"
	// FirstRunPartial marks an import that stopped before it finished, so the
	// next launch offers to complete it instead of treating the destination as
	// a profile the user already chose.
	FirstRunPartial = "partial"
)

// FirstRunPlan is one migration from the legacy profile to MyGo's directory.
type FirstRunPlan struct {
	Source string
	Dest   string
	// Legacy reports whether Source is the Electron shell's default
	// directory rather than another directory the user pointed at.
	Legacy bool
}

// FirstRunResult reports what a migration copied.
type FirstRunResult struct {
	Copied  int
	Skipped int
	Failed  int
	Message string
	Journal []string
}

// FirstRunMarkerPath is the outcome record inside a data directory.
func FirstRunMarkerPath(dest string) string {
	return filepath.Join(dest, FirstRunMarkerName)
}

type firstRunMarker struct {
	Status  string `json:"status"`
	At      string `json:"at"`
	Copied  int    `json:"copied"`
	Skipped int    `json:"skipped"`
	Failed  int    `json:"failed"`
}

// FirstRunStatus reports the recorded outcome: FirstRunDone, FirstRunSkipped,
// or "" when the offer has not been answered for this directory.
func FirstRunStatus(dest string) string {
	raw, err := os.ReadFile(FirstRunMarkerPath(dest))
	if err != nil {
		return ""
	}
	var marker firstRunMarker
	if err := json.Unmarshal(raw, &marker); err != nil {
		return ""
	}
	switch marker.Status {
	case FirstRunDone, FirstRunSkipped, FirstRunDismissed, FirstRunPartial:
		return marker.Status
	}
	return ""
}

// DetectFirstRun decides whether the legacy profile should be offered. It is
// offered when MyGo's directory holds none of its own artifacts, the legacy
// directory has something worth copying, and the offer has not been answered
// for this directory yet.
func DetectFirstRun(dest, source string) (FirstRunPlan, bool) {
	clean := filepath.Clean(strings.TrimSpace(dest))
	switch FirstRunStatus(clean) {
	case FirstRunDone, FirstRunSkipped, FirstRunDismissed:
		return FirstRunPlan{}, false
	case FirstRunPartial:
		// A stopped import is finished from where it ended; the destination
		// already holds part of the data.
	default:
		if !isEmptyProfile(clean) {
			return FirstRunPlan{}, false
		}
	}
	if !HasLegacyData(source) {
		return FirstRunPlan{}, false
	}
	plan := PlanFirstRun(dest, source)
	if plan.Source == "" || plan.Dest == "" {
		return FirstRunPlan{}, false
	}
	return plan, true
}

// HasLegacyData reports whether source looks like a profile with something to
// copy: the configuration, the secrets, or one of the imported data areas.
func HasLegacyData(source string) bool {
	source = filepath.Clean(strings.TrimSpace(source))
	if source == "" || source == "." {
		return false
	}
	report, err := ImportReport(source)
	if err != nil || !report.DirExists {
		return false
	}
	return report.ConfigJSON || report.CredentialsJSON || anyCategory(report.CategoryFound)
}

// The marker records an answer to the launch prompt, not a refusal of the
// import: 导入 leaves nothing to offer, while 跳过 and 关闭 keep the import as
// an action a person can take later from the data-location block.
func FirstRunOffer(dest, source string) (FirstRunPlan, bool) {
	if FirstRunStatus(filepath.Clean(strings.TrimSpace(dest))) == FirstRunDone {
		return FirstRunPlan{}, false
	}
	if !HasLegacyData(source) {
		return FirstRunPlan{}, false
	}
	plan := PlanFirstRun(dest, source)
	if plan.Source == "" || plan.Dest == "" {
		return FirstRunPlan{}, false
	}
	return plan, true
}

// PlanFirstRun builds the migration plan without deciding whether to offer it.
// Settings uses it to run the same import again after the first offer was
// skipped; the copy keeps whatever the destination already holds. It reads no
// session log: what the plan carries is where the copy starts and ends, and
// the catalog that decides what to copy is built by the run itself.
func PlanFirstRun(dest, source string) FirstRunPlan {
	dest = filepath.Clean(strings.TrimSpace(dest))
	source = filepath.Clean(strings.TrimSpace(source))
	if dest == "" || dest == "." || source == "" || source == "." || strings.EqualFold(dest, source) {
		return FirstRunPlan{}
	}
	return FirstRunPlan{Source: source, Dest: dest, Legacy: source == ElectronUserDataDir()}
}

// isEmptyProfile reports whether dir holds none of the artifacts MyGo or the
// Electron shell create, so writing there cannot overwrite live data.
func isEmptyProfile(dir string) bool {
	if dir == "" {
		return false
	}
	if entryExists(filepath.Join(dir, config.FileName), false) ||
		entryExists(filepath.Join(dir, credentials.FileName), false) {
		return false
	}
	for _, name := range []string{"sessions", "settings.yaml", "skills", "agent-presets", ".agent-presets", "attachments"} {
		if presence(filepath.Join(dir, "dsh-home", name)) {
			return false
		}
	}
	return true
}

func anyCategory(found map[string]bool) bool {
	for _, ok := range found {
		if ok {
			return true
		}
	}
	return false
}

// SkipFirstRun answers the offer with "skip" for this directory, so the next
// launch does not ask again. It never touches the legacy directory.
func SkipFirstRun(dest string) error {
	if strings.TrimSpace(dest) == "" {
		return ErrNotDirectory
	}
	return writeFirstRunMarker(dest, firstRunMarker{Status: FirstRunSkipped, At: time.Now().UTC().Format(time.RFC3339)})
}

// DismissFirstRun records that the offer was closed without an answer. It is
// its own outcome: the first-run dialog does not come back, and Settings offers
// the same import from its data-location block, which reads no marker.
func DismissFirstRun(dest string) error {
	if strings.TrimSpace(dest) == "" {
		return ErrNotDirectory
	}
	return writeFirstRunMarker(dest, firstRunMarker{Status: FirstRunDismissed, At: time.Now().UTC().Format(time.RFC3339)})
}

// RunFirstRun copies the plan into MyGo's directory. It is re-entrant: every
// step keeps an existing destination and reports it as skipped.
func RunFirstRun(ctx context.Context, plan FirstRunPlan) (FirstRunResult, error) {
	var result FirstRunResult
	if plan.Source == "" || plan.Dest == "" {
		return result, ErrNotDirectory
	}
	if err := ValidateRoots(plan.Source, plan.Dest); err != nil {
		return result, err
	}
	if err := ctx.Err(); err != nil {
		return result, err
	}
	if err := os.MkdirAll(plan.Dest, 0o700); err != nil {
		return result, err
	}
	note := func(line string) { result.Journal = append(result.Journal, line) }

	if src := filepath.Join(plan.Source, config.FileName); entryExists(src, false) {
		dst := filepath.Join(plan.Dest, config.FileName)
		if entryExists(dst, false) {
			result.Skipped++
			note("config.json: skipped (already present)")
		} else if err := copyFileAtomic(src, dst); err != nil {
			result.Failed++
			note("config.json: failed: " + err.Error())
		} else {
			result.Copied++
			note("config.json: copied")
		}
	}
	if src := filepath.Join(plan.Source, credentials.FileName); entryExists(src, false) {
		dst := filepath.Join(plan.Dest, credentials.FileName)
		if entryExists(dst, false) {
			result.Skipped++
			note("credentials.json: skipped (already present)")
		} else if err := migrateCredentials(plan.Source, plan.Dest); err != nil {
			// A credentials write right the platform refuses, or an envelope
			// only the original could read, is permanent for this run: the
			// user signs in again instead. Counting it as skipped rather than
			// failed keeps the offer from coming back on every launch for
			// something that cannot succeed.
			result.Skipped++
			note("credentials.json: skipped: " + err.Error())
		} else {
			result.Copied++
			note("credentials.json: copied")
		}
	}
	if entryExists(filepath.Join(plan.Source, "dsh-home"), true) {
		// The catalog is built here, where the copy needs it, and not in the
		// plan: planning runs while a page is being drawn, and reading a
		// session title costs a decode of its log.
		cat, catErr := BuildCatalogContext(ctx, filepath.Join(plan.Source, "dsh-home"), filepath.Join(plan.Dest, "dsh-home"))
		selection := SelectionFromCatalog(cat, cat.HasAttachments, false)
		// A first run has no marketplace session to reinstall plugins with;
		// the plugin page owns that afterwards.
		selection.SelectedPluginIDs = nil
		if catErr == nil && cat.SourceHome != "" && selection.Count() > 0 {
			applied, err := ApplyCatalogWithPlugins(ctx, cat, selection, nil, nil)
			result.Copied += applied.Copied
			result.Skipped += applied.Skipped
			result.Failed += applied.Failed
			result.Journal = append(result.Journal, applied.Journal...)
			if err != nil {
				// Completed items stay and the marker records the partial run,
				// so the next launch offers to finish it instead of treating the
				// destination as a profile the user already chose.
				result.Message = applied.Message
				_ = writeFirstRunMarker(plan.Dest, firstRunMarkerFor(result, err))
				return result, err
			}
		}
	}
	result.Message = fmt.Sprintf("copied %d, skipped %d, failed %d", result.Copied, result.Skipped, result.Failed)
	note(result.Message)
	if err := writeFirstRunMarker(plan.Dest, firstRunMarkerFor(result, nil)); err != nil {
		return result, err
	}
	return result, nil
}

// firstRunMarkerFor decides what the next launch should do with this run. Any
// item that did not land, or a run that stopped, is partial: the offer comes
// back and the copy finishes what is missing. Only a clean run is done.
func firstRunMarkerFor(result FirstRunResult, runErr error) firstRunMarker {
	status := FirstRunDone
	if runErr != nil || result.Failed > 0 {
		status = FirstRunPartial
	}
	return firstRunMarker{
		Status:  status,
		At:      time.Now().UTC().Format(time.RFC3339),
		Copied:  result.Copied,
		Skipped: result.Skipped,
		Failed:  result.Failed,
	}
}

// migrateCredentials reads the source profile's secrets and writes them into
// the destination through the credentials package, so the destination seals
// them with its own key. The source file is only read.
func migrateCredentials(source, dest string) error {
	creds, err := credentials.Load(source)
	if err != nil {
		return err
	}
	return credentials.Save(dest, creds)
}

// copyFileAtomic publishes src at dst through a private temporary file, so a
// failure never leaves a half-written destination file.
func copyFileAtomic(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	dir := filepath.Dir(dst)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, "."+filepath.Base(dst)+".*.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if _, err := io.Copy(tmp, in); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Chmod(tmpName, 0o600); err != nil {
		return err
	}
	return os.Rename(tmpName, dst)
}

func writeFirstRunMarker(dest string, marker firstRunMarker) error {
	raw, err := json.MarshalIndent(marker, "", "  ")
	if err != nil {
		return err
	}
	raw = append(raw, '\n')
	if err := os.MkdirAll(dest, 0o700); err != nil {
		return err
	}
	tmpFile, err := os.CreateTemp(dest, "."+FirstRunMarkerName+".*.tmp")
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
	if err := os.Chmod(tmp, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, FirstRunMarkerPath(dest))
}

// presence reports whether path exists at all, of any type: the emptiness
// check only cares that nothing either app writes is already there.
func presence(path string) bool {
	_, err := os.Lstat(path)
	return err == nil
}
