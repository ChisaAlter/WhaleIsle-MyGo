package main

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/i18n"
)

// Header-only classification and stat sizes avoid loading files just to enforce
// limits. No Host projection means no invented client-side image limits.
func validateComposerImageIntake(existing []session.ComposerAttachment, paths []string, limits *session.ComposerImageLimits) error {
	if limits == nil {
		return nil
	}
	files, err := inspectComposerFiles(context.Background(), paths, false)
	if err != nil {
		return err
	}
	return validateComposerImageMetadata(context.Background(), existing, files, limits)
}

func validateComposerImageMetadata(ctx context.Context, existing []session.ComposerAttachment, files []composerFileInfo, limits *session.ComposerImageLimits) error {
	if limits == nil {
		return nil
	}
	supported := map[string]bool{}
	for _, typ := range limits.MediaTypes {
		supported[typ] = true
	}
	var count, total int64
	check := func(typ string, size int64) error {
		if !supported[typ] {
			return nil
		}
		count++
		if count > limits.MaxImagesPerMessage {
			return fmt.Errorf(i18n.T(i18n.KeyCmdComposerImageCountLimit), limits.MaxImagesPerMessage)
		}
		if size > limits.MaxImageBytes {
			return fmt.Errorf(i18n.T(i18n.KeyCmdComposerImageSizeLimit), humanBytes(limits.MaxImageBytes))
		}
		if size > limits.MaxMessageImageBytes-total {
			return fmt.Errorf(i18n.T(i18n.KeyCmdComposerImageTotalLimit), humanBytes(limits.MaxMessageImageBytes))
		}
		total += size
		return nil
	}
	for _, a := range existing {
		if err := ctx.Err(); err != nil {
			return err
		}
		if a.MediaType != "" && !supported[a.MediaType] {
			continue
		}
		if a.ImageData != "" && supported[a.MediaType] {
			size := int64(len(a.ImageData)/4*3 - (len(a.ImageData) - len(strings.TrimRight(a.ImageData, "="))))
			if err := check(a.MediaType, size); err != nil {
				return err
			}
			continue
		}
		inspected, err := inspectComposerFiles(ctx, []string{a.Path}, false)
		if err != nil {
			return err
		}
		for _, file := range inspected {
			if !file.info.IsDir() {
				typ := file.mediaType
				if a.MediaType != "" {
					typ = a.MediaType
				}
				if err := check(typ, file.info.Size()); err != nil {
					return err
				}
			}
		}
	}
	for _, file := range files {
		if !file.info.IsDir() {
			if !file.info.Mode().IsRegular() {
				return fmt.Errorf(i18n.T(i18n.KeyCmdComposerFileNotRegular), filepath.Base(file.path))
			}
			if err := check(file.mediaType, file.info.Size()); err != nil {
				return err
			}
		}
	}
	return nil
}

func humanBytes(n int64) string {
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%d B", n)
	}
	div, exp := int64(unit), 0
	for v := n / unit; v >= unit; v /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(n)/float64(div), "KMGTPE"[exp])
}
