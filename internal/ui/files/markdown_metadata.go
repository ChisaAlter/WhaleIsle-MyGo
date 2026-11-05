package files

import (
	"github.com/egoist/mygo/ui"
	"gopkg.in/yaml.v3"
	"strings"
	"whaleisle/internal/ui/widgets/markdown"
)

type metadataField struct{ key, value string }

func splitMarkdownMetadata(text string) (source, body string, found bool) {
	textWithoutBOM := strings.TrimPrefix(text, "\ufeff")
	lines := strings.SplitAfter(textWithoutBOM, "\n")
	if len(lines) < 2 || strings.TrimRight(lines[0], " \t\r\n") != "---" {
		return "", text, false
	}
	offset := len(lines[0])
	for _, line := range lines[1:] {
		delimiter := strings.TrimRight(line, " \t\r\n")
		if delimiter == "---" || delimiter == "..." {
			return strings.TrimSuffix(strings.TrimSuffix(textWithoutBOM[len(lines[0]):offset], "\n"), "\r"), textWithoutBOM[offset+len(line):], true
		}
		offset += len(line)
	}
	return "", text, false
}

func parseMetadataFields(source string) ([]metadataField, bool) {
	var document yaml.Node
	if err := yaml.Unmarshal([]byte(source), &document); err != nil {
		return nil, false
	}
	if len(document.Content) == 0 {
		return nil, true
	}
	mapping := document.Content[0]
	if mapping.Kind != yaml.MappingNode {
		return nil, false
	}
	// Decode validates duplicate mapping keys; aliases remain syntax nodes below.
	var check map[string]any
	if err := mapping.Decode(&check); err != nil {
		return nil, false
	}
	fields := make([]metadataField, 0, len(mapping.Content)/2)
	for i := 0; i < len(mapping.Content); i += 2 {
		key, value := mapping.Content[i], mapping.Content[i+1]
		end := len(source)
		if i+2 < len(mapping.Content) {
			end = metadataOffset(source, mapping.Content[i+2].Line, 1)
		}
		fields = append(fields, metadataField{metadataNodeText(source, key, metadataOffset(source, value.Line, value.Column)), metadataNodeText(source, value, end)})
	}
	return fields, true
}

func metadataOffset(source string, line, column int) int {
	offset := 0
	for n := 1; n < line; n++ {
		next := strings.IndexByte(source[offset:], '\n')
		if next < 0 {
			return len(source)
		}
		offset += next + 1
	}
	offset += column - 1
	if offset > len(source) {
		return len(source)
	}
	return offset
}

func metadataNodeText(source string, node *yaml.Node, end int) string {
	if node.Kind == yaml.ScalarNode {
		return strings.TrimRight(node.Value, "\r\n \t")
	}
	if node.Kind == yaml.AliasNode {
		return "*" + node.Value
	}
	start := metadataOffset(source, node.Line, node.Column)
	if end < start {
		end = start
	}
	lines := strings.Split(strings.TrimRight(source[start:end], "\r\n \t"), "\n")
	for i := 1; i < len(lines); i++ {
		for n := 0; n < node.Column-1 && strings.HasPrefix(lines[i], " "); n++ {
			lines[i] = lines[i][1:]
		}
	}
	return strings.Join(lines, "\n")
}

func paintMarkdownResource(c *ui.Context, m Model, fg, muted ui.Color) {
	source, body, found := splitMarkdownMetadata(previewText(m))
	if found {
		fields, valid := parseMetadataFields(source)
		if !valid {
			ui.Text(c, source).TextColor(muted)
		} else {
			for _, field := range fields {
				ui.Row(c).Gap(12).FillWidth().Children(func() { ui.Text(c, field.key).Bold().TextColor(muted); ui.Text(c, field.value).TextColor(fg).Grow(1) })
			}
		}
	}
	style := sourceStyle(m, fg, muted)
	style.RenderImage = func(ctx *ui.Context, ref, alt string) bool { return paintMarkdownImage(ctx, m, ref, alt, muted) }
	markdown.ViewInteractive(c, body, style, m.OnOpenLink)
}
