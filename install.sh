#!/usr/bin/env bash
set -euo pipefail

RELEASE_VERSION="v0.0.2"
RELEASE_BASE="https://github.com/Flyinsky2004/SSHGuard/releases/download/$RELEASE_VERSION"
INSTALL_DIR="/opt/SSHGuard"
ENV_FILE="/etc/sshguard/env"
LEGACY_ENV_FILE="/etc/sshguard.env"
SERVICE_FILE="/etc/systemd/system/sshguard.service"
PAM_FILE="/etc/pam.d/sshd"
PAM_HELPER="$INSTALL_DIR/sshguard-pam-helper"
SOCKET_PATH="/run/sshguard.sock"

LOCAL_BINARY=""
UPDATE_ONLY=false
INSTALLED=false
INSTALL_SERVICE=true
RUN_MODE=""
LOG_PATH=""
SOURCE_ENV=""
STAGING_DIR=""
BACKUP_DIR=""

info() { printf '[+] %s\n' "$*"; }
warn() { printf '[!] %s\n' "$*" >&2; }
die() { printf '[✗] %s\n' "$*" >&2; exit 1; }

usage() {
    cat <<'EOF'
用法: sudo bash install.sh [--update] [--binary /path/to/sshguard]

默认检测旧安装并升级；未安装时进入交互式安装。
--update 仅更新已有安装，保留 Telegram 配置和原有运行模式。
--binary 使用本地 v0.0.2 二进制文件，供发布前或离线安装使用。
EOF
}

parse_args() {
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --update) UPDATE_ONLY=true; shift ;;
            --binary)
                [[ $# -ge 2 ]] || die "--binary 缺少文件路径"
                LOCAL_BINARY="$2"
                shift 2
                ;;
            -h|--help) usage; exit 0 ;;
            *) die "未知参数: $1" ;;
        esac
    done
}

check_platform() {
    [[ $EUID -eq 0 ]] || die "请以 root 身份运行"
    [[ $(uname -s) == Linux && $(uname -m) == x86_64 ]] || die "v0.0.2 预编译文件仅支持 Linux amd64"
    command -v systemctl >/dev/null || die "未找到 systemctl"
    if [[ -z $LOCAL_BINARY ]]; then
        command -v curl >/dev/null || die "未找到 curl"
        command -v sha256sum >/dev/null || die "未找到 sha256sum"
    fi
}

read_env_value() {
    local key="$1" file="$2"
    awk -v key="$key" 'index($0, key "=") == 1 { value = substr($0, length(key) + 2) } END { print value }' "$file"
}

detect_installation() {
    if [[ -f $ENV_FILE ]]; then
        SOURCE_ENV="$ENV_FILE"
    elif [[ -f $LEGACY_ENV_FILE ]]; then
        SOURCE_ENV="$LEGACY_ENV_FILE"
    fi
    if [[ -e $INSTALL_DIR/sshguard || -e $SERVICE_FILE || -n $SOURCE_ENV ]]; then
        INSTALLED=true
    fi
}

detect_log_path() {
    if [[ -f /var/log/auth.log ]]; then
        printf '%s\n' /var/log/auth.log
    elif [[ -f /var/log/secure ]]; then
        printf '%s\n' /var/log/secure
    else
        printf '%s\n' /var/log/auth.log
    fi
}

