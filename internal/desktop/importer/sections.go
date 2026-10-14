package importer

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

var settingSections = []string{"llm-deepseek", "llm-pi-ai", "agent-default-model", "vision-fallback", "ui-theme"}
var sectionKey = regexp.MustCompile(`^([A-Za-z0-9_-]+):(?:\s|$)`)

type textSections struct {
	prefix string
	order  []string
	blocks map[string]string
}

// Keep raw blocks, including nested fields and comments. Unsupported document
// shapes fail closed rather than being reinterpreted or losing unknown fields.
func splitSections(raw string) (textSections, error) {
	d := textSections{blocks: map[string]string{}}
	key := ""
	for _, line := range strings.SplitAfter(raw, "\n") {
		trim := strings.TrimSpace(line)
		if m := sectionKey.FindStringSubmatch(line); m != nil {
			key = m[1]
			if _, exists := d.blocks[key]; exists {
				return d, fmt.Errorf("duplicate YAML section")
			}
			d.order = append(d.order, key)
		} else if trim != "" && !strings.HasPrefix(trim, "#") && line[0] != ' ' {
			return d, fmt.Errorf("unsupported YAML document shape")
		}
		if key == "" {
			d.prefix += line
		} else {
			d.blocks[key] += line
		}
	}
	return d, nil
}
func (d textSections) render() string {
	out := d.prefix
	for _, id := range d.order {
		if out != "" && !strings.HasSuffix(out, "\n") {
			out += "\n"
		}
		out += d.blocks[id]
	}
	return out
}

// MCP is the official servers sequence, not top-level server keys. Copy each
// selected raw record so headers, tokens, args and future fields survive intact.
func splitMCP(raw string) (textSections, textSections, error) {
	doc, err := splitSections(raw)
	rows := textSections{blocks: map[string]string{}}
	if err != nil {
		return doc, rows, err
	}
	block, ok := doc.blocks["servers"]
	if !ok {
		if strings.TrimSpace(raw) == "" {
			return doc, rows, nil
		}
		return doc, rows, fmt.Errorf("MCP servers sequence missing")
	}
	lines := strings.SplitAfter(block, "\n")
	header := strings.TrimSpace(lines[0])
	if header == "servers: []" {
		if len(lines) > 1 && strings.TrimSpace(strings.Join(lines[1:], "")) != "" {
			return doc, rows, fmt.Errorf("unsupported MCP sequence")
		}
		return doc, rows, nil
	}
	if header != "servers:" {
		return doc, rows, fmt.Errorf("unsupported MCP sequence")
	}
	var record strings.Builder
	id := ""
	flush := func() error {
		if record.Len() == 0 {
			return nil
		}
		if id == "" {
			return fmt.Errorf("MCP record missing id")
		}
		if _, ok := rows.blocks[id]; ok {
			return fmt.Errorf("duplicate MCP id")
		}
		rows.order = append(rows.order, id)
		rows.blocks[id] = record.String()
		record.Reset()
		id = ""
		return nil
	}
	indent := -1
	for _, line := range lines[1:] {
		trim := strings.TrimSpace(line)
		n := len(line) - len(strings.TrimLeft(line, " "))
		start := strings.HasPrefix(trim, "- ") && (indent < 0 || n == indent)
		if start {
			if indent < 0 {
				indent = n
				if indent != 2 {
					return doc, rows, fmt.Errorf("unsupported MCP record indentation")
				}
			}
			if err := flush(); err != nil {
				return doc, rows, err
			}
		}
		if record.Len() == 0 && !start {
			if trim != "" && !strings.HasPrefix(trim, "#") {
				return doc, rows, fmt.Errorf("unsupported MCP sequence")
			}
			rows.prefix += line
			continue
		}
		record.WriteString(line)
		field := trim
		if start {
			field = strings.TrimSpace(strings.TrimPrefix(field, "- "))
		}
		if (start || n == indent+2) && strings.HasPrefix(field, "id:") {
			if id != "" {
				return doc, rows, fmt.Errorf("duplicate MCP record id field")
			}
			value := strings.TrimSpace(strings.TrimPrefix(field, "id:"))
			if strings.HasPrefix(value, `"`) {
				v, e := strconv.Unquote(value)
				if e != nil {
					return doc, rows, fmt.Errorf("unsupported MCP id")
				}
				id = v
			} else if strings.HasPrefix(value, "'") && strings.HasSuffix(value, "'") {
				id = strings.ReplaceAll(value[1:len(value)-1], "''", "'")
			} else {
				id = strings.TrimSpace(strings.Split(value, " #")[0])
				if strings.ContainsAny(id, "{}[]&*!|>\t") {
					return doc, rows, fmt.Errorf("unsupported MCP id")
				}
			}
		}
	}
	if err := flush(); err != nil {
		return doc, rows, err
	}
	return doc, rows, nil
}

