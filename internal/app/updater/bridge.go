package updater

import (
	"context"
	"strings"

	desktopupdate "whaleisle/internal/desktop/update"
	"whaleisle/internal/paths"
	"whaleisle/internal/ui/i18n"
)

// WindowRef is an opaque comparable handle identifying a host shell window.
type WindowRef any

// NoticeHandle represents a desktop notification instance created by the shell.
type NoticeHandle interface {
	OnClick(fn func())
	Show() error
	Close()
}

// ConfirmRequest describes a confirmation prompt requested by the updater.
type ConfirmRequest struct {
	Owner            WindowRef
	Width            int
	Title            string
	Body             string
	ConfirmLabel     string
	CancelLabel      string
	TechnicalDetails string
	Run              func()
	Cancel           func()
}

// ShellEnvironment captures the host shell's current flags without holding
// updater locks across shell mutexes.
type ShellEnvironment struct {
	BuildVersion       string
	DownloadRoute      string
	Quitting           bool
	LauncherOpen       bool
	LauncherStartBusy  bool
	LauncherImportBusy bool
	TaskRestartPending bool
	AskOnUpdate        bool
	HostID             string
	HostGeneration     int
}

// ShellBridge abstracts all windowing, notification, and runtime lifecycle
// operations needed by Center so internal/app/updater stays toolkit-free.
type ShellBridge interface {
	Environment() ShellEnvironment
	AutoDownloadPref() (value bool, ok bool)
	InitAutoDownloadPref(migratedValue bool) error
	SetAutoDownloadPref(enabled bool) error
	LifecycleContext() context.Context
	UserDataDir() string

	OwnedWindow(win WindowRef) bool
	WindowAlive(win WindowRef) bool
	WindowVisibility(win WindowRef) (visible, focused bool)
	WindowHasPage(win WindowRef) bool
	WindowKey(win WindowRef) string
	DialogOwner(preferred WindowRef) WindowRef
	PrimaryWindows() (mainWin, launcherWin WindowRef)

	RunOnMain(fn func())
	UpdateWindow(win WindowRef, fn func())
	InvalidateWindow(win WindowRef)
	RestoreShowFocus(win WindowRef)
	FlashFrame(win WindowRef, flash bool)

	SystemNotificationsSupported() bool
	NewNotification(title, body string, silent bool) NoticeHandle
	WatchFocusAndClose(win WindowRef, onFocus, onClose func()) (stop func())
	WatchModalWindows(parent, owner WindowRef, onDismiss func()) (stop func())
	OpenModalWindow(parent WindowRef, title string) WindowRef
	DestroyWindow(win WindowRef)

	RefreshTray()
	AskConfirm(req ConfirmRequest)
	AskConfirmBlocking(req ConfirmRequest) bool
	InstallStagedWithConsent(win WindowRef, staged desktopupdate.ApplyResult, confirmedByInstallButton bool) bool
	SaveFailedToast(action string, err error) string
	SetLauncherUpdateNotice(text string)
}

func lifecycleText(zh, en string) string {
	if i18n.Locale() == i18n.LocaleEN {
		return en
	}
	return zh
}

func (c *Center) env() ShellEnvironment {
	if c != nil && c.bridge != nil {
		e := c.bridge.Environment()
		if e.BuildVersion == "" {
			e.BuildVersion = c.fallbackVersion()
		}
		if e.DownloadRoute == "" {
			c.mu.Lock()
			e.DownloadRoute = c.fallbackRoute
			c.mu.Unlock()
		}
		return e
	}
	c.mu.Lock()
	ver, route := c.fallbackVersionLocked(), c.fallbackRoute
	c.mu.Unlock()
	return ShellEnvironment{
		BuildVersion:  ver,
		DownloadRoute: route,
	}
}

func (c *Center) fallbackVersion() string {
	if c == nil {
		return ""
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.fallbackVersionLocked()
}

func (c *Center) fallbackVersionLocked() string {
	if c.buildVersion != "" {
		return c.buildVersion
	}
	return "dev"
}

func (c *Center) userDataDir() string {
	if c != nil && c.bridge != nil {
		if dir := strings.TrimSpace(c.bridge.UserDataDir()); dir != "" {
			return dir
		}
	}
	return paths.UserDataDir()
}

func (c *Center) lifecycleContext() context.Context {
	if c != nil && c.bridge != nil {
		if ctx := c.bridge.LifecycleContext(); ctx != nil {
			return ctx
		}
	}
	return context.Background()
}
