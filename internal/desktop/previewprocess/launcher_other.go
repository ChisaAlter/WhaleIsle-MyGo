//go:build (!windows && !linux) || (!amd64 && !arm64)

package previewprocess

import (
	"context"
	"errors"
)

func openPlatform(context.Context, Options, Document) (*Session, error) {
	return nil, errors.New("isolated preview is not implemented for this operating system or architecture")
}
