package htmldocument

import (
	"context"
	"encoding/base64"
	"errors"
	"os"
	"strings"
)

// Done closes when the snapshot's lease expires or its owner is revoked.
// It is observation only; no document bytes or filesystem authority are exposed.
func (d *Document) Done() <-chan struct{} {
	if d == nil || d.done == nil {
		closed := make(chan struct{})
		close(closed)
		return closed
	}
	return d.done
}

// InteractiveSnapshot is called only for the user's explicit "run scripts"
// gesture. It reuses the original HTML bytes, not editor drafts or a reread
// of a changed file. Declared dependencies are frozen under the same directory
// identity with os.Root; no path/URL is ever passed to the helper.
func (d *Document) InteractiveSnapshot(ctx context.Context) (string, ResourceReport, error) {
	authorized := func() bool { return ctx.Err() == nil && d.Valid() }
	if !authorized() {
		return "", ResourceReport{}, errors.New("HTML snapshot is no longer authorized")
	}
	d.mu.RLock()
	source, rootPath, relative := d.source, d.root, d.relative
	rootIdentity, identity := d.rootIdentity, d.identity
	closed := d.closed
	d.mu.RUnlock()
	if closed || source == "" || rootIdentity == nil || identity == nil {
		return "", ResourceReport{}, errors.New("HTML snapshot is closed")
	}
	root, err := os.OpenRoot(rootPath)
	if err != nil {
		return "", ResourceReport{}, err
	}
	defer root.Close()
	check := func() bool {
		if !authorized() {
			return false
		}
		currentRoot, err := root.Stat(".")
		if err != nil || !os.SameFile(rootIdentity, currentRoot) {
			return false
		}
		currentFile, err := root.Stat(relative)
		return err == nil && os.SameFile(identity, currentFile) && currentFile.Size() == identity.Size() && currentFile.ModTime().Equal(identity.ModTime())
	}
	if !check() {
		return "", ResourceReport{}, errors.New("HTML or its workspace changed; reopen the saved file")
	}
	prepared, report, err := prepareResourceMode(root, relative, source, authorized, true)
	if err != nil {
		return "", report, err
	}
	if !check() {
		return "", report, errors.New("HTML authorization changed while preparing scripts")
	}
	return prepared, report, nil
}

func resourceKindAllows(kind, extension string) bool {
	switch kind {
	case "css":
		return extension == ".css"
	case "script":
		return extension == ".js" || extension == ".mjs"
	case "asset":
		return binaryExtension(extension)
	default:
		return false
	}
}

func resourceKindMIME(kind, mime string) bool {
	switch kind {
	case "css":
		return mime == "text/css"
	case "script":
		return mime == "text/javascript"
	case "asset":
		return strings.HasPrefix(mime, "image/") || strings.HasPrefix(mime, "font/")
	default:
		return false
	}
}

func (g *resourceGraph) script(base, reference string) (string, error) {
	reference = strings.TrimSpace(reference)
	if strings.HasPrefix(strings.ToLower(reference), "data:") {
		header := strings.ToLower(strings.SplitN(reference, ",", 2)[0])
		mime := strings.SplitN(strings.TrimPrefix(header, "data:"), ";", 2)[0]
		switch mime {
		case "text/javascript", "application/javascript", "text/ecmascript", "application/ecmascript":
			return reference, nil
		}
		return "", errors.New("不支持的内嵌脚本类型")
	}
	file, _, _, err := g.readKind(base, reference, "script")
	if err != nil {
		return "", err
	}
	return "data:text/javascript;base64," + base64.StdEncoding.EncodeToString(file.data), nil
}
