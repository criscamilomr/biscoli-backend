package notifications

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

// TelegramNotifier sends order notifications via Telegram Bot API.
type TelegramNotifier struct {
	BotToken string
	ChatID   string
}

// NewTelegramNotifier creates a new TelegramNotifier.
func NewTelegramNotifier(botToken, chatID string) *TelegramNotifier {
	return &TelegramNotifier{
		BotToken: botToken,
		ChatID:   chatID,
	}
}

// Name returns the notifier channel name.
func (t *TelegramNotifier) Name() string {
	return "Telegram"
}

// Send sends an order notification message via Telegram.
func (t *TelegramNotifier) Send(order OrderNotification) error {
	text := FormatOrderText(order)

	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", t.BotToken)

	payload := map[string]string{
		"chat_id":    t.ChatID,
		"text":       text,
		"parse_mode": "HTML",
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("error marshalling telegram payload: %w", err)
	}

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("error sending telegram message: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var result map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&result)
		return fmt.Errorf("telegram API error (status %d): %v", resp.StatusCode, result)
	}

	return nil
}
