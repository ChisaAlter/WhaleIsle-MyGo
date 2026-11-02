//go:build windows

package notifications

import (
	"fmt"
	"os"
	"strings"

	"golang.org/x/sys/windows/registry"
)

// UnregisterOwnedIntegration preflights both objects before changing anything.
// It removes only this installation's exact COM command and uncustomized Go
// shortcut. Notification history, Electron identities, other installs and any
// user changes to registration values or links remain untouched.
func (s Shortcut) UnregisterOwnedIntegration() error {
	return s.unregisterOwnedIntegration(registry.CURRENT_USER, `Software\Classes\CLSID\`+ToastActivatorCLSID)
}
func (s Shortcut) unregisterOwnedIntegration(root registry.Key, clsidPath string) error {
	if s.Target == "" {
		return nil
	}
	serverPath := clsidPath + `\LocalServer32`
	expected := `"` + s.Target + `" ` + ToastServerFlag
	removeServer := false
	key, err := registry.OpenKey(root, serverPath, registry.QUERY_VALUE|registry.ENUMERATE_SUB_KEYS)
	if err == nil {
		command, _, e := key.GetStringValue("")
		values, ve := key.ReadValueNames(-1)
		subkeys, se := key.ReadSubKeyNames(-1)
		key.Close()
		if e != nil {
			return e
		}
		if ve != nil {
			return ve
		}
		if se != nil {
			return se
		}
		removeServer = strings.EqualFold(command, expected) && len(values) == 1 && values[0] == "" && len(subkeys) == 0
	} else if err != registry.ErrNotExist {
		return err
	}
	removeLink := false
	var before os.FileInfo
	if s.Path() != "" {
		before, err = os.Lstat(s.Path())
		if err != nil && !os.IsNotExist(err) {
			return err
		}
		if err == nil && before.Mode().IsRegular() {
			d, err := readShortcut(s.Path())
			if err != nil {
				return err
			}
			removeLink = d.AppID == AppUserModelID && sameShortcutTarget(d.Target, s.Target) && d.Args == "" && d.IconIndex == 0 && sameShortcutTarget(d.Icon, d.Target) && d.HasActivator && d.Activator == toastActivatorGUID
		}
	}
	// Recheck every planned mutation before the first delete. The parent CLSID
	// is deleted only when it has no values and no remaining subkeys.
	if removeLink {
		after, err := os.Lstat(s.Path())
		if err != nil {
			return err
		}
		if !os.SameFile(before, after) || before.Size() != after.Size() || !before.ModTime().Equal(after.ModTime()) {
			return fmt.Errorf("notifications: owned shortcut changed before removal")
		}
	}
	if removeServer {
		again, e := registry.OpenKey(root, serverPath, registry.QUERY_VALUE|registry.ENUMERATE_SUB_KEYS)
		if e != nil {
			return e
		}
		command, _, e := again.GetStringValue("")
		values, ve := again.ReadValueNames(-1)
		subs, se := again.ReadSubKeyNames(-1)
		again.Close()
		if e != nil || ve != nil || se != nil || !strings.EqualFold(command, expected) || len(values) != 1 || values[0] != "" || len(subs) != 0 {
			return fmt.Errorf("notifications: owned COM server changed before removal")
		}
	}
	if removeLink {
		if err := os.Remove(s.Path()); err != nil {
			return err
		}
	}
	if removeServer {
		if err := registry.DeleteKey(root, serverPath); err != nil {
			return err
		}
		parent, e := registry.OpenKey(root, clsidPath, registry.QUERY_VALUE|registry.ENUMERATE_SUB_KEYS)
		if e == nil {
			values, ve := parent.ReadValueNames(-1)
			subs, se := parent.ReadSubKeyNames(-1)
			parent.Close()
			if ve == nil && se == nil && len(values) == 0 && len(subs) == 0 {
				if err := registry.DeleteKey(root, clsidPath); err != nil {
					return err
				}
			}
		}
	}
	return nil
}