func privatePath(path string) error {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	for p := absolute; ; p = filepath.Dir(p) {
		info, e := os.Lstat(p)
		if e != nil && !os.IsNotExist(e) {
			return e
		}
		if e == nil && (info.Mode()&os.ModeSymlink != 0 || (p == absolute && !info.Mode().IsRegular())) {
			return fmt.Errorf("unsafe import file path")
		}
		if p == filepath.Dir(p) {
			break
		}
	}
	return nil
}
func readImportText(path string) ([]byte, error) {
	if err := privatePath(path); err != nil {
		return nil, err
	}
	raw, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	return raw, err
}

// Publish only after the private same-directory temporary file is complete.
// Never truncate/delete the old destination, including on rename failure.
func writeMergedFile(path string, before, data []byte) error {
	return writeMergedFileWithRename(path, before, data, os.Rename)
}
func writeMergedFileWithRename(path string, before, data []byte, rename func(string, string) error) error {
	if err := privatePath(path); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	f, err := os.CreateTemp(filepath.Dir(path), ".import-section-*")
	if err != nil {
		return err
	}
	temp := f.Name()
	owned, statErr := f.Stat()
	if statErr != nil {
		f.Close()
		return statErr
	}
	defer func() {
		if current, err := os.Lstat(temp); err == nil && os.SameFile(owned, current) {
			_ = os.Remove(temp)
		}
	}()
	if _, err = f.Write(data); err == nil {
		err = f.Sync()
	}
	closeErr := f.Close()
	if err != nil {
		return err
	}
	if closeErr != nil {
		return closeErr
	}
	current, err := readImportText(path)
	if err != nil {
		return err
	}
	if (current == nil) != (before == nil) || !bytes.Equal(current, before) {
		return fmt.Errorf("destination changed; rescan before importing")
	}
	if err := privatePath(temp); err != nil {
		return err
	}
	tempInfo, err := os.Lstat(temp)
	if err != nil {
		return err
	}
	if !os.SameFile(owned, tempInfo) {
		return fmt.Errorf("temporary import file changed")
	}
	return rename(temp, path)
}

func mergeSelected(ctx context.Context, cat Catalog, ids []string, mcp, overwrite bool) (copied, skipped int, err error) {
	file := "settings.yaml"
	if mcp {
		file = "mcp-servers.yaml"
	}
	src, err := readImportText(filepath.Join(cat.SourceHome, file))
	if err != nil {
		return 0, 0, err
	}
	dstPath := filepath.Join(cat.DestHome, file)
	before, err := readImportText(dstPath)
	if err != nil {
		return 0, 0, err
	}
	var source, dest, doc textSections
	if mcp {
		_, source, err = splitMCP(string(src))
		if err == nil {
			doc, dest, err = splitMCP(string(before))
		}
	} else {
		source, err = splitSections(string(src))
		if err == nil {
			dest, err = splitSections(string(before))
		}
	}
	if err != nil {
		return 0, 0, err
	}
	seen := map[string]bool{}
	for _, id := range ids {
		if err := ctx.Err(); err != nil {
			return 0, 0, err
		}
		if seen[id] {
			continue
		}
		seen[id] = true
		block, exists := source.blocks[id]
		if !exists || (!mcp && !contains(settingSections, id)) {
			skipped++
			continue
		}
		_, exists = dest.blocks[id]
		if exists && !overwrite {
			skipped++
			continue
		}
		if !exists {
			dest.order = append(dest.order, id)
		}
		dest.blocks[id] = block
		copied++
	}
	if copied == 0 {
		return copied, skipped, nil
	}
	var data string
	if mcp {
		if _, ok := doc.blocks["servers"]; !ok {
			doc.order = append(doc.order, "servers")
		}
		doc.blocks["servers"] = "servers:\n" + dest.render()
		data = doc.render()
	} else {
		data = dest.render()
	}
	if err := ctx.Err(); err != nil {
		return 0, skipped, err
	}
	if err := writeMergedFile(dstPath, before, []byte(data)); err != nil {
		return 0, skipped, err
	}
	return copied, skipped, nil
}
