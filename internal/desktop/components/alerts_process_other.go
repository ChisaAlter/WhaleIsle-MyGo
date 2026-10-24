//go:build !windows

package components

import (
	"errors"

	"whaleisle/internal/store/storelocale"
)

// Same contract as the Windows half: the identity cannot be verified here, so
// the caller must treat the bridge request as unsupported rather than trusted.
func bridgeAlertProcessIdentity(int, string) (string, error) {
	return "", errors.New(storelocale.T(storelocale.KeyStoreComponentsWhalebridgeComponentRemindersRequireWind))
}
