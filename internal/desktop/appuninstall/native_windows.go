//go:build windows

package appuninstall

import (
	"golang.org/x/sys/windows/registry"
	"os"
	"os/exec"
)

func DetectCurrent() (Plan, error) {
	current, err := os.Executable()
	if err != nil {
		return Plan{}, err
	}
	key, err := registry.OpenKey(registry.CURRENT_USER, RegistryKey, registry.QUERY_VALUE)
	if err != nil {
		return Plan{}, ErrNotNative
	}
	defer key.Close()
	read := func(name string) string {
		value, typ, err := key.GetStringValue(name)
		if err != nil || typ != registry.SZ {
			return ""
		}
		return value
	}
	p, err := Resolve(current, Record{read("DisplayName"), read("NativeRuntime"), read("InstallLocation"), read("UninstallString")})
	if err != nil {
		return Plan{}, err
	}
	return p, p.Validate()
}

func Start(p Plan) error {
	// Re-read registration and identity after the confirmation/task-drain delay.
	fresh, err := DetectCurrent()
	if err != nil {
		return err
	}
	if fresh != p {
		return ErrNotNative
	}
	return p.Launch(func(path string) error {
		cmd := exec.Command(path)
		if err := cmd.Start(); err != nil {
			return err
		}
		go func() { _ = cmd.Wait() }()
		return nil
	})
}
