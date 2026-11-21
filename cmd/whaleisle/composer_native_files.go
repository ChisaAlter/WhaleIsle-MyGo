package main

import (
	"context"
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
	"whaleisle/internal/ui/i18n"

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

type composerFileInfo struct {
	path      string
	info      os.FileInfo
	mediaType string
}

// inspectComposerFiles is worker-only. One sequential pass bounds open files
// and identifies all inputs before any reference or upload enters the draft.
func inspectComposerFiles(ctx context.Context, paths []string, regularOnly bool) ([]composerFileInfo, error) {
	var files []composerFileInfo
	seen := make(map[string]bool, len(paths))
	for _, path := range paths {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if strings.TrimSpace(path) == "" {
			continue
		}
		if !filepath.IsAbs(path) || strings.ContainsRune(path, 0) {
			return nil, errors.New(i18n.T(i18n.KeyCmdComposerAbsoluteFilePath))
		}
		path = filepath.Clean(path)
		if seen[path] {
			continue
		}
		seen[path] = true
		info, err := os.Stat(path)
		if err != nil {
			return nil, err
		}
		if regularOnly && !info.Mode().IsRegular() {
			return nil, errors.New(i18n.T(i18n.KeyCmdComposerIntakeFilesOnly))
		}
		if !info.IsDir() && !info.Mode().IsRegular() {
			return nil, errors.New(i18n.T(i18n.KeyCmdComposerRegularFileDirectory))
		}
		file := composerFileInfo{path: path, info: info}
		if !info.IsDir() {
			file.mediaType = mime.TypeByExtension(strings.ToLower(filepath.Ext(path)))
			if !strings.HasPrefix(file.mediaType, "image/") {
				if err := ctx.Err(); err != nil {
					return nil, err
				}
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
				file.mediaType = http.DetectContentType(prefix[:n])
			}
		}
		files = append(files, file)
	}
	return files, ctx.Err()
}

type composerFilePlan struct {
	uploads    []session.ComposerAttachment
	references []string
}

func planNativeComposerFiles(cwd string, files []composerFileInfo, local bool) (composerFilePlan, error) {
	var plan composerFilePlan
	for _, file := range files {
		path, info := file.path, file.info
		if !local || strings.HasPrefix(file.mediaType, "image/") {
			plan.uploads = append(plan.uploads, session.ComposerAttachment{Path: path, Name: filepath.Base(path), MediaType: file.mediaType})
			continue
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
			return composerFilePlan{}, errors.New(i18n.T(i18n.KeyCmdComposerUnsafeFilePath))
		}
		plan.references = append(plan.references, mention)
	}
	return plan, nil
}

// Synchronous adapter for non-window callers. Interactive intake uses the same
// inspector and planner on a bounded worker, then applies through Window.Update.
func insertNativeComposerFiles(d *composer.DraftState, cwd string, paths []string, local bool) ([]string, error) {
	if d == nil || d.Sending {
		return nil, errors.New(i18n.T(i18n.KeyCmdComposerWaitSendFiles))
	}
	if !local {
		return paths, nil
	}
	files, err := inspectComposerFiles(context.Background(), paths, false)
	if err != nil {
		return nil, err
	}
	plan, err := planNativeComposerFiles(cwd, files, local)
	if err != nil {
		return nil, err
	}
	if !d.InsertPreparedReferences(plan.references) {
		return nil, errors.New(i18n.T(i18n.KeyCmdComposerFinishComposition))
	}
	uploads := make([]string, 0, len(plan.uploads))
	for _, a := range plan.uploads {
		uploads = append(uploads, a.Path)
	}
	return uploads, nil
}
