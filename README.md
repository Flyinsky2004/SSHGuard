# SSHGuard

SSH 登录成功后发送 Telegram 通知。支持 PAM Socket 模式，以及从 `/var/log/auth.log` 或 `/var/log/secure` 读取登录事件的日志模式。v0.0.2 可识别 Debian 13 的 `sshd-session` 日志；同一条成功登录日志重复写入时只通知一次。

## 安装与更新

预编译文件支持 Linux amd64。需要 root 权限、systemd、Telegram Bot Token 和 Chat ID。

```bash
curl -fsSL https://raw.githubusercontent.com/Flyinsky2004/SSHGuard/main/install.sh | sudo bash
```

安装脚本会检测 `/opt/SSHGuard/sshguard`、systemd 服务以及新旧环境文件。已有安装会自动更新到 v0.0.2，并保留 Telegram 凭据及运行模式；首次安装进入交互式配置。旧版 `/etc/sshguard.env` 会迁移到 `/etc/sshguard/env`。旧版仅支持日志模式，因此迁移时继续使用日志模式，不需要改动 PAM 配置。

只允许更新已有安装时使用：

```bash
curl -fsSL https://raw.githubusercontent.com/Flyinsky2004/SSHGuard/main/install.sh | sudo bash -s -- --update
```

发布前或离线安装可使用本地二进制文件：

```bash
sudo bash install.sh --update --binary /path/to/sshguard
```

安装脚本从 `v0.0.2` release 下载 `sshguard` 和 `checksums.txt`，校验 SHA-256 和二进制版本之后才替换程序。更新时会备份已有二进制、配置和服务单元；服务启动失败时尝试恢复。**只有发布实际标签 `v0.0.2` 并上传这两个资产后，远程下载才可用。**将 release 标题改为 v0.0.2 而继续使用旧标签 `main` 不会生成脚本所用的下载地址。

## 运行模式

| 模式 | 用途 | 安装要求 |
| --- | --- | --- |
| `socket` | 新安装默认模式，PAM 在 SSH 会话开启时发送事件 | `/etc/pam.d/sshd` 中的 `pam_exec.so` helper；脚本自动配置 |
| `log` | 兼容旧安装，保留认证方式、来源 IP 和端口 | 服务器持续写入 `/var/log/auth.log` 或 `/var/log/secure` |

Socket 模式通知中的认证方式为 `pam`，不包含 SSH 来源端口。日志模式支持旧版 `sshd[PID]` 和 Debian 13 的 `sshd-session[PID]` 成功登录行。日志监控只处理程序启动后新写入的内容。

配置文件为 `/etc/sshguard/env`；升级后的服务单元会使用此路径。可通过环境变量或命令行参数设置 `SSHGUARD_MODE`、`SSHGUARD_LOG_PATH`、`SSHGUARD_SOCKET_PATH` 和 `SSHGUARD_ALIAS`。凭据变量为 `SSHGUARD_TELEGRAM_TOKEN` 和 `SSHGUARD_TELEGRAM_CHAT_ID`。

```bash
/opt/SSHGuard/sshguard -version
systemctl status sshguard
journalctl -u sshguard -f
```

## 构建和发布 v0.0.2

使用 Go 1.25.5 构建 Linux amd64 静态文件：

```bash
mkdir -p dist/v0.0.2
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -buildvcs=false \
  -ldflags='-s -w -X main.version=v0.0.2' -o dist/v0.0.2/sshguard .
(cd dist/v0.0.2 && sha256sum sshguard > checksums.txt)
```

将 `dist/v0.0.2/sshguard` 和 `dist/v0.0.2/checksums.txt` 作为 **`v0.0.2` 标签**的 release 资产发布。仓库现有 GitHub Actions 在推送 `v*` 标签时也会构建并附加同名资产；手动上传时请让校验和与所上传二进制来自同一次构建。

## 开源协议

MIT
