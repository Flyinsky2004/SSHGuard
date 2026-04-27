package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"time"
)

// SSHEvent represents a parsed SSH login event.
type SSHEvent struct {
	Timestamp time.Time
	Hostname  string
	User      string
	SourceIP  string
	SourcePort string
	AuthMethod string
}

type telegramMessage struct {
	ChatID    string `json:"chat_id"`
	Text      string `json:"text"`
	ParseMode string `json:"parse_mode"`
}

type telegramResponse struct {
	OK          bool   `json:"ok"`
	ErrorCode   int    `json:"error_code"`
	Description string `json:"description"`
}

func notifyTelegram(token, chatID string, ev *SSHEvent) error {
	text := fmt.Sprintf(
		"<b>SSH Login</b> on <code>%s</code>\n\n"+
			"User: <b>%s</b>\n"+
			"From: <code>%s</code>\n"+
			"Method: %s\n"+
			"Time: %s",
		escapeHTML(serverIdent),
		escapeHTML(ev.User),
		fmt.Sprintf("%s:%s", ev.SourceIP, ev.SourcePort),
		ev.AuthMethod,
		ev.Timestamp.Format("2006-01-02 15:04:05"),
	)

	body, err := json.Marshal(telegramMessage{
		ChatID:    chatID,
		Text:      text,
		ParseMode: "HTML",
	})
	if err != nil {
		return fmt.Errorf("marshal message: %w", err)
	}

	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", token)
	resp, err := http.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	var result telegramResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}
	if !result.OK {
		return fmt.Errorf("telegram API error %d: %s", result.ErrorCode, result.Description)
	}

	log.Printf("notification sent: user=%s ip=%s", ev.User, ev.SourceIP)
	return nil
}

var serverIdent string

func init() {
	hn, _ := os.Hostname()

	ip := primaryIPv4()
	if ip != "" {
		if hn != "" && hn != "localhost" && hn != "localhost.localdomain" {
			serverIdent = fmt.Sprintf("%s (%s)", hn, ip)
		} else {
			serverIdent = ip
		}
	} else {
		if hn != "" {
			serverIdent = hn
		} else {
			serverIdent = "unknown"
		}
	}
}

// primaryIPv4 returns the first non-loopback IPv4 address.
func primaryIPv4() string {
	// Prefer the outbound IP used to reach the internet.
	conn, err := net.Dial("udp", "8.8.8.8:53")
	if err == nil {
		addr := conn.LocalAddr().(*net.UDPAddr)
		conn.Close()
		if ip := addr.IP.To4(); ip != nil && !ip.IsLoopback() {
			return ip.String()
		}
	}

	// Fallback: scan network interfaces.
	ifaces, err := net.Interfaces()
	if err != nil {
		return ""
	}
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			if ipnet, ok := addr.(*net.IPNet); ok {
				if ip := ipnet.IP.To4(); ip != nil && !ip.IsLoopback() {
					return ip.String()
				}
			}
		}
	}
	return ""
}

func notifyStatus(token, chatID, status string) error {
	emoji := "🟢"
	if status == "offline" {
		emoji = "🔴"
	}

	text := fmt.Sprintf(
		"%s SSHHGuard <b>%s</b> on <code>%s</code>\nTime: %s",
		emoji,
		status,
		escapeHTML(serverIdent),
		time.Now().Format("2006-01-02 15:04:05"),
	)

	body, err := json.Marshal(telegramMessage{
		ChatID:    chatID,
		Text:      text,
		ParseMode: "HTML",
	})
	if err != nil {
		return fmt.Errorf("marshal status message: %w", err)
	}

	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", token)
	resp, err := http.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("send status request: %w", err)
	}
	defer resp.Body.Close()

	var result telegramResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("decode status response: %w", err)
	}
	if !result.OK {
		return fmt.Errorf("telegram API error %d: %s", result.ErrorCode, result.Description)
	}

	return nil
}

func escapeHTML(s string) string {
	repl := map[byte]string{
		'&':  "&amp;",
		'<':  "&lt;",
		'>':  "&gt;",
		'"':  "&quot;",
		'\'': "&#39;",
	}
	var buf []byte
	for i := 0; i < len(s); i++ {
		if r, ok := repl[s[i]]; ok {
			buf = append(buf, []byte(r)...)
		} else {
			buf = append(buf, s[i])
		}
	}
	return string(buf)
}
