package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"whaleisle/internal/desktop/pluginui"
	"whaleisle/internal/paths"
	"whaleisle/internal/store/session"
)

var _ pluginui.ComposerFileTransfer = nativePluginComposerBridge{}

func composerPublicMediaType(file session.ComposerAttachment) string {
	if file.SourceMediaType != nil {
		return *file.SourceMediaType
	}
	return file.MediaType
}

func (b nativePluginComposerBridge) composerPost() func(func()) {
	if b.state == nil {
		return nil
	}
	b.state.mu.Lock()
	win := b.state.win
	b.state.mu.Unlock()
	if win == nil {
		return nil
	}
	return func(fn func()) { win.Update(fn) }
}

func composerFileError(err error) error {
	var pathError *os.PathError
	if errors.As(err, &pathError) {
		return fmt.Errorf("attachment %s: %w", pathError.Op, pathError.Err)
	}
	return err
}

func removeStagedComposerFiles(files []session.ComposerAttachment) {
	for _, file := range files {
		_ = os.Remove(file.Path)
	}
}

type composerFileWriter struct {
	ctx context.Context
	io.Writer
}

func (w composerFileWriter) Write(p []byte) (int, error) {
	if err := w.ctx.Err(); err != nil {
		return 0, err
	}
	return w.Writer.Write(p)
}

// File/drop bytes share the existing clipboard attachment cache. The private
// random pathname is never a draft @reference or part of the public protocol.
func stagePluginComposerFiles(ctx context.Context, source pluginui.ComposerFileIterator) (files []session.ComposerAttachment, err error) {
	defer func() {
		if err != nil {
			removeStagedComposerFiles(files)
			files = nil
			err = composerFileError(err)
		}
	}()
	if source == nil {
		return nil, errors.New("Missing attachment stream")
	}
	buffer := make([]byte, 64*1024)
	for {
		if err = ctx.Err(); err != nil {
			return files, err
		}
		var part pluginui.ComposerFileContent
		part, err = source.Next()
		if err == io.EOF {
			return files, nil
		}
		if err != nil {
			return files, err
		}
		if part.Content == nil {
			return files, errors.New("Missing attachment content")
		}
		err = func() error {
			defer part.Content.Close()
			if strings.TrimSpace(part.Name) == "" || part.Name == "." || part.Name == ".." || strings.ContainsAny(part.Name, "/\\\x00\r\n") {
				return errors.New("Invalid attachment filename")
			}
			dir := filepath.Join(paths.UserDataDir(), "cache", "attachments")
			if err := os.MkdirAll(dir, 0700); err != nil {
				return err
			}
			file, err := os.CreateTemp(dir, "plugin-*")
			if err != nil {
				return err
			}
			defer file.Close()
			index := len(files)
			sourceMediaType := part.MediaType
			files = append(files, session.ComposerAttachment{Path: file.Name(), Name: part.Name, SourceMediaType: &sourceMediaType})
			size, err := io.CopyBuffer(composerFileWriter{ctx, file}, part.Content, buffer)
			if err != nil {
				return err
			}
			if part.Size >= 0 && size != part.Size {
				return errors.New("Attachment stream ended with a different size")
			}
			files[index].TotalBytes = size
			var header [512]byte
			n, err := file.ReadAt(header[:], 0)
			if err != nil && err != io.EOF {
				return err
			}
			files[index].MediaType = http.DetectContentType(header[:n])
			return file.Close()
		}()
		if err != nil {
			return files, err
		}
	}
}

func (b nativePluginComposerBridge) adoptComposerFilesOnUI(ctx context.Context, binding pluginui.ComposerBinding, original *nativePluginComposerOwner, files []session.ComposerAttachment, post func(func())) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	o, err := b.validate(binding)
	if err != nil || o != original || post == nil {
		return pluginComposerStale()
	}
	paths := make([]string, 0, len(files))
	for _, file := range files {
		paths = append(paths, file.Path)
	}
	if err := validateComposerImageIntake(o.draft.Attachments, paths, o.sessions.ComposerImageLimits(o.subject.SessionID)); err != nil {
		return composerFileError(err)
	}
	first := len(o.draft.Attachments)
	o.draft.Attachments = append(o.draft.Attachments, files...)
	for index := first; index < len(o.draft.Attachments); index++ {
		if !prepareComposerAttachment(o.target, o.draft, index, false, post) {
			for len(o.draft.Attachments) > first {
				o.draft.RemoveAttachment(len(o.draft.Attachments) - 1)
			}
			return pluginComposerStale()
		}
		o.draft.Attachments[index].TotalBytes = files[index-first].TotalBytes
	}
	return nil
}

