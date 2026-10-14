package importer

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestImportReportEmpty(t *testing.T) {
	_, err := ImportReport("")
	if err != ErrNotDirectory {
		t.Fatalf("err = %v, want ErrNotDirectory", err)
	}
}

func TestImportReportMissingDir(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "no-such-user-data")
	rep, err := ImportReport(dir)
	if err != nil {
		t.Fatal(err)
	}
	if rep.DirExists {
		t.Fatal("DirExists on missing path")
	}
	if rep.ConfigJSON || rep.CredentialsJSON || rep.DshHome {
		t.Fatal("expected all artifacts false")
	}
}

func TestImportReportFullElectronLayout(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "config.json"), `{}`)
	writeFile(t, filepath.Join(root, "credentials.json"), `{}`)
	if err := os.MkdirAll(filepath.Join(root, "dsh-home"), 0o755); err != nil {
		t.Fatal(err)
	}

	rep, err := ImportReport(root)
	if err != nil {
		t.Fatal(err)
	}
	if !rep.DirExists {
		t.Fatal("DirExists")
	}
	if !rep.ConfigJSON || !rep.CredentialsJSON || !rep.DshHome {
		t.Fatalf("report = %+v", rep)
	}
}

func TestImportReportPartialAndFileVsDir(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "config.json"), `{}`)
	// credentials missing; dsh-home as file should not count as DshHome
	writeFile(t, filepath.Join(root, "dsh-home"), "not-a-dir")

	rep, err := ImportReport(root)
	if err != nil {
		t.Fatal(err)
	}
	if !rep.ConfigJSON {
		t.Fatal("ConfigJSON")
	}
	if rep.CredentialsJSON {
		t.Fatal("CredentialsJSON should be false")
	}
	if rep.DshHome {
		t.Fatal("dsh-home file must not set DshHome")
	}
}

func TestImportReportNotDirectory(t *testing.T) {
	f := filepath.Join(t.TempDir(), "file.txt")
	writeFile(t, f, "x")
	_, err := ImportReport(f)
	if err != ErrNotDirectory {
		t.Fatalf("err = %v", err)
	}
}

func TestFormatReport(t *testing.T) {
	text := FormatReport(Report{
		UserData:              "/data",
		DirExists:             true,
		IsDefaultElectronPath: true,
		ConfigJSON:            true,
	})
	if !strings.Contains(text, "config.json: 有") || !strings.Contains(text, "Electron 默认") {
		t.Fatalf("FormatReport = %q", text)
	}
}

func TestElectronUserDataDirMatchesPathsDefault(t *testing.T) {
	appData := t.TempDir()
	t.Setenv("AppData", appData)
	t.Setenv("WHALEISLE_USER_DATA", filepath.Join(appData, "override"))

	got := ElectronUserDataDir()
	want := filepath.Join(appData, "Deepseek-Harness-Desktop")
	if got != want {
		t.Fatalf("ElectronUserDataDir() = %q, want %q (override ignored)", got, want)
	}

	rep, err := ImportReport(got)
	if err != nil {
		t.Fatal(err)
	}
	if !rep.IsDefaultElectronPath {
		t.Fatal("IsDefaultElectronPath")
	}
}

func writeFile(t *testing.T, path, body string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}
