# SSHHGuard

实时监控 SSH 认证日志，在每次登录成功时发送 Telegram 通知。

## 一键安装

```bash
curl -fsSL https://raw.githubusercontent.com/Flyinsky2004/SSHGuard/main/install.sh | sudo bash
```

或者克隆仓库后本地运行：

```bash
git clone https://github.com/Flyinsky2004/SSHGuard.git
cd SSHHGuard
sudo bash install.sh
```

## 安装脚本做了什么

交互式 `install.sh` 会引导你完成以下步骤：

1. **下载** — 从 GitHub Releases 拉取最新的预编译 Linux amd64 二进制文件到 `/opt/SSHGuard/sshguard`
2. **chmod +x** — 赋予二进制可执行权限
3. **配置** — 交互式询问：
   - Telegram Bot Token
   - Telegram Chat ID
   - SSH 日志路径（留空则自动检测）
4. **systemd 服务**（可选）— 创建并启用 `sshguard.service`，实现开机自启和异常自动重启

### 安装过程演示

```
$ sudo bash install.sh

  ╔══════════════════════════════════╗
  ║        SSHHGuard 安装程序        ║
  ╚══════════════════════════════════╝

[+] 检查依赖...
[+] 依赖检查通过

[+] 正在下载 SSHHGuard 二进制文件...
[+] 下载地址：https://github.com/Flyinsky2004/SSHGuard/releases/download/main/sshguard
[+] 二进制文件已安装至 /opt/SSHGuard/sshguard

  ─── 配置参数 ───
  按 Enter 使用方括号中的默认值。

[?] Telegram Bot Token:
> 123456:ABC-DEF1234ghijk

[?] Telegram Chat ID:
> 987654321

  SSH 认证日志路径（留空则自动检测）：
    检测到：/var/log/auth.log（Debian/Ubuntu）

[?] 日志路径 [/var/log/auth.log]:
> （直接按 Enter）

[?] 是否安装 systemd 服务（开机自启）？[Y/n]:
> y

  ─── 安装确认 ───

  安装目录：      /opt/SSHGuard
  二进制文件：    /opt/SSHGuard/sshguard
  日志文件：      /var/log/auth.log
  Telegram Token：123456:...
  Telegram Chat： 987654321
  systemd 服务：  是

[?] 确认开始安装？[Y/n]:
> y

[+] 环境变量文件已写入 /etc/sshguard/env
[+] systemd 服务已安装至 /etc/systemd/system/sshguard.service
[?] 是否现在启动 SSHHGuard？[Y/n]:
> y
[+] 服务已启动并设为开机自启。

  ✓ 安装完成！
```

## 前置条件

- **Linux amd64**（预编译二进制仅支持该平台）
- **root / sudo** 权限
- `curl`（安装脚本会自动检测并提供安装）
- **Telegram Bot Token** — 通过 [@BotFather](https://t.me/BotFather) 创建 Bot 获取
- **Telegram Chat ID** — 向你的 Bot 发送 `/start`，然后访问 `https://api.telegram.org/bot<TOKEN>/getUpdates`，在返回的 JSON 中找到 `"chat":{"id": ...}`

## 手动运行

直接运行二进制文件：

```bash
/opt/SSHGuard/sshguard -token <bot_token> -chat-id <chat_id> [-log <日志路径>]
```

或使用环境变量：

```bash
export SSHGUARD_TELEGRAM_TOKEN=你的_token
export SSHGUARD_TELEGRAM_CHAT_ID=你的_chat_id
export SSHGUARD_LOG_PATH=/var/log/auth.log   # 可选

/opt/SSHGuard/sshguard
```

## 配置参考

| 参数 | 环境变量 | 必填 | 默认值 |
|------|---------|------|--------|
| `-token` | `SSHGUARD_TELEGRAM_TOKEN` | 是 | — |
| `-chat-id` | `SSHGUARD_TELEGRAM_CHAT_ID` | 是 | — |
| `-log` | `SSHGUARD_LOG_PATH` | 否 | 自动检测 |

日志路径会自动在 `/var/log/auth.log`（Debian/Ubuntu）和 `/var/log/secure`（RHEL/CentOS）之间检测。

## 服务管理

```bash
systemctl status sshguard     # 查看运行状态
systemctl stop sshguard       # 停止服务
systemctl start sshguard      # 启动服务
systemctl restart sshguard    # 重启服务

journalctl -u sshguard -f     # 实时查看日志
```

## 从源码构建

```bash
git clone https://github.com/Flyinsky2004/SSHGuard.git
cd SSHHGuard
go build -ldflags="-s -w" -o sshguard .
```

## 开源协议

MIT
