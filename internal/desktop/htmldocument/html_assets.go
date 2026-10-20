package htmldocument

import (
	"errors"
	"io"
	"os"
	"strconv"
	"strings"

	xhtml "golang.org/x/net/html"
	"whaleisle/internal/desktop/htmlpreview"
	"whaleisle/internal/store/storelocale"
)

const maxHTMLTokens = 200000

func prepareResources(root *os.Root, relative, source string, valid func() bool) (string, ResourceReport, error) {
	return prepareResourceMode(root, relative, source, valid, false)
}

// interactive is never selected by the static URL-serving path.
func prepareResourceMode(root *os.Root, relative, source string, valid func() bool, interactive bool) (string, ResourceReport, error) {
	g, err := newResourceGraph(root, len(source), valid)
	if err != nil {
		return "", ResourceReport{}, err
	}
	return prepareResourceGraph(g, relative, source, valid, interactive)
}

func prepareResourceGraph(g *resourceGraph, relative, source string, valid func() bool, interactive bool) (string, ResourceReport, error) {
	// A base URL would change which files the HTML author meant. Rather than
	// guess, keep that document static and explicitly decline relative assets.
	hasBase := false
	scan := xhtml.NewTokenizer(strings.NewReader(source))
	for n := 0; ; n++ {
		kind := scan.Next()
		if kind == xhtml.ErrorToken {
			if scan.Err() != io.EOF {
				return "", g.report, scan.Err()
			}
			break
		}
		if n >= maxHTMLTokens {
			return "", g.report, errors.New("HTML token limit exceeded")
		}
		if kind == xhtml.StartTagToken || kind == xhtml.SelfClosingTagToken {
			name, _ := scan.TagName()
			if string(name) == "base" {
				hasBase = true
			}
		}
	}
	if hasBase {
		g.warn(storelocale.T(storelocale.KeyStoreHtmldocumentUnsupportedBaseUrlRelativeResourcesWere))
	}
	loadImage := func(ref string) string {
		if hasBase && !embeddedReference(ref) {
			g.warn(storelocale.T(storelocale.KeyStoreHtmldocumentRelativeResourcesWereSkippedBecauseOfT))
			return "data:,"
		}
		out, err := g.image(relative, ref)
		if err != nil {
			g.warn(err.Error())
			return "data:,"
		}
		return out
	}
	loadScript := func(ref string) string {
		if hasBase && !embeddedReference(ref) {
			g.warn(storelocale.T(storelocale.KeyStoreHtmldocumentRelativeScriptsWereSkippedBecauseOfThe))
			return "data:text/javascript,"
		}
		value, err := g.script(relative, ref)
		if err != nil {
			g.warn(err.Error())
			return "data:text/javascript,"
		}
		return value
	}
	loadURL := g.asset
	loadCSS := g.stylesheet
	if hasBase {
		loadURL = func(base, ref string) (string, error) {
			if embeddedReference(ref) {
				return g.asset(base, ref)
			}
			return "", errors.New(storelocale.T(storelocale.KeyStoreHtmldocumentRelativeResourcesWereSkippedBecauseOfT))
		}
		loadCSS = func(string, string, int) (string, error) { return "", errors.New(storelocale.T(storelocale.KeyStoreHtmldocumentRelativeResourcesWereSkippedBecauseOfT)) }
	}
	z := xhtml.NewTokenizer(strings.NewReader(source))
	var out strings.Builder
	style, script, picture := false, false, 0
	appendText := func(s string) error {
		if len(s) > htmlpreview.MaxPreparedHTMLBytes-out.Len() {
			return errors.New("HTML resource snapshot exceeds encoded size limit")
		}
		out.WriteString(s)
		return nil
	}
	for n := 0; ; n++ {
		kind := z.Next()
		if kind == xhtml.ErrorToken {
			if z.Err() != io.EOF {
				return "", g.report, z.Err()
			}
			break
		}
		if n >= maxHTMLTokens {
			return "", g.report, errors.New("HTML token limit exceeded")
		}
		if !valid() {
			return "", g.report, errors.New("HTML preview owner changed")
		}
		raw := string(z.Raw())
		if kind == xhtml.StartTagToken || kind == xhtml.SelfClosingTagToken {
			t := z.Token()
			tag := t.Data
			if tag == "script" {
				script = true
				if !interactive {
					g.warn(storelocale.T(storelocale.KeyStoreHtmldocumentScriptsDisabled))
					continue
				}
				for i := range t.Attr {
					if key := strings.ToLower(t.Attr[i].Key); key == "src" || key == "href" || key == "xlink:href" {
						t.Attr[i].Val = loadScript(t.Attr[i].Val)
					}
				}
				if strings.EqualFold(attr(t, "type"), "module") {
					g.warn(storelocale.T(storelocale.KeyStoreHtmldocumentRuntimeImportFetchDependenciesOfModules))
				}
				if err := appendText(t.String()); err != nil {
					return "", g.report, err
				}
				continue
			}
			if script && !interactive {
				continue
			}
			if tag == "base" {
				continue
			}
			if tag == "picture" {
				picture++
			}
			if tag == "link" {
				rel := strings.Fields(strings.ToLower(attr(t, "rel")))
				if containsWord(rel, "stylesheet") && !containsWord(rel, "alternate") && !hasAttr(t, "disabled") {
					ref := attr(t, "href")
					css, err := loadCSS(relative, ref, 0)
					if err != nil {
						g.warn(err.Error())
						continue
					}
					attrs := []xhtml.Attribute{}
					if media := attr(t, "media"); media != "" {
						attrs = append(attrs, xhtml.Attribute{Key: "media", Val: media})
					}
					opening := xhtml.Token{Type: xhtml.StartTagToken, Data: "style", Attr: attrs}.String()
					if err := appendText(opening + safeStyle(css) + "</style>"); err != nil {
						return "", g.report, err
					}
				} else if attr(t, "href") != "" {
					g.warn(storelocale.T(storelocale.KeyStoreHtmldocumentNonPlainStylesheetLinksWereNotLoaded))
				}
				continue
			}
			attrs := make([]xhtml.Attribute, 0, len(t.Attr))
			for _, a := range t.Attr {
				key := strings.ToLower(a.Key)
				if strings.HasPrefix(key, "on") && !interactive {
					g.warn(storelocale.T(storelocale.KeyStoreHtmldocumentEventHandlerScriptsDisabled))
					continue
				}
				switch {
				case key == "style":
					// @import is invalid in a style declaration; never use it to read files.
					css, err := inlineCSS(a.Val, relative, 0, func(string, string, int) (string, error) { return "", errors.New(storelocale.T(storelocale.KeyStoreHtmldocumentTheStyleAttributeDoesNotSupportImport)) }, loadURL, g.warn)
					if err != nil {
						g.warn(storelocale.T(storelocale.KeyStoreHtmldocumentInlineStyleCouldNotBeParsed))
						continue
					}
					a.Val = css
				case key == "src" && (tag == "img" || (tag == "input" && strings.EqualFold(attr(t, "type"), "image"))):
					a.Val = loadImage(a.Val)
				case key == "srcset" && (tag == "img" || (tag == "source" && picture > 0)):
					set, ok := inlineSrcset(a.Val, loadImage)
					if !ok {
						g.warn(storelocale.T(storelocale.KeyStoreHtmldocumentUnsupportedSrcsetSyntax))
						continue
					}
					a.Val = set
				case (key == "href" || key == "xlink:href") && tag == "image":
					a.Val = loadImage(a.Val)
				case key == "poster" && tag == "video":
					a.Val = loadImage(a.Val)
				case key == "background":
					a.Val = loadImage(a.Val)
				case key == "src" || key == "srcdoc" || key == "data" || ((key == "href" || key == "xlink:href") && tag == "use"):
					if strings.HasPrefix(a.Val, "#") && tag == "use" {
						break
					}
					g.warn(storelocale.T(storelocale.KeyStoreHtmldocumentMediaOrNestedDocumentsWereNotLoaded))
					continue
				}
				attrs = append(attrs, a)
			}
			t.Attr = attrs
			raw = t.String()
			if tag == "style" {
				style = true
			}
		} else if kind == xhtml.EndTagToken {
			t := z.Token()
			if t.Data == "script" {
				script = false
				if !interactive {
					continue
				}
			}
			if script && !interactive {
				continue
			}
			if t.Data == "style" {
				style = false
			}
			if t.Data == "picture" && picture > 0 {
				picture--
			}
		} else if script {
			if !interactive {
				continue
			}
		} else if style && kind == xhtml.TextToken {
			css, err := inlineCSS(raw, relative, 0, loadCSS, loadURL, g.warn)
			if err != nil {
				g.warn(storelocale.T(storelocale.KeyStoreHtmldocumentStylesheetCouldNotBeParsed))
				raw = ""
			} else {
				raw = safeStyle(css)
			}
		}
		if err := appendText(raw); err != nil {
			return "", g.report, err
		}
	}
	return out.String(), g.report, nil
}
func attr(t xhtml.Token, key string) string {
	for _, a := range t.Attr {
		if a.Key == key {
			return a.Val
		}
	}
	return ""
}
func hasAttr(t xhtml.Token, key string) bool {
	for _, a := range t.Attr {
		if a.Key == key {
			return true
		}
	}
	return false
}
func containsWord(words []string, want string) bool {
	for _, word := range words {
		if word == want {
			return true
		}
	}
	return false
}
func embeddedReference(ref string) bool {
	ref = strings.TrimSpace(ref)
	return strings.HasPrefix(ref, "#") || strings.HasPrefix(strings.ToLower(ref), "data:")
}
func safeStyle(css string) string { return strings.ReplaceAll(css, "</", `\3c /`) }

