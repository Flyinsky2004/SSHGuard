#!/usr/bin/env bash
set -euo pipefail

# ============================================================
# SSHHGuard — Interactive Installer
# ============================================================

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
BOLD='\033[1m'
NC='\033[0m'

INSTALL_DIR="/opt/SSHGuard"
BINARY_NAME="sshguard"
ENV_FILE="/etc/sshguard/env"
SERVICE_FILE="/etc/systemd/system/sshguard.service"

# Default artifact — latest CI build on main branch
ARTIFACT_RUN_ID="24980111221"
ARTIFACT_ID="6654616310"
REPO="Flyinsky2004/SSHGuard"

# -----------------------------------------------------------
# Helpers
# -----------------------------------------------------------
banner() {
    echo -e "${CYAN}${BOLD}"
    echo "  ╔══════════════════════════════════╗"
    echo "  ║        SSHHGuard Installer       ║"
    echo "  ╚══════════════════════════════════╝"
    echo -e "${NC}"
}

info()    { echo -e "${GREEN}[+]${NC} $*"; }
warn()    { echo -e "${YELLOW}[!]${NC} $*"; }
error()   { echo -e "${RED}[x]${NC} $*"; }
prompt()  { echo -ne "${BOLD}[?]${NC} $* "; }

die() {
    error "$*"
    exit 1
}

require_root() {
    if [[ $EUID -ne 0 ]]; then
        die "This script must be run as root. Use: sudo bash install.sh"
    fi
}

