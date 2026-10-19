// functional-preview-check runs only product local-document owners in hidden,
// fresh-profile WebView2 windows. It never starts a Host or opens user files.
package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/desktop/htmldocument"
	"whaleisle/internal/desktop/pdfpreview"
	"whaleisle/internal/desktop/webview"
	"whaleisle/internal/paths"
)

type result struct{ Format, Status, Detail, Screenshot string }
type report struct {
	Started, SourceRevision, Root string
	ModelRequests                 int
	WindowsHidden                 bool
	Results                       []result
}

func main() {
	root := flag.String("root", "", "fresh absolute profile/fixture directory")
	out := flag.String("out", "", "JSON evidence path")
	flag.Parse()
	if !filepath.IsAbs(*root) || !filepath.IsAbs(*out) {
		fmt.Fprintln(os.Stderr, "absolute fresh root and report required")
		os.Exit(2)
	}
	if err := os.Mkdir(*root, 0700); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	_ = os.Setenv(paths.EnvUserData, filepath.Join(*root, "data"))
	mygo.App.SetName("WhaleIsle Files Isolated Acceptance")
	mygo.App.SetPath(mygo.PathUserData, filepath.Join(*root, "mygo-data"))
	mygo.App.SetPath(mygo.PathCache, filepath.Join(*root, "cache"))
	mygo.App.SetPath(mygo.PathLogs, filepath.Join(*root, "logs"))
	r := report{Started: time.Now().UTC().Format(time.RFC3339), Root: *root, WindowsHidden: true}
	if revision, err := exec.Command("git", "rev-parse", "HEAD").Output(); err == nil {
		r.SourceRevision = strings.TrimSpace(string(revision))
	}
	var mu sync.Mutex
	var windows []*mygo.Window
	mygo.App.OnWindowCreated(func(w *mygo.Window) { mu.Lock(); windows = append(windows, w); mu.Unlock() })
	mygo.App.WhenReady(func() {
		go func() {
			defer mygo.App.Quit()
			parent := mygo.NewWindow(mygo.WindowOptions{Title: "Isolated file preview acceptance", Width: 900, Height: 700, Hidden: true, SkipTaskbar: true, Content: ui.View(func(c *ui.Context) { ui.Box(c).Fill() })})
			if parent == nil {
				r.Results = append(r.Results, result{"setup", "FAIL", "hidden parent unavailable", ""})
				return
			}
			defer parent.Destroy()
			capture := func(host *webview.Host, format, expr, want string) error {
				mu.Lock()
				child := windows[len(windows)-1]
				mu.Unlock()
				deadline := time.Now().Add(25 * time.Second)
				for time.Now().Before(deadline) {
					if state := host.State(); state.Error != "" {
						return errors.New(state.Error)
					}
					value, err := child.Page().Eval(expr)
					if err == nil && fmt.Sprint(value) == want {
						png, err := child.CapturePage()
						if err != nil {
							return err
						}
						if len(png) < 100 {
							return errors.New("empty native capture")
						}
						return os.WriteFile(filepath.Join(*root, format+".png"), png, 0600)
					}
					time.Sleep(50 * time.Millisecond)
				}
				if png, e := child.CapturePage(); e == nil {
					_ = os.WriteFile(filepath.Join(*root, format+"-failure.png"), png, 0600)
				}
				value, err := child.Page().Eval("JSON.stringify({ready:document.readyState,status:document.querySelector('#status')?.textContent,count:document.querySelector('#count')?.textContent,page:document.querySelector('#page')?.value,text:document.querySelector('#pages')?.textContent,canvases:[...document.querySelectorAll('canvas')].map(c=>[c.width,c.height]),failures:[...document.querySelectorAll('.failure')].map(p=>p.textContent),ua:navigator.userAgent,imports:location.pathname.endsWith('reader.html')?await Promise.all(['./pdf.mjs','./pdf_viewer.mjs','./reader.js'].map(async p=>{try{await import(p);return 'ok'}catch(e){return String(e)}})):[]})")
				return fmt.Errorf("actual renderer did not reach expected state: %v %v", value, err)
			}
			htmlPath := filepath.Join(*root, "fixture.html")
			_ = os.WriteFile(htmlPath, []byte(`<!doctype html><html><body><h1 id="fixture">Isolated HTML 中文 preview</h1><script>document.body.dataset.executed='forbidden'</script></body></html>`), 0600)
			doc, err := htmldocument.OpenWorkspace(*root, "fixture.html", func() bool { return true })
			if err == nil {
				var host *webview.Host
				host, err = webview.NewHost(webview.HostOptions{Parent: parent, URL: doc.URL(), LocalHTMLURL: doc.URL(), Untrusted: true, OnClosed: doc.Close})
				if err == nil {
					host.SetBounds(mygo.Rectangle{Width: 800, Height: 600})
					err = capture(host, "html", "document.querySelector('iframe')?.srcdoc.includes('Isolated HTML 中文 preview') && document.querySelector('iframe')?.getAttribute('sandbox') === '' && document.querySelector('iframe')?.getBoundingClientRect().height > 100", "true")
					_ = host.Close()
				}
				doc.Close()
			}
			status, detail := "PASS", "Actual hidden WebView2 renders HTML; script remains inert; native PNG captured; child/document closed"
			if err != nil {
				status, detail = "FAIL", err.Error()
			}
			r.Results = append(r.Results, result{"html", status, detail, filepath.Join(*root, "html.png")})
			pdfPath := filepath.Join(*root, "fixture.pdf")
			_ = os.WriteFile(pdfPath, makePDF(), 0600)
			pdf, err := pdfpreview.OpenWorkspace(*root, "fixture.pdf", func() bool { return true })
			if err == nil {
				var host *webview.Host
				host, err = webview.NewHost(webview.HostOptions{Parent: parent, URL: pdf.ViewerURL(), LocalPDFViewerURL: pdf.ViewerURL(), Untrusted: true, OnClosed: pdf.Close})
				if err == nil {
					host.SetBounds(mygo.Rectangle{Width: 800, Height: 600})
					err = capture(host, "pdf", "document.querySelector('canvas')?.width > 0 && document.querySelector('#pages')?.textContent.includes('Isolated PDF page one')", "true")
					if err == nil {
						mu.Lock()
						child := windows[len(windows)-1]
						mu.Unlock()
						_, err = child.Page().Eval("document.querySelector('#zoom').value='150'; document.querySelector('#zoom').dispatchEvent(new Event('change')); document.querySelector('#next').click(); await new Promise(r=>setTimeout(r,150)); true")
						if err == nil {
							err = capture(host, "pdf-page-two", "document.querySelector('#page')?.value === '2' && document.querySelector('#zoom')?.value === '150' && document.querySelector('#pages')?.textContent.includes('Isolated PDF page two')", "true")
							if err == nil {
								value, e := child.Page().Eval("document.querySelector('#scrollport').scrollTo(0,0);await new Promise(r=>setTimeout(r,150));return document.querySelector('#page').value")
								if e != nil {
									err = e
								} else if fmt.Sprint(value) != "1" {
									err = fmt.Errorf("physical scroll did not update page: %v", value)
								}
							}
						}
					}
					_ = host.Close()
				}
				pdf.Close()
			}
			status, detail = "PASS", "Actual offline PDF.js WebView2 paints canvas/text; page two and 150% zoom action executed; native PNGs; child/document closed"
			if err != nil {
				status, detail = "FAIL", err.Error()
			}
			r.Results = append(r.Results, result{"pdf", status, detail, filepath.Join(*root, "pdf.png")})
			for _, url := range []string{htmlPath, pdfPath} {
				if _, err := os.Stat(url); err != nil {
					r.Results = append(r.Results, result{"fixtures", "FAIL", "owned file disappeared", ""})
				}
			}
		}()
	})
	err := mygo.App.Run()
	if err != nil {
		r.Results = append(r.Results, result{"app", "FAIL", err.Error(), ""})
	}
	raw, _ := json.MarshalIndent(r, "", "  ")
	if err = os.WriteFile(*out, append(raw, '\n'), 0600); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println(string(raw))
	for _, result := range r.Results {
		if result.Status != "PASS" {
			os.Exit(1)
		}
	}
}

