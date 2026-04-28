package main

import (
	"encoding/json"
	"log"
	"net"
	"os"
	"time"
)

// pamEvent is the JSON wire format sent over the Unix socket.
type pamEvent struct {
	Type       string `json:"type"`
	User       string `json:"user"`
	SourceIP   string `json:"source_ip"`
	SourcePort string `json:"source_port"`
	Timestamp  string `json:"timestamp"`
	Hostname   string `json:"hostname"`
	AuthMethod string `json:"auth_method"`
}

func runPAMHelper(socketPath string) {
	pamType := os.Getenv("PAM_TYPE")
	if pamType != "open_session" {
		return
	}

	user := os.Getenv("PAM_USER")
	if user == "" {
		return
	}

	sourceIP := os.Getenv("PAM_RHOST")
	hostname, _ := os.Hostname()

	ev := pamEvent{
		Type:       "ssh_login",
		User:       user,
		SourceIP:   sourceIP,
		SourcePort: "",
		Timestamp:  time.Now().UTC().Format(time.RFC3339),
		Hostname:   hostname,
		AuthMethod: "pam",
	}

	data, err := json.Marshal(ev)
	if err != nil {
		log.Printf("PAM helper: 序列化事件失败: %v", err)
		return
	}
	data = append(data, '\n')

	conn, err := net.DialUnix("unix", nil, &net.UnixAddr{Name: socketPath, Net: "unix"})
	if err != nil {
		log.Printf("PAM helper: 连接 socket 失败 (%s): %v", socketPath, err)
		return
	}
	defer conn.Close()

	if err := conn.SetWriteDeadline(time.Now().Add(3 * time.Second)); err != nil {
		return
	}

	if _, err := conn.Write(data); err != nil {
		log.Printf("PAM helper: 写入 socket 失败: %v", err)
		return
	}
}
