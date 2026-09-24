package main

import (
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestSocketReceivesSplitPAMEvent(t *testing.T) {
	dir, err := os.MkdirTemp("/tmp", "sshguard-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(dir) })
	path := filepath.Join(dir, "sshguard.sock")
	events := make(chan *SSHEvent, 1)
	listener, err := listenSocket(path, events)
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()

	conn, err := net.Dial("unix", path)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	payload := []byte(`{"type":"ssh_login","user":"root","source_ip":"14.137.235.102","timestamp":"2026-09-25T02:48:45+08:00","hostname":"localhost","auth_method":"pam"}` + "\n")
	if _, err := conn.Write(payload[:8]); err != nil {
		t.Fatal(err)
	}
	time.Sleep(20 * time.Millisecond)
	if _, err := conn.Write(payload[8:]); err != nil {
		t.Fatal(err)
	}
	select {
	case ev := <-events:
		if ev.User != "root" || ev.SourceIP != "14.137.235.102" || ev.AuthMethod != "pam" {
			t.Fatalf("unexpected event: %+v", ev)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("socket did not deliver the PAM event")
	}
}

func TestSocketDoesNotDeleteNonSocketFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sshguard.sock")
	if err := os.WriteFile(path, []byte("keep me"), 0600); err != nil {
		t.Fatal(err)
	}
	if listener, err := listenSocket(path, make(chan *SSHEvent)); err == nil {
		listener.Close()
		t.Fatal("ordinary file was accepted as a socket")
	}
	if data, err := os.ReadFile(path); err != nil || string(data) != "keep me" {
		t.Fatalf("ordinary file was modified: %q, %v", data, err)
	}
}