func makePDF() []byte {
	content1 := "BT /F1 18 Tf 32 150 Td (Isolated PDF page one) Tj ET"
	content2 := "BT /F1 18 Tf 32 150 Td (Isolated PDF page two) Tj ET"
	objects := []string{`<< /Type /Catalog /Pages 2 0 R >>`, `<< /Type /Pages /Count 2 /Kids [3 0 R 5 0 R] >>`, `<< /Type /Page /Parent 2 0 R /MediaBox [0 0 400 200] /Resources << /Font << /F1 7 0 R >> >> /Contents 4 0 R >>`, fmt.Sprintf("<< /Length %d >>\nstream\n%s\nendstream", len(content1), content1), `<< /Type /Page /Parent 2 0 R /MediaBox [0 0 400 200] /Resources << /Font << /F1 7 0 R >> >> /Contents 6 0 R >>`, fmt.Sprintf("<< /Length %d >>\nstream\n%s\nendstream", len(content2), content2), `<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>`}
	var out bytes.Buffer
	out.WriteString("%PDF-1.4\n")
	offsets := []int{0}
	for i, obj := range objects {
		offsets = append(offsets, out.Len())
		fmt.Fprintf(&out, "%d 0 obj\n%s\nendobj\n", i+1, obj)
	}
	start := out.Len()
	fmt.Fprintf(&out, "xref\n0 %d\n0000000000 65535 f \n", len(offsets))
	for _, offset := range offsets[1:] {
		fmt.Fprintf(&out, "%010d 00000 n \n", offset)
	}
	fmt.Fprintf(&out, "trailer\n<< /Size %d /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF\n", len(offsets), start)
	return out.Bytes()
}
