package dsh

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/textproto"
	"strings"
)

// MultipartResult is a Typert Remote response whose value carries binary parts.
type MultipartResult struct {
	// Value is the JSON result.value after binary slots are assembled (base64
	// strings where placeholders were null). Callers that need raw parts use Parts.
	Value json.RawMessage
	// Parts maps Content-Disposition name → body bytes.
	Parts map[string][]byte
}

// parseMultipartResult parses a multipart/form-data RPC body.
// The JSON envelope part is named "metadata" (Connection binary transport) or
// "response" / the first application/json part (legacy).
func parseMultipartResult(contentType string, body []byte, rpcID, method string) (*MultipartResult, error) {
	_, params, err := mime.ParseMediaType(contentType)
	if err != nil {
		return nil, fmt.Errorf("dsh %s: multipart Content-Type: %w", method, err)
	}
	boundary := params["boundary"]
	if boundary == "" {
		return nil, fmt.Errorf("dsh %s: multipart missing boundary", method)
	}
	mr := multipart.NewReader(bytes.NewReader(body), boundary)
	parts := make(map[string][]byte)
	var envelope []byte
	for {
		p, err := mr.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("dsh %s: multipart: %w", method, err)
		}
		data, err := io.ReadAll(io.LimitReader(p, 32<<20))
		if err != nil {
			return nil, err
		}
		name := partName(p.Header)
		if name == "" {
			continue
		}
		parts[name] = data
		ct := p.Header.Get("Content-Type")
		if name == "metadata" || name == "response" || strings.HasPrefix(ct, "application/json") {
			if envelope == nil {
				envelope = data
			}
		}
	}
	if envelope == nil {
		return nil, fmt.Errorf("dsh %s: multipart has no JSON envelope", method)
	}
	var parsed struct {
		Type   string `json:"type"`
		RPCID  string `json:"rpcId"`
		Result struct {
			OK    bool            `json:"ok"`
			Value json.RawMessage `json:"value"`
			Error *struct {
				Code    string          `json:"code"`
				Message string          `json:"message"`
				Details json.RawMessage `json:"details"`
			} `json:"error"`
		} `json:"result"`
		Attachments []attachmentMeta `json:"attachments"`
	}
	if err := json.Unmarshal(envelope, &parsed); err != nil {
		return nil, fmt.Errorf("dsh %s: multipart envelope: %w", method, err)
	}
	if parsed.Type != "server-response" || parsed.RPCID != rpcID {
		return nil, fmt.Errorf("dsh %s: invalid multipart envelope", method)
	}
	if !parsed.Result.OK {
		msg := method + " failed"
		code := ""
		var details json.RawMessage
		if parsed.Result.Error != nil {
			if parsed.Result.Error.Message != "" {
				msg = parsed.Result.Error.Message
			}
			code = parsed.Result.Error.Code
			details = parsed.Result.Error.Details
		}
		return nil, &Error{Code: code, Message: msg, Details: details}
	}
	value := parsed.Result.Value
	if len(parsed.Attachments) > 0 {
		assembled, err := assembleAttachments(value, parsed.Attachments, parts)
		if err != nil {
			return nil, fmt.Errorf("dsh %s: multipart attachments: %w", method, err)
		}
		value = assembled
	}
	// Parts exposed to callers exclude the JSON envelope field(s).
	outParts := make(map[string][]byte, len(parts))
	for k, v := range parts {
		if k == "metadata" || k == "response" {
			continue
		}
		outParts[k] = v
	}
	return &MultipartResult{Value: value, Parts: outParts}, nil
}

type attachmentMeta struct {
	Path  []json.RawMessage `json:"path"`
	Codec string            `json:"codec"`
	Part  string            `json:"part"`
}

