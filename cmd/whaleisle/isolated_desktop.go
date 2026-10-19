package main

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/egoist/mygo"

	"whaleisle/internal/config"
	"whaleisle/internal/desktop/importer"
	"whaleisle/internal/paths"
)

// An isolated desktop uses an explicit data directory and cannot take over the
// installed application's protocol, login registration or single-instance lock.
// It is opt-in so normal installed startup keeps its existing behavior.
func isolatedDesktopRun() bool {
	return os.Getenv("WHALEISLE_ISOLATED") == "1" || portableDesktopRoot != ""
}

// A portable double-click opens a visible desktop. Only the explicit QA mode
// requests a hidden window; both modes retain the private integration guards.
func isolatedDesktopHidden() bool { return os.Getenv("WHALEISLE_ISOLATED") == "1" }

func prepareIsolatedDesktop() error {
	if !isolatedDesktopRun() {
		return nil
	}
	data := strings.TrimSpace(os.Getenv(paths.EnvUserData))
	if data == "" || !filepath.IsAbs(data) {
		return fmt.Errorf("WHALEISLE_ISOLATED requires an absolute %s directory", paths.EnvUserData)
	}
	data = filepath.Clean(data)
	base, err := os.UserConfigDir()
	if err != nil {
		return err
	}
	if strings.EqualFold(data, filepath.Join(base, config.DataDirName)) {
		return fmt.Errorf("WHALEISLE_ISOLATED cannot use the installed application's data directory")
	}
	sum := sha256.Sum256([]byte(strings.ToLower(data)))
	mygo.App.SetName(fmt.Sprintf("Whale Isle Isolated %x", sum[:6]))
	mygo.App.SetPath(mygo.PathUserData, filepath.Join(data, "native"))
	mygo.App.SetPath(mygo.PathCache, filepath.Join(data, "native", "cache"))
	mygo.App.SetPath(mygo.PathLogs, filepath.Join(data, "native", "logs"))
	return nil
}

func desktopSystemIntegration(apply func() error) error {
	if isolatedDesktopRun() {
		return nil
	}
	return apply()
}

func desktopComponentRoots() []string {
	if isolatedDesktopRun() {
		return []string{paths.UserDataDir()}
	}
	return []string{paths.UserDataDir(), importer.ElectronUserDataDir()}
}
