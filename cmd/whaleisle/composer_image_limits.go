package main

import (
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
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
	supported := map[string]bool{}
	for _, typ := range limits.MediaTypes {
		supported[typ] = true
	}
	var count, total int64
	check := func(path, knownType string) error {
		info, err := os.Stat(path)
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf(i18n.T(i18n.KeyCmdComposerFileNotRegular), filepath.Base(path))
		}
		typ := knownType
		if typ == "" {
			typ = mime.TypeByExtension(strings.ToLower(filepath.Ext(path)))
			if !supported[typ] {
				f, err := os.Open(path)
				if err != nil {
					return err
				}
				var b [512]byte
				n, e := f.Read(b[:])
				f.Close()
				if e != nil && e != io.EOF {
					return e
				}
				typ = http.DetectContentType(b[:n])
			}
		}
		if !supported[typ] {
			return nil
		}
		count++
		if count > limits.MaxImagesPerMessage {
			return fmt.Errorf(i18n.T(i18n.KeyCmdComposerImageCountLimit), limits.MaxImagesPerMessage)
		}
		if info.Size() > limits.MaxImageBytes {
			return fmt.Errorf(i18n.T(i18n.KeyCmdComposerImageSizeLimit), limits.MaxImageBytes)
		}
		if info.Size() > limits.MaxMessageImageBytes-total {
			return fmt.Errorf(i18n.T(i18n.KeyCmdComposerImageTotalLimit), limits.MaxMessageImageBytes)
		}
		total += info.Size()
		return nil
	}
	for _, a := range existing {
		if a.MediaType != "" && !supported[a.MediaType] {
			continue
		}
		if a.ImageData != "" && supported[a.MediaType] {
			size := int64(len(a.ImageData)/4*3 - (len(a.ImageData) - len(strings.TrimRight(a.ImageData, "="))))
			count++
			if count > limits.MaxImagesPerMessage {
				return fmt.Errorf(i18n.T(i18n.KeyCmdComposerImageCountLimit), limits.MaxImagesPerMessage)
			}
			if size > limits.MaxImageBytes {
				return fmt.Errorf(i18n.T(i18n.KeyCmdComposerImageSizeLimit), limits.MaxImageBytes)
			}
			if size > limits.MaxMessageImageBytes-total {
				return fmt.Errorf(i18n.T(i18n.KeyCmdComposerImageTotalLimit), limits.MaxMessageImageBytes)
			}
			total += size
			continue
		}
		if err := check(a.Path, a.MediaType); err != nil {
			return err
		}
	}
	for _, p := range paths {
		if strings.TrimSpace(p) == "" {
			continue
		}
		if err := check(p, ""); err != nil {
			return err
		}
	}
	return nil
}
