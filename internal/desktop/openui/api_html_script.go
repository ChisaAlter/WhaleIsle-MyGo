package openui

import (
	"context"
	"crypto/sha256"
	"crypto/sha512"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"mime"
	"strings"
	"unicode/utf8"
)

// scriptIntegrity retains only the strongest valid supported hash algorithm.
// Multiple digests of that strength are alternatives; weaker matches can never
// override a stronger mismatch. No upstream JavaScript is evaluated by Go.
type scriptIntegrity struct {
	strength int
	digests  [][]byte
}

func parseScriptIntegrity(raw string) (scriptIntegrity, error) {
	var result scriptIntegrity
	if len(raw) > MaxEventBytes {
		return result, errors.New("script integrity exceeds the event budget")
	}
	if strings.TrimSpace(raw) == "" {
		return result, nil
	}
	for _, token := range strings.Fields(raw) {
		algorithm, encoded, ok := strings.Cut(token, "-")
		if !ok {
			continue
		}
		strength := 0
		switch algorithm {
		case "sha256":
			strength = 256
		case "sha384":
			strength = 384
		case "sha512":
			strength = 512
		default:
			continue
		}
		// SRI metadata may carry an option suffix; it does not alter the hash.
		encoded, _, _ = strings.Cut(encoded, "?")
		encoded = strings.NewReplacer("-", "+", "_", "/").Replace(encoded)
		digest, err := base64.StdEncoding.Strict().DecodeString(encoded)
		if err != nil {
			digest, err = base64.RawStdEncoding.Strict().DecodeString(encoded)
		}
		if err != nil || len(digest) != strength/8 || strength < result.strength {
			continue
		}
		if strength > result.strength {
			result.strength, result.digests = strength, nil
		}
		result.digests = append(result.digests, digest)
	}
	if result.strength == 0 {
		return result, errors.New("script integrity contains no valid sha256, sha384 or sha512 digest")
	}
	return result, nil
}

func (integrity scriptIntegrity) matches(data []byte) bool {
	var actual []byte
	switch integrity.strength {
	case 0:
		return true
	case 256:
		sum := sha256.Sum256(data)
		actual = sum[:]
	case 384:
		sum := sha512.Sum384(data)
		actual = sum[:]
	case 512:
		sum := sha512.Sum512(data)
		actual = sum[:]
	}
	for _, expected := range integrity.digests {
		if subtle.ConstantTimeCompare(actual, expected) == 1 {
			return true
		}
	}
	return false
}

func classicScriptMIME(raw string) bool {
	mediaType, parameters, err := mime.ParseMediaType(raw)
	if err != nil {
		return false
	}
	switch strings.ToLower(mediaType) {
	case "text/javascript", "application/javascript", "text/ecmascript", "application/ecmascript", "application/x-javascript", "text/x-javascript":
	default:
		return false
	}
	charset := strings.ToLower(parameters["charset"])
	return charset == "" || charset == "utf-8" || charset == "utf8"
}

func (a *ResourceAPI) htmlScript(ctx context.Context, e Event) APIResponse {
	integrity, err := parseScriptIntegrity(e.Query["integrity"])
	if err != nil {
		return apiError(e, 400, err.Error())
	}
	resource, failure := readPublicResource(ctx, e)
	if failure != nil {
		return *failure
	}
	// MIME and UTF-8 checks do not prove script semantics or execute its code.
	// JSON/HTML/octet-stream responses and legacy character sets are rejected.
	// The owned HtmlView must use this only as a classic script in its opaque
	// inner document, whose CSP still denies external fetches and workers. This
	// adapter does not resolve module imports or any dynamic dependencies.
	if !classicScriptMIME(resource.contentType) {
		return apiError(e, 415, "HTML script requires a supported JavaScript MIME type and UTF-8 charset")
	}
	if !utf8.Valid(resource.data) {
		return apiError(e, 415, "HTML script is not valid UTF-8")
	}
	if !integrity.matches(resource.data) {
		return apiError(e, 412, "HTML script integrity does not match the strongest valid supplied digest")
	}
	return apiResult(e, 200, map[string]any{
		"dataUrl":           "data:text/javascript;base64," + base64.StdEncoding.EncodeToString(resource.data),
		"source":            resource.source,
		"integrityVerified": integrity.strength != 0,
	})
}
