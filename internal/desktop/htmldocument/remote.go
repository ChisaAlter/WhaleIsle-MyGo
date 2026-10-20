package htmldocument

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"net"
	"net/http"
	pathpkg "path"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"
	"whaleisle/internal/desktop/htmlpreview"
	"whaleisle/internal/desktop/mediapreview"
	"whaleisle/internal/store/storelocale"
)

// OpenRemote freezes the document and its declared static dependencies through
// the original session's binary reader. Host paths are never opened locally.
func OpenRemote(ctx context.Context, reader mediapreview.Caller, sid, path string, valid func() bool) (*Document, error) {
	data, err := mediapreview.ReadSnapshot(ctx, reader, sid, path, MaxBytes, valid)
	if err != nil {
		return nil, err
	}
	current := func() bool { return ctx.Err() == nil && valid != nil && valid() }
	g := &resourceGraph{
		valid: current,
		remoteRead: func(assetPath string, limit int) ([]byte, error) {
			return mediapreview.ReadSnapshot(ctx, reader, sid, assetPath, limit, current)
		},
		files:  map[string]resourceFile{},
		active: map[string]bool{},
		report: ResourceReport{Bytes: len(data)},
	}
	d, err := openBytes(path, data, valid, g)
	if err != nil {
		return nil, err
	}
	// Retain the original source and scoped Host reader, not the short-lived
	// initial read context. Each gesture gets its own cancellation context.
	d.source = string(data)
	d.remoteInteractive = func(ctx context.Context) (string, ResourceReport, error) {
		current := func() bool { return ctx.Err() == nil && d.Valid() }
		graph := &resourceGraph{valid: current, files: map[string]resourceFile{}, active: map[string]bool{}, report: ResourceReport{Bytes: len(data)}}
		graph.remoteRead = func(asset string, limit int) ([]byte, error) {
			return mediapreview.ReadSnapshot(ctx, reader, sid, asset, limit, current)
		}
		prepared, report, err := prepareResourceGraph(graph, strings.ReplaceAll(path, `\`, "/"), string(data), current, true)
		if err == nil && !current() {
			err = errors.New("HTML session changed while preparing scripts")
		}
		return prepared, report, err
	}
	return d, nil
}

// OpenBytes serves bytes without a file reader. The browser receives neither
// Host credentials nor filesystem/network authority in either remote path.
func OpenBytes(path string, data []byte, valid func() bool) (*Document, error) {
	return openBytes(path, data, valid, nil)
}

func openBytes(path string, data []byte, valid func() bool, g *resourceGraph) (*Document, error) {
	if valid == nil || !valid() {
		return nil, errors.New("HTML request is no longer active")
	}
	if len(data) == 0 || len(data) > MaxBytes || !utf8.Valid(data) || bytes.IndexByte(data, 0) >= 0 {
		return nil, errors.New("HTML preview requires a nonempty UTF-8 document within 8 MiB")
	}
	prepared := string(data)
	resources := ResourceReport{Bytes: len(data), Skipped: 1, Warnings: []string{storelocale.T(storelocale.KeyStoreHtmldocumentRemoteStaticSnapshotRelativeResourcesAn)}}
	if g != nil {
		var err error
		prepared, resources, err = prepareResourceGraph(g, strings.ReplaceAll(path, `\`, "/"), prepared, g.valid, false)
		if err != nil {
			return nil, err
		}
		if !g.valid() {
			return nil, errors.New("HTML session changed while reading")
		}
	}
	data = []byte(htmlpreview.DocumentShell(prepared))
	if len(data) > htmlpreview.MaxPreparedHTMLBytes {
		return nil, errors.New("HTML document is too large after sandbox encoding")
	}
	secret := make([]byte, 32)
	if _, err := rand.Read(secret); err != nil {
		return nil, err
	}
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		return nil, err
	}
	d := &Document{Name: filepath.Base(path), Resources: resources, data: data, done: make(chan struct{}), origin: "http://" + listener.Addr().String(), route: "/" + hex.EncodeToString(secret) + "/document.html", valid: valid}
	d.server = &http.Server{Handler: http.HandlerFunc(d.serve), ReadHeaderTimeout: 5 * time.Second, IdleTimeout: 30 * time.Second, MaxHeaderBytes: 16 << 10}
	d.timer = time.AfterFunc(Lifetime, func() { d.Close() })
	go func() { _ = d.server.Serve(listener) }()
	return d, nil
}

// Remote dependency names use Host-neutral slash paths. The existing
// workspaceFileScopeId reader resolves them inside the initiating session;
// no local cwd or selectedID participates in nested CSS resolution.
func (g *resourceGraph) readRemote(base, reference, kind string) (resourceFile, string, string, error) {
	ref, err := relativeAssetPath(reference)
	if err != nil {
		return resourceFile{}, "", "", err
	}
	rel := pathpkg.Join(pathpkg.Dir(base), ref)
	if rel == ".." || strings.HasPrefix(rel, "../") || gitPath(rel) {
		return resourceFile{}, "", "", errors.New(storelocale.T(storelocale.KeyStoreHtmldocumentTheResourceIsOutsideTheAuthorizedWorks))
	}
	ext := strings.ToLower(pathpkg.Ext(rel))
	if !resourceKindAllows(kind, ext) {
		return resourceFile{}, "", "", errors.New(storelocale.T(storelocale.KeyStoreHtmldocumentUnsupportedResourceType))
	}
	if cached, ok := g.files[rel]; ok {
		if !resourceKindMIME(kind, cached.mime) {
			return resourceFile{}, "", "", errors.New(storelocale.T(storelocale.KeyStoreHtmldocumentTheResourceTypeAliasDoesNotMatch))
		}
		return cached, rel, rel, nil
	}
	if len(g.files) >= MaxAssets {
		return resourceFile{}, "", "", errors.New(storelocale.T(storelocale.KeyStoreHtmldocumentTheNumberOfResourceFilesExceedsTheLim))
	}
	limit := min(MaxAssetBytes, MaxGraphBytes-g.report.Bytes)
	if limit < 1 {
		return resourceFile{}, "", "", errors.New(storelocale.T(storelocale.KeyStoreHtmldocumentTheResourceSizeExceedsTheLimit))
	}
	data, err := g.remoteRead(rel, limit)
	if err != nil {
		return resourceFile{}, "", "", errors.New(storelocale.T(storelocale.KeyStoreHtmldocumentTheResourceIsMissingOrUnreadable))
	}
	if len(data) < 1 || len(data) > limit {
		return resourceFile{}, "", "", errors.New(storelocale.T(storelocale.KeyStoreHtmldocumentTheResourceSizeExceedsTheLimit))
	}
	file, err := g.rememberResource(rel, ext, data)
	return file, rel, rel, err
}
