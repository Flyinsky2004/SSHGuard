package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
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
		escapeHTML(ev.Hostname),
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
