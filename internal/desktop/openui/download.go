package openui

import (
	"encoding/base64"
	"errors"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"
)

const MaxDownloadBytes = 24 << 20

var downloadMIME = regexp.MustCompile(`^[\w.+-]+/[\w.+-]+(?:;[\w=.+ -]+)*$`)
var badFilename = regexp.MustCompile(`[<>:"/\\|?*\x00-\x1f\x7f]`)
var deviceFilename = regexp.MustCompile(`(?i)^(con|prn|aux|nul|com[0-9]|lpt[0-9])(?:\.|$)`)

func ValidateDownload(e Event) error {
	if e.Filename == "" || len(e.Filename) > 240 || !utf8.ValidString(e.Filename) || badFilename.MatchString(e.Filename) || deviceFilename.MatchString(e.Filename) || e.Filename == "." || e.Filename == ".." || strings.HasSuffix(e.Filename, ".") || strings.HasSuffix(e.Filename, " ") || !downloadMIME.MatchString(e.MIME) || len(e.Content) > MaxDownloadBytes || !utf8.ValidString(e.Content) {
		return errors.New("invalid Whale UI download filename, media type or content")
	}
	return nil
}
func ValidateDownloadPart(e Event) error {
	if !e.UserGesture || e.Sequence == 0 || e.Sequence > 9007199254740991 || e.RequestID != "native-action-"+strconv.FormatUint(e.Sequence, 10) || e.Part < 0 || e.Parts < 1 || e.Parts > 3000 || e.Part >= e.Parts || len(e.Content) > 48000 {
		return errors.New("invalid Whale UI download part")
	}
	return ValidateDownload(e)
}
func DownloadBytes(e Event) ([]byte, error) {
	if err := ValidateDownload(e); err != nil {
		return nil, err
	}
	if !strings.HasPrefix(e.Content, "data:") {
		return []byte(e.Content), nil
	}
	header, payload, ok := strings.Cut(e.Content, ",")
	if !ok {
		return nil, errors.New("invalid Whale UI download data URL")
	}
	var b []byte
	var err error
	if strings.HasSuffix(header, ";base64") {
		b, err = base64.StdEncoding.DecodeString(payload)
	} else {
		var text string
		text, err = url.PathUnescape(payload)
		b = []byte(text)
	}
	if err != nil || len(b) > MaxDownloadBytes {
		return nil, errors.New("invalid or oversized Whale UI download data")
	}
	return b, nil
}
