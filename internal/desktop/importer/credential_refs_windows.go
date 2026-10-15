//go:build windows

package importer

import (
	"os"

	"golang.org/x/sys/windows"
)

func credentialFileHasSingleLink(file *os.File, _ os.FileInfo) bool {
	var info windows.ByHandleFileInformation
	return windows.GetFileInformationByHandle(windows.Handle(file.Fd()), &info) == nil && info.NumberOfLinks == 1
}
