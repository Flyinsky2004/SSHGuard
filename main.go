package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	cfg := parseFlags()

	events := make(chan *SSHEvent, 64)

	if err := monitorLog(cfg.LogPath, events); err != nil {
		log.Fatalf("monitor: %v", err)
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	for {
		select {
		case ev, ok := <-events:
			if !ok {
				log.Println("monitor stopped, exiting")
				return
			}
			if err := notifyTelegram(cfg.Token, cfg.ChatID, ev); err != nil {
				log.Printf("notify error: %v", err)
			}
		case sig := <-sigCh:
			log.Printf("received signal %v, shutting down", sig)
			return
		}
	}
}
