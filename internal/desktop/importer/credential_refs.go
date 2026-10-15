package importer

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// CredentialResult deliberately contains names/statuses only, never values.
// Ref is preserved as-is: settings apiKeyEnv -> .credentials.yaml refs[Ref].
type CredentialResult struct {
	Ref    string
	Status string // copied, skipped, missing, unsupported, failed
	Reason string
}

const referencedCredentialsFile = ".credentials.yaml"

var credentialRefName = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)
var credentialRefEntry = regexp.MustCompile(`^  ([A-Za-z_][A-Za-z0-9_]*):(.*)$`)
var credentialSettingField = regexp.MustCompile(`^\s+apiKeyEnv:\s*(.*?)\s*$`)
var errCredentialDocument = errors.New("unsupported credential document")

type credentialEntry struct {
	raw       string
	line      int
	supported bool
}
type credentialDocument struct {
	lines      []string
	refs       map[string]credentialEntry
	start, end int
	emptyMap   bool
}

// credentialRefsOfSection recognizes the two legacy LLM sections only. Names
// are restricted to the harness ref grammar; comments/scalar prose are not refs.
func credentialRefsOfSection(id, block string) []string {
	if id != "llm-deepseek" && id != "llm-pi-ai" {
		return nil
	}
	var refs []string
	seen := map[string]bool{}
	scalarIndent := -1
	for _, line := range strings.Split(block, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		indent := len(line) - len(strings.TrimLeft(line, " "))
		if scalarIndent >= 0 {
			if indent > scalarIndent {
				continue
			}
			scalarIndent = -1
		}
		// apiKeyEnv-looking lines inside YAML block-scalar text are not settings.
		if colon := strings.Index(trimmed, ":"); colon >= 0 {
			value := strings.TrimSpace(trimmed[colon+1:])
			if strings.HasPrefix(value, "|") || strings.HasPrefix(value, ">") {
				scalarIndent = indent
				continue
			}
		}
		match := credentialSettingField.FindStringSubmatch(line)
		if match == nil {
			continue
		}
		value := strings.TrimSpace(strings.SplitN(match[1], "#", 2)[0])
		if len(value) >= 2 && ((value[0] == '\'' && value[len(value)-1] == '\'') || (value[0] == '"' && value[len(value)-1] == '"')) {
			value = value[1 : len(value)-1]
		}
		if credentialRefName.MatchString(value) && !seen[value] {
			refs = append(refs, value)
			seen[value] = true
		}
	}
	if id == "llm-deepseek" && len(refs) == 0 {
		return []string{"DEEPSEEK_API_KEY"}
	}
	return refs
}

// Only complete single-line scalar values can be transplanted unchanged.
// Anchors, aliases, tags, collections, nested/block values and encryption
// objects are intentionally unsupported, rather than mistaken for secrets.
func credentialScalarSupported(raw string) bool {
	value := strings.TrimSpace(raw)
	if value == "" || strings.ContainsAny(value, "\r\n\x00\t") {
		return false
	}
	if strings.ContainsRune("|>&*!{[#", rune(value[0])) {
		return false
	}
	if value[0] == '\'' || value[0] == '"' {
		quote := value[0]
		for i := 1; i < len(value); i++ {
			if quote == '"' && value[i] == '\\' {
				i++
				continue
			}
			if value[i] != quote {
				continue
			}
			if quote == '\'' && i+1 < len(value) && value[i+1] == '\'' {
				i++
				continue
			}
			rest := strings.TrimSpace(value[i+1:])
			return rest == "" || strings.HasPrefix(rest, "#")
		}
		return false
	}
	value = strings.TrimSpace(strings.SplitN(value, " #", 2)[0])
	return value != "" && value != "~" && value != "null" && value != "Null" && value != "NULL" && !strings.Contains(value, ": ") && !strings.HasPrefix(value, "- ") && !strings.HasPrefix(value, "? ")
}