# -----------------------------------------------------------
# Dependency checks
# -----------------------------------------------------------
check_deps() {
    info "Checking dependencies..."

    local missing=()

    command -v curl >/dev/null 2>&1 || missing+=("curl")
    command -v unzip >/dev/null 2>&1 || missing+=("unzip")
    command -v tar  >/dev/null 2>&1 || missing+=("tar")

    if [[ ${#missing[@]} -gt 0 ]]; then
        warn "Missing: ${missing[*]}"
        prompt "Install them now? (apt-get install ${missing[*]}) [Y/n]"
        read -r ans
        if [[ "${ans:-y}" =~ ^[Yy]$ ]]; then
            apt-get update -qq && apt-get install -y "${missing[@]}"
        else
            die "Cannot continue without: ${missing[*]}"
        fi
    fi

    # Check for gh CLI (preferred) or prompt for token later
    if command -v gh >/dev/null 2>&1; then
        if gh auth status >/dev/null 2>&1; then
            info "GitHub CLI (gh) is authenticated — will use it to download artifacts."
            DOWNLOAD_METHOD="gh"
        else
            info "GitHub CLI found but not logged in. Run 'gh auth login' or use a PAT."
            DOWNLOAD_METHOD="curl"
        fi
    else
        DOWNLOAD_METHOD="curl"
    fi
}

# -----------------------------------------------------------
# Download artifact
# -----------------------------------------------------------
download_artifact() {
    info "Downloading SSHHGuard binary..."

    mkdir -p "$INSTALL_DIR"

    local archive="$INSTALL_DIR/sshguard-artifact.zip"

    if [[ "$DOWNLOAD_METHOD" == "gh" ]]; then
        info "Using gh CLI to download artifact (run ID: $ARTIFACT_RUN_ID)..."
        cd "$INSTALL_DIR"
        gh run download "$ARTIFACT_RUN_ID" \
            --repo "$REPO" \
            --dir "$INSTALL_DIR" \
            || die "Download failed. Check that the run exists and gh is authenticated."
        # gh run download extracts the zip; the artifact name is sshguard-linux-amd64
        # It creates a subdirectory named after the artifact, or extracts directly.
        # Handle both cases.
        if [[ -d "$INSTALL_DIR/sshguard-linux-amd64" ]]; then
            mv "$INSTALL_DIR/sshguard-linux-amd64/$BINARY_NAME" "$INSTALL_DIR/$BINARY_NAME" 2>/dev/null || true
            rm -rf "$INSTALL_DIR/sshguard-linux-amd64"
        fi
    else
        # Fallback: download via API with a PAT
        warn "GitHub CLI not available or not authenticated."
        echo ""
        echo -e "  ${BOLD}Option A${NC}: Install GitHub CLI and authenticate:"
        echo "    sudo apt install gh && gh auth login"
        echo ""
        echo -e "  ${BOLD}Option B${NC}: Provide a GitHub Personal Access Token (PAT)"
        echo "    Create one at: https://github.com/settings/tokens"
        echo "    Required scopes: none (public repo only)"
        echo ""

        prompt "Enter GitHub PAT (or press Enter to cancel):"
        read -rs token
        echo ""

        if [[ -z "$token" ]]; then
            die "No token provided. Aborting."
        fi

        info "Downloading via API..."
        curl -fsSL \
            -H "Authorization: Bearer $token" \
            -H "Accept: application/vnd.github+json" \
            -o "$archive" \
            "https://api.github.com/repos/$REPO/actions/artifacts/$ARTIFACT_ID/zip" \
            || die "Download failed. Check your token and network."

        # Extract
        cd "$INSTALL_DIR"
        unzip -o "$archive" -d "$INSTALL_DIR" >/dev/null
        rm -f "$archive"

        # The zip from GitHub API contains a flat file named sshguard
        if [[ ! -f "$INSTALL_DIR/$BINARY_NAME" ]]; then
            # Maybe it's in a subdirectory
            local found
            found=$(find "$INSTALL_DIR" -name "$BINARY_NAME" -type f 2>/dev/null | head -1)
            if [[ -n "$found" && "$found" != "$INSTALL_DIR/$BINARY_NAME" ]]; then
                mv "$found" "$INSTALL_DIR/$BINARY_NAME"
            fi
        fi
    fi

    # Verify binary exists
    if [[ ! -f "$INSTALL_DIR/$BINARY_NAME" ]]; then
        die "Binary not found after extraction. Check archive contents."
    fi

    chmod +x "$INSTALL_DIR/$BINARY_NAME"
    info "Binary installed to $INSTALL_DIR/$BINARY_NAME"
    info "Version: $("$INSTALL_DIR/$BINARY_NAME" -help 2>&1 | head -1 || echo 'unknown')"
}

# -----------------------------------------------------------
# Interactive configuration
# -----------------------------------------------------------
configure() {
    echo ""
    echo -e "${CYAN}${BOLD}  ─── Configuration ───${NC}"
    echo -e "  Press Enter to accept defaults (shown in brackets)."
    echo ""

    # --- Telegram Bot Token ---
    while true; do
        prompt "Telegram Bot Token:"
        read -r TELEGRAM_TOKEN

        if [[ -z "$TELEGRAM_TOKEN" ]]; then
            warn "Telegram Bot Token is required."
            echo "  Get one from @BotFather on Telegram: https://t.me/BotFather"
            continue
        fi
        break
    done

    # --- Telegram Chat ID ---
    while true; do
        prompt "Telegram Chat ID:"
        read -r TELEGRAM_CHAT_ID

        if [[ -z "$TELEGRAM_CHAT_ID" ]]; then
            warn "Telegram Chat ID is required."
            echo "  Send /start to your bot, then visit:"
            echo "  https://api.telegram.org/bot<TOKEN>/getUpdates"
            echo "  Look for 'chat':{'id': ...} in the JSON response."
            continue
        fi
        break
    done

    # --- Log path ---
    echo ""
    echo "  SSH auth log location (auto-detected if left blank):"
    if [[ -f /var/log/auth.log ]]; then
        echo -e "    ${GREEN}Found:${NC} /var/log/auth.log (Debian/Ubuntu)"
        DETECTED_LOG="/var/log/auth.log"
    elif [[ -f /var/log/secure ]]; then
        echo -e "    ${GREEN}Found:${NC} /var/log/secure (RHEL/CentOS)"
        DETECTED_LOG="/var/log/secure"
    else
        DETECTED_LOG="/var/log/auth.log"
    fi

    prompt "Log path [$DETECTED_LOG]:"
    read -r LOG_PATH
    LOG_PATH="${LOG_PATH:-$DETECTED_LOG}"

    # --- systemd service ---
    echo ""
    prompt "Install systemd service so SSHHGuard starts on boot? [Y/n]"
    read -r INSTALL_SERVICE
    INSTALL_SERVICE="${INSTALL_SERVICE:-y}"

    # --- Summary ---
    echo ""
    echo -e "${CYAN}${BOLD}  ─── Review ───${NC}"
    echo ""
    echo -e "  ${BOLD}Install dir:${NC}    $INSTALL_DIR"
    echo -e "  ${BOLD}Binary:${NC}         $INSTALL_DIR/$BINARY_NAME"
    echo -e "  ${BOLD}Log file:${NC}      $LOG_PATH"
    echo -e "  ${BOLD}Telegram token:${NC} ${TELEGRAM_TOKEN:0:8}..."
    echo -e "  ${BOLD}Telegram chat:${NC}  $TELEGRAM_CHAT_ID"
    echo -e "  ${BOLD}systemd service:${NC} $([[ "$INSTALL_SERVICE" =~ ^[Yy]$ ]] && echo 'Yes' || echo 'No')"
    echo ""

    prompt "Proceed with installation? [Y/n]"
    read -r CONFIRM
    if [[ ! "${CONFIRM:-y}" =~ ^[Yy]$ ]]; then
        die "Installation cancelled."
    fi
}

# -----------------------------------------------------------
# Write env file
# -----------------------------------------------------------
write_env() {
    mkdir -p "$(dirname "$ENV_FILE")"
    cat > "$ENV_FILE" <<EOF
# SSHHGuard environment — managed by install.sh
SSHGUARD_TELEGRAM_TOKEN=$TELEGRAM_TOKEN
SSHGUARD_TELEGRAM_CHAT_ID=$TELEGRAM_CHAT_ID
SSHGUARD_LOG_PATH=$LOG_PATH
EOF
    chmod 600 "$ENV_FILE"
    info "Environment file written to $ENV_FILE"
}

# -----------------------------------------------------------
# Install systemd service
# -----------------------------------------------------------
install_service() {
    if [[ ! "$INSTALL_SERVICE" =~ ^[Yy]$ ]]; then
        info "Skipping systemd service. Run manually:"
        echo ""
        echo "  $INSTALL_DIR/$BINARY_NAME -token <token> -chat-id <id>"
        return
    fi

    cat > "$SERVICE_FILE" <<EOF
[Unit]
Description=SSHHGuard — SSH Login Monitor & Telegram Notifier
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
EnvironmentFile=$ENV_FILE
ExecStart=$INSTALL_DIR/$BINARY_NAME
Restart=always
RestartSec=30
StandardOutput=journal
StandardError=journal

# Security hardening
NoNewPrivileges=yes
PrivateTmp=yes
ProtectSystem=strict
ProtectHome=yes
ReadWritePaths=$INSTALL_DIR
ReadOnlyPaths=$(dirname "$LOG_PATH")

[Install]
WantedBy=multi-user.target
EOF

    systemctl daemon-reload
    info "systemd service installed at $SERVICE_FILE"

    prompt "Start SSHHGuard now? [Y/n]"
    read -r START_NOW
    if [[ "${START_NOW:-y}" =~ ^[Yy]$ ]]; then
        systemctl enable --now sshguard
        info "Service started and enabled on boot."
        echo ""
        echo "  Manage with:"
        echo "    systemctl status sshguard"
        echo "    journalctl -u sshguard -f"
    else
        systemctl enable sshguard
        info "Service enabled (will start on next boot)."
        echo ""
        echo "  Start manually:  systemctl start sshguard"
    fi
}

# -----------------------------------------------------------
# Main
# -----------------------------------------------------------
main() {
    banner
    require_root
    check_deps

    echo ""
    download_artifact
    configure
    write_env
    install_service

    echo ""
    echo -e "${GREEN}${BOLD}  ✓ Installation complete!${NC}"
    echo ""
}

main "$@"
