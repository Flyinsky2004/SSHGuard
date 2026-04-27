package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	cfg := parseFlags()

	// 上线通知
	if err := notifyStatus(cfg.Token, cfg.ChatID, "在线"); err != nil {
		log.Printf("上线通知发送失败: %v", err)
	}

	events := make(chan *SSHEvent, 64)

	if err := monitorLog(cfg.LogPath, events); err != nil {
		log.Fatalf("启动日志监控失败: %v", err)
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	for {
		select {
		case ev, ok := <-events:
			if !ok {
				log.Println("日志监控已停止，程序退出")
				notifyStatus(cfg.Token, cfg.ChatID, "离线")
				return
			}
			if err := notifyTelegram(cfg.Token, cfg.ChatID, ev); err != nil {
				log.Printf("通知发送失败: %v", err)
			}
		case sig := <-sigCh:
			log.Printf("收到信号 %v，正在关闭", sig)
			notifyStatus(cfg.Token, cfg.ChatID, "离线")
			return
		}
	}
}