func assembleAttachments(value json.RawMessage, attachments []attachmentMeta, parts map[string][]byte) (json.RawMessage, error) {
	if len(attachments) == 0 {
		return value, nil
	}
	usedParts := make(map[string]struct{}, len(attachments))
	var root any
	if len(value) == 0 || string(value) == "null" {
		root = nil
	} else if err := json.Unmarshal(value, &root); err != nil {
		return nil, err
	}
	wrap := map[string]any{"value": root}
	for _, att := range attachments {
		if att.Codec != "bytes" || att.Part == "" {
			return nil, fmt.Errorf("invalid attachment")
		}
		if _, dup := usedParts[att.Part]; dup {
			return nil, fmt.Errorf("duplicate part %q", att.Part)
		}
		usedParts[att.Part] = struct{}{}
		raw, ok := parts[att.Part]
		if !ok {
			return nil, fmt.Errorf("missing part %q", att.Part)
		}
		path := make([]any, len(att.Path))
		for i, seg := range att.Path {
			segVal, err := parseAttachmentPathSegment(seg)
			if err != nil {
				return nil, err
			}
			path[i] = segVal
		}
		fullPath := append([]any{"value"}, path...)
		if err := setAtPath(wrap, fullPath, base64.StdEncoding.EncodeToString(raw)); err != nil {
			return nil, err
		}
	}
	for name := range parts {
		if name == "metadata" || name == "response" {
			continue
		}
		if _, ok := usedParts[name]; !ok {
			return nil, fmt.Errorf("unexpected part %q", name)
		}
	}
	out, err := json.Marshal(wrap["value"])
	if err != nil {
		return nil, err
	}
	return out, nil
}

func parseAttachmentPathSegment(raw json.RawMessage) (any, error) {
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return s, nil
	}
	var n json.Number
	if err := json.Unmarshal(raw, &n); err == nil {
		i, err := n.Int64()
		if err != nil || i < 0 {
			return nil, fmt.Errorf("invalid attachment path segment")
		}
		return int(i), nil
	}
	return nil, fmt.Errorf("invalid attachment path segment")
}

func setAtPath(root any, path []any, value any) error {
	if len(path) == 0 {
		return fmt.Errorf("empty path")
	}
	parent := root
	for i := 0; i < len(path)-1; i++ {
		next, err := step(parent, path[i])
		if err != nil {
			return err
		}
		parent = next
	}
	key := path[len(path)-1]
	switch p := parent.(type) {
	case map[string]any:
		k, ok := key.(string)
		if !ok {
			return fmt.Errorf("invalid path")
		}
		if cur, ok := p[k]; !ok || cur != nil {
			return fmt.Errorf("invalid placeholder at %q", k)
		}
		p[k] = value
	case []any:
		idx, err := pathIndex(key)
		if err != nil {
			return err
		}
		if idx < 0 || idx >= len(p) {
			return fmt.Errorf("invalid path index")
		}
		if p[idx] != nil {
			return fmt.Errorf("invalid placeholder at index %d", idx)
		}
		p[idx] = value
	default:
		return fmt.Errorf("invalid path")
	}
	return nil
}

func step(parent any, key any) (any, error) {
	switch p := parent.(type) {
	case map[string]any:
		k, ok := key.(string)
		if !ok {
			return nil, fmt.Errorf("invalid path")
		}
		v, ok := p[k]
		if !ok {
			return nil, fmt.Errorf("invalid path")
		}
		if v == nil {
			return nil, fmt.Errorf("invalid path")
		}
		return v, nil
	case []any:
		idx, err := pathIndex(key)
		if err != nil {
			return nil, err
		}
		if idx < 0 || idx >= len(p) {
			return nil, fmt.Errorf("invalid path index")
		}
		if p[idx] == nil {
			return nil, fmt.Errorf("invalid path")
		}
		return p[idx], nil
	default:
		return nil, fmt.Errorf("invalid path")
	}
}

func pathIndex(key any) (int, error) {
	switch k := key.(type) {
	case int:
		return k, nil
	case float64:
		return int(k), nil
	default:
		return 0, fmt.Errorf("invalid path index")
	}
}

func partName(h textproto.MIMEHeader) string {
	_, params, err := mime.ParseMediaType(h.Get("Content-Disposition"))
	if err != nil {
		return ""
	}
	return params["name"]
}