load_existing_config() {
    [[ -n $SOURCE_ENV ]] || die "发现旧安装，但未找到 $ENV_FILE 或 $LEGACY_ENV_FILE；无法安全迁移 Telegram 配置"
    [[ -n $(read_env_value SSHGUARD_TELEGRAM_TOKEN "$SOURCE_ENV") ]] || die "$SOURCE_ENV 缺少 SSHGUARD_TELEGRAM_TOKEN"
    [[ -n $(read_env_value SSHGUARD_TELEGRAM_CHAT_ID "$SOURCE_ENV") ]] || die "$SOURCE_ENV 缺少 SSHGUARD_TELEGRAM_CHAT_ID"

    RUN_MODE=$(read_env_value SSHGUARD_MODE "$SOURCE_ENV")
    if [[ -z $RUN_MODE ]]; then
        # The legacy release had only log monitoring. Preserve that behavior.
        if [[ $SOURCE_ENV == "$LEGACY_ENV_FILE" || -n $(read_env_value SSHGUARD_LOG_PATH "$SOURCE_ENV") ]]; then
            RUN_MODE=log
        else
            RUN_MODE=socket
        fi
    fi
    [[ $RUN_MODE == log || $RUN_MODE == socket ]] || die "旧配置中的 SSHGUARD_MODE 无效: $RUN_MODE"

    LOG_PATH=$(read_env_value SSHGUARD_LOG_PATH "$SOURCE_ENV")
    if [[ $RUN_MODE == log ]]; then
        LOG_PATH=${LOG_PATH:-$(detect_log_path)}
        [[ -f $LOG_PATH ]] || die "SSH 日志文件不存在: $LOG_PATH"
    else
        SOCKET_PATH=$(read_env_value SSHGUARD_SOCKET_PATH "$SOURCE_ENV")
        SOCKET_PATH=${SOCKET_PATH:-/run/sshguard.sock}
        [[ $SOCKET_PATH != *"'"* && $SOCKET_PATH != *$'\n'* ]] || die "Socket 路径包含不支持的字符"
        [[ -f $PAM_FILE ]] || die "Socket 模式需要 $PAM_FILE"
    fi

    if [[ ! -e $SERVICE_FILE ]]; then
        INSTALL_SERVICE=false
    fi
    info "检测到已安装 SSHGuard；保留 $SOURCE_ENV 中的凭据，运行模式: $RUN_MODE"
    if [[ -x $INSTALL_DIR/sshguard ]]; then
        local old_version
        old_version=$("$INSTALL_DIR/sshguard" -version 2>/dev/null || true)
        [[ $old_version == "$RELEASE_VERSION" ]] && info "当前二进制为 $old_version，将重新部署配置" || info "当前二进制为旧版或无版本标识"
    fi
}

prompt() {
    local label="$1" default="$2" answer
    printf '%s [%s]: ' "$label" "$default" >&2
    IFS= read -r answer || die "读取输入失败"
    printf '%s\n' "${answer:-$default}"
}

configure_new() {
    [[ -r /dev/tty ]] || die "首次安装需要交互式终端"
    # curl | bash consumes stdin; read prompts from the user's terminal.
    exec </dev/tty
    printf 'Telegram Bot Token: ' >&2
    IFS= read -r TELEGRAM_TOKEN
    [[ -n $TELEGRAM_TOKEN && $TELEGRAM_TOKEN != *$'\n'* ]] || die "Telegram Bot Token 不能为空"
    printf 'Telegram Chat ID: ' >&2
    IFS= read -r TELEGRAM_CHAT_ID
    [[ -n $TELEGRAM_CHAT_ID && $TELEGRAM_CHAT_ID != *$'\n'* ]] || die "Telegram Chat ID 不能为空"
    RUN_MODE=$(prompt '运行模式 (socket/log)' socket)
    [[ $RUN_MODE == socket || $RUN_MODE == log ]] || die "无效的运行模式: $RUN_MODE"
    if [[ $RUN_MODE == log ]]; then
        LOG_PATH=$(prompt 'SSH 日志路径' "$(detect_log_path)")
        [[ -f $LOG_PATH ]] || die "SSH 日志文件不存在: $LOG_PATH"
    else
        [[ -f $PAM_FILE ]] || die "Socket 模式需要 $PAM_FILE"
    fi
    local service_answer
    service_answer=$(prompt '安装 systemd 服务？(Y/n)' Y)
    [[ $service_answer == y || $service_answer == Y ]] || INSTALL_SERVICE=false
}

