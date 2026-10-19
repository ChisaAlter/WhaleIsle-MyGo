// functional-workspace-check exercises directory/workspace controllers against
// one fresh owned Host. No prompt, model endpoint or user profile is accessed.
package main

import (
	"archive/zip"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/dsh"
	"whaleisle/internal/dsh/api"
	"whaleisle/internal/harness"
	"whaleisle/internal/paths"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/dirpick"
	"whaleisle/internal/ui/i18n"
)

type result struct{ ID, Case, Status, Method, Detail string }
type report struct {
	Started, SourceRevision, Root         string
	ModelRequests                         int
	OwnedHostStopped, OwnedListenerClosed bool
	Results                               []result
}

func main() {
	stage := flag.String("stage", "", "verified staged runtime")
	base := flag.String("root", "", "fresh absolute directory")
	out := flag.String("out", "", "JSON report")
	officeOnly := flag.Bool("office-only", false, "only validate existing Office converters against an owned blank session")
	flag.Parse()
	r := report{Started: time.Now().UTC().Format(time.RFC3339), Root: *base}
	if revision, err := exec.Command("git", "rev-parse", "HEAD").Output(); err == nil {
		r.SourceRevision = strings.TrimSpace(string(revision))
	}
	err := run(*stage, *base, *officeOnly, &r)
	if err != nil {
		r.Results = append(r.Results, result{"WORK-01/02", "run", "FAIL", "isolated real Host", err.Error()})
	}
	raw, e := json.MarshalIndent(r, "", "  ")
	if e == nil {
		e = os.WriteFile(*out, append(raw, '\n'), 0600)
	}
	if e != nil {
		fmt.Fprintln(os.Stderr, e)
		os.Exit(1)
	}
	fmt.Println(string(raw))
	if err != nil {
		os.Exit(1)
	}
}

func wait(ctx context.Context, condition func() (bool, error)) error {
	for {
		ok, err := condition()
		if err != nil {
			return err
		}
		if ok {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(20 * time.Millisecond):
		}
	}
}

