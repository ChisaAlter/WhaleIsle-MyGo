package main

import (
	"errors"
	"fmt"
	"io"
	"mime"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"unicode"
	"unicode/utf8"
	"whaleisle/internal/ui/i18n"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/harness"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/composer"
)

// A loopback URL alone could be an SSH tunnel. Require the owned local process,
// its live generation/endpoint, and the selected session's actual client identity.
func (s *mainState) nativeComposerFileScope(sessions *session.Store, id string, d *composer.DraftState) (string, bool) {
	if sessions == nil || id == "" {
		return "", false
	}
	s.mu.Lock()
	sup, status := s.sup, s.bootStatus
	current := s.sessions == sessions && s.composerDrafts[id] == d
	s.mu.Unlock()
	if !current || sup == nil {
		return "", false
	}
	live := sup.Status()
	if live.State != harness.Ready || live.Generation == 0 || live.Generation != status.Generation || live.Endpoint != status.Endpoint {
		return "", false
	}
	u, err := url.Parse(live.Endpoint.Origin)
	if err != nil || (u.Hostname() != "localhost" && !net.ParseIP(u.Hostname()).IsLoopback()) {
		return "", false
	}
	s.dshMu.Lock()
	client, endpoint := s.dshClient, s.dshEndpoint
	s.dshMu.Unlock()
	if client == nil || endpoint != live.Endpoint.Origin+"\x00"+live.Endpoint.Cookie {
		return "", false
	}
	binding, sid, cwd := sessions.WorkspaceFileReferenceOrigin()
	return cwd, sid == id && binding == fmt.Sprintf("%p\x00%s", client, id)
}

// Batch validation precedes the single undoable selection replacement. Images
// and paths whose Host accessibility is unproven retain the upload path.
func insertNativeComposerFiles(d *composer.DraftState, cwd string, paths []string, local bool) ([]string, error) {
	if d == nil || d.Sending {
		return nil, errors.New(i18n.T(i18n.KeyCmdComposerWaitSendFiles))
	}
	if !local {
		return paths, nil
	}
	var uploads, mentions []string
	for _, path := range paths {
		if strings.TrimSpace(path) == "" {
			continue
		}
		if !filepath.IsAbs(path) {
			return nil, errors.New(i18n.T(i18n.KeyCmdComposerAbsoluteFilePath))
		}
		info, err := os.Stat(path)
		if err != nil {
			return nil, err
		}
		if !info.IsDir() && !info.Mode().IsRegular() {
			return nil, errors.New(i18n.T(i18n.KeyCmdComposerRegularFileDirectory))
		}
		if !info.IsDir() {
			image := strings.HasPrefix(mime.TypeByExtension(strings.ToLower(filepath.Ext(path))), "image/")
			if !image {
				f, err := os.Open(path)
				if err != nil {
					return nil, err
				}
				var prefix [512]byte
				n, err := f.Read(prefix[:])
				f.Close()
				if err != nil && err != io.EOF {
					return nil, err
				}
				image = strings.HasPrefix(http.DetectContentType(prefix[:n]), "image/")
			}
			if image {
				uploads = append(uploads, path)
				continue
			}
		}
		value := path
		if cwd != "" {
			if rel, err := filepath.Rel(cwd, path); err == nil && rel != "." && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
				value = rel
			}
		}
		value = filepath.ToSlash(value)
		if info.IsDir() {
			value = strings.TrimSuffix(value, "/") + "/"
		}
		// Completed directory references close quotes; suggestions alone leave them open.
		mention, ok := composer.FormatFileMention(value, false, false)
		if !ok {
			return nil, errors.New(i18n.T(i18n.KeyCmdComposerUnsafeFilePath))
		}
		mentions = append(mentions, mention)
	}
	if len(mentions) == 0 {
		return uploads, nil
	}
	if d.Client == nil {
		d.Client = composer.NewDraftClient(&d.Text)
	}
	if _, marked := d.Client.MarkedRange(); marked {
		return nil, errors.New(i18n.T(i18n.KeyCmdComposerFinishComposition))
	}
	selection := d.Client.Selection().Range
	start, end := ui.UTF16ByteOffset(d.Text, selection.Start), ui.UTF16ByteOffset(d.Text, selection.End)
	replacement := strings.Join(mentions, " ")
	if start > 0 {
		r, _ := utf8.DecodeLastRuneInString(d.Text[:start])
		if !unicode.IsSpace(r) {
			replacement = " " + replacement
		}
	}
	if end == len(d.Text) {
		replacement += " "
	} else {
		r, _ := utf8.DecodeRuneInString(d.Text[end:])
		if !unicode.IsSpace(r) {
			replacement += " "
		}
	}
	d.Client.ReplaceText(&selection, replacement)
	d.EditFocus = true
	return uploads, nil
}
