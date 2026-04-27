package main

import (
	"flag"
	"fmt"
	"os"
)

// Config holds all configuration for SSHGuard.
type Config struct {
	LogPath string
	Token   string
	ChatID  string
}

func parseFlags() *Config {
	cfg := &Config{}

	flag.StringVar(&cfg.Token, "token", os.Getenv("SSHGUARD_TELEGRAM_TOKEN"), "Telegram Bot Token (环境变量: SSHGUARD_TELEGRAM_TOKEN)")
	flag.StringVar(&cfg.ChatID, "chat-id", os.Getenv("SSHGUARD_TELEGRAM_CHAT_ID"), "Telegram Chat ID (环境变量: SSHGUARD_TELEGRAM_CHAT_ID)")
	flag.StringVar(&cfg.LogPath, "log", "", "SSH 日志路径 (留空则自动检测; 环境变量: SSHGUARD_LOG_PATH)")
	flag.Parse()

	if cfg.LogPath == "" {
		if v := os.Getenv("SSHGUARD_LOG_PATH"); v != "" {
			cfg.LogPath = v
		} else {
			cfg.LogPath = detectLogPath()
		}
	}

	if cfg.Token == "" {
		exitErr("缺少 Telegram Bot Token (-token 或环境变量 SSHGUARD_TELEGRAM_TOKEN)")
	}
	if cfg.ChatID == "" {
		exitErr("缺少 Telegram Chat ID (-chat-id 或环境变量 SSHGUARD_TELEGRAM_CHAT_ID)")
	}

	return cfg
}

func exitErr(msg string) {
	fmt.Fprintln(os.Stderr, "sshguard:", msg)
	fmt.Fprintln(os.Stderr, "用法: sshguard -token <bot_token> -chat-id <chat_id> [-log <路径>]")
	os.Exit(1)
}

func detectLogPath() string {
	paths := []string{"/var/log/auth.log", "/var/log/secure"}
	for _, p := range paths {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return "/var/log/auth.log"
}
