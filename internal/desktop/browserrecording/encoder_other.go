//go:build (!windows && !darwin && !linux) || (!amd64 && !arm64)

package browserrecording

import (
	"context"
	"errors"
	"whaleisle/internal/store/storelocale"
)

func newEncoder(context.Context) (encoder, error) {
	return nil, errors.New(storelocale.T(storelocale.KeyStoreBrowserrecordingWebmWebpageRecordingIsUnsupportedOnThi))
}
