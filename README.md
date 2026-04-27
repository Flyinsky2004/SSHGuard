# SSHHGuard

Monitors SSH authentication logs in real time and sends Telegram notifications on every successful login.

## Quick Install

```bash
curl -fsSL https://raw.githubusercontent.com/Flyinsky2004/SSHGuard/main/install.sh | sudo bash
```

Or clone and run locally:

```bash
git clone https://github.com/Flyinsky2004/SSHGuard.git
cd SSHHGuard
sudo bash install.sh
```

## What the Installer Does

The interactive `install.sh` script walks you through:

1. **Download** — pulls the latest prebuilt Linux amd64 binary from GitHub Actions artifacts into `/opt/SSHGuard/sshguard`
2. **chmod +x** — makes the binary executable
3. **Configuration** — prompts for:
   - Telegram Bot Token
   - Telegram Chat ID
   - SSH log path (auto-detected if left blank)
4. **systemd service** (optional) — creates and enables `sshguard.service` so it starts on boot and restarts automatically

### Walkthrough

```
$ sudo bash install.sh

  ╔══════════════════════════════════╗
  ║        SSHHGuard Installer       ║
  ╚══════════════════════════════════╝

[+] Checking dependencies...
[+] GitHub CLI (gh) is authenticated — will use it to download artifacts.

[+] Downloading SSHHGuard binary...
[+] Binary installed to /opt/SSHGuard/sshguard

  ─── Configuration ───
  Press Enter to accept defaults (shown in brackets).

[?] Telegram Bot Token:
> 123456:ABC-DEF1234ghijk

[?] Telegram Chat ID:
> 987654321

  SSH auth log location (auto-detected if left blank):
    Found: /var/log/auth.log (Debian/Ubuntu)

[?] Log path [/var/log/auth.log]:
> (press Enter)

[?] Install systemd service so SSHHGuard starts on boot? [Y/n]:
> y

  ─── Review ───

  Install dir:    /opt/SSHGuard
  Binary:         /opt/SSHGuard/sshguard
  Log file:       /var/log/auth.log
  Telegram token: 123456:...
  Telegram chat:  987654321
  systemd service: Yes

[?] Proceed with installation? [Y/n]:
> y

[+] Environment file written to /etc/sshguard/env
[+] systemd service installed at /etc/systemd/system/sshguard.service
[?] Start SSHHGuard now? [Y/n]:
> y
[+] Service started and enabled on boot.

  ✓ Installation complete!
```

## Prerequisites

- **Linux amd64** (the prebuilt binary targets this platform)
- **root / sudo** access
- `curl` and `unzip` (script will offer to install them if missing)
- A **Telegram Bot Token** from [@BotFather](https://t.me/BotFather)
- A **Telegram Chat ID** — send `/start` to your bot, then check `https://api.telegram.org/bot<TOKEN>/getUpdates`

### GitHub Authentication (for downloading artifacts)

The installer prefers the **GitHub CLI (`gh`)**. If `gh` is installed and logged in, downloads happen automatically.

If `gh` is not available, you'll be prompted for a **Personal Access Token** (no scopes needed for public repos). Create one at [github.com/settings/tokens](https://github.com/settings/tokens).

## Manual Usage

If you prefer to run SSHHGuard directly:

```bash
/opt/SSHGuard/sshguard -token <bot_token> -chat-id <chat_id> [-log <path>]
```

Or with environment variables:

```bash
export SSHGUARD_TELEGRAM_TOKEN=your_token
export SSHGUARD_TELEGRAM_CHAT_ID=your_chat_id
export SSHGUARD_LOG_PATH=/var/log/auth.log   # optional

/opt/SSHGuard/sshguard
```

## Configuration Reference

| Flag | Env Variable | Required | Default |
|------|-------------|----------|---------|
| `-token` | `SSHGUARD_TELEGRAM_TOKEN` | Yes | — |
| `-chat-id` | `SSHGUARD_TELEGRAM_CHAT_ID` | Yes | — |
| `-log` | `SSHGUARD_LOG_PATH` | No | `/var/log/auth.log` |

The log path auto-detects between `/var/log/auth.log` (Debian/Ubuntu) and `/var/log/secure` (RHEL/CentOS).

## Service Management

```bash
systemctl status sshguard     # check status
systemctl stop sshguard       # stop
systemctl start sshguard      # start
systemctl restart sshguard    # restart

journalctl -u sshguard -f     # follow logs
```

## Build from Source

```bash
git clone https://github.com/Flyinsky2004/SSHGuard.git
cd SSHHGuard
go build -ldflags="-s -w" -o sshguard .
```

## License

MIT
