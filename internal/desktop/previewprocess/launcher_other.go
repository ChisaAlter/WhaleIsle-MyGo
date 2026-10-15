//go:build !windows || (!amd64 && !arm64)

package previewprocess

import (
	"context"
	"errors"
)

func openPlatform(context.Context, Options, Document) (*Session, error) {
	return nil, errors.New("interactive AppContainer preview requires Windows amd64 or arm64")
}
