//go:build !windows

package components

import "fmt"

func openBridgeProcess(int, string) (bridgeProcess, error) {
	return nil, errors.New(storelocale.T(storelocale.KeyStoreComponentsWhalebridgeComponentProcessManagementReq))
}
