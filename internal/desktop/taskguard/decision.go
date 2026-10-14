package taskguard

// HideOnClose applies the close preference independently of running work.
// Explicit quit has already passed its own guard and must not hide instead.
func HideOnClose(quitting, closeToTray bool) bool { return !quitting && closeToTray }

// ConfirmRunning gates a destructive action on explicit consent when the local
// session snapshot contains running work. It is not a Host admission lock.
func ConfirmRunning(running int, ask func() (bool, error)) (bool, error) {
	if running <= 0 {
		return true, nil
	}
	approved, err := ask()
	return err == nil && approved, err
}
