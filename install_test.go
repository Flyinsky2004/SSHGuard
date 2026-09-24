package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestInstallerMigratesLegacyLogMode(t *testing.T) {
	root := t.TempDir()
	logPath := filepath.Join(root, "auth.log")
	writeFixture(t, logPath, "")
	writeFixture(t, filepath.Join(root, "etc/sshguard.env"), "SSHGUARD_TELEGRAM_TOKEN=secret-token\nSSHGUARD_TELEGRAM_CHAT_ID=12345\nSSHGUARD_LOG_PATH="+logPath+"\n")
	writeFixture(t, filepath.Join(root, "etc/systemd/system/sshguard.service"), "old unit\n")

	output := runInstallerMigration(t, root)
	if strings.Contains(output, "secret-token") {
		t.Fatal("migration printed the Telegram token")
	}
	env := readFixture(t, filepath.Join(root, "etc/sshguard/env"))
	if !strings.Contains(env, "SSHGUARD_TELEGRAM_TOKEN=secret-token\n") || !strings.Contains(env, "SSHGUARD_MODE=log\n") || !strings.Contains(env, "SSHGUARD_LOG_PATH="+logPath+"\n") {
		t.Fatalf("legacy configuration was not preserved: %q", env)
	}
	unit := readFixture(t, filepath.Join(root, "etc/systemd/system/sshguard.service"))
	if !strings.Contains(unit, "EnvironmentFile="+filepath.Join(root, "etc/sshguard/env")) || !strings.Contains(unit, "ReadWritePaths=/run") {
		t.Fatalf("service unit was not migrated: %q", unit)
	}
}

func TestInstallerPreservesSocketMode(t *testing.T) {
	root := t.TempDir()
	socketPath := filepath.Join(root, "custom.sock")
	if err := os.MkdirAll(filepath.Join(root, "opt/SSHGuard"), 0755); err != nil {
		t.Fatal(err)
	}
	writeFixture(t, filepath.Join(root, "etc/sshguard/env"), "SSHGUARD_TELEGRAM_TOKEN=secret-token\nSSHGUARD_TELEGRAM_CHAT_ID=12345\nSSHGUARD_MODE=socket\nSSHGUARD_SOCKET_PATH="+socketPath+"\n")
	writeFixture(t, filepath.Join(root, "etc/pam.d/sshd"), "session required pam_unix.so\n")
	writeFixture(t, filepath.Join(root, "etc/systemd/system/sshguard.service"), "old unit\n")

	runInstallerMigration(t, root)
	env := readFixture(t, filepath.Join(root, "etc/sshguard/env"))
	if strings.Count(env, "SSHGUARD_MODE=socket") != 1 || !strings.Contains(env, "SSHGUARD_SOCKET_PATH="+socketPath) {
		t.Fatalf("socket configuration changed: %q", env)
	}
	helper := readFixture(t, filepath.Join(root, "opt/SSHGuard/sshguard-pam-helper"))
	if !strings.Contains(helper, "-socket '"+socketPath+"'") {
		t.Fatalf("PAM helper does not use the service socket: %q", helper)
	}
	pam := readFixture(t, filepath.Join(root, "etc/pam.d/sshd"))
	if !strings.Contains(pam, "type=open_session ") {
		t.Fatalf("PAM hook does not target session opens: %q", pam)
	}
}

func TestInstallerChecksLocalBinaryVersion(t *testing.T) {
	for _, tc := range []struct {
		version string
		wantOK  bool
	}{
		{version: "v0.0.2", wantOK: true},
		{version: "v0.0.1", wantOK: false},
	} {
		t.Run(tc.version, func(t *testing.T) {
			binary := filepath.Join(t.TempDir(), "sshguard")
			writeFixture(t, binary, "#!/bin/sh\nprintf '%s\\n' '"+tc.version+"'\n")
			if err := os.Chmod(binary, 0755); err != nil {
				t.Fatal(err)
			}
			cmd := exec.Command("bash", "-c", `source ./install.sh
LOCAL_BINARY="$1"
trap cleanup_stage EXIT
stage_binary`, "bash", binary)
			output, err := cmd.CombinedOutput()
			if (err == nil) != tc.wantOK {
				t.Fatalf("stage_binary result: err=%v output=%s", err, output)
			}
		})
	}
}

func runInstallerMigration(t *testing.T, root string) string {
	t.Helper()
	script := `source ./install.sh
INSTALL_DIR="$1/opt/SSHGuard"
ENV_FILE="$1/etc/sshguard/env"
LEGACY_ENV_FILE="$1/etc/sshguard.env"
SERVICE_FILE="$1/etc/systemd/system/sshguard.service"
PAM_FILE="$1/etc/pam.d/sshd"
PAM_HELPER="$INSTALL_DIR/sshguard-pam-helper"
detect_installation
load_existing_config
write_env
write_service
configure_pam`
	cmd := exec.Command("bash", "-c", script, "bash", root)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("installer migration failed: %v\n%s", err, output)
	}
	return string(output)
}

func writeFixture(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
}

func readFixture(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}
