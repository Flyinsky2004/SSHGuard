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

	flag.StringVar(&cfg.Token, "token", os.Getenv("SSHGUARD_TELEGRAM_TOKEN"), "Telegram bot token (env: SSHGUARD_TELEGRAM_TOKEN)")
	flag.StringVar(&cfg.ChatID, "chat-id", os.Getenv("SSHGUARD_TELEGRAM_CHAT_ID"), "Telegram chat ID (env: SSHGUARD_TELEGRAM_CHAT_ID)")
	flag.StringVar(&cfg.LogPath, "log", "", "SSH auth log path (auto-detect if empty; env: SSHGUARD_LOG_PATH)")
	flag.Parse()

	if cfg.LogPath == "" {
		if v := os.Getenv("SSHGUARD_LOG_PATH"); v != "" {
			cfg.LogPath = v
		} else {
			cfg.LogPath = detectLogPath()
		}
	}

	if cfg.Token == "" {
		exitErr("telegram bot token is required (-token or SSHGUARD_TELEGRAM_TOKEN env)")
	}
	if cfg.ChatID == "" {
		exitErr("telegram chat ID is required (-chat-id or SSHGUARD_TELEGRAM_CHAT_ID env)")
	}

	return cfg
}

func exitErr(msg string) {
	fmt.Fprintln(os.Stderr, "sshguard:", msg)
	fmt.Fprintln(os.Stderr, "usage: sshguard -token <bot_token> -chat-id <chat_id> [-log <path>]")
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
