//go:build !windows

package components

import "fmt"

func bridgeAlertProcessIdentity(int, string) (string, error) {
	return "", errors.New(storelocale.T(storelocale.KeyStoreComponentsWhalebridgeComponentRemindersRequireWind))
}