// Parse refs structurally while retaining every untouched target line verbatim.
// OAuth records are never inspected or copied. Unknown/encrypted formats fail
// closed; malformed or duplicate mappings must not become an empty destination.
func parseCredentialDocument(raw []byte) (credentialDocument, error) {
	doc := credentialDocument{lines: strings.SplitAfter(string(raw), "\n"), refs: map[string]credentialEntry{}, start: -1, end: -1}
	topSeen := map[string]bool{}
	for i, line := range doc.lines {
		trim := strings.TrimSpace(line)
		if trim == "" || strings.HasPrefix(trim, "#") || line[0] == ' ' {
			continue
		}
		match := sectionKey.FindStringSubmatch(line)
		if match == nil || topSeen[match[1]] {
			return doc, errCredentialDocument
		}
		key := match[1]
		topSeen[key] = true
		if doc.start >= 0 && doc.end < 0 {
			doc.end = i
		}
		value := strings.TrimSpace(strings.TrimPrefix(line, key+":"))
		value = strings.TrimSpace(strings.SplitN(value, "#", 2)[0])
		if key == "version" && value != "1" && value != "'1'" && value != "\"1\"" {
			return doc, errCredentialDocument
		}
		if key == "payload" || key == "encrypted" || key == "ciphertext" {
			return doc, errCredentialDocument
		}
		if key == "refs" {
			if value != "" && value != "{}" {
				return doc, errCredentialDocument
			}
			doc.start = i
			doc.end = -1
			doc.emptyMap = value == "{}"
		}
	}
	if doc.start < 0 {
		return doc, nil
	}
	if doc.end < 0 {
		doc.end = len(doc.lines)
	}
	last := ""
	for i := doc.start + 1; i < doc.end; i++ {
		line := strings.TrimSuffix(strings.TrimSuffix(doc.lines[i], "\n"), "\r")
		trim := strings.TrimSpace(line)
		if trim == "" || strings.HasPrefix(trim, "#") {
			continue
		}
		if doc.emptyMap {
			return doc, errCredentialDocument
		}
		if match := credentialRefEntry.FindStringSubmatch(line); match != nil {
			if _, duplicate := doc.refs[match[1]]; duplicate {
				return doc, errCredentialDocument
			}
			last = match[1]
			doc.refs[last] = credentialEntry{raw: match[2], line: i, supported: credentialScalarSupported(match[2])}
		} else if strings.HasPrefix(line, "    ") && last != "" {
			entry := doc.refs[last]
			entry.supported = false
			doc.refs[last] = entry
		} else {
			return doc, errCredentialDocument
		}
	}
	return doc, nil
}

func credentialResults(refs []string, status, reason string) []CredentialResult {
	rows := make([]CredentialResult, 0, len(refs))
	for _, ref := range refs {
		rows = append(rows, CredentialResult{Ref: ref, Status: status, Reason: reason})
	}
	return rows
}

// A credential source must be a regular, single-link file inside the selected
// home. Check the opened inode as well as the path, and bound parsing memory.
// These checks do not claim protection from privileged concurrent ancestor swaps.
func readCredentialImportFile(path string) ([]byte, error) {
	if err := privatePath(path); err != nil {
		return nil, errCredentialDocument
	}
	file, err := os.Open(path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, errCredentialDocument
	}
	defer file.Close()
	opened, err := file.Stat()
	if err != nil || !opened.Mode().IsRegular() || !credentialFileHasSingleLink(file, opened) {
		return nil, errCredentialDocument
	}
	current, err := os.Lstat(path)
	if err != nil || !os.SameFile(opened, current) || current.Mode()&os.ModeSymlink != 0 {
		return nil, errCredentialDocument
	}
	const maxCredentialBytes = 4 << 20
	raw, err := io.ReadAll(io.LimitReader(file, maxCredentialBytes+1))
	if err != nil || len(raw) > maxCredentialBytes {
		return nil, errCredentialDocument
	}
	if err := privatePath(path); err != nil {
		return nil, errCredentialDocument
	}
	after, err := file.Stat()
	if err != nil || after.Size() != opened.Size() || !after.ModTime().Equal(opened.ModTime()) || !credentialFileHasSingleLink(file, after) {
		return nil, errCredentialDocument
	}
	current, err = os.Lstat(path)
	if err != nil || !os.SameFile(opened, current) {
		return nil, errCredentialDocument
	}
	return raw, nil
}