func run(stage, root string, officeOnly bool, r *report) error {
	if !filepath.IsAbs(stage) || !filepath.IsAbs(root) {
		return errors.New("absolute stage and root required")
	}
	if err := os.Mkdir(root, 0700); err != nil {
		return fmt.Errorf("new isolated root required: %w", err)
	}
	workspace := filepath.Join(root, "workspace-one")
	second := filepath.Join(root, "workspace-two")
	data := filepath.Join(root, "user-data")
	home := filepath.Join(data, "dsh-home")
	for _, p := range []string{workspace, second, filepath.Join(workspace, "visible"), filepath.Join(workspace, ".hidden")} {
		if err := os.MkdirAll(p, 0700); err != nil {
			return err
		}
	}
	if err := os.WriteFile(filepath.Join(second, "retained.txt"), []byte("registration removal must preserve this file\n"), 0600); err != nil {
		return err
	}
	for key, value := range map[string]string{paths.EnvUserData: data, paths.EnvVendor: filepath.Join(stage, "vendor"), "DSH_HOME": home} {
		if err := os.Setenv(key, value); err != nil {
			return err
		}
	}
	// Converter/font caches stay in the owned task profile, not the user's
	// application cache. The kit resolves installed engines without downloads.
	if officeOnly {
		for _, name := range []string{"LOCALAPPDATA", "APPDATA"} {
			if err := os.Setenv(name, filepath.Join(root, strings.ToLower(name))); err != nil {
				return err
			}
		}
	}
	vendor := filepath.Join(stage, "vendor")
	harnessRoot := filepath.Join(vendor, "deepseek-harness")
	node := filepath.Join(stage, "node", "node.exe")
	supervisor := harness.New(harness.Options{Install: harness.NewInstallServer(harness.InstallServerOptions{NodeBin: node}), Launch: harness.LaunchOptions{NodeBin: node, HarnessRoot: harnessRoot, VendorDir: vendor, Host: "127.0.0.1", Port: 32361, Home: home}, Prepare: harness.PrepareOptions{VendorDir: vendor, HarnessRoot: harnessRoot, DshHome: home, ProfileDir: filepath.Join(home, "profiles", "web"), RemoteSet: true, Remote: false}, Workspace: workspace, ReadyTimeout: 3 * time.Minute})
	var ep harness.Endpoint
	defer func() {
		supervisor.Stop()
		r.OwnedHostStopped = supervisor.Status().State == harness.Stopped
		if u, err := url.Parse(ep.Origin); err == nil && u.Host != "" {
			conn, e := net.DialTimeout("tcp", u.Host, time.Second)
			r.OwnedListenerClosed = e != nil
			if conn != nil {
				conn.Close()
			}
		}
	}()
	fmt.Println(`{"checkpoint":"starting-owned-workspace-host"}`)
	supervisor.Start()
	ctx, cancel := context.WithTimeout(context.Background(), 195*time.Second)
	defer cancel()
	if err := wait(ctx, func() (bool, error) {
		status := supervisor.Status()
		if status.State == harness.Failed {
			return false, status.Err
		}
		if status.State == harness.Ready {
			ep = status.Endpoint
			return true, nil
		}
		return false, nil
	}); err != nil {
		return err
	}
	r.Results = append(r.Results, result{"WORK-01/02", "fresh-host-ready", "PASS", "real staged Host, isolated user-data", "No model requests; cookie retained in memory only"})
	fmt.Println(`{"checkpoint":"owned-workspace-host-ready"}`)
	client := &dsh.Client{Endpoint: dsh.Endpoint{Origin: ep.Origin, Cookie: ep.Cookie}, HTTP: &http.Client{Timeout: 20 * time.Second}}
	mux := &dsh.Mux{Endpoint: client.Endpoint, DialTimeout: 10 * time.Second}
	defer mux.Close()
	store := session.New()
	store.Attach(client, nil)
	store.AttachMux(mux)
	defer store.Detach()
	if officeOnly {
		client.HTTP.Timeout = 120 * time.Second
		return checkOffice(client, store, workspace, r)
	}
	ctx2, cancel2 := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel2()
	store.FollowWorkspace(ctx2)
	if err := wait(ctx2, func() (bool, error) {
		v := store.Snapshot()
		if v.WorkspaceLayout.Error != "" {
			return false, errors.New(v.WorkspaceLayout.Error)
		}
		return v.WorkspaceReady, nil
	}); err != nil {
		return err
	}
	check := func(id, name, method, detail string, fn func() error) error {
		if err := fn(); err != nil {
			r.Results = append(r.Results, result{id, name, "FAIL", method, err.Error()})
			return err
		}
		r.Results = append(r.Results, result{id, name, "PASS", method, detail})
		return nil
	}
	list := func(path string) (api.DirectoryListing, error) {
		var listing api.DirectoryListing
		err := client.Call(ctx2, "directoryPicker/list", map[string]any{"path": path}, &listing)
		return listing, err
	}
	if err := check("WORK-02", "miller-list-create-missing-retry", "real directoryPicker RPC + real FS + headless Miller input", "Real directory listing and empty child, nested creation, invalid-name rejection, missing path retry and Miller Open; no ACL changes", func() error {
		listing, err := list(workspace)
		if err != nil {
			return err
		}
		if listing.Path != workspace {
			return errors.New("directory listing path mismatch")
		}
		var visible *api.DirectoryListingCrumbsItem
		for _, entry := range listing.Entries {
			if entry.Name == "visible" {
				copy := entry
				visible = &copy
			}
		}
		if visible == nil {
			return errors.New("real directory missing from Host listing")
		}
		child, err := list(visible.Path)
		if err != nil {
			return err
		}
		if len(child.Entries) != 0 {
			return errors.New("fresh child directory not empty")
		}
		var created api.CreateDirectoryResult
		if err = client.Call(ctx2, "directoryPicker/createDirectory", map[string]any{"path": visible.Path, "name": "created"}, &created); err != nil {
			return err
		}
		if info, err := os.Stat(filepath.Join(visible.Path, "created")); err != nil || !info.IsDir() {
			return errors.New("Host folder create not present on disk")
		}
		if err = client.Call(ctx2, "directoryPicker/createDirectory", map[string]any{"path": visible.Path, "name": "../escape"}, &created); err == nil {
			return errors.New("Host accepted non-segment create name")
		}
		missing := filepath.Join(workspace, "missing-before-retry")
		if _, err = list(missing); err == nil {
			return errors.New("Host returned missing path as success")
		}
		if err = os.Mkdir(missing, 0700); err != nil {
			return err
		}
		if _, err = list(missing); err != nil {
			return err
		}
		old := i18n.Locale()
		i18n.SetLocale("en")
		defer i18n.SetLocale(old)
		state := dirpick.BrowserState{Parent: &listing, Selected: visible, Child: &child}
		picked := ""
		tt := ui.NewTester(func(c *ui.Context) {
			dirpick.View(c, dirpick.Model{State: &state, OnPick: func(path string) { picked = path }})
		}, 680, 500)
		if err = tt.Click("Open"); err != nil {
			return err
		}
		if picked != visible.Path {
			return errors.New("Miller Open changed selected Host path")
		}
		return nil
	}); err != nil {
		return err
	}
	if err := check("WORK-01", "workspace-register-choose-persistence-retry", "real Store controller + Host workspace/follow", "Creates only owned registration; rejected persistence keeps current selection; retry/stale token checked", func() error {
		store.OpenWorkspacePicker()
		view := store.Snapshot().WorkspaceNavigation
		if view == nil {
			return errors.New("picker failed to open")
		}
		store.RegisterWorkspace(ctx2, view.Token, second, func(path string) error {
			if path != second {
				return errors.New("wrong persistence target")
			}
			return nil
		}, nil)
		if err := wait(ctx2, func() (bool, error) {
			v := store.Snapshot()
			if v.WorkspaceNavigation != nil && v.WorkspaceNavigation.Error != "" {
				return false, errors.New(v.WorkspaceNavigation.Error)
			}
			return store.WorkspaceCwd() == second && v.WorkspaceNavigation == nil, nil
		}); err != nil {
			return err
		}
		store.OpenWorkspacePicker()
		view = store.Snapshot().WorkspaceNavigation
		var initial string
		for _, row := range store.Snapshot().Workspaces {
			if row.Path == workspace {
				initial = row.WorkspaceID
			}
		}
		if initial == "" {
			return errors.New("initial Host workspace missing")
		}
		if store.ChooseRegisteredWorkspace(view.Token, initial, func(string) error { return errors.New("owned persistence rejection") }) {
			return errors.New("failed persistence selected workspace")
		}
		if store.WorkspaceCwd() != second || store.Snapshot().WorkspaceNavigation.Error == "" {
			return errors.New("failed selection lost cwd/error")
		}
		if !store.ChooseRegisteredWorkspace(view.Token, initial, func(string) error { return nil }) {
			return errors.New("selection retry failed")
		}
		if store.ChooseRegisteredWorkspace(view.Token, initial, nil) {
			return errors.New("stale picker token accepted")
		}
		return nil
	}); err != nil {
		return err
	}
	if err := check("WORK-01", "workspace-rename-cancel-remove-preserves-files", "real Store + Host RPC and follow receipts + disk verification", "Renames and removes only self-created registry row; cancel does not mutate, files survive confirmed removal", func() error {
		var id string
		for _, row := range store.Snapshot().Workspaces {
			if row.Path == second {
				id = row.WorkspaceID
			}
		}
		if id == "" {
			return errors.New("registered workspace not projected")
		}
		store.BeginWorkspaceRename(id)
		view := store.Snapshot().WorkspaceNavigation
		if view == nil {
			return errors.New("rename did not open")
		}
		store.ConfirmWorkspaceRename(ctx2, view.Token, "Owned renamed workspace")
		if err := wait(ctx2, func() (bool, error) {
			v := store.Snapshot()
			if v.WorkspaceNavigation != nil && v.WorkspaceNavigation.Error != "" {
				return false, errors.New(v.WorkspaceNavigation.Error)
			}
			for _, row := range v.Workspaces {
				if row.WorkspaceID == id && row.Title == "Owned renamed workspace" {
					return true, nil
				}
			}
			return false, nil
		}); err != nil {
			return err
		}
		store.BeginWorkspaceRemoval(id)
		view = store.Snapshot().WorkspaceNavigation
		store.CancelWorkspaceNavigation(view.Token)
		found := false
		for _, row := range store.Snapshot().Workspaces {
			found = found || row.WorkspaceID == id
		}
		if !found {
			return errors.New("cancel removed registry")
		}
		store.BeginWorkspaceRemoval(id)
		view = store.Snapshot().WorkspaceNavigation
		store.ConfirmWorkspaceRemoval(ctx2, view.Token)
		if err := wait(ctx2, func() (bool, error) {
			v := store.Snapshot()
			if v.WorkspaceNavigation != nil && v.WorkspaceNavigation.Error != "" {
				return false, errors.New(v.WorkspaceNavigation.Error)
			}
			for _, row := range v.Workspaces {
				if row.WorkspaceID == id {
					return false, nil
				}
			}
			return v.WorkspaceNavigation == nil, nil
		}); err != nil {
			return err
		}
		data, err := os.ReadFile(filepath.Join(second, "retained.txt"))
		if err != nil || string(data) != "registration removal must preserve this file\n" {
			return errors.New("registration removal changed owned file")
		}
		return nil
	}); err != nil {
		return err
	}
	return nil
}

