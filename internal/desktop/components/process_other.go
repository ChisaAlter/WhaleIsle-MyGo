//go:build !windows

package components

import (
	"errors"

	"whaleisle/internal/store/storelocale"
)

// Same contract as the Windows half: process management is Windows-only, and
// the caller gets the localized "not supported here" error with a nil process.
func openBridgeProcess(int, string) (bridgeProcess, error) {
	return nil, errors.New(storelocale.T(storelocale.KeyStoreComponentsWhalebridgeComponentProcessManagementReq))
}
