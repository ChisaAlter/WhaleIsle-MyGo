//go:build !windows || (!amd64 && !arm64)

package htmlpreview

import (
	"context"
	"errors"
)

func capturePrepared(context.Context, request) (captureResult, error) {
	return captureResult{}, errors.New(UnavailableReason)
}

func captureNative(context.Context, request) (captureResult, error) {
	return captureResult{}, errors.New(UnavailableReason)
}
