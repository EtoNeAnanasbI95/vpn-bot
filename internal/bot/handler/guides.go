package handler

import (
	"context"
	"fmt"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/EtoNeAnanasbI95/vpn-bot/internal/bot/keyboard"
)

// HandleGuideList sends the platform selection keyboard.
func HandleGuideList(
	ctx context.Context,
	bot *tgbotapi.BotAPI,
	chatID int64,
	callbackID string,
	uc *UseCases,
) {
	platforms, err := uc.Guide.ListPlatforms(ctx)
	if err != nil || len(platforms) == 0 {
		answerCallback(bot, callbackID, "")
		send(bot, tgbotapi.NewMessage(chatID, "Гайды временно недоступны."))
		return
	}
	answerCallback(bot, callbackID, "")

	msg := tgbotapi.NewMessage(chatID, "📖 <b>Гайды по подключению</b>\n\nВыберите платформу:")
	msg.ParseMode = tgbotapi.ModeHTML
	msg.ReplyMarkup = keyboard.PlatformList(platforms)
	send(bot, msg)
}

// HandleGuideGet sends the PDF for the selected platform.
func HandleGuideGet(
	ctx context.Context,
	bot *tgbotapi.BotAPI,
	chatID int64,
	callbackID string,
	platformKey string,
	uc *UseCases,
) {
	answerCallback(bot, callbackID, "")

	data, err := uc.Guide.GetGuide(ctx, platformKey)
	if err != nil {
		send(bot, tgbotapi.NewMessage(chatID, "Гайд для этой платформы не найден."))
		return
	}

	// Resolve label for caption.
	platforms, _ := uc.Guide.ListPlatforms(ctx)
	label := platformKey
	for _, p := range platforms {
		if p.Key == platformKey {
			label = p.Label
			break
		}
	}

	doc := tgbotapi.NewDocument(chatID, tgbotapi.FileBytes{
		Name:  fmt.Sprintf("vpn-guide-%s.pdf", platformKey),
		Bytes: data,
	})
	doc.Caption = fmt.Sprintf("📖 Инструкция по подключению на %s", label)
	send(bot, doc)
}

// HandleGuideListFromMessage handles the "📖 Гайды" text button.
func HandleGuideListFromMessage(
	ctx context.Context,
	bot *tgbotapi.BotAPI,
	chatID int64,
	uc *UseCases,
) {
	platforms, err := uc.Guide.ListPlatforms(ctx)
	if err != nil || len(platforms) == 0 {
		send(bot, tgbotapi.NewMessage(chatID, "Гайды временно недоступны."))
		return
	}

	msg := tgbotapi.NewMessage(chatID, "📖 <b>Гайды по подключению</b>\n\nВыберите платформу:")
	msg.ParseMode = tgbotapi.ModeHTML
	msg.ReplyMarkup = keyboard.PlatformList(platforms)
	send(bot, msg)
}
