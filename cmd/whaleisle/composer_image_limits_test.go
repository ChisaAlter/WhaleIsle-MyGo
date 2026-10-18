package main

import (
	"os"
	"path/filepath"
	"testing"
	"whaleisle/internal/store/session"
)

func TestComposerImageIntakeProjectedLimits(t *testing.T) {
	root := t.TempDir()
	image := filepath.Join(root, "x.png")
	generic := filepath.Join(root, "large.txt")
	os.WriteFile(image, []byte("\x89PNG\r\n\x1a\n1234"), 0600)
	os.WriteFile(generic, make([]byte, 100), 0600)
	limits := &session.ComposerImageLimits{MediaTypes: []string{"image/png"}, MaxImagesPerMessage: 2, MaxImageBytes: 12, MaxMessageImageBytes: 24}
	if err := validateComposerImageIntake(nil, []string{image, generic}, limits); err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name              string
		count, one, total int64
	}{{"count", 0, 12, 24}, {"single", 2, 11, 24}, {"total", 2, 12, 20}} {
		t.Run(test.name, func(t *testing.T) {
			copy := *limits
			copy.MaxImagesPerMessage = test.count
			copy.MaxImageBytes = test.one
			copy.MaxMessageImageBytes = test.total
			if validateComposerImageIntake([]session.ComposerAttachment{{Path: image}}, []string{image, generic}, &copy) == nil {
				t.Fatal("limit accepted")
			}
		})
	}
	if err := validateComposerImageIntake(nil, []string{image}, nil); err != nil {
		t.Fatal("invented limit")
	}
}
