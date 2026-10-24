package chat

import (
	"encoding/json"
	"math"
	"reflect"
	"strings"
	"testing"

	"whaleisle/internal/store/session"
)

const (
	toolImageModelEnvelopeForTest   = "<path>envelope only</path>\n<type>image</type>\n<content>\nbody\n</content>"
	toolImageModelAttachmentForTest = `{"attachmentId":"provider:id","mediaType":"image/png","bytes":12,"width":3,"height":4}`
	toolImageModelArgsForTest       = `{"file_path":" authored path "}`
	toolImageModelMetaForTest       = `{"path":" recorded path "}`
)

func toolImageModelJSONForTest(t *testing.T, value any) string {
	t.Helper()
	raw, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

func toolImageModelContentForTest(t *testing.T, extra ...string) string {
	t.Helper()
	parts := []string{`{"type":"text","text":` + toolImageModelJSONForTest(t, toolImageModelEnvelopeForTest) + `}`, `{"type":"image","attachment":` + toolImageModelAttachmentForTest + `}`}
	return "[" + strings.Join(append(parts, extra...), ",") + "]"
}

// These are actual retained root and nested event shapes. Deliberately unrelated
// projection/flattened fields ensure the model uses the paired wire records.
func toolImageModelCallForTest(t *testing.T, nested bool, name, args, meta, content string) *session.ToolCallView {
	t.Helper()
	metaField := ""
	if meta != "" {
		metaField = `,"meta":` + meta
	}
	headType, resultType := "tool/call", "tool/result"
	headData := `{"callId":"image-122","name":` + toolImageModelJSONForTest(t, name) + `,"arguments":` + toolImageModelJSONForTest(t, args) + `}`
	resultData := `{"message":{"toolCallId":"image-122","source":{"kind":"tool","callId":"image-122"},"content":` + content + `}` + metaField + `}`
	if nested {
		headType, resultType = "tool/ptc-dispatch-start", "tool/ptc-dispatch"
		headData = `{"rootCallId":"root","parentCallId":"parent","subCallId":"image-122","name":` + toolImageModelJSONForTest(t, name) + `,"arguments":` + args + `}`
		resultData = strings.TrimSuffix(headData, "}") + `,"content":` + content + metaField + `}`
	}
	call := &session.ToolCallView{
		CallID: "image-122", Name: name, Arguments: args, State: "ok",
		Call: &session.Message{Kind: session.MessageKindTool, EventType: headType, Seq: 1,
			Raw:  json.RawMessage(`{"type":` + toolImageModelJSONForTest(t, headType) + `,"data":` + headData + `}`),
			Body: "flattened head is unrelated", Args: `{"file_path":"wrong head Args"}`},
		Result: &session.Message{Kind: session.MessageKindTool, EventType: resultType, Seq: 2,
			Raw:  json.RawMessage(`{"type":` + toolImageModelJSONForTest(t, resultType) + `,"data":` + resultData + `}`),
			Body: "flattened result is unrelated", Args: `{"file_path":"wrong result Args"}`},
	}
	if nested {
		call.ParentCallID, call.RootCallID = "parent", "root"
	}
	return call
}

func toolImageModelWithAttachmentForTest(t *testing.T, nested bool, attachment string) *session.ToolCallView {
	t.Helper()
	content := `[{"type":"text","text":` + toolImageModelJSONForTest(t, toolImageModelEnvelopeForTest) + `},{"type":"image","attachment":` + attachment + `}]`
	return toolImageModelCallForTest(t, nested, "read_image", toolImageModelArgsForTest, toolImageModelMetaForTest, content)
}

func toolImageModelWantForTest(path string, hasPath bool, extra *string) *toolImageModel {
	return &toolImageModel{Path: path, HasPath: hasPath, Images: []toolImageAttachment{{AttachmentID: "provider:id", MediaType: "image/png", Bytes: 12, Width: 3, Height: 4}}, ExtraText: extra}
}

// Installed goldens below were recorded by executing the CURRENT installed
// image-card-model.js and raw-tool-call.js in isolated node:vm SourceTextModules:
// image-card-model.js SHA256 59ad72e7be1465b129926a2ad1c4010e4f091b3cb4ac3f1df89adb2063a162b0
// raw-tool-call.js SHA256 c97f8fa5e1b85e63b2bb4c65c1b7bbb265793dd0631ec0a3f32c38b3c15b44a8
// The tests retain the installed output, require no Node/browser/filesystem, and
// adapt only the installed images[].attachment wrapper to the requested Go API.
type toolImageModelGoldenForTest struct {
	Path   *string `json:"path"`
	Images []struct {
		Attachment struct {
			AttachmentID       string  `json:"attachmentId"`
			MediaType          string  `json:"mediaType"`
			Bytes              float64 `json:"bytes"`
			Width              float64 `json:"width"`
			Height             float64 `json:"height"`
			Name               *string `json:"name"`
			OriginalDimensions *struct {
				Width  float64 `json:"width"`
				Height float64 `json:"height"`
			} `json:"originalDimensions"`
		} `json:"attachment"`
	} `json:"images"`
	ExtraText *string `json:"extraText"`
}

func (golden *toolImageModelGoldenForTest) model() *toolImageModel {
	if golden == nil {
		return nil
	}
	model := &toolImageModel{ExtraText: golden.ExtraText}
	if golden.Path != nil {
		model.Path, model.HasPath = *golden.Path, true
	}
	for _, item := range golden.Images {
		ref := item.Attachment
		image := toolImageAttachment{AttachmentID: ref.AttachmentID, MediaType: ref.MediaType, Bytes: ref.Bytes, Width: ref.Width, Height: ref.Height, Name: ref.Name}
		if ref.OriginalDimensions != nil {
			image.OriginalDimensions = &toolImageDimensions{Width: ref.OriginalDimensions.Width, Height: ref.OriginalDimensions.Height}
		}
		model.Images = append(model.Images, image)
	}
	return model
}

func TestToolImageModelInstalledGoldens(t *testing.T) {
	var fixtures []struct {
		Name, Tool, ArgsRaw, MetaRaw, ContentRaw string
		Nested, OmitMeta, NoCall, IsError        bool
		Specialized, Generic                     *toolImageModelGoldenForTest
	}
	if err := json.Unmarshal([]byte(toolImageModelInstalledGoldensForTest), &fixtures); err != nil {
		t.Fatal(err)
	}
	for _, fixture := range fixtures {
		t.Run(fixture.Name, func(t *testing.T) {
			meta := fixture.MetaRaw
			if fixture.OmitMeta {
				meta = ""
			}
			call := toolImageModelCallForTest(t, fixture.Nested, fixture.Tool, fixture.ArgsRaw, meta, fixture.ContentRaw)
			if fixture.NoCall {
				call.Call = nil
				if fixture.Nested {
					// A direct model fixture with no embedded call differs from a
					// real childResult, which synthesizes it from receipt name/args.
					fields, _ := terminalModelObject(call.Result.Raw)
					data, _ := terminalModelObject(fields["data"])
					delete(data, "name")
					fields["data"] = json.RawMessage(toolImageModelJSONForTest(t, data))
					call.Result.Raw = json.RawMessage(toolImageModelJSONForTest(t, fields))
				}
			}
			if fixture.IsError {
				call.State = "error"
			}
			before := toolImageModelJSONForTest(t, call)
			if got, want := toolImageModelFor(call), fixture.Specialized.model(); !reflect.DeepEqual(got, want) {
				t.Fatalf("specialized = %#v, want %#v", got, want)
			}
			if got, want := toolGenericImageModelFor(call), fixture.Generic.model(); !reflect.DeepEqual(got, want) {
				t.Fatalf("generic = %#v, want %#v", got, want)
			}
			if toolImageModelJSONForTest(t, call) != before {
				t.Fatal("derivation rewrote a retained source record")
			}
		})
	}
}

func TestToolImageModelMetadataAndNestedFallback(t *testing.T) {
	for _, meta := range []string{"", `null`, `[]`, `true`, `1`, `"meta"`, `{}`, `{"path":null}`, `{"path":1}`, `{"path":[]}`, `{"path":""}`} {
		for _, nested := range []bool{false, true} {
			t.Run(toolImageModelJSONForTest(t, []any{nested, meta}), func(t *testing.T) {
				call := toolImageModelCallForTest(t, nested, "read_image", toolImageModelArgsForTest, meta, toolImageModelContentForTest(t))
				got := toolImageModelFor(call)
				if nested {
					if want := toolImageModelWantForTest(" authored path ", true, nil); !reflect.DeepEqual(got, want) {
						t.Fatalf("invalid nested metadata did not use authored path: %#v, want %#v", got, want)
					}
				} else if got != nil {
					t.Fatalf("invalid root metadata produced image card: %#v", got)
				}
				if toolGenericImageModelFor(call) == nil {
					t.Fatal("generic gallery incorrectly required metadata")
				}
			})
		}
	}
	for _, path := range []string{" \t\r\n", "relative/../unchanged.png", "javascript:opaque", "C:\\Mixed/path\r\n\x00\ufeff"} {
		for _, nested := range []bool{false, true} {
			meta := `{"path":` + toolImageModelJSONForTest(t, path) + `,"ignored":1e9999}`
			call := toolImageModelCallForTest(t, nested, "read_image", toolImageModelArgsForTest, meta, toolImageModelContentForTest(t))
			if got := toolImageModelFor(call); got == nil || got.Path != path || !got.HasPath {
				t.Fatalf("recorded path spelling changed: %#v, want %q", got, path)
			}
		}
	}
}

func TestToolImageModelCallHeadAndJavaScriptTrim(t *testing.T) {
	for _, args := range []string{"", "{", `null`, `[]`, `true`, `1`, `"args"`, `{}`, `{"file_path":null}`, `{"file_path":1}`, `{"file_path":true}`, `{"file_path":[]}`, `{"file_path":{}}`, `{"file_path":"x"} {}`, `{"file_path":"x",}`} {
		call := toolImageModelCallForTest(t, false, "read_image", args, toolImageModelMetaForTest, toolImageModelContentForTest(t))
		if got := toolImageModelFor(call); got != nil {
			t.Fatalf("invalid whole argument object %s produced %#v", args, got)
		}
		if toolGenericImageModelFor(call) == nil {
			t.Fatalf("generic incorrectly parsed args %s", args)
		}
	}
	for _, whitespace := range []string{"", "\t", "\n", "\v", "\f", "\r", " ", "\u00a0", "\u1680", "\u2000", "\u2001", "\u2002", "\u2003", "\u2004", "\u2005", "\u2006", "\u2007", "\u2008", "\u2009", "\u200a", "\u2028", "\u2029", "\u202f", "\u205f", "\u3000", "\ufeff"} {
		for _, nonblank := range []bool{false, true} {
			path := whitespace
			if nonblank {
				path += "x" + whitespace
			}
			args := `{"file_path":` + toolImageModelJSONForTest(t, path) + `,"ignored":1e9999}`
			call := toolImageModelCallForTest(t, false, "read_image", args, toolImageModelMetaForTest, toolImageModelContentForTest(t))
			if got := toolImageModelFor(call); (got != nil) != nonblank {
				t.Fatalf("path=%q nonblank=%v model=%#v", path, nonblank, got)
			}
		}
	}
	for _, path := range []string{"\u0085", "\u180e", "\u200b", "\x00"} {
		args := `{"file_path":` + toolImageModelJSONForTest(t, path) + `}`
		call := toolImageModelCallForTest(t, true, "read_image", args, "", toolImageModelContentForTest(t))
		if got := toolImageModelFor(call); got == nil || got.Path != path {
			t.Fatalf("non-JS-trim path %q declined or changed: %#v", path, got)
		}
	}
	for _, name := range []string{"", "read", "Read_image", "read_image ", "functions.read_image", "plugin:read_image", "screenshot"} {
		call := toolImageModelCallForTest(t, false, name, toolImageModelArgsForTest, toolImageModelMetaForTest, toolImageModelContentForTest(t))
		if got := toolImageModelFor(call); got != nil {
			t.Fatalf("nonexact tool name %q acquired read_image model: %#v", name, got)
		}
		if toolGenericImageModelFor(call) == nil {
			t.Fatalf("generic incorrectly required read_image for %q", name)
		}
	}
}

func TestToolImageModelAllAttachmentReferencesMustValidate(t *testing.T) {
	for _, attachment := range []string{"", `null`, `[]`, `true`, `1`, `"ref"`, `{}`} {
		if attachment == "" {
			continue
		}
		for _, nested := range []bool{false, true} {
			call := toolImageModelWithAttachmentForTest(t, nested, attachment)
			if toolImageModelFor(call) != nil || toolGenericImageModelFor(call) != nil {
				t.Fatalf("invalid attachment %s produced gallery", attachment)
			}
		}
	}
	for _, field := range []string{"attachmentId", "mediaType", "bytes", "width", "height"} {
		fields := map[string]json.RawMessage{}
		if err := json.Unmarshal([]byte(toolImageModelAttachmentForTest), &fields); err != nil {
			t.Fatal(err)
		}
		delete(fields, field)
		call := toolImageModelWithAttachmentForTest(t, false, toolImageModelJSONForTest(t, fields))
		if toolImageModelFor(call) != nil || toolGenericImageModelFor(call) != nil {
			t.Fatalf("missing %s produced gallery", field)
		}
		for _, value := range []string{`null`, `true`, `[]`, `{}`} {
			fields[field] = json.RawMessage(value)
			call := toolImageModelWithAttachmentForTest(t, false, toolImageModelJSONForTest(t, fields))
			if toolImageModelFor(call) != nil || toolGenericImageModelFor(call) != nil {
				t.Fatalf("invalid %s=%s produced gallery", field, value)
			}
		}
	}
	for _, field := range []string{"name", "originalDimensions"} {
		for _, value := range []string{`null`, `1`, `true`, `[]`, `{}`} {
			attachment := strings.TrimSuffix(toolImageModelAttachmentForTest, "}") + `,"` + field + `":` + value + `}`
			call := toolImageModelWithAttachmentForTest(t, false, attachment)
			if toolImageModelFor(call) != nil || toolGenericImageModelFor(call) != nil {
				t.Fatalf("invalid optional %s=%s produced gallery", field, value)
			}
		}
	}
	for _, dimensions := range []string{`{"width":1}`, `{"height":1}`, `{"width":0,"height":1}`, `{"width":1,"height":0}`, `{"width":1,"height":1.5}`, `{"width":1,"height":null}`, `{"width":1e-9999,"height":1}`, `{"width":1e9999,"height":1}`} {
		attachment := strings.TrimSuffix(toolImageModelAttachmentForTest, "}") + `,"originalDimensions":` + dimensions + `}`
		call := toolImageModelWithAttachmentForTest(t, false, attachment)
		if toolImageModelFor(call) != nil || toolGenericImageModelFor(call) != nil {
			t.Fatalf("invalid original dimensions %s produced gallery", dimensions)
		}
	}
	for _, media := range []string{"image/png", "image/jpeg", "image/webp", "image/gif", "", "IMAGE/PNG", "image/avif", "image/png ", "text/plain"} {
		attachment := strings.TrimSuffix(toolImageModelAttachmentForTest, "}") + `,"mediaType":` + toolImageModelJSONForTest(t, media) + `}`
		call := toolImageModelWithAttachmentForTest(t, false, attachment)
		valid := media == "image/png" || media == "image/jpeg" || media == "image/webp" || media == "image/gif"
		if got := toolImageModelFor(call); (got != nil) != valid {
			t.Fatalf("media %q valid=%v model=%#v", media, valid, got)
		}
		if got := toolGenericImageModelFor(call); (got != nil) != valid {
			t.Fatalf("generic media %q valid=%v model=%#v", media, valid, got)
		}
	}
	// A valid first reference cannot hide a later malformed one.
	call := toolImageModelCallForTest(t, false, "read_image", toolImageModelArgsForTest, toolImageModelMetaForTest, toolImageModelContentForTest(t, `{"type":"image","attachment":null}`))
	if toolImageModelFor(call) != nil || toolGenericImageModelFor(call) != nil {
		t.Fatal("malformed later image was silently hidden")
	}
}

func TestToolImageModelPositiveJavaScriptIntegersAndOpaqueIDs(t *testing.T) {
	for _, measure := range []struct {
		raw  string
		want float64
	}{
		{"1", 1}, {"1e1", 10}, {"10.0", 10}, {"1.00000000000000001", 1}, {"1.99999999999999999", 2},
		{"9007199254740993", 9007199254740992}, {"9223372036854775807", 9223372036854775808},
		{"1e300", 1e300}, {"1.7976931348623157e308", math.MaxFloat64},
	} {
		attachment := `{"attachmentId":" \t\r\n\u0000opaque:id ","mediaType":"image/png","bytes":` + measure.raw + `,"width":` + measure.raw + `,"height":` + measure.raw + `,"name":"","originalDimensions":{"width":` + measure.raw + `,"height":` + measure.raw + `},"ignored":1e9999}`
		call := toolImageModelWithAttachmentForTest(t, false, attachment)
		for _, got := range []*toolImageModel{toolImageModelFor(call), toolGenericImageModelFor(call)} {
			if got == nil || len(got.Images) != 1 {
				t.Fatalf("positive JS integer %s declined: %#v", measure.raw, got)
			}
			ref := got.Images[0]
			if ref.AttachmentID != " \t\r\n\x00opaque:id " || ref.Bytes != measure.want || ref.Width != measure.want || ref.Height != measure.want || ref.Name == nil || *ref.Name != "" || !reflect.DeepEqual(ref.OriginalDimensions, &toolImageDimensions{measure.want, measure.want}) {
				t.Fatalf("positive JS integer %s or opaque fields changed: %#v", measure.raw, ref)
			}
		}
	}
	for _, field := range []string{"bytes", "width", "height"} {
		for _, value := range []string{`0`, `-0`, `-1`, `1.5`, `1e-300`, `1e-9999`, `-1e-9999`, `1e9999`, `-1e9999`, `"1"`} {
			attachment := strings.TrimSuffix(toolImageModelAttachmentForTest, "}") + `,"` + field + `":` + value + `}`
			call := toolImageModelWithAttachmentForTest(t, false, attachment)
			if toolImageModelFor(call) != nil || toolGenericImageModelFor(call) != nil {
				t.Fatalf("nonpositive/noninteger %s=%s produced gallery", field, value)
			}
		}
	}
	for _, id := range []string{"", " ", "\t\n", "provider-owned", "https://attachment.invalid/opaque", "sha256:not necessarily a digest"} {
		attachment := strings.TrimSuffix(toolImageModelAttachmentForTest, "}") + `,"attachmentId":` + toolImageModelJSONForTest(t, id) + `}`
		call := toolImageModelWithAttachmentForTest(t, false, attachment)
		for _, got := range []*toolImageModel{toolImageModelFor(call), toolGenericImageModelFor(call)} {
			if (got != nil) != (id != "") || got != nil && got.Images[0].AttachmentID != id {
				t.Fatalf("opaque ID %q was parsed or trimmed: %#v", id, got)
			}
		}
	}
}

func TestToolImageModelExactEnvelopeAndExtraText(t *testing.T) {
	for _, fixture := range []struct {
		name, text string
		valid      bool
	}{
		{"standard", toolImageModelEnvelopeForTest, true},
		{"empty path and body", "<path></path>\n<type>image</type>\n<content>\n\n</content>", true},
		{"CR unicode path", "<path>a\r\u2028\u2029b</path>\n<type>image</type>\n<content>\nx\n</content>", true},
		{"tags in path", "<path>a</path>tail</path>\n<type>image</type>\n<content>\nx\n</content>", true},
		{"any body", "<path>a</path>\n<type>image</type>\n<content>\n</content>\n\x00\r\u2028\u2029\n</content>", true},
		{"prefix", "prefix" + toolImageModelEnvelopeForTest, false},
		{"leading LF", "\n" + toolImageModelEnvelopeForTest, false},
		{"BOM", "\ufeff" + toolImageModelEnvelopeForTest, false},
		{"suffix", toolImageModelEnvelopeForTest + "suffix", false},
		{"trailing LF", toolImageModelEnvelopeForTest + "\n", false},
		{"trailing CR", toolImageModelEnvelopeForTest + "\r", false},
		{"trailing CRLF", toolImageModelEnvelopeForTest + "\r\n", false},
		{"trailing line separator", toolImageModelEnvelopeForTest + "\u2028", false},
		{"trailing paragraph separator", toolImageModelEnvelopeForTest + "\u2029", false},
		{"path LF", strings.Replace(toolImageModelEnvelopeForTest, "envelope only", "a\nb", 1), false},
		{"structural CRLF", strings.ReplaceAll(toolImageModelEnvelopeForTest, "\n", "\r\n"), false},
		{"missing body separator", "<path>a</path>\n<type>image</type>\n<content>\n</content>", false},
		{"wrong type", strings.Replace(toolImageModelEnvelopeForTest, "image", "file", 1), false},
		{"tag spacing", strings.Replace(toolImageModelEnvelopeForTest, "<content>", "<content >", 1), false},
	} {
		t.Run(fixture.name, func(t *testing.T) {
			content := `[{"type":"text","text":` + toolImageModelJSONForTest(t, fixture.text) + `},{"type":"image","attachment":` + toolImageModelAttachmentForTest + `}]`
			call := toolImageModelCallForTest(t, false, "read_image", toolImageModelArgsForTest, toolImageModelMetaForTest, content)
			if got := toolImageModelFor(call); (got != nil) != fixture.valid {
				t.Fatalf("envelope valid=%v model=%#v", fixture.valid, got)
			}
			if got := toolGenericImageModelFor(call); got == nil || got.ExtraText == nil || *got.ExtraText != fixture.text {
				t.Fatalf("generic altered the verbatim envelope/text: %#v", got)
			}
		})
	}
	for _, fixture := range []struct {
		extra []string
		want  *string
	}{
		{nil, nil},
		{[]string{`{"type":"text","text":""}`}, func() *string { text := ""; return &text }()},
		{[]string{`{"type":"text","text":""}`, `{"type":"text","text":"tail\r\n"}`}, func() *string { text := "\ntail\r\n"; return &text }()},
		{[]string{`{"type":"text","text":` + toolImageModelJSONForTest(t, toolImageModelEnvelopeForTest) + `}`, `{"type":"text","text":"tail"}`}, func() *string { text := "tail"; return &text }()},
	} {
		call := toolImageModelCallForTest(t, false, "read_image", toolImageModelArgsForTest, toolImageModelMetaForTest, toolImageModelContentForTest(t, fixture.extra...))
		if got := toolImageModelFor(call); got == nil || !reflect.DeepEqual(got.ExtraText, fixture.want) {
			t.Fatalf("extra text nil/empty/order mismatch: %#v, want %v", got, fixture.want)
		}
	}
}

func TestToolGenericImageModelUnknownBlocksAndJSONSemantics(t *testing.T) {
	for _, fixture := range []struct {
		extra, want string
		hasText     bool
	}{
		{`null`, "", false}, {`1`, "", false}, {`true`, "", false}, {`"primitive"`, "", false},
		{`[]`, "[]", true}, {`[null,1,{"z":-0}]`, "[\n  null,\n  1,\n  {\n    \"z\": 0\n  }\n]", true},
		{`{}`, "{}", true}, {`{"type":"reasoning","text":"evidence"}`, "{\n  \"type\": \"reasoning\",\n  \"text\": \"evidence\"\n}", true},
		{`{"type":"text","text":null}`, "{\n  \"type\": \"text\",\n  \"text\": null\n}", true},
		{`{"type":"text","text":1}`, "{\n  \"type\": \"text\",\n  \"text\": 1\n}", true},
		{`{"type":"text","text":""}`, "", true},
		{`{"type":"text","text":" exact\r\n\u0000 "}`, " exact\r\n\x00 ", true},
		{`{"z":"first","10":"ten","2":"two","z":"last","nested":[-0,9007199254740993,1e9999,1e-7],"literal":"<>&\u2028\u2029"}`, "{\n  \"2\": \"two\",\n  \"10\": \"ten\",\n  \"z\": \"last\",\n  \"nested\": [\n    0,\n    9007199254740992,\n    null,\n    1e-7\n  ],\n  \"literal\": \"<>&\u2028\u2029\"\n}", true},
	} {
		t.Run(fixture.extra, func(t *testing.T) {
			content := `[{"type":"image","attachment":` + toolImageModelAttachmentForTest + `},` + fixture.extra + `]`
			call := toolImageModelCallForTest(t, false, "anything", "invalid arguments", "null", content)
			call.Call = nil
			got := toolGenericImageModelFor(call)
			if got == nil || got.HasPath || got.Path != "" || (got.ExtraText != nil) != fixture.hasText || got.ExtraText != nil && *got.ExtraText != fixture.want {
				t.Fatalf("generic unknown-block evidence = %#v, want text=%q presence=%v", got, fixture.want, fixture.hasText)
			}
		})
	}
	for _, extra := range []string{`null`, `1`, `true`, `"primitive"`, `[]`, `{}`, `{"type":"reasoning"}`, `{"type":"text"}`, `{"type":"text","text":null}`} {
		call := toolImageModelCallForTest(t, false, "read_image", toolImageModelArgsForTest, toolImageModelMetaForTest, toolImageModelContentForTest(t, extra))
		if got := toolImageModelFor(call); got != nil {
			t.Fatalf("specialized silently hid unknown block %s: %#v", extra, got)
		}
		if toolGenericImageModelFor(call) == nil {
			t.Fatalf("generic failed to retain usable images beside %s", extra)
		}
	}
}

func TestToolImageModelSettledRawReceiptAndProjectedErrors(t *testing.T) {
	for _, nested := range []bool{false, true} {
		for _, state := range []string{"", "preparing", "running", "error", "failed", "stopped", "cancelled", "canceled", "OK", "success"} {
			call := toolImageModelCallForTest(t, nested, "read_image", toolImageModelArgsForTest, toolImageModelMetaForTest, toolImageModelContentForTest(t))
			call.State = state
			if toolImageModelFor(call) != nil || toolGenericImageModelFor(call) != nil {
				t.Fatalf("projected state %q produced gallery", state)
			}
		}
		for _, failure := range []string{`,"isError":true`, `,"error":{"code":"failed"}`, `,"error":{"code":"interrupted"}`, `,"error":false`, `,"error":"failure"`} {
			call := toolImageModelCallForTest(t, nested, "read_image", toolImageModelArgsForTest, toolImageModelMetaForTest, toolImageModelContentForTest(t))
			call.Result.Raw = json.RawMessage(strings.TrimSuffix(string(call.Result.Raw), "}}") + failure + "}}")
			if toolImageModelFor(call) != nil || toolGenericImageModelFor(call) != nil {
				t.Fatalf("wire failure %s overruled by projected success", failure)
			}
		}
		call := toolImageModelCallForTest(t, nested, "read_image", toolImageModelArgsForTest, toolImageModelMetaForTest, toolImageModelContentForTest(t))
		call.Result.Raw = json.RawMessage(strings.TrimSuffix(string(call.Result.Raw), "}}") + `,"error":null}}`)
		if toolImageModelFor(call) == nil || toolGenericImageModelFor(call) == nil {
			t.Fatal("explicit null error should not decline successful result")
		}
	}
	call := toolImageModelCallForTest(t, false, "read_image", toolImageModelArgsForTest, toolImageModelMetaForTest, toolImageModelContentForTest(t))
	call.Result.Raw = json.RawMessage(strings.Replace(string(call.Result.Raw), `"message":{`, `"message":{"isError":true,`, 1))
	if toolImageModelFor(call) != nil || toolGenericImageModelFor(call) != nil {
		t.Fatal("root message.isError overruled by projected success")
	}
	for _, raw := range []string{"", "{", `null`, `[]`, `true`, `1`, `"receipt"`, `{}`, `{"type":"unknown","data":{}}`, `{"type":"tool/result"}`, `{"type":"tool/result","data":null}`, `{"type":"tool/result","data":[]}`, `{"type":"tool/result","data":{"message":null}}`} {
		call := toolImageModelCallForTest(t, false, "read_image", toolImageModelArgsForTest, toolImageModelMetaForTest, toolImageModelContentForTest(t))
		call.Result.Raw = json.RawMessage(raw)
		if toolImageModelFor(call) != nil || toolGenericImageModelFor(call) != nil {
			t.Fatalf("malformed raw receipt %s produced gallery", raw)
		}
	}
	for _, content := range []string{`null`, `{}`, `"content"`, `[]`, `[null]`, `[{"type":"image"}]`, `[{"type":"text","text":"no images"}]`} {
		call := toolImageModelCallForTest(t, false, "read_image", toolImageModelArgsForTest, toolImageModelMetaForTest, content)
		if toolImageModelFor(call) != nil || toolGenericImageModelFor(call) != nil {
			t.Fatalf("unusable raw content %s produced gallery", content)
		}
	}
	for _, derive := range []func(*session.ToolCallView) *toolImageModel{toolImageModelFor, toolGenericImageModelFor} {
		if derive(nil) != nil || derive(&session.ToolCallView{State: "ok"}) != nil {
			t.Fatal("missing result produced gallery")
		}
		call := toolImageModelCallForTest(t, false, "read_image", toolImageModelArgsForTest, toolImageModelMetaForTest, toolImageModelContentForTest(t))
		call.Result.EventType = "tool/ptc-dispatch"
		if derive(call) != nil {
			t.Fatal("EventType/raw result mismatch produced gallery")
		}
		call.Result.EventType = ""
		if derive(call) == nil {
			t.Fatal("omitted EventType should use raw result type")
		}
	}
}

func TestToolImageModelRawHeadOwnsPresentation(t *testing.T) {
	for _, nested := range []bool{false, true} {
		call := toolImageModelCallForTest(t, nested, "read_image", toolImageModelArgsForTest, toolImageModelMetaForTest, toolImageModelContentForTest(t))
		call.Name, call.Arguments = "projection is not raw call", `{"file_path":"projected lie"}`
		call.Call.Body, call.Result.Body = "flattened lie", "flattened lie"
		call.Call.Args, call.Result.Args = "invalid flattened args", "invalid flattened args"
		// Parent/root projection does not turn a raw root receipt into a nested
		// receipt or erase the real parent of a nested receipt.
		call.ParentCallID, call.RootCallID = "", ""
		if !nested {
			call.ParentCallID, call.RootCallID = "projected parent", "projected root"
		}
		if got, want := toolImageModelFor(call), toolImageModelWantForTest(" recorded path ", true, nil); !reflect.DeepEqual(got, want) {
			t.Fatalf("projection replaced raw presentation: %#v, want %#v", got, want)
		}
		if nested {
			call.Result.Raw = json.RawMessage(strings.Replace(string(call.Result.Raw), `"name":"read_image"`, `"name":"other"`, 1))
		} else {
			call.Call.Raw = json.RawMessage(strings.Replace(string(call.Call.Raw), `"name":"read_image"`, `"name":"other"`, 1))
		}
		call.Name = "read_image"
		if toolImageModelFor(call) != nil {
			t.Fatal("projected name granted specialized card to a different raw call")
		}
		if toolGenericImageModelFor(call) == nil {
			t.Fatal("generic unnecessarily parsed the raw name")
		}
	}
	for _, raw := range []string{"", "{", `null`, `[]`, `{}`, `{"type":"tool/call","data":null}`, `{"type":"tool/call","data":{"name":"read_image","arguments":{}}}`} {
		call := toolImageModelCallForTest(t, false, "read_image", toolImageModelArgsForTest, toolImageModelMetaForTest, toolImageModelContentForTest(t))
		call.Call.Raw = json.RawMessage(raw)
		if toolImageModelFor(call) != nil || toolGenericImageModelFor(call) == nil {
			t.Fatalf("raw head %s affected wrong gate", raw)
		}
	}
	call := toolImageModelCallForTest(t, false, "read_image", toolImageModelArgsForTest, toolImageModelMetaForTest, toolImageModelContentForTest(t))
	call.Call.EventType = "tool/ptc-dispatch-start"
	if toolImageModelFor(call) != nil || toolGenericImageModelFor(call) == nil {
		t.Fatal("head EventType/raw mismatch affected wrong gate")
	}
}

func TestToolImageModelProjectedRootNestedAndResultOnly(t *testing.T) {
	for _, nested := range []bool{false, true} {
		call := toolImageModelCallForTest(t, nested, "read_image", toolImageModelArgsForTest, "", toolImageModelContentForTest(t))
		if !nested {
			call = toolImageModelCallForTest(t, false, "read_image", toolImageModelArgsForTest, toolImageModelMetaForTest, toolImageModelContentForTest(t))
		}
		messages := []session.Message{*call.Call, *call.Result}
		before := toolImageModelJSONForTest(t, messages)
		for _, window := range []struct {
			name     string
			messages []session.Message
			retained bool
		}{
			{"paired", messages, true}, {"result only", messages[1:], false}, {"head only", messages[:1], true},
		} {
			t.Run(toolImageModelJSONForTest(t, []any{nested, window.name}), func(t *testing.T) {
				projected := session.ProjectToolMessages(window.messages)
				if len(projected) != 1 || projected[0].Tool == nil {
					t.Fatalf("actual wire records did not project: %#v", projected)
				}
				view := projected[0].Tool
				settled := window.name != "head only"
				if got := toolImageModelFor(view); (got != nil) != ((window.retained || nested) && settled) {
					t.Fatalf("retained=%v settled=%v specialized=%#v", window.retained, settled, got)
				}
				if got := toolGenericImageModelFor(view); (got != nil) != settled {
					t.Fatalf("settled=%v generic=%#v", settled, got)
				}
			})
		}
		if toolImageModelJSONForTest(t, messages) != before {
			t.Fatal("projection or presentation mutated original messages")
		}
	}
}

func TestToolImageModelSourceImmutabilityAndIndependentModels(t *testing.T) {
	attachment := strings.TrimSuffix(toolImageModelAttachmentForTest, "}") + `,"name":" name ","originalDimensions":{"width":900,"height":600}}`
	for _, derive := range []func(*session.ToolCallView) *toolImageModel{toolImageModelFor, toolGenericImageModelFor} {
		call := toolImageModelWithAttachmentForTest(t, true, attachment)
		call.Result.Raw = json.RawMessage(strings.Replace(string(call.Result.Raw), `"content":[`, `"content":[{"type":"text","text":" tail "},`, 1))
		call.Events = []session.Message{*call.Call, *call.Result}
		call.Children = []*session.ToolCallView{{CallID: "unrelated child", Name: "other", State: "running"}}
		before := toolImageModelJSONForTest(t, call)
		first, second := derive(call), derive(call)
		if first == nil || !reflect.DeepEqual(first, second) {
			t.Fatalf("repeat derivation differs: %#v, %#v", first, second)
		}
		secondBefore := toolImageModelJSONForTest(t, second)
		first.Path, first.HasPath = "changed", false
		first.Images[0].AttachmentID, first.Images[0].MediaType = "changed", "changed"
		first.Images[0].Bytes, first.Images[0].Width, first.Images[0].Height = 99, 99, 99
		*first.Images[0].Name = "changed"
		first.Images[0].OriginalDimensions.Width = 99
		first.Images = append(first.Images, toolImageAttachment{AttachmentID: "appended"})
		*first.ExtraText = "changed"
		if toolImageModelJSONForTest(t, second) != secondBefore || !reflect.DeepEqual(derive(call), second) {
			t.Fatal("returned references, names, dimensions or extra text alias another model")
		}
		if toolImageModelJSONForTest(t, call) != before {
			t.Fatal("derivation or returned-model mutation rewrote retained call/events")
		}
		call.Result.Raw = json.RawMessage(strings.Replace(string(call.Result.Raw), `"attachmentId":"provider:id"`, `"attachmentId":""`, 1))
		declinedBefore := toolImageModelJSONForTest(t, call)
		if derive(call) != nil || toolImageModelJSONForTest(t, call) != declinedBefore {
			t.Fatal("declined derivation reused stale data or mutated source")
		}
	}
}

// Populated from the independently executed installed module, never from the Go
// implementation. Each line is one complete fixture to keep inputs reviewable.
const toolImageModelInstalledGoldensForTest = `[
{"name":"root","tool":"read_image","argsRaw":"{\"file_path\":\" authored path \"}","metaRaw":"{\"path\":\" recorded path \"}","contentRaw":"[{\"type\":\"text\",\"text\":\"<path>envelope only</path>\\n<type>image</type>\\n<content>\\nbody\\n</content>\"},{\"type\":\"image\",\"attachment\":{\"attachmentId\":\"provider:id\",\"mediaType\":\"image/png\",\"bytes\":12,\"width\":3,\"height\":4}}]","specialized":{"path":" recorded path ","images":[{"attachment":{"attachmentId":"provider:id","mediaType":"image/png","bytes":12,"width":3,"height":4}}],"extraText":null},"generic":{"images":[{"attachment":{"attachmentId":"provider:id","mediaType":"image/png","bytes":12,"width":3,"height":4}}],"extraText":"<path>envelope only</path>\n<type>image</type>\n<content>\nbody\n</content>"}},
{"name":"two images named original dimensions","tool":"read_image","argsRaw":"{\"file_path\":\" authored path \"}","metaRaw":"{\"path\":\" recorded path \"}","contentRaw":"[{\"type\":\"image\",\"attachment\":{\"attachmentId\":\" \\t\\r\\n\\u0000id \",\"mediaType\":\"image/webp\",\"bytes\":1e300,\"width\":9007199254740993,\"height\":1.99999999999999999,\"name\":\"\",\"originalDimensions\":{\"width\":1e300,\"height\":9223372036854775807},\"ignored\":1e9999}},{\"type\":\"text\",\"text\":\"<path>envelope only</path>\\n<type>image</type>\\n<content>\\nbody\\n</content>\"},{\"type\":\"image\",\"attachment\":{\"attachmentId\":\"external:second\",\"mediaType\":\"image/gif\",\"bytes\":1,\"width\":2,\"height\":3,\"name\":\" raw\\r\\nname \"}},{\"type\":\"text\",\"text\":\"\"},{\"type\":\"text\",\"text\":\" tail\\r\\n\"}]","specialized":{"path":" recorded path ","images":[{"attachment":{"attachmentId":" \t\r\n\u0000id ","mediaType":"image/webp","bytes":1e+300,"width":9007199254740992,"height":2,"name":"","originalDimensions":{"width":1e+300,"height":9223372036854776000}}},{"attachment":{"attachmentId":"external:second","mediaType":"image/gif","bytes":1,"width":2,"height":3,"name":" raw\r\nname "}}],"extraText":"\n tail\r\n"},"generic":{"images":[{"attachment":{"attachmentId":" \t\r\n\u0000id ","mediaType":"image/webp","bytes":1e+300,"width":9007199254740992,"height":2,"name":"","originalDimensions":{"width":1e+300,"height":9223372036854776000}}},{"attachment":{"attachmentId":"external:second","mediaType":"image/gif","bytes":1,"width":2,"height":3,"name":" raw\r\nname "}}],"extraText":"<path>envelope only</path>\n<type>image</type>\n<content>\nbody\n</content>\n\n tail\r\n"}},
{"name":"whitespace metadata path","tool":"read_image","argsRaw":"{\"file_path\":\" authored path \"}","metaRaw":"{\"path\":\" \\t\\r\\n\"}","contentRaw":"[{\"type\":\"text\",\"text\":\"<path>envelope only</path>\\n<type>image</type>\\n<content>\\nbody\\n</content>\"},{\"type\":\"image\",\"attachment\":{\"attachmentId\":\"provider:id\",\"mediaType\":\"image/png\",\"bytes\":12,\"width\":3,\"height\":4}}]","specialized":{"path":" \t\r\n","images":[{"attachment":{"attachmentId":"provider:id","mediaType":"image/png","bytes":12,"width":3,"height":4}}],"extraText":null},"generic":{"images":[{"attachment":{"attachmentId":"provider:id","mediaType":"image/png","bytes":12,"width":3,"height":4}}],"extraText":"<path>envelope only</path>\n<type>image</type>\n<content>\nbody\n</content>"}},
{"name":"nested absent meta","tool":"read_image","argsRaw":"{\"file_path\":\" authored path \"}","metaRaw":"{\"path\":\" recorded path \"}","contentRaw":"[{\"type\":\"text\",\"text\":\"<path>envelope only</path>\\n<type>image</type>\\n<content>\\nbody\\n</content>\"},{\"type\":\"image\",\"attachment\":{\"attachmentId\":\"provider:id\",\"mediaType\":\"image/png\",\"bytes\":12,\"width\":3,\"height\":4}}]","nested":true,"omitMeta":true,"specialized":{"path":" authored path ","images":[{"attachment":{"attachmentId":"provider:id","mediaType":"image/png","bytes":12,"width":3,"height":4}}],"extraText":null},"generic":{"images":[{"attachment":{"attachmentId":"provider:id","mediaType":"image/png","bytes":12,"width":3,"height":4}}],"extraText":"<path>envelope only</path>\n<type>image</type>\n<content>\nbody\n</content>"}},
{"name":"root invalid meta null","tool":"read_image","argsRaw":"{\"file_path\":\" authored path \"}","metaRaw":"null","contentRaw":"[{\"type\":\"text\",\"text\":\"<path>envelope only</path>\\n<type>image</type>\\n<content>\\nbody\\n</content>\"},{\"type\":\"image\",\"attachment\":{\"attachmentId\":\"provider:id\",\"mediaType\":\"image/png\",\"bytes\":12,\"width\":3,\"height\":4}}]","specialized":null,"generic":{"images":[{"attachment":{"attachmentId":"provider:id","mediaType":"image/png","bytes":12,"width":3,"height":4}}],"extraText":"<path>envelope only</path>\n<type>image</type>\n<content>\nbody\n</content>"}},
{"name":"nested invalid meta null","tool":"read_image","argsRaw":"{\"file_path\":\" authored path \"}","metaRaw":"null","contentRaw":"[{\"type\":\"text\",\"text\":\"<path>envelope only</path>\\n<type>image</type>\\n<content>\\nbody\\n</content>\"},{\"type\":\"image\",\"attachment\":{\"attachmentId\":\"provider:id\",\"mediaType\":\"image/png\",\"bytes\":12,\"width\":3,\"height\":4}}]","nested":true,"specialized":{"path":" authored path ","images":[{"attachment":{"attachmentId":"provider:id","mediaType":"image/png","bytes":12,"width":3,"height":4}}],"extraText":null},"generic":{"images":[{"attachment":{"attachmentId":"provider:id","mediaType":"image/png","bytes":12,"width":3,"height":4}}],"extraText":"<path>envelope only</path>\n<type>image</type>\n<content>\nbody\n</content>"}},
{"name":"root no call","tool":"read_image","argsRaw":"{\"file_path\":\" authored path \"}","metaRaw":"{\"path\":\" recorded path \"}","contentRaw":"[{\"type\":\"text\",\"text\":\"<path>envelope only</path>\\n<type>image</type>\\n<content>\\nbody\\n</content>\"},{\"type\":\"image\",\"attachment\":{\"attachmentId\":\"provider:id\",\"mediaType\":\"image/png\",\"bytes\":12,\"width\":3,\"height\":4}}]","noCall":true,"specialized":null,"generic":{"images":[{"attachment":{"attachmentId":"provider:id","mediaType":"image/png","bytes":12,"width":3,"height":4}}],"extraText":"<path>envelope only</path>\n<type>image</type>\n<content>\nbody\n</content>"}},
{"name":"nested no call","tool":"read_image","argsRaw":"{\"file_path\":\" authored path \"}","metaRaw":"{\"path\":\" recorded path \"}","contentRaw":"[{\"type\":\"text\",\"text\":\"<path>envelope only</path>\\n<type>image</type>\\n<content>\\nbody\\n</content>\"},{\"type\":\"image\",\"attachment\":{\"attachmentId\":\"provider:id\",\"mediaType\":\"image/png\",\"bytes\":12,\"width\":3,\"height\":4}}]","nested":true,"noCall":true,"specialized":null,"generic":{"images":[{"attachment":{"attachmentId":"provider:id","mediaType":"image/png","bytes":12,"width":3,"height":4}}],"extraText":"<path>envelope only</path>\n<type>image</type>\n<content>\nbody\n</content>"}},
{"name":"invalid args {\"file_path\":\"\"}","tool":"read_image","argsRaw":"{\"file_path\":\"\"}","metaRaw":"{\"path\":\" recorded path \"}","contentRaw":"[{\"type\":\"text\",\"text\":\"<path>envelope only</path>\\n<type>image</type>\\n<content>\\nbody\\n</content>\"},{\"type\":\"image\",\"attachment\":{\"attachmentId\":\"provider:id\",\"mediaType\":\"image/png\",\"bytes\":12,\"width\":3,\"height\":4}}]","specialized":null,"generic":{"images":[{"attachment":{"attachmentId":"provider:id","mediaType":"image/png","bytes":12,"width":3,"height":4}}],"extraText":"<path>envelope only</path>\n<type>image</type>\n<content>\nbody\n</content>"}},
{"name":"NEL path","tool":"read_image","argsRaw":"{\"file_path\":\"\"}","metaRaw":"{\"path\":\" recorded path \"}","contentRaw":"[{\"type\":\"text\",\"text\":\"<path>envelope only</path>\\n<type>image</type>\\n<content>\\nbody\\n</content>\"},{\"type\":\"image\",\"attachment\":{\"attachmentId\":\"provider:id\",\"mediaType\":\"image/png\",\"bytes\":12,\"width\":3,\"height\":4}}]","nested":true,"omitMeta":true,"specialized":{"path":"","images":[{"attachment":{"attachmentId":"provider:id","mediaType":"image/png","bytes":12,"width":3,"height":4}}],"extraText":null},"generic":{"images":[{"attachment":{"attachmentId":"provider:id","mediaType":"image/png","bytes":12,"width":3,"height":4}}],"extraText":"<path>envelope only</path>\n<type>image</type>\n<content>\nbody\n</content>"}},
{"name":"duplicate last fields","tool":"read_image","argsRaw":"{\"file_path\":null,\"file_path\":\"x\"}","metaRaw":"{\"path\":null,\"path\":\"last\"}","contentRaw":"[{\"type\":\"text\",\"text\":\"<path>envelope only</path>\\n<type>image</type>\\n<content>\\nbody\\n</content>\"},{\"type\":\"image\",\"attachment\":{\"attachmentId\":null,\"attachmentId\":\"last\",\"mediaType\":\"image/png\",\"bytes\":null,\"bytes\":10.0,\"width\":1e1,\"height\":1,\"name\":null,\"name\":\"\"}}]","specialized":{"path":"last","images":[{"attachment":{"attachmentId":"last","mediaType":"image/png","bytes":10,"width":10,"height":1,"name":""}}],"extraText":null},"generic":{"images":[{"attachment":{"attachmentId":"last","mediaType":"image/png","bytes":10,"width":10,"height":1,"name":""}}],"extraText":"<path>envelope only</path>\n<type>image</type>\n<content>\nbody\n</content>"}},
{"name":"tool extension/screenshot","tool":"extension/screenshot","argsRaw":"{\"file_path\":\" authored path \"}","metaRaw":"{\"path\":\" recorded path \"}","contentRaw":"[{\"type\":\"text\",\"text\":\"<path>envelope only</path>\\n<type>image</type>\\n<content>\\nbody\\n</content>\"},{\"type\":\"image\",\"attachment\":{\"attachmentId\":\"provider:id\",\"mediaType\":\"image/png\",\"bytes\":12,\"width\":3,\"height\":4}}]","specialized":null,"generic":{"images":[{"attachment":{"attachmentId":"provider:id","mediaType":"image/png","bytes":12,"width":3,"height":4}}],"extraText":"<path>envelope only</path>\n<type>image</type>\n<content>\nbody\n</content>"}},
{"name":"null or malformed name null","tool":"read_image","argsRaw":"{\"file_path\":\" authored path \"}","metaRaw":"{\"path\":\" recorded path \"}","contentRaw":"[{\"type\":\"text\",\"text\":\"<path>envelope only</path>\\n<type>image</type>\\n<content>\\nbody\\n</content>\"},{\"type\":\"image\",\"attachment\":{\"attachmentId\":\"provider:id\",\"mediaType\":\"image/png\",\"bytes\":12,\"width\":3,\"height\":4,\"name\":null}}]","specialized":null,"generic":null},
{"name":"null or malformed dimensions null","tool":"read_image","argsRaw":"{\"file_path\":\" authored path \"}","metaRaw":"{\"path\":\" recorded path \"}","contentRaw":"[{\"type\":\"text\",\"text\":\"<path>envelope only</path>\\n<type>image</type>\\n<content>\\nbody\\n</content>\"},{\"type\":\"image\",\"attachment\":{\"attachmentId\":\"provider:id\",\"mediaType\":\"image/png\",\"bytes\":12,\"width\":3,\"height\":4,\"originalDimensions\":null}}]","specialized":null,"generic":null},
{"name":"invalid positive integer width 1e-9999","tool":"read_image","argsRaw":"{\"file_path\":\" authored path \"}","metaRaw":"{\"path\":\" recorded path \"}","contentRaw":"[{\"type\":\"text\",\"text\":\"<path>envelope only</path>\\n<type>image</type>\\n<content>\\nbody\\n</content>\"},{\"type\":\"image\",\"attachment\":{\"attachmentId\":\"provider:id\",\"mediaType\":\"image/png\",\"bytes\":12,\"width\":3,\"height\":4,\"width\":1e-9999}}]","specialized":null,"generic":null},
{"name":"JS positive integer 1e300","tool":"read_image","argsRaw":"{\"file_path\":\" authored path \"}","metaRaw":"{\"path\":\" recorded path \"}","contentRaw":"[{\"type\":\"text\",\"text\":\"<path>envelope only</path>\\n<type>image</type>\\n<content>\\nbody\\n</content>\"},{\"type\":\"image\",\"attachment\":{\"attachmentId\":\"x\",\"mediaType\":\"image/jpeg\",\"bytes\":1e300,\"width\":1e300,\"height\":1e300}}]","specialized":{"path":" recorded path ","images":[{"attachment":{"attachmentId":"x","mediaType":"image/jpeg","bytes":1e+300,"width":1e+300,"height":1e+300}}],"extraText":null},"generic":{"images":[{"attachment":{"attachmentId":"x","mediaType":"image/jpeg","bytes":1e+300,"width":1e+300,"height":1e+300}}],"extraText":"<path>envelope only</path>\n<type>image</type>\n<content>\nbody\n</content>"}},
{"name":"JS positive integer 1.7976931348623157e308","tool":"read_image","argsRaw":"{\"file_path\":\" authored path \"}","metaRaw":"{\"path\":\" recorded path \"}","contentRaw":"[{\"type\":\"text\",\"text\":\"<path>envelope only</path>\\n<type>image</type>\\n<content>\\nbody\\n</content>\"},{\"type\":\"image\",\"attachment\":{\"attachmentId\":\"x\",\"mediaType\":\"image/jpeg\",\"bytes\":1.7976931348623157e308,\"width\":1.7976931348623157e308,\"height\":1.7976931348623157e308}}]","specialized":{"path":" recorded path ","images":[{"attachment":{"attachmentId":"x","mediaType":"image/jpeg","bytes":1.7976931348623157e+308,"width":1.7976931348623157e+308,"height":1.7976931348623157e+308}}],"extraText":null},"generic":{"images":[{"attachment":{"attachmentId":"x","mediaType":"image/jpeg","bytes":1.7976931348623157e+308,"width":1.7976931348623157e+308,"height":1.7976931348623157e+308}}],"extraText":"<path>envelope only</path>\n<type>image</type>\n<content>\nbody\n</content>"}},
{"name":"extra block null","tool":"read_image","argsRaw":"{\"file_path\":\" authored path \"}","metaRaw":"{\"path\":\" recorded path \"}","contentRaw":"[{\"type\":\"text\",\"text\":\"<path>envelope only</path>\\n<type>image</type>\\n<content>\\nbody\\n</content>\"},{\"type\":\"image\",\"attachment\":{\"attachmentId\":\"provider:id\",\"mediaType\":\"image/png\",\"bytes\":12,\"width\":3,\"height\":4}},null]","specialized":null,"generic":{"images":[{"attachment":{"attachmentId":"provider:id","mediaType":"image/png","bytes":12,"width":3,"height":4}}],"extraText":"<path>envelope only</path>\n<type>image</type>\n<content>\nbody\n</content>"}},
{"name":"extra block []","tool":"read_image","argsRaw":"{\"file_path\":\" authored path \"}","metaRaw":"{\"path\":\" recorded path \"}","contentRaw":"[{\"type\":\"text\",\"text\":\"<path>envelope only</path>\\n<type>image</type>\\n<content>\\nbody\\n</content>\"},{\"type\":\"image\",\"attachment\":{\"attachmentId\":\"provider:id\",\"mediaType\":\"image/png\",\"bytes\":12,\"width\":3,\"height\":4}},[]]","specialized":null,"generic":{"images":[{"attachment":{"attachmentId":"provider:id","mediaType":"image/png","bytes":12,"width":3,"height":4}}],"extraText":"<path>envelope only</path>\n<type>image</type>\n<content>\nbody\n</content>\n[]"}},
{"name":"extra block [null,1,{\"z\":-0}]","tool":"read_image","argsRaw":"{\"file_path\":\" authored path \"}","metaRaw":"{\"path\":\" recorded path \"}","contentRaw":"[{\"type\":\"text\",\"text\":\"<path>envelope only</path>\\n<type>image</type>\\n<content>\\nbody\\n</content>\"},{\"type\":\"image\",\"attachment\":{\"attachmentId\":\"provider:id\",\"mediaType\":\"image/png\",\"bytes\":12,\"width\":3,\"height\":4}},[null,1,{\"z\":-0}]]","specialized":null,"generic":{"images":[{"attachment":{"attachmentId":"provider:id","mediaType":"image/png","bytes":12,"width":3,"height":4}}],"extraText":"<path>envelope only</path>\n<type>image</type>\n<content>\nbody\n</content>\n[\n  null,\n  1,\n  {\n    \"z\": 0\n  }\n]"}},
{"name":"extra block {\"type\":\"reasoning\",\"text\":\"evidence\"}","tool":"read_image","argsRaw":"{\"file_path\":\" authored path \"}","metaRaw":"{\"path\":\" recorded path \"}","contentRaw":"[{\"type\":\"text\",\"text\":\"<path>envelope only</path>\\n<type>image</type>\\n<content>\\nbody\\n</content>\"},{\"type\":\"image\",\"attachment\":{\"attachmentId\":\"provider:id\",\"mediaType\":\"image/png\",\"bytes\":12,\"width\":3,\"height\":4}},{\"type\":\"reasoning\",\"text\":\"evidence\"}]","specialized":null,"generic":{"images":[{"attachment":{"attachmentId":"provider:id","mediaType":"image/png","bytes":12,"width":3,"height":4}}],"extraText":"<path>envelope only</path>\n<type>image</type>\n<content>\nbody\n</content>\n{\n  \"type\": \"reasoning\",\n  \"text\": \"evidence\"\n}"}},
{"name":"extra block {\"type\":\"text\",\"text\":null}","tool":"read_image","argsRaw":"{\"file_path\":\" authored path \"}","metaRaw":"{\"path\":\" recorded path \"}","contentRaw":"[{\"type\":\"text\",\"text\":\"<path>envelope only</path>\\n<type>image</type>\\n<content>\\nbody\\n</content>\"},{\"type\":\"image\",\"attachment\":{\"attachmentId\":\"provider:id\",\"mediaType\":\"image/png\",\"bytes\":12,\"width\":3,\"height\":4}},{\"type\":\"text\",\"text\":null}]","specialized":null,"generic":{"images":[{"attachment":{"attachmentId":"provider:id","mediaType":"image/png","bytes":12,"width":3,"height":4}}],"extraText":"<path>envelope only</path>\n<type>image</type>\n<content>\nbody\n</content>\n{\n  \"type\": \"text\",\n  \"text\": null\n}"}},
{"name":"generic primitive ignored","tool":"read_image","argsRaw":"{\"file_path\":\" authored path \"}","metaRaw":"{\"path\":\" recorded path \"}","contentRaw":"[null,1,true,\"primitive\",{\"type\":\"image\",\"attachment\":{\"attachmentId\":\"provider:id\",\"mediaType\":\"image/png\",\"bytes\":12,\"width\":3,\"height\":4}}]","specialized":null,"generic":{"images":[{"attachment":{"attachmentId":"provider:id","mediaType":"image/png","bytes":12,"width":3,"height":4}}],"extraText":null}},
{"name":"generic empty text","tool":"read_image","argsRaw":"{\"file_path\":\" authored path \"}","metaRaw":"{\"path\":\" recorded path \"}","contentRaw":"[{\"type\":\"image\",\"attachment\":{\"attachmentId\":\"provider:id\",\"mediaType\":\"image/png\",\"bytes\":12,\"width\":3,\"height\":4}},{\"type\":\"text\",\"text\":\"\"}]","specialized":null,"generic":{"images":[{"attachment":{"attachmentId":"provider:id","mediaType":"image/png","bytes":12,"width":3,"height":4}}],"extraText":""}},
{"name":"generic JSON semantics","tool":"screenshot","argsRaw":"invalid","metaRaw":"{\"path\":\" recorded path \"}","contentRaw":"[{\"type\":\"image\",\"attachment\":{\"attachmentId\":\"provider:id\",\"mediaType\":\"image/png\",\"bytes\":12,\"width\":3,\"height\":4}},{\"z\":\"first\",\"10\":\"ten\",\"2\":\"two\",\"z\":\"last\",\"nested\":[-0,9007199254740993,1e9999,1e-7],\"literal\":\"<>&\\u2028\\u2029\"},[1,{\"b\":2,\"1\":\"index\"}],{\"type\":\"text\",\"text\":null},{\"type\":\"text\",\"text\":\"exact\\r\\n\\u0000\"}]","omitMeta":true,"specialized":null,"generic":{"images":[{"attachment":{"attachmentId":"provider:id","mediaType":"image/png","bytes":12,"width":3,"height":4}}],"extraText":"{\n  \"2\": \"two\",\n  \"10\": \"ten\",\n  \"z\": \"last\",\n  \"nested\": [\n    0,\n    9007199254740992,\n    null,\n    1e-7\n  ],\n  \"literal\": \"<>&  \"\n}\n[\n  1,\n  {\n    \"1\": \"index\",\n    \"b\": 2\n  }\n]\n{\n  \"type\": \"text\",\n  \"text\": null\n}\nexact\r\n\u0000"}},
{"name":"generic image only","tool":"extension/screenshot","argsRaw":"[]","metaRaw":"{\"path\":\" recorded path \"}","contentRaw":"[{\"type\":\"image\",\"attachment\":{\"attachmentId\":\"provider:id\",\"mediaType\":\"image/png\",\"bytes\":12,\"width\":3,\"height\":4}}]","specialized":null,"generic":{"images":[{"attachment":{"attachmentId":"provider:id","mediaType":"image/png","bytes":12,"width":3,"height":4}}],"extraText":null}},
{"name":"multiple envelopes excluded","tool":"read_image","argsRaw":"{\"file_path\":\" authored path \"}","metaRaw":"{\"path\":\" recorded path \"}","contentRaw":"[{\"type\":\"text\",\"text\":\"<path>envelope only</path>\\n<type>image</type>\\n<content>\\nbody\\n</content>\"},{\"type\":\"image\",\"attachment\":{\"attachmentId\":\"provider:id\",\"mediaType\":\"image/png\",\"bytes\":12,\"width\":3,\"height\":4}},{\"type\":\"text\",\"text\":\"<path>envelope only</path>\\n<type>image</type>\\n<content>\\nbody\\n</content>\"},{\"type\":\"text\",\"text\":\"\"},{\"type\":\"text\",\"text\":\"tail\"}]","specialized":{"path":" recorded path ","images":[{"attachment":{"attachmentId":"provider:id","mediaType":"image/png","bytes":12,"width":3,"height":4}}],"extraText":"\ntail"},"generic":{"images":[{"attachment":{"attachmentId":"provider:id","mediaType":"image/png","bytes":12,"width":3,"height":4}}],"extraText":"<path>envelope only</path>\n<type>image</type>\n<content>\nbody\n</content>\n<path>envelope only</path>\n<type>image</type>\n<content>\nbody\n</content>\n\ntail"}},
{"name":"envelope empty path and body","tool":"read_image","argsRaw":"{\"file_path\":\" authored path \"}","metaRaw":"{\"path\":\" recorded path \"}","contentRaw":"[{\"type\":\"text\",\"text\":\"<path></path>\\n<type>image</type>\\n<content>\\n\\n</content>\"},{\"type\":\"image\",\"attachment\":{\"attachmentId\":\"provider:id\",\"mediaType\":\"image/png\",\"bytes\":12,\"width\":3,\"height\":4}}]","specialized":{"path":" recorded path ","images":[{"attachment":{"attachmentId":"provider:id","mediaType":"image/png","bytes":12,"width":3,"height":4}}],"extraText":null},"generic":{"images":[{"attachment":{"attachmentId":"provider:id","mediaType":"image/png","bytes":12,"width":3,"height":4}}],"extraText":"<path></path>\n<type>image</type>\n<content>\n\n</content>"}},
{"name":"envelope CR unicode path","tool":"read_image","argsRaw":"{\"file_path\":\" authored path \"}","metaRaw":"{\"path\":\" recorded path \"}","contentRaw":"[{\"type\":\"text\",\"text\":\"<path>a\\r  b</path>\\n<type>image</type>\\n<content>\\nx\\n</content>\"},{\"type\":\"image\",\"attachment\":{\"attachmentId\":\"provider:id\",\"mediaType\":\"image/png\",\"bytes\":12,\"width\":3,\"height\":4}}]","specialized":{"path":" recorded path ","images":[{"attachment":{"attachmentId":"provider:id","mediaType":"image/png","bytes":12,"width":3,"height":4}}],"extraText":null},"generic":{"images":[{"attachment":{"attachmentId":"provider:id","mediaType":"image/png","bytes":12,"width":3,"height":4}}],"extraText":"<path>a\r  b</path>\n<type>image</type>\n<content>\nx\n</content>"}},
{"name":"envelope trailing LF","tool":"read_image","argsRaw":"{\"file_path\":\" authored path \"}","metaRaw":"{\"path\":\" recorded path \"}","contentRaw":"[{\"type\":\"text\",\"text\":\"<path>envelope only</path>\\n<type>image</type>\\n<content>\\nbody\\n</content>\\n\"},{\"type\":\"image\",\"attachment\":{\"attachmentId\":\"provider:id\",\"mediaType\":\"image/png\",\"bytes\":12,\"width\":3,\"height\":4}}]","specialized":null,"generic":{"images":[{"attachment":{"attachmentId":"provider:id","mediaType":"image/png","bytes":12,"width":3,"height":4}}],"extraText":"<path>envelope only</path>\n<type>image</type>\n<content>\nbody\n</content>\n"}},
{"name":"envelope structural CRLF","tool":"read_image","argsRaw":"{\"file_path\":\" authored path \"}","metaRaw":"{\"path\":\" recorded path \"}","contentRaw":"[{\"type\":\"text\",\"text\":\"<path>envelope only</path>\\r\\n<type>image</type>\\r\\n<content>\\r\\nbody\\r\\n</content>\"},{\"type\":\"image\",\"attachment\":{\"attachmentId\":\"provider:id\",\"mediaType\":\"image/png\",\"bytes\":12,\"width\":3,\"height\":4}}]","specialized":null,"generic":{"images":[{"attachment":{"attachmentId":"provider:id","mediaType":"image/png","bytes":12,"width":3,"height":4}}],"extraText":"<path>envelope only</path>\r\n<type>image</type>\r\n<content>\r\nbody\r\n</content>"}},
{"name":"isError","tool":"read_image","argsRaw":"{\"file_path\":\" authored path \"}","metaRaw":"{\"path\":\" recorded path \"}","contentRaw":"[{\"type\":\"text\",\"text\":\"<path>envelope only</path>\\n<type>image</type>\\n<content>\\nbody\\n</content>\"},{\"type\":\"image\",\"attachment\":{\"attachmentId\":\"provider:id\",\"mediaType\":\"image/png\",\"bytes\":12,\"width\":3,\"height\":4}}]","isError":true,"specialized":null,"generic":null}
]`
