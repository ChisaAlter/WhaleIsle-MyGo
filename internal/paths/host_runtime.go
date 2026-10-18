package paths

// 功能优先；全部功能完成前不运行测试、不做独立审查、不新增防御性扩展

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

// HostRuntimeSelection names one complete offline runtime installed by the
// component manager. Vendor and Node always come from the same directory.
type HostRuntimeSelection struct {
	Version   string `json:"version"`
	Directory string `json:"directory"`
	Previous  string `json:"previous,omitempty"`
}

func HostComponentDir(userData string) string {
	return filepath.Join(userData, "components", "whaleisle-host")
}

func ReadHostRuntimeSelection(userData string) (HostRuntimeSelection, error) {
	var selection HostRuntimeSelection
	data, err := os.ReadFile(filepath.Join(HostComponentDir(userData), "installed.json"))
	if err != nil {
		return selection, err
	}
	if err := json.Unmarshal(data, &selection); err != nil {
		return selection, err
	}
	if selection.Version == "" || selection.Directory == "" ||
		selection.Directory == "." || selection.Directory == ".." ||
		filepath.Base(selection.Directory) != selection.Directory {
		return selection, fmt.Errorf("Host runtime selection is invalid")
	}
	return selection, nil
}

// SelectedHostRuntime keeps an explicit external vendor ahead of managed
// installs. A selected but missing runtime is returned so launch reports the
// missing installation instead of silently switching to a different version.
func SelectedHostRuntime() string {
	if runtime.GOOS != "darwin" || os.Getenv(EnvVendor) != "" {
		return ""
	}
	selection, err := ReadHostRuntimeSelection(UserDataDir())
	if err != nil {
		return ""
	}
	return filepath.Join(HostComponentDir(UserDataDir()), "versions", selection.Directory, "runtime")
}
