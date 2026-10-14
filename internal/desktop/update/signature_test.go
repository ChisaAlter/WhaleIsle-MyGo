package update

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestParseAuthenticodeOutput(t *testing.T) {
	info := parseAuthenticodeOutput("STATUS=Valid\nPUBLISHER=CN=WhaleIsle Inc\n")
	if info.Status != SignatureValid || !strings.Contains(info.Publisher, "WhaleIsle") {
		t.Fatalf("%+v", info)
	}
	none := parseAuthenticodeOutput("STATUS=NotSigned\nPUBLISHER=\n")
	if none.Status != SignatureNone {
		t.Fatalf("%+v", none)
	}
	bad := parseAuthenticodeOutput("STATUS=HashMismatch\nPUBLISHER=CN=X\n")
	if bad.Status != SignatureInvalid {
		t.Fatalf("%+v", bad)
	}
}

func TestCheckSignatureNotRequested(t *testing.T) {
	old := authenticodeProbe
	t.Cleanup(func() { authenticodeProbe = old })
	called := false
	authenticodeProbe = func(context.Context, string) (SignatureInfo, error) {
		called = true
		return SignatureInfo{}, nil
	}
	info, err := CheckSignature(context.Background(), "x.exe", VerifyOptions{})
	if err != nil || info.Status != SignatureSkipped {
		t.Fatalf("info=%+v err=%v", info, err)
	}
	if called {
		t.Fatal("probe should not run when signature not requested")
	}
}

func TestCheckSignatureRequireFailClosed(t *testing.T) {
	old := authenticodeProbe
	t.Cleanup(func() { authenticodeProbe = old })
	authenticodeProbe = func(context.Context, string) (SignatureInfo, error) {
		return SignatureInfo{Status: SignatureNone, Detail: "NotSigned"}, nil
	}
	_, err := CheckSignature(context.Background(), "x.exe", VerifyOptions{RequireSignature: true})
	if err == nil {
		t.Fatal("expected fail closed")
	}
}

func TestCheckSignatureProbeErrorFailClosed(t *testing.T) {
	old := authenticodeProbe
	t.Cleanup(func() { authenticodeProbe = old })
	authenticodeProbe = func(context.Context, string) (SignatureInfo, error) {
		return SignatureInfo{}, errors.New("powershell missing")
	}
	_, err := CheckSignature(context.Background(), "x.exe", VerifyOptions{RequireSignature: true})
	if err == nil {
		t.Fatal("expected probe error")
	}
}

func TestCheckSignatureExpectedPublisher(t *testing.T) {
	old := authenticodeProbe
	t.Cleanup(func() { authenticodeProbe = old })
	authenticodeProbe = func(context.Context, string) (SignatureInfo, error) {
		return SignatureInfo{Status: SignatureValid, Publisher: "CN=WhaleIsle Inc, O=WhaleIsle"}, nil
	}
	info, err := CheckSignature(context.Background(), "x.exe", VerifyOptions{ExpectedPublisher: "whaleisle"})
	if err != nil || info.Status != SignatureValid {
		t.Fatalf("info=%+v err=%v", info, err)
	}
	_, err = CheckSignature(context.Background(), "x.exe", VerifyOptions{ExpectedPublisher: "Other Corp"})
	if err == nil {
		t.Fatal("expected publisher mismatch")
	}
}

func TestCheckSignatureSkippedNoOp(t *testing.T) {
	old := authenticodeProbe
	t.Cleanup(func() { authenticodeProbe = old })
	authenticodeProbe = func(context.Context, string) (SignatureInfo, error) {
		return SignatureInfo{Status: SignatureSkipped, Detail: "Authenticode unavailable on linux"}, nil
	}
	info, err := CheckSignature(context.Background(), "x.exe", VerifyOptions{RequireSignature: true})
	if err != nil {
		t.Fatal(err)
	}
	if info.Status != SignatureSkipped {
		t.Fatalf("%+v", info)
	}
}

func TestEnforceSignatureValid(t *testing.T) {
	err := enforceSignature(SignatureInfo{Status: SignatureValid, Publisher: "CN=Acme"}, VerifyOptions{
		RequireSignature:  true,
		ExpectedPublisher: "Acme",
	})
	if err != nil {
		t.Fatal(err)
	}
}