func (b nativePluginComposerBridge) ImportFiles(ctx context.Context, binding pluginui.ComposerBinding, source pluginui.ComposerFileIterator) error {
	return b.importFiles(ctx, binding, source, b.onUI, b.composerPost())
}

// The stream worker and resident draft have different owners. Keeping the UI
// handoff explicit lets both native dispatch and headless consumers use it.
func (b nativePluginComposerBridge) importFiles(ctx context.Context, binding pluginui.ComposerBinding, source pluginui.ComposerFileIterator, onUI func(func() error) error, post func(func())) error {
	var original *nativePluginComposerOwner
	if err := onUI(func() (err error) { original, err = b.validate(binding); return err }); err != nil {
		return err
	}
	files, err := stagePluginComposerFiles(ctx, source)
	if err != nil {
		return err
	}
	adopted := false
	defer func() {
		if !adopted {
			removeStagedComposerFiles(files)
		}
	}()
	err = onUI(func() error { return b.adoptComposerFilesOnUI(ctx, binding, original, files, post) })
	adopted = err == nil
	return err
}

// Reads are allowed while a file is preparing, but never across a changed
// binding/revision or after the opaque preparation ID leaves this draft.
func (b nativePluginComposerBridge) composerFileOnUI(binding pluginui.ComposerBinding, id uint64) (session.ComposerAttachment, error) {
	o, err := b.capture(binding.Subject)
	if err != nil || composerPluginBinding(o) != binding || id == 0 {
		return session.ComposerAttachment{}, pluginComposerStale()
	}
	for _, file := range o.draft.Attachments {
		if file.PreparationID == id {
			return file, nil
		}
	}
	return session.ComposerAttachment{}, pluginComposerStale()
}

type boundComposerFileReader struct {
	io.ReadCloser
	ctx     context.Context
	current func() error
}

func (r *boundComposerFileReader) Read(p []byte) (int, error) {
	if err := r.ctx.Err(); err != nil {
		return 0, err
	}
	if err := r.current(); err != nil {
		return 0, err
	}
	n, err := r.ReadCloser.Read(p)
	if currentErr := r.current(); currentErr != nil {
		return 0, currentErr
	}
	return n, composerFileError(err)
}

func (b nativePluginComposerBridge) ReadFile(ctx context.Context, binding pluginui.ComposerBinding, id uint64) (out pluginui.ComposerFileContent, err error) {
	return b.readFile(ctx, binding, id, b.onUI)
}

func (b nativePluginComposerBridge) readFile(ctx context.Context, binding pluginui.ComposerBinding, id uint64, onUI func(func() error) error) (out pluginui.ComposerFileContent, err error) {
	var attachment session.ComposerAttachment
	if err = onUI(func() (err error) { attachment, err = b.composerFileOnUI(binding, id); return err }); err != nil {
		return out, err
	}
	file, err := os.Open(attachment.Path)
	if err != nil {
		return out, composerFileError(err)
	}
	info, err := file.Stat()
	if err != nil || !info.Mode().IsRegular() {
		file.Close()
		if err == nil {
			err = errors.New("Attachment is not a regular file")
		}
		return out, composerFileError(err)
	}
	current := func() error {
		if err := ctx.Err(); err != nil {
			return err
		}
		return onUI(func() error {
			live, err := b.composerFileOnUI(binding, id)
			if err == nil && live.Path != attachment.Path {
				err = pluginComposerStale()
			}
			return err
		})
	}
	if err = current(); err != nil {
		file.Close()
		return out, err
	}
	name := attachment.Name
	if name == "" {
		name = filepath.Base(attachment.Path)
	}
	out = pluginui.ComposerFileContent{Name: name, MediaType: composerPublicMediaType(attachment), Size: info.Size(), Content: &boundComposerFileReader{file, ctx, current}}
	return out, nil
}
