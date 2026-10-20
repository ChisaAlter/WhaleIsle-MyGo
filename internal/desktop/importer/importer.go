package importer

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

	"whaleisle/internal/config"
	"whaleisle/internal/credentials"
	"whaleisle/internal/store/storelocale"
)

// ErrNotDirectory means userData is missing or not a directory.
var ErrNotDirectory = errors.New("importer: not a directory")

// Report is a read-only snapshot of known Electron user-data artifacts.
type Report struct {
	// UserData is the cleaned path that was inspected.
	UserData string
	// DirExists is true when UserData is an existing directory.
	DirExists bool
	// IsDefaultElectronPath is true when UserData equals ElectronUserDataDir().
	IsDefaultElectronPath bool
	ConfigJSON            bool
	CredentialsJSON       bool
	DshHome               bool
	// CategoryFound maps launcher import chips (sessions/skills/plugins/mcp/settings/presets)
	// to whether a corresponding artifact was detected (probe-level, not item counts).
	CategoryFound map[string]bool
}

// ElectronUserDataDir returns the Electron shell's default user data directory
// (%AppData%\Deepseek-Harness-Desktop on Windows). WHALEISLE_USER_DATA is
// ignored so import UI can point at the real Electron install even while tests
// run with an override.
func ElectronUserDataDir() string {
	base, err := os.UserConfigDir()
	if err != nil {
		wd, _ := os.Getwd()
		return filepath.Join(wd, ".whaleisle")
	}
	return filepath.Join(base, config.DataDirName)
}

// ImportReport inspects userData without reading file contents.
func ImportReport(userData string) (Report, error) {
	root := filepath.Clean(strings.TrimSpace(userData))
	if root == "" || root == "." {
		return Report{}, ErrNotDirectory
	}

	report := Report{
		UserData:              root,
		IsDefaultElectronPath: root == ElectronUserDataDir(),
	}

	info, err := os.Stat(root)
	if err != nil {
		if os.IsNotExist(err) {
			return report, nil
		}
		return Report{}, err
	}
	if !info.IsDir() {
		return Report{}, ErrNotDirectory
	}
	report.DirExists = true

	report.ConfigJSON = entryExists(filepath.Join(root, config.FileName), false)
	report.CredentialsJSON = entryExists(filepath.Join(root, credentials.FileName), false)
	report.DshHome = entryExists(filepath.Join(root, "dsh-home"), true)
	dsh := filepath.Join(root, "dsh-home")
	report.CategoryFound = map[string]bool{
		"sessions": report.DshHome && (entryExists(filepath.Join(dsh, "sessions"), true) ||
			entryExists(filepath.Join(dsh, "session"), true)),
		"skills": report.DshHome && entryExists(filepath.Join(dsh, "skills"), true),
		"plugins": report.DshHome && (entryExists(filepath.Join(dsh, "plugins"), true) ||
			entryExists(filepath.Join(dsh, "user-plugins"), true) ||
			entryExists(filepath.Join(dsh, "profiles", "web", "package.json"), false)),
		"mcp": report.DshHome && (entryExists(filepath.Join(dsh, "mcp"), true) ||
			entryExists(filepath.Join(dsh, "mcpServers"), true) ||
			entryExists(filepath.Join(dsh, "mcp-servers.yaml"), false)),
		"settings": report.ConfigJSON || report.CredentialsJSON ||
			(report.DshHome && entryExists(filepath.Join(dsh, "settings.yaml"), false)),
		"presets": report.DshHome && (entryExists(filepath.Join(dsh, "agent-presets"), true) ||
			entryExists(filepath.Join(dsh, ".agent-presets"), true)),
	}

	return report, nil
}

// FormatReport returns a multi-line summary suitable for info dialogs.
func FormatReport(r Report) string {
	var b strings.Builder
	b.WriteString(storelocale.T(storelocale.KeyStoreImporterPath))
	b.WriteString(r.UserData)
	b.WriteString("\n")
	if r.IsDefaultElectronPath {
		b.WriteString(storelocale.T(storelocale.KeyStoreImporterElectronDefaultDataDirectoryN))
	}
	if !r.DirExists {
		b.WriteString(storelocale.T(storelocale.KeyStoreImporterDirectoryMissingN))
	} else {
		b.WriteString(storelocale.T(storelocale.KeyStoreImporterDirectoryExistsN))
	}
	b.WriteString("config.json: ")
	b.WriteString(yesNo(r.ConfigJSON))
	b.WriteString("\ncredentials.json: ")
	b.WriteString(yesNo(r.CredentialsJSON))
	b.WriteString("\ndsh-home: ")
	b.WriteString(yesNo(r.DshHome))
	return b.String()
}

func yesNo(ok bool) string {
	if ok {
		return "有"
	}
	return "无"
}

func entryExists(path string, wantDir bool) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	if wantDir {
		return info.IsDir()
	}
	return !info.IsDir()
}
