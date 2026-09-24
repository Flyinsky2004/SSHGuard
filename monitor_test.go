package main

import "testing"

func TestParseLineDebian13(t *testing.T) {
	line := "2026-09-25T02:48:45.511991+08:00 localhost sshd-session[14193]: Accepted publickey for root from 14.137.235.102 port 43156 ssh2: ED25519 SHA256:5nOdyW0xpPL8w7/CTZPg1y8mTwHWy8CTkmlcyqVhlRU"
	ev := parseLine(line)
	if ev == nil {
		t.Fatal("Debian 13 accepted login was ignored")
	}
	if ev.Hostname != "localhost" || ev.User != "root" || ev.AuthMethod != "publickey" || ev.SourceIP != "14.137.235.102" || ev.SourcePort != "43156" {
		t.Fatalf("unexpected event: %+v", ev)
	}
	if got := ev.Timestamp.Format("2006-01-02T15:04:05.999999Z07:00"); got != "2026-09-25T02:48:45.511991+08:00" {
		t.Fatalf("unexpected timestamp: %s", got)
	}
}

func TestParseLineLegacyAndNoise(t *testing.T) {
	legacy := "Jan 22 10:15:30 myhost sshd[12345]: Accepted password for admin from 1.2.3.4 port 12345 ssh2"
	if ev := parseLine(legacy); ev == nil || ev.User != "admin" || ev.AuthMethod != "password" {
		t.Fatalf("legacy SSH login not parsed: %+v", ev)
	}
	paddedDay := "Jan  2 10:15:30 myhost sshd[12345]: Accepted password for admin from 1.2.3.4 port 12345 ssh2"
	if ev := parseLine(paddedDay); ev == nil || ev.Timestamp.Day() != 2 {
		t.Fatalf("space-padded syslog day not parsed: %+v", ev)
	}
	for _, line := range []string{
		"2026-09-25T02:48:45.512700+08:00 localhost sshd-session[14193]: pam_unix(sshd:session): session opened for user root(uid=0) by root(uid=0)",
		"2026-09-25T02:48:43.230992+08:00 localhost sshd-session[14158]: Disconnected from user root 14.137.235.102 port 34396",
		"2026-09-25T02:48:45.515984+08:00 localhost systemd-logind[541]: New session 77 of user root.",
	} {
		if ev := parseLine(line); ev != nil {
			t.Fatalf("non-login event accepted: %+v", ev)
		}
	}
}

func TestDuplicateAcceptedLine(t *testing.T) {
	line := "2026-09-25T02:48:45.511991+08:00 localhost sshd-session[14193]: Accepted publickey for root from 14.137.235.102 port 43156 ssh2"
	var previous string
	if ev := parseUniqueLogin(line, &previous); ev == nil {
		t.Fatal("first login was ignored")
	}
	parseUniqueLogin("2026-09-25T02:48:45.512700+08:00 localhost systemd-logind[541]: New session 77 of user root.", &previous)
	if ev := parseUniqueLogin(line, &previous); ev != nil {
		t.Fatalf("duplicate login was emitted: %+v", ev)
	}
}
