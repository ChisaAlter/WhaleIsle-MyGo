package update

import (
	"fmt"
	"strings"
	"whaleisle/internal/store/storelocale"
)

// NativeReleasePrompt summarizes the already fetched, prepared Versions catalog.
// It performs no network, download or installation operation.
func NativeReleasePrompt(current string, release NativeRelease, rows []NativeRelease, chinese bool) string {
	text := func(zh, en string) string {
		if chinese {
			return zh
		}
		return en
	}
	lines := []string{fmt.Sprintf(text(storelocale.T(storelocale.KeyStoreUpdateCurrentVersionSNavailableVersionS), "Current version: %s\nAvailable version: %s"), current, release.Version)}
	methods := []string{}
	seen := map[string]bool{}
	for _, row := range rows {
		if row.Tag != release.Tag || row.Platform != release.Platform || row.Arch != release.Arch || row.UnavailableReason != "" {
			continue
		}
		kind := text(storelocale.T(storelocale.KeyStoreUpdateFull), "Full")
		size := row.Size
		if row.IsDelta() {
			if row.Delta == nil {
				continue
			}
			kind = fmt.Sprintf(text(storelocale.T(storelocale.KeyStoreUpdateDeltaBasedOnS), "Delta (from %s)"), row.Delta.BaseVersion)
			size = row.Delta.DownloadSize
		}
		variant := text(storelocale.T(storelocale.KeyStoreUpdateDesktopOnly), "desktop only")
		if row.RuntimeIncluded {
			variant = text(storelocale.T(storelocale.KeyStoreUpdateIncludesRuntime), "includes runtime")
		}
		method := kind + " · " + variant
		if size > 0 {
			method += fmt.Sprintf(" · %.1f MiB", float64(size)/(1<<20))
		}
		if !seen[method] {
			seen[method] = true
			methods = append(methods, method)
		}
	}
	if len(methods) == 0 {
		methods = append(methods, text(storelocale.T(storelocale.KeyStoreUpdateCheckAvailableInstallersOnTheVersionsP), "Review available packages on the Versions page"))
	}
	lines = append(lines, text(storelocale.T(storelocale.KeyStoreUpdateAvailableDownloadModesN), "Available downloads:\n")+strings.Join(methods, "\n"))
	notes := strings.TrimSpace(release.Notes)
	if notes == "" {
		notes = text(storelocale.T(storelocale.KeyStoreUpdateThisReleaseHasNoNotes), "No release notes were provided.")
	}
	if chars := []rune(notes); len(chars) > 3000 {
		notes = string(chars[:3000]) + text(storelocale.T(storelocale.KeyStoreUpdateNSeeTheVersionsPageForTheFullNotes), "\n…Full notes are on the Versions page.")
	}
	lines = append(lines, text(storelocale.T(storelocale.KeyStoreUpdateReleaseNotesN), "Release notes:\n")+notes)
	lines = append(lines, text(storelocale.T(storelocale.KeyStoreUpdateContinueOpensTheVersionsPageWhereYouC), "Continue to Versions to choose a full or available delta package. Nothing is downloaded or installed now. After verification, the install button confirms installation."))
	return strings.Join(lines, "\n\n")
}
