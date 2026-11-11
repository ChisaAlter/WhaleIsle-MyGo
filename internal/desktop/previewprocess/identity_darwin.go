//go:build darwin && (amd64 || arm64)

package previewprocess

import (
	"bytes"
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"unsafe"

	"github.com/ebitengine/purego"
)

func macCodesign(ctx context.Context, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "/usr/bin/codesign", args...)
	cmd.Env = []string{"PATH=/usr/bin:/bin", "LANG=C", "LC_ALL=C"}
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("preview signature check failed: %w", err)
	}
	return out, nil
}
func macCodeField(raw []byte, name, pattern string) (string, error) {
	var found string
	for _, line := range strings.Split(string(raw), "\n") {
		if v, ok := strings.CutPrefix(line, name+"="); ok {
			if found != "" || !regexp.MustCompile(pattern).MatchString(v) {
				return "", errors.New("ambiguous preview signing identity")
			}
			found = v
		}
	}
	if found == "" {
		return "", errors.New("missing preview signing identity")
	}
	return found, nil
}
func macSandboxEntitlements(raw []byte) error {
	if len(raw) == 0 || len(raw) > 65536 {
		return errors.New("invalid preview entitlements")
	}
	d := xml.NewDecoder(bytes.NewReader(raw))
	var tokens []string
	for {
		t, err := d.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		switch v := t.(type) {
		case xml.StartElement:
			if v.Name.Space != "" {
				return errors.New("unexpected entitlement namespace")
			}
			for _, a := range v.Attr {
				if v.Name.Local != "plist" || a.Name.Local != "version" || a.Value != "1.0" {
					return errors.New("unexpected entitlement attribute")
				}
			}
			tokens = append(tokens, "<"+v.Name.Local+">")
		case xml.EndElement:
			tokens = append(tokens, "</"+v.Name.Local+">")
		case xml.CharData:
			if s := strings.TrimSpace(string(v)); s != "" {
				tokens = append(tokens, s)
			}
		}
	}
	if strings.Join(tokens, "|") != "<plist>|<dict>|<key>|com.apple.security.app-sandbox|</key>|<true>|</true>|</dict>|</plist>" {
		return errors.New("preview requires exactly App Sandbox without network, file selection or inheritance grants")
	}
	return nil
}
func verifyMacBundle(ctx context.Context, helper string) (string, error) {
	host, err := os.Executable()
	if err != nil {
		return "", err
	}
	host, err = filepath.EvalSymlinks(host)
	if err != nil {
		return "", err
	}
	expected := HelperPath(host)
	actual, err := filepath.EvalSymlinks(helper)
	if err != nil {
		return "", err
	}
	if filepath.Clean(helper) != expected || actual != expected {
		return "", errors.New("preview helper must be the embedded nonsymlink companion")
	}
	hostReq := `anchor apple generic and identifier "ai.deepseek.harness.gui.mygo"`
	if _, err = macCodesign(ctx, "--verify", "--strict", "-R", hostReq, host); err != nil {
		return "", err
	}
	info, err := macCodesign(ctx, "-d", "--verbose=4", host)
	if err != nil {
		return "", err
	}
	team, err := macCodeField(info, "TeamIdentifier", `^[A-Z0-9]{10}$`)
	if err != nil {
		return "", err
	}
	bundle := filepath.Dir(filepath.Dir(filepath.Dir(helper)))
	req := `anchor apple generic and identifier "org.whaleisle.preview" and certificate leaf[subject.OU] = "` + team + `"`
	if _, err = macCodesign(ctx, "--verify", "--deep", "--strict", "-R", req, bundle); err != nil {
		return "", err
	}
	cmd := exec.CommandContext(ctx, "/usr/bin/codesign", "-d", "--entitlements", ":-", helper)
	cmd.Env = []string{"PATH=/usr/bin:/bin", "LANG=C", "LC_ALL=C"}
	ent, err := cmd.Output()
	if err != nil {
		return "", err
	}
	if err = macSandboxEntitlements(ent); err != nil {
		return "", err
	}
	info, err = macCodesign(ctx, "-d", "--verbose=4", helper)
	if err != nil {
		return "", err
	}
	hash, err := macCodeField(info, "CDHash", `^[a-fA-F0-9]{40}$`)
	if err != nil {
		return "", err
	}
	return `cdhash H"` + hash + `"`, nil
}

// Check the LIVE child, not just a pathname that could be replaced after the
// static check. Security.framework asks the kernel for the running code object.
// App Sandbox activation is enforced by macOS for its verified signed image;
// the helper additionally validates its own entitlements before sending ready.
func verifyMacProcess(pid int, requirement string) (err error) {
	security, err := purego.Dlopen("/System/Library/Frameworks/Security.framework/Security", purego.RTLD_NOW|purego.RTLD_LOCAL)
	if err != nil {
		return err
	}
	defer purego.Dlclose(security)
	cf, err := purego.Dlopen("/System/Library/Frameworks/CoreFoundation.framework/CoreFoundation", purego.RTLD_NOW|purego.RTLD_LOCAL)
	if err != nil {
		return err
	}
	defer purego.Dlclose(cf)
	// RegisterLibFunc panics on missing symbols; an older OS must fail closed.
	defer func() {
		if v := recover(); v != nil {
			err = fmt.Errorf("macOS identity API unavailable: %v", v)
		}
	}()
	var stringCreate func(uintptr, string, uint32) uintptr
	var numberCreate func(uintptr, int, unsafe.Pointer) uintptr
	var dictionaryCreate func(uintptr, unsafe.Pointer, unsafe.Pointer, int, uintptr, uintptr) uintptr
	var release func(uintptr)
	var guest func(uintptr, uintptr, uint32, *uintptr) int32
	var createRequirement func(uintptr, uint32, *uintptr) int32
	var check func(uintptr, uint32, uintptr) int32
	purego.RegisterLibFunc(&stringCreate, cf, "CFStringCreateWithCString")
	purego.RegisterLibFunc(&numberCreate, cf, "CFNumberCreate")
	purego.RegisterLibFunc(&dictionaryCreate, cf, "CFDictionaryCreate")
	purego.RegisterLibFunc(&release, cf, "CFRelease")
	purego.RegisterLibFunc(&guest, security, "SecCodeCopyGuestWithAttributes")
	purego.RegisterLibFunc(&createRequirement, security, "SecRequirementCreateWithString")
	purego.RegisterLibFunc(&check, security, "SecCodeCheckValidity")
	symbol, err := purego.Dlsym(security, "kSecGuestAttributePid")
	if err != nil {
		return err
	}
	key := *(*uintptr)(unsafe.Pointer(symbol))
	nativePID := int32(pid)
	value := numberCreate(0, 3, unsafe.Pointer(&nativePID))
	if value == 0 {
		return errors.New("cannot construct preview process identity")
	}
	defer release(value)
	attrs := dictionaryCreate(0, unsafe.Pointer(&key), unsafe.Pointer(&value), 1, 0, 0)
	if attrs == 0 {
		return errors.New("cannot construct preview process attributes")
	}
	defer release(attrs)
	var code uintptr
	if guest(0, attrs, 0, &code) != 0 {
		return errors.New("cannot identify running preview helper")
	}
	defer release(code)
	text := stringCreate(0, requirement, 0x08000100)
	if text == 0 {
		return errors.New("cannot construct preview requirement")
	}
	defer release(text)
	var req uintptr
	if createRequirement(text, 0, &req) != 0 {
		return errors.New("invalid preview code requirement")
	}
	defer release(req)
	if check(code, 0, req) != 0 {
		return errors.New("running preview helper differs from verified signed companion")
	}
	return nil
}
