package bot

import (
	"log"
	"telegram-bot/internal/config"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
)

func StartBot(cfg *config.Config) error {
	bot, err := tgbotapi.NewBotAPI(cfg.BotToken)
	if err != nil {
		return err
	}

	log.Printf("Authorized on account %s", bot.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates, err := bot.GetUpdatesChan(u)
	if err != nil {
		return err
	}

	for update := range updates {
		if update.Message != nil {
			// Обробка повідомлень
			log.Printf("[%s] %s", update.Message.From.UserName, update.Message.Text)
		}
	}

	return nil
}