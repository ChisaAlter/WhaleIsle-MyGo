package directorypicker

import (
	"context"
	"errors"
	"path/filepath"
	"strings"

	"whaleisle/internal/desktop/workspaceauth"
)

var ErrCancelled = errors.New("directory selection cancelled")

// SelectAndRegister grants authority only to an explicit native picker result.
// An empty/cancelled result never falls back to a proposed or previous cwd.
func SelectAndRegister(ctx context.Context, pick func() ([]string, error), register func(context.Context, string) error) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if pick == nil || register == nil {
		return "", errors.New("directory selection unavailable")
	}
	selected, err := pick()
	if err != nil {
		return "", err
	}
	if err = ctx.Err(); err != nil {
		return "", err
	}
	if len(selected) == 0 {
		return "", ErrCancelled
	}
	if len(selected) != 1 || !SelectablePath(selected[0]) || !filepath.IsAbs(strings.TrimSpace(selected[0])) {
		return "", errors.New("invalid selected directory")
	}
	real, err := filepath.EvalSymlinks(strings.TrimSpace(selected[0]))
	if err != nil {
		return "", errors.New("selected directory unavailable")
	}
	if err = workspaceauth.Validate(real); err != nil {
		return "", errors.New("selected directory cannot be authorized")
	}
	if err = register(ctx, real); err != nil {
		return "", err
	}
	return real, nil
}
