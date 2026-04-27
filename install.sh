#!/usr/bin/env bash
set -euo pipefail

# 管道执行时，stdin 是脚本内容而不是终端，会导致 read 读到 EOF 立即退出。
# 显式将 stdin 重定向到 /dev/tty，确保 curl | bash 场景下交互输入正常。
exec </dev/tty

# ============================================================
# SSHHGuard — 交互式安装脚本
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

# 下载地址 — GitHub Releases 中的预编译二进制
DOWNLOAD_URL="https://github.com/Flyinsky2004/SSHGuard/releases/download/main/sshguard"

# -----------------------------------------------------------
# 工具函数
# -----------------------------------------------------------
banner() {
    echo -e "${CYAN}${BOLD}"
    echo "  ╔══════════════════════════════════╗"
    echo "  ║        SSHHGuard 安装程序        ║"
    echo "  ╚══════════════════════════════════╝"
    echo -e "${NC}"
}

info()    { echo -e "${GREEN}[+]${NC} $*"; }
warn()    { echo -e "${YELLOW}[!]${NC} $*"; }
error()   { echo -e "${RED}[✗]${NC} $*"; }
prompt()  { echo -ne "${BOLD}[?]${NC} $* "; }

die() {
    error "$*"
    exit 1
}

require_root() {
    if [[ $EUID -ne 0 ]]; then
        die "请使用 root 权限运行此脚本：sudo bash install.sh"
    fi
}