stage_binary() {
    STAGING_DIR=$(mktemp -d)
    if [[ -n $LOCAL_BINARY ]]; then
        [[ -f $LOCAL_BINARY ]] || die "找不到本地二进制文件: $LOCAL_BINARY"
        cp "$LOCAL_BINARY" "$STAGING_DIR/sshguard"
    else
        info "下载 $RELEASE_VERSION 二进制文件与校验和"
        curl -fsSL --retry 3 -o "$STAGING_DIR/sshguard" "$RELEASE_BASE/sshguard" || die "二进制文件下载失败；请确认 v0.0.2 已发布"
        curl -fsSL --retry 3 -o "$STAGING_DIR/checksums.txt" "$RELEASE_BASE/checksums.txt" || die "校验和下载失败"
        awk '$2 == "sshguard" && length($1) == 64 { print }' "$STAGING_DIR/checksums.txt" > "$STAGING_DIR/sshguard.sha256"
        [[ $(wc -l < "$STAGING_DIR/sshguard.sha256") -eq 1 ]] || die "checksums.txt 缺少唯一的 sshguard 校验和"
        (cd "$STAGING_DIR" && sha256sum -c --status sshguard.sha256) || die "二进制文件 SHA-256 校验失败"
    fi
    chmod 755 "$STAGING_DIR/sshguard"
    [[ $("$STAGING_DIR/sshguard" -version 2>/dev/null) == "$RELEASE_VERSION" ]] || die "二进制文件版本不是 $RELEASE_VERSION，或无法在本机运行"
}

cleanup_stage() {
    if [[ -n $STAGING_DIR && -d $STAGING_DIR ]]; then
        rm -r -- "$STAGING_DIR"
    fi
}

backup_existing() {
    [[ $INSTALLED == true ]] || return 0
    BACKUP_DIR=$(mktemp -d /var/tmp/sshguard-backup.XXXXXX)
    chmod 700 "$BACKUP_DIR"
    local path
    for path in "$INSTALL_DIR/sshguard" "$ENV_FILE" "$SERVICE_FILE" "$PAM_FILE"; do
        if [[ -f $path ]]; then
            mkdir -p "$BACKUP_DIR$(dirname "$path")"
            cp -p "$path" "$BACKUP_DIR$path"
        fi
    done
    info "旧安装备份到 $BACKUP_DIR"
}

install_binary() {
    mkdir -p "$INSTALL_DIR"
    local staged
    staged=$(mktemp "$INSTALL_DIR/.sshguard.XXXXXX")
    install -m 755 "$STAGING_DIR/sshguard" "$staged"
    mv -f "$staged" "$INSTALL_DIR/sshguard"
}

set_env_value() {
    local key="$1" value="$2" staged
    staged=$(mktemp "$ENV_FILE.XXXXXX")
    awk -v key="$key" 'index($0, key "=") != 1 { print }' "$ENV_FILE" > "$staged"
    printf '%s=%s\n' "$key" "$value" >> "$staged"
    chmod 600 "$staged"
    mv -f "$staged" "$ENV_FILE"
}

write_env() {
    mkdir -p "$(dirname "$ENV_FILE")"
    if [[ $INSTALLED == true ]]; then
        if [[ $SOURCE_ENV != "$ENV_FILE" ]]; then
            install -m 600 "$SOURCE_ENV" "$ENV_FILE"
        fi
        set_env_value SSHGUARD_MODE "$RUN_MODE"
        if [[ $RUN_MODE == log ]]; then
            set_env_value SSHGUARD_LOG_PATH "$LOG_PATH"
        else
            set_env_value SSHGUARD_SOCKET_PATH "$SOCKET_PATH"
        fi
    else
        umask 077
        {
            printf 'SSHGUARD_TELEGRAM_TOKEN=%s\n' "$TELEGRAM_TOKEN"
            printf 'SSHGUARD_TELEGRAM_CHAT_ID=%s\n' "$TELEGRAM_CHAT_ID"
            printf 'SSHGUARD_MODE=%s\n' "$RUN_MODE"
            if [[ $RUN_MODE == log ]]; then
                printf 'SSHGUARD_LOG_PATH=%s\n' "$LOG_PATH"
            else
                printf 'SSHGUARD_SOCKET_PATH=%s\n' "$SOCKET_PATH"
            fi
        } > "$ENV_FILE"
    fi
    chmod 600 "$ENV_FILE"
}

