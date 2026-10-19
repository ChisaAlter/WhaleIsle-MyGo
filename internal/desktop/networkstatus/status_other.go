//go:build !windows

package networkstatus

import (
	"context"
	"errors"
)

// Watch explicitly leaves unsupported platforms unknown.
func Watch(ctx context.Context, emit func(Availability)) (func() error, error) {
	if ctx.Err() != nil {
		return func() error { return nil }, ctx.Err()
	}
	emit(Unknown)
	return func() error { return nil }, errors.New("native network availability is unavailable on this platform")
}