# -----------------------------------------------------------
# 检查依赖
# -----------------------------------------------------------
check_deps() {
    info "检查依赖..."

    local missing=()

    command -v curl >/dev/null 2>&1 || missing+=("curl")

    if [[ ${#missing[@]} -gt 0 ]]; then
        warn "缺少依赖：${missing[*]}"
        prompt "是否现在安装？（apt-get install ${missing[*]}）[Y/n]"
        read -r ans
        if [[ "${ans:-y}" =~ ^[Yy]$ ]]; then
            apt-get update -qq && apt-get install -y "${missing[@]}"
        else
            die "缺少依赖，无法继续：${missing[*]}"
        fi
    fi

    info "依赖检查通过"
}

# -----------------------------------------------------------
# 下载二进制
# -----------------------------------------------------------
download_binary() {
    info "正在下载 SSHHGuard 二进制文件..."
    info "下载地址：$DOWNLOAD_URL"

    mkdir -p "$INSTALL_DIR"

    curl -fsSL --progress-bar -o "$INSTALL_DIR/$BINARY_NAME" "$DOWNLOAD_URL" \
        || die "下载失败，请检查网络连接和下载地址。"

    # 校验有效性
    if [[ ! -s "$INSTALL_DIR/$BINARY_NAME" ]]; then
        die "下载的文件为空，请稍后重试。"
    fi

    chmod +x "$INSTALL_DIR/$BINARY_NAME"
    info "二进制文件已安装至 $INSTALL_DIR/$BINARY_NAME"
}

# -----------------------------------------------------------
# 交互式配置
# -----------------------------------------------------------
configure() {
    echo ""
    echo -e "${CYAN}${BOLD}  ─── 配置参数 ───${NC}"
    echo -e "  按 Enter 使用方括号中的默认值。"
    echo ""

    # --- Telegram Bot Token ---
    while true; do
        prompt "Telegram Bot Token:"
        read -r TELEGRAM_TOKEN

        if [[ -z "$TELEGRAM_TOKEN" ]]; then
            warn "Telegram Bot Token 为必填项。"
            echo "  请在 Telegram 上向 @BotFather 获取：https://t.me/BotFather"
            continue
        fi
        break
    done

    # --- Telegram Chat ID ---
    while true; do
        prompt "Telegram Chat ID:"
        read -r TELEGRAM_CHAT_ID

        if [[ -z "$TELEGRAM_CHAT_ID" ]]; then
            warn "Telegram Chat ID 为必填项。"
            echo "  向你的 Bot 发送 /start，然后访问以下地址查看 chat id："
            echo "  https://api.telegram.org/bot<TOKEN>/getUpdates"
            echo "  在返回的 JSON 中找到 'chat':{'id': ...}"
            continue
        fi
        break
    done

    # --- 日志路径 ---
    echo ""
    echo "  SSH 认证日志路径（留空则自动检测）："
    if [[ -f /var/log/auth.log ]]; then
        echo -e "    ${GREEN}检测到：${NC} /var/log/auth.log（Debian/Ubuntu）"
        DETECTED_LOG="/var/log/auth.log"
    elif [[ -f /var/log/secure ]]; then
        echo -e "    ${GREEN}检测到：${NC} /var/log/secure（RHEL/CentOS）"
        DETECTED_LOG="/var/log/secure"
    else
        DETECTED_LOG="/var/log/auth.log"
    fi

    prompt "日志路径 [$DETECTED_LOG]:"
    read -r LOG_PATH
    LOG_PATH="${LOG_PATH:-$DETECTED_LOG}"

    # --- systemd 服务 ---
    echo ""
    prompt "是否安装 systemd 服务（开机自启）？[Y/n]"
    read -r INSTALL_SERVICE
    INSTALL_SERVICE="${INSTALL_SERVICE:-y}"

    # --- 确认摘要 ---
    echo ""
    echo -e "${CYAN}${BOLD}  ─── 安装确认 ───${NC}"
    echo ""
    echo -e "  ${BOLD}安装目录：${NC}      $INSTALL_DIR"
    echo -e "  ${BOLD}二进制文件：${NC}    $INSTALL_DIR/$BINARY_NAME"
    echo -e "  ${BOLD}日志文件：${NC}      $LOG_PATH"
    echo -e "  ${BOLD}Telegram Token：${NC} ${TELEGRAM_TOKEN:0:8}..."
    echo -e "  ${BOLD}Telegram Chat：${NC}  $TELEGRAM_CHAT_ID"
    echo -e "  ${BOLD}systemd 服务：${NC}   $([[ "$INSTALL_SERVICE" =~ ^[Yy]$ ]] && echo '是' || echo '否')"
    echo ""

    prompt "确认开始安装？[Y/n]"
    read -r CONFIRM
    if [[ ! "${CONFIRM:-y}" =~ ^[Yy]$ ]]; then
        die "安装已取消。"
    fi
}

# -----------------------------------------------------------
# 写入环境变量文件
# -----------------------------------------------------------
write_env() {
    mkdir -p "$(dirname "$ENV_FILE")"
    cat > "$ENV_FILE" <<EOF
# SSHHGuard 环境变量 — 由 install.sh 管理
SSHGUARD_TELEGRAM_TOKEN=$TELEGRAM_TOKEN
SSHGUARD_TELEGRAM_CHAT_ID=$TELEGRAM_CHAT_ID
SSHGUARD_LOG_PATH=$LOG_PATH
EOF
    chmod 600 "$ENV_FILE"
    info "环境变量文件已写入 $ENV_FILE"
}

# -----------------------------------------------------------
# 安装 systemd 服务
# -----------------------------------------------------------
install_service() {
    if [[ ! "$INSTALL_SERVICE" =~ ^[Yy]$ ]]; then
        info "已跳过 systemd 服务安装。手动运行方式："
        echo ""
        echo "  $INSTALL_DIR/$BINARY_NAME -token <token> -chat-id <id>"
        return
    fi

    cat > "$SERVICE_FILE" <<EOF
[Unit]
Description=SSHHGuard - SSH 登录监控与 Telegram 通知
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

# 安全加固
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
    info "systemd 服务已安装至 $SERVICE_FILE"

    prompt "是否现在启动 SSHHGuard？[Y/n]"
    read -r START_NOW
    if [[ "${START_NOW:-y}" =~ ^[Yy]$ ]]; then
        systemctl enable --now sshguard
        info "服务已启动并设为开机自启。"
        echo ""
        echo "  管理命令："
        echo "    systemctl status sshguard    # 查看状态"
        echo "    journalctl -u sshguard -f    # 查看日志"
    else
        systemctl enable sshguard
        info "服务已设为开机自启（将在下次重启后启动）。"
        echo ""
        echo "  手动启动：systemctl start sshguard"
    fi
}

# -----------------------------------------------------------
# 主流程
# -----------------------------------------------------------
main() {
    banner
    require_root
    check_deps

    echo ""
    download_binary
    configure
    write_env
    install_service

    echo ""
    echo -e "${GREEN}${BOLD}  ✓ 安装完成！${NC}"
    echo ""
}

main "$@"