// importCredentialRefs reads only the source-home file, and publishes only refs
// collected from selected settings. It uses the same private atomic CAS writer
// as settings, under ApplyCatalog's existing import lock and recovery admission.
// No credential value can enter a result, error, journal or recovery manifest.
func importCredentialRefs(ctx context.Context, cat Catalog, refs []string, overwrite bool) ([]CredentialResult, error) {
	wanted := make([]string, 0, len(refs))
	seen := map[string]bool{}
	for _, ref := range refs {
		if credentialRefName.MatchString(ref) && !seen[ref] {
			wanted = append(wanted, ref)
			seen[ref] = true
		}
	}
	if len(wanted) == 0 {
		return nil, nil
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	sourceRaw, err := readCredentialImportFile(filepath.Join(cat.SourceHome, referencedCredentialsFile))
	if err != nil {
		return credentialResults(wanted, "failed", "source-unreadable-or-unsafe"), nil
	}
	source, err := parseCredentialDocument(sourceRaw)
	if err != nil {
		return credentialResults(wanted, "unsupported", "source-format"), nil
	}
	path := filepath.Join(cat.DestHome, referencedCredentialsFile)
	before, err := readCredentialImportFile(path)
	if err != nil {
		return credentialResults(wanted, "failed", "target-unreadable-or-unsafe"), nil
	}
	dest, err := parseCredentialDocument(before)
	if err != nil {
		return credentialResults(wanted, "failed", "target-format"), nil
	}
	rows := make([]CredentialResult, 0, len(wanted))
	additions := []string{}
	replacements := map[int]string{}
	newline := "\n"
	if strings.Contains(string(before), "\r\n") {
		newline = "\r\n"
	}
	for _, ref := range wanted {
		entry, exists := source.refs[ref]
		row := CredentialResult{Ref: ref}
		old, conflict := dest.refs[ref]
		switch {
		case !exists:
			row.Status, row.Reason = "missing", "source-ref-missing"
		case !entry.supported:
			row.Status, row.Reason = "unsupported", "source-value-format"
		case conflict && !overwrite:
			row.Status, row.Reason = "skipped", "target-conflict"
		case conflict && !old.supported:
			row.Status, row.Reason = "unsupported", "target-value-format"
		default:
			row.Status = "copied"
			line := "  " + ref + ":" + entry.raw + newline
			if conflict {
				replacements[old.line] = line
			} else {
				additions = append(additions, line)
			}
		}
		rows = append(rows, row)
	}
	if len(additions) == 0 && len(replacements) == 0 {
		return rows, nil
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	var out strings.Builder
	if before == nil {
		out.WriteString("version: 1" + newline)
	}
	if dest.start < 0 {
		out.Write(before)
		if len(before) > 0 && !strings.HasSuffix(string(before), "\n") {
			out.WriteString(newline)
		}
		out.WriteString("refs:" + newline)
		for _, line := range additions {
			out.WriteString(line)
		}
	} else {
		for i, line := range dest.lines {
			if i == dest.end {
				for _, addition := range additions {
					out.WriteString(addition)
				}
			}
			if i == dest.start && dest.emptyMap {
				line = "refs:" + newline
			}
			if replacement, ok := replacements[i]; ok {
				line = replacement
			}
			out.WriteString(line)
			if i == dest.end-1 && len(additions) > 0 && line != "" && !strings.HasSuffix(line, "\n") {
				out.WriteString(newline)
			}
		}
		if dest.end == len(dest.lines) {
			for _, line := range additions {
				out.WriteString(line)
			}
		}
	}
	if err := writeMergedFile(path, before, []byte(out.String())); err != nil {
		for i := range rows {
			if rows[i].Status == "copied" {
				rows[i].Status, rows[i].Reason = "failed", "publication-failed"
			}
		}
	}
	return rows, nil
}
