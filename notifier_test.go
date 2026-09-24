package main

import (
	"errors"
	"net/url"
	"strings"
	"testing"
)

func TestTelegramRequestErrorDoesNotExposeToken(t *testing.T) {
	err := &url.Error{
		Op:  "Post",
		URL: "https://api.telegram.org/botsecret-token/sendMessage",
		Err: errors.New("network timeout"),
	}
	message := telegramRequestError(err).Error()
	if strings.Contains(message, "secret-token") || !strings.Contains(message, "network timeout") {
		t.Fatalf("unsafe or unhelpful error: %q", message)
	}
}
