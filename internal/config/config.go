package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	TelegramToken   string
	WebhookURL      string
	WebhookPath     string
	ListenAddr      string
	RegisterWebhook bool
	AdminIDs        []int64
	DBPath          string
	XUIBaseURL      string
	XUIUsername     string
	XUIPassword     string
	XUIServerAddr string // public VPN server address used to build VLESS URIs, e.g. vpn.example.com
	XUIInboundID  int    // ID of the Reality inbound to create clients on
	GuidesDir       string
	ReminderTZ      string
}

func Load() (*Config, error) {
	cfg := &Config{
		TelegramToken: mustEnv("TELEGRAM_TOKEN"),
		WebhookURL:      getEnv("WEBHOOK_URL", ""),
		WebhookPath:     getEnv("WEBHOOK_PATH", "/webhook"),
		RegisterWebhook: getEnv("REGISTER_WEBHOOK", "true") == "true",
		ListenAddr:    getEnv("LISTEN_ADDR", ":8080"),
		DBPath:        getEnv("DB_PATH", "./data/bot.db"),
		XUIBaseURL:    getEnv("XUI_BASE_URL", ""),
		XUIUsername:   getEnv("XUI_USERNAME", ""),
		XUIPassword:   getEnv("XUI_PASSWORD", ""),
		XUIServerAddr: getEnv("XUI_SERVER_ADDR", ""),
		GuidesDir:     getEnv("GUIDES_DIR", "./assets/guides"),
		ReminderTZ:    getEnv("REMINDER_TZ", "Europe/Moscow"),
	}

	if inboundIDRaw := getEnv("XUI_INBOUND_ID", "0"); inboundIDRaw != "0" {
		id, err := strconv.Atoi(inboundIDRaw)
		if err != nil {
			return nil, fmt.Errorf("invalid XUI_INBOUND_ID %q: %w", inboundIDRaw, err)
		}
		cfg.XUIInboundID = id
	}

	adminIDsRaw := getEnv("ADMIN_IDS", "")
	if adminIDsRaw == "" {
		return nil, fmt.Errorf("ADMIN_IDS is required")
	}
	for _, part := range strings.Split(adminIDsRaw, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		id, err := strconv.ParseInt(part, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid admin ID %q: %w", part, err)
		}
		cfg.AdminIDs = append(cfg.AdminIDs, id)
	}
	if len(cfg.AdminIDs) == 0 {
		return nil, fmt.Errorf("ADMIN_IDS must contain at least one ID")
	}

	return cfg, nil
}

func (c *Config) IsAdmin(telegramID int64) bool {
	for _, id := range c.AdminIDs {
		if id == telegramID {
			return true
		}
	}
	return false
}

func mustEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		panic(fmt.Sprintf("required environment variable %s is not set", key))
	}
	return v
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
