package main

import (
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
)

func TestRuntimeLinksPartialDetachCompensates(t *testing.T) {
	root, _ := fixtureGraph(t)
	if err := operate("restore", root); err != nil {
		t.Fatal(err)
	}
	second := filepath.Join(root, "peer/node_modules/package")
	name, err := syscall.UTF16PtrFromString(second)
	if err != nil {
		t.Fatal(err)
	}
	h, err := syscall.CreateFile(name, syscall.GENERIC_READ, syscall.FILE_SHARE_READ|syscall.FILE_SHARE_WRITE, nil, syscall.OPEN_EXISTING, syscall.FILE_FLAG_OPEN_REPARSE_POINT|syscall.FILE_FLAG_BACKUP_SEMANTICS, 0)
	if err != nil {
		t.Fatal(err)
	}
	if err = operate("detach", root); err == nil {
		syscall.CloseHandle(h)
		t.Fatal("locked second junction unexpectedly detached")
	}
	syscall.CloseHandle(h)
	if err = operate("verify", root); err != nil {
		t.Fatalf("partial detach did not compensate: %v", err)
	}
}
func TestRuntimeLinksUnknownReparseTagRejected(t *testing.T) {
	root, _ := fixtureGraph(t)
	shortRoot, err := os.MkdirTemp(filepath.Join(os.Getenv("SystemRoot"), "Temp"), "wi-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(shortRoot)
	if err = os.Rename(root, filepath.Join(shortRoot, "v")); err != nil {
		t.Fatal(err)
	}
	root = filepath.Join(shortRoot, "v")
	filename := filepath.Join(root, "sock")
	var winsock syscall.WSAData
	if err := syscall.WSAStartup(0x202, &winsock); err != nil {
		t.Fatal(err)
	}
	defer syscall.WSACleanup()
	socket, err := syscall.Socket(syscall.AF_UNIX, syscall.SOCK_STREAM, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer syscall.Closesocket(socket)
	if err = syscall.Bind(socket, &syscall.SockaddrUnix{Name: filename}); err != nil {
		t.Fatal(err)
	}
	if err = operate("restore", root); err == nil || !strings.Contains(err.Error(), "reparse") {
		t.Fatalf("unknown AF_UNIX reparse not rejected: %v", err)
	}
}
func TestRuntimeLinksFileAliasAndStrictMissing(t *testing.T) {
	root, g := fixtureGraph(t)
	if err := operate("verify", root); err == nil {
		t.Fatal("strict verify accepted missing aliases")
	}
	g.Links["package/entry.js"] = linkSpec{Target: "package/index.js", Kind: "file"}
	saveGraph(t, root, g)
	for _, command := range []string{"restore", "verify", "restore", "detach", "restore", "verify"} {
		if err := operate(command, root); err != nil {
			t.Fatalf("%s file alias: %v", command, err)
		}
	}
	original, err := os.Stat(filepath.Join(root, "package/index.js"))
	if err != nil {
		t.Fatal(err)
	}
	alias, err := os.Stat(filepath.Join(root, "package/entry.js"))
	if err != nil || !os.SameFile(original, alias) {
		t.Fatal("file alias is not the declared hardlink")
	}
}
func TestRuntimeLinksChangedPhysicalFilesFailBeforeDetach(t *testing.T) {
	for _, extra := range []bool{false, true} {
		root, _ := fixtureGraph(t)
		if err := operate("restore", root); err != nil {
			t.Fatal(err)
		}
		filename := filepath.Join(root, "package/index.js")
		if extra {
			filename = filepath.Join(root, "unknown.js")
		}
		if err := os.WriteFile(filename, []byte("user modification"), 0600); err != nil {
			t.Fatal(err)
		}
		if err := operate("verify", root); err == nil {
			t.Fatal("modified old tree passed the upgrade preflight")
		}
		for _, relative := range []string{"package/node_modules/peer", "peer/node_modules/package"} {
			if _, err := resolveDirectoryLink(filepath.Join(root, relative)); err != nil {
				t.Fatal("modified old tree lost its dependency link")
			}
		}
	}
}