write_service() {
    [[ $INSTALL_SERVICE == true ]] || return 0
    cat > "$SERVICE_FILE" <<EOF
[Unit]
Description=SSHGuard - SSH 登录监控与 Telegram 通知
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
EnvironmentFile=$ENV_FILE
ExecStart=$INSTALL_DIR/sshguard
Restart=always
RestartSec=30
StandardOutput=journal
StandardError=journal
NoNewPrivileges=yes
PrivateTmp=yes
ProtectSystem=strict
ProtectHome=yes
ReadWritePaths=/run

[Install]
WantedBy=multi-user.target
EOF
}

start_service() {
    [[ $INSTALL_SERVICE == true ]] || return 0
    systemctl daemon-reload
    if [[ $INSTALLED == false ]]; then
        systemctl enable sshguard >/dev/null
    fi
    systemctl restart sshguard || return 1
    sleep 1
    systemctl is-active --quiet sshguard || return 1
    if [[ $RUN_MODE == socket ]]; then
        [[ -S $SOCKET_PATH ]] || return 1
    fi
}

restore_on_failure() {
    warn "新服务未能启动，正在恢复旧安装"
    if [[ -n $BACKUP_DIR ]]; then
        local path
        for path in "$INSTALL_DIR/sshguard" "$ENV_FILE" "$SERVICE_FILE" "$PAM_FILE"; do
            if [[ -f $BACKUP_DIR$path ]]; then
                if [[ $path == "$INSTALL_DIR/sshguard" ]]; then
                    local restored
                    restored=$(mktemp "$INSTALL_DIR/.sshguard.restore.XXXXXX")
                    cp -p "$BACKUP_DIR$path" "$restored"
                    mv -f "$restored" "$path"
                else
                    cp -p "$BACKUP_DIR$path" "$path"
                fi
            elif [[ $path == "$ENV_FILE" ]]; then
                rm -f -- "$path"
            fi
        done
        systemctl daemon-reload || true
        systemctl restart sshguard || warn "旧服务也未能重启，请检查 journalctl -u sshguard"
    fi
    die "安装失败；备份位于 $BACKUP_DIR"
}

configure_pam() {
    [[ $RUN_MODE == socket ]] || return 0
    cat > "$PAM_HELPER" <<EOF || return 1
#!/bin/sh
exec '$INSTALL_DIR/sshguard' -pam -socket '$SOCKET_PATH'
EOF
    chmod 755 "$PAM_HELPER" || return 1
    if ! awk -v helper="$PAM_HELPER" '$0 !~ /^[[:space:]]*#/ && index($0, helper) { found=1 } END { exit !found }' "$PAM_FILE"; then
        printf 'session optional pam_exec.so type=open_session %s\n' "$PAM_HELPER" >> "$PAM_FILE" || return 1
    fi
}

main() {
    parse_args "$@"
    trap cleanup_stage EXIT
    check_platform
    detect_installation
    if [[ $UPDATE_ONLY == true && $INSTALLED == false ]]; then
        die "本机未安装 SSHGuard，无法执行 --update"
    fi
    if [[ $INSTALLED == true ]]; then
        load_existing_config
    else
        configure_new
    fi
    stage_binary
    backup_existing
    install_binary
    write_env
    write_service
    if ! start_service; then
        restore_on_failure
    fi
    if ! configure_pam; then
        restore_on_failure
    fi
    info "SSHGuard $RELEASE_VERSION 安装完成，运行模式: $RUN_MODE"
    if [[ $INSTALL_SERVICE == true ]]; then
        info "服务状态: $(systemctl is-active sshguard)"
    else
        info "未安装 systemd 服务；请手动启动 $INSTALL_DIR/sshguard"
    fi
}

# Sourcing the script is useful for installer migration tests. Piped bash has
# an empty BASH_SOURCE[0], so it still runs the installer.
if [[ -z ${BASH_SOURCE[0]-} || ${BASH_SOURCE[0]-} == "$0" ]]; then
    main "$@"
fi
