package main

import (
	"github.com/egoist/mygo"
	"whaleisle/internal/desktop/importer"
	"whaleisle/internal/ui/i18n"
)

// Detection is read-only; its long report uses the same bounded shell dialog
// on the invoking surface instead of a separate operating-system message box.
func (s *mainState) showElectronImportReportNative(win *mygo.Window) {
	go func() {
		report, err := importer.ImportReport(importer.ElectronUserDataDir())
		message := importer.FormatReport(report)
		if err != nil {
			message = err.Error()
		}
		if win == nil || win.IsDestroyed() {
			return
		}
		s.askConfirm(confirmAskRequest{Owner: win, Title: i18n.T(i18n.KeySettingsDesktopDetectElectronTitle), Body: message, ConfirmLabel: i18n.T(i18n.KeyCmdOk), CancelLabel: i18n.T(i18n.KeyCmdClose)})
	}()
}