func checkOffice(client *dsh.Client, store *session.Store, root string, r *report) error {
	ctx, cancel := context.WithTimeout(context.Background(), 150*time.Second)
	defer cancel()
	var created api.SessionCreateValue
	if err := client.Call(ctx, "session/create", map[string]any{"request": api.SessionCreateRequest{Cwd: &root}}, &created); err != nil {
		return err
	}
	if created.SessionId == "" {
		return errors.New("owned blank session was not created")
	}
	store.Select(created.SessionId)
	fixtures := map[string]map[string]string{
		"document.docx": {
			"[Content_Types].xml": `<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types"><Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/><Default Extension="xml" ContentType="application/xml"/><Override PartName="/word/document.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.document.main+xml"/></Types>`,
			"_rels/.rels":         `<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="word/document.xml"/></Relationships>`,
			"word/document.xml":   `<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"><w:body><w:p><w:r><w:t>Isolated Office document</w:t></w:r></w:p><w:sectPr><w:pgSz w:w="12240" w:h="15840"/></w:sectPr></w:body></w:document>`,
		},
		"workbook.xlsx": {
			"[Content_Types].xml":        `<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types"><Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/><Default Extension="xml" ContentType="application/xml"/><Override PartName="/xl/workbook.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.sheet.main+xml"/><Override PartName="/xl/worksheets/sheet1.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.worksheet+xml"/></Types>`,
			"_rels/.rels":                `<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="xl/workbook.xml"/></Relationships>`,
			"xl/workbook.xml":            `<workbook xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships"><sheets><sheet name="Sheet" sheetId="1" r:id="rId1"/></sheets></workbook>`,
			"xl/_rels/workbook.xml.rels": `<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/worksheet" Target="worksheets/sheet1.xml"/></Relationships>`,
			"xl/worksheets/sheet1.xml":   `<worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main"><sheetData><row r="1"><c r="A1" t="inlineStr"><is><t>Isolated workbook</t></is></c><c r="B1"><v>42</v></c></row></sheetData></worksheet>`,
		},
		"slides.pptx": {
			"[Content_Types].xml":             `<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types"><Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/><Default Extension="xml" ContentType="application/xml"/><Override PartName="/ppt/presentation.xml" ContentType="application/vnd.openxmlformats-officedocument.presentationml.presentation.main+xml"/><Override PartName="/ppt/slides/slide1.xml" ContentType="application/vnd.openxmlformats-officedocument.presentationml.slide+xml"/></Types>`,
			"_rels/.rels":                     `<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="ppt/presentation.xml"/></Relationships>`,
			"ppt/presentation.xml":            `<p:presentation xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships" xmlns:p="http://schemas.openxmlformats.org/presentationml/2006/main"><p:sldIdLst><p:sldId id="256" r:id="rId1"/></p:sldIdLst><p:sldSz cx="9144000" cy="6858000"/><p:notesSz cx="6858000" cy="9144000"/></p:presentation>`,
			"ppt/_rels/presentation.xml.rels": `<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/slide" Target="slides/slide1.xml"/></Relationships>`,
			"ppt/slides/slide1.xml":           `<p:sld xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main" xmlns:p="http://schemas.openxmlformats.org/presentationml/2006/main"><p:cSld><p:spTree><p:nvGrpSpPr><p:cNvPr id="1" name=""/><p:cNvGrpSpPr/><p:nvPr/></p:nvGrpSpPr><p:grpSpPr/><p:sp><p:nvSpPr><p:cNvPr id="2" name="Title"/><p:cNvSpPr txBox="1"/><p:nvPr/></p:nvSpPr><p:spPr><a:xfrm><a:off x="914400" y="914400"/><a:ext cx="7315200" cy="914400"/></a:xfrm></p:spPr><p:txBody><a:bodyPr/><a:lstStyle/><a:p><a:r><a:t>Isolated slide</a:t></a:r></a:p></p:txBody></p:sp></p:spTree></p:cSld><p:clrMapOvr><a:masterClrMapping/></p:clrMapOvr></p:sld>`,
		},
	}
	var failure error
	for _, name := range []string{"document.docx", "workbook.xlsx", "slides.pptx"} {
		file, err := os.Create(filepath.Join(root, name))
		if err != nil {
			return err
		}
		archive := zip.NewWriter(file)
		for path, text := range fixtures[name] {
			part, e := archive.Create(path)
			if e != nil {
				file.Close()
				return e
			}
			if _, e = part.Write([]byte(text)); e != nil {
				file.Close()
				return e
			}
		}
		if err = archive.Close(); err == nil {
			err = file.Close()
		} else {
			_ = file.Close()
		}
		if err != nil {
			return err
		}
		document, err := store.RenderOfficePDFBytesForSession(ctx, created.SessionId, name)
		status, detail := "PASS", "Real Host authorization and installed LibreOffice kit conversion; complete PDF bytes decoded through production Store adapter"
		if err != nil {
			status, detail = "FAIL", err.Error()
			failure = err
		} else if !strings.HasPrefix(string(document.Data), "%PDF-") || document.SourcePath != filepath.Join(root, name) {
			status, detail = "FAIL", "conversion did not retain source identity or complete PDF signature"
			failure = errors.New(detail)
		} else if err = os.WriteFile(filepath.Join(root, name+".pdf"), document.Data, 0600); err != nil {
			return err
		}
		r.Results = append(r.Results, result{"WORK-06", "office-" + name, status, "actual isolated Host RPC + Store multipart adapter + installed native converter", detail})
	}
	return failure
}
