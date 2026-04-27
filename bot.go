package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"
)

// Telegram Bot API structures for getUpdates.
type tgUpdate struct {
	UpdateID int        `json:"update_id"`
	Message  *tgMessage `json:"message"`
}

type tgMessage struct {
	MessageID int    `json:"message_id"`
	Chat      tgChat `json:"chat"`
	Text      string `json:"text"`
}

type tgChat struct {
	ID int64 `json:"id"`
}

type tgUpdatesResponse struct {
	OK     bool       `json:"ok"`
	Result []tgUpdate `json:"result"`
}

// setMyCommands registers bot commands visible in the Telegram chat UI.
type tgBotCommand struct {
	Command     string `json:"command"`
	Description string `json:"description"`
}

func setMyCommands(token string) error {
	body, _ := json.Marshal(map[string][]tgBotCommand{
		"commands": {
			{Command: "name", Description: "设置 IP 别名: /name <IP> <别名>"},
			{Command: "delname", Description: "删除 IP 别名: /delname <IP>"},
			{Command: "listnames", Description: "列出所有 IP 别名"},
		},
	})

	url := fmt.Sprintf("https://api.telegram.org/bot%s/setMyCommands", token)
	resp, err := http.Post(url, "application/json", strings.NewReader(stringify(body)))
	if err != nil {
		return fmt.Errorf("setMyCommands 请求失败: %w", err)
	}
	defer resp.Body.Close()

	var result telegramResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("setMyCommands 响应解析失败: %w", err)
	}
	if !result.OK {
		return fmt.Errorf("setMyCommands API 错误 %d: %s", result.ErrorCode, result.Description)
	}
	return nil
}

func startBot(token, chatID string) {
	go func() {
		if err := setMyCommands(token); err != nil {
			log.Printf("注册机器人命令失败: %v", err)
		}

		offset := 0
		for {
			updates, err := getUpdates(token, offset)
			if err != nil {
				log.Printf("获取 Telegram 更新失败: %v", err)
				time.Sleep(5 * time.Second)
				continue
			}
			for _, upd := range updates {
				offset = upd.UpdateID + 1
				if upd.Message == nil || upd.Message.Text == "" {
					continue
				}
				handleCommand(token, chatID, upd.Message)
			}
		}
	}()
}

func getUpdates(token string, offset int) ([]tgUpdate, error) {
	url := fmt.Sprintf("https://api.telegram.org/bot%s/getUpdates?offset=%d&timeout=30", token, offset)
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("getUpdates 请求失败: %w", err)
	}
	defer resp.Body.Close()

	var result tgUpdatesResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("getUpdates 响应解析失败: %w", err)
	}
	if !result.OK {
		return nil, fmt.Errorf("getUpdates API 失败")
	}
	return result.Result, nil
}

func handleCommand(token, chatID string, msg *tgMessage) {
	// only respond to the configured chat
	if fmt.Sprintf("%d", msg.Chat.ID) != chatID {
		return
	}

	text := strings.TrimSpace(msg.Text)
	if !strings.HasPrefix(text, "/") {
		return
	}

	parts := strings.Fields(text)
	if len(parts) == 0 {
		return
	}
	cmd := parts[0]

	var reply string
	switch cmd {
	case "/name":
		reply = handleNameCommand(parts)
	case "/delname":
		reply = handleDelNameCommand(parts)
	case "/listnames":
		reply = handleListNamesCommand()
	default:
		return // unknown command, silently ignore
	}

	if reply != "" {
		if err := sendTelegramReply(token, chatID, reply); err != nil {
			log.Printf("发送命令回复失败: %v", err)
		}
	}
}

func handleNameCommand(parts []string) string {
	if len(parts) < 3 {
		return "用法: /name <IP地址> <别名>\n示例: /name 1.2.3.4 bage-hks"
	}
	ip := parts[1]
	name := parts[2]
	if err := aliases.set(ip, name); err != nil {
		return fmt.Sprintf("设置别名失败: %v", err)
	}
	return fmt.Sprintf("已设置: %s → %s", ip, name)
}

func handleDelNameCommand(parts []string) string {
	if len(parts) < 2 {
		return "用法: /delname <IP地址>"
	}
	ip := parts[1]
	if aliases.lookup(ip) == "" {
		return fmt.Sprintf("IP %s 没有设置别名", ip)
	}
	if err := aliases.del(ip); err != nil {
		return fmt.Sprintf("删除别名失败: %v", err)
	}
	return fmt.Sprintf("已删除: %s 的别名", ip)
}

func handleListNamesCommand() string {
	m := aliases.list()
	if len(m) == 0 {
		return "暂无 IP 别名"
	}
	var sb strings.Builder
	sb.WriteString("IP 别名列表:\n")
	for ip, name := range m {
		sb.WriteString(fmt.Sprintf("  %s → %s\n", ip, name))
	}
	return sb.String()
}

func sendTelegramReply(token, chatID, text string) error {
	body, _ := json.Marshal(telegramMessage{
		ChatID:    chatID,
		Text:      text,
		ParseMode: "",
	})

	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", token)
	resp, err := http.Post(url, "application/json", strings.NewReader(stringify(body)))
	if err != nil {
		return fmt.Errorf("发送回复失败: %w", err)
	}
	defer resp.Body.Close()

	var result telegramResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("解析回复响应失败: %w", err)
	}
	if !result.OK {
		return fmt.Errorf("发送回复 API 错误 %d: %s", result.ErrorCode, result.Description)
	}
	return nil
}

func stringify(b []byte) string { return string(b) }
