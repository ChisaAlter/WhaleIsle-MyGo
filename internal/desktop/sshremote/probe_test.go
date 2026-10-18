package sshremote

import (
	"context"
	"net"
	"testing"
)

func TestProbeTCPLocalListener(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	port := ln.Addr().(*net.TCPAddr).Port
	go func() {
		c, err := ln.Accept()
		if err == nil {
			_ = c.Close()
		}
	}()
	p := ProbeTCP(context.Background(), Snapshot{
		RemoteBindAddress: "127.0.0.1",
		RemotePort:        port,
	})
	if !p.OK {
		t.Fatalf("%+v", p)
	}
}

func TestProbeTCPFailsClosed(t *testing.T) {
	p := ProbeTCP(context.Background(), Snapshot{
		RemoteBindAddress: "127.0.0.1",
		RemotePort:        1, // typically closed
	})
	if p.OK {
		t.Fatal("expected fail")
	}
}