// Keep bounded ordinary srcset candidates; URLs containing commas are supported
// when followed by whitespace, as required by the HTML srcset tokenization.
func inlineSrcset(source string, load func(string) string) (string, bool) {
	var result []string
	for len(strings.TrimSpace(source)) > 0 {
		if len(result) >= 64 {
			return "", false
		}
		source = strings.TrimLeft(source, " \t\r\n\f,")
		if source == "" {
			break
		}
		i := strings.IndexAny(source, " \t\r\n\f")
		if i < 0 {
			i = len(source)
		}
		ref := source[:i]
		source = source[i:]
		descriptor := ""
		if strings.HasSuffix(ref, ",") {
			ref = strings.TrimRight(ref, ",")
		} else {
			if j := strings.IndexByte(source, ','); j >= 0 {
				descriptor = strings.TrimSpace(source[:j])
				source = source[j+1:]
			} else {
				descriptor = strings.TrimSpace(source)
				source = ""
			}
		}
		if ref == "" {
			return "", false
		}
		fields := strings.Fields(descriptor)
		if len(fields) > 1 {
			return "", false
		}
		if descriptor != "" {
			if len(descriptor) < 2 {
				return "", false
			}
			suffix := descriptor[len(descriptor)-1]
			v, err := strconv.ParseFloat(descriptor[:len(descriptor)-1], 64)
			if err != nil || !(v > 0) || (suffix != 'x' && suffix != 'w') {
				return "", false
			}
			if suffix == 'w' && strings.ContainsAny(descriptor, ".eE+") {
				return "", false
			}
		}
		ref = load(ref)
		if descriptor != "" {
			ref += " " + descriptor
		}
		result = append(result, ref)
	}
	return strings.Join(result, ", "), len(result) > 0
}
