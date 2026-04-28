package main

import (
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	cfg := parseFlags()

	// PAM helper mode: send event and exit immediately.
	if cfg.PAMMode {
		runPAMHelper(cfg.SocketPath)
		return
	}

	// Set server alias.
	if cfg.Alias != "" {
		applyServerName(cfg.Alias)
	}

	// Online notification.
	if err := notifyStatus(cfg.Token, cfg.ChatID, "在线"); err != nil {
		log.Printf("上线通知发送失败: %v", err)
	}

	events := make(chan *SSHEvent, 64)
	var listener net.Listener

	if cfg.Mode == "log" {
		if err := monitorLog(cfg.LogPath, events); err != nil {
			log.Fatalf("启动日志监控失败: %v", err)
		}
	} else {
		ln, err := listenSocket(cfg.SocketPath, events)
		if err != nil {
			log.Fatalf("启动 Socket 监听失败: %v", err)
		}
		listener = ln
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	for {
		select {
		case ev, ok := <-events:
			if !ok {
				log.Println("事件源已停止，程序退出")
				notifyStatus(cfg.Token, cfg.ChatID, "离线")
				return
			}
			if err := notifyTelegram(cfg.Token, cfg.ChatID, ev); err != nil {
				log.Printf("通知发送失败: %v", err)
			}
		case sig := <-sigCh:
			log.Printf("收到信号 %v，正在关闭", sig)
			if listener != nil {
				listener.Close()
			}
			notifyStatus(cfg.Token, cfg.ChatID, "离线")
			return
		}
	}
}
