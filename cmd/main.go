package main

import (
	"log"
	"telegram-bot/internal/bot"
	"telegram-bot/internal/config"
)

func main() {
	// Завантаження конфігурації
	cfg, err := config.LoadConfig("config.json")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Ініціалізація бота
	b, err := bot.NewBot(cfg.BotToken, cfg.AdminChatID, cfg.GroupChatID)
	if err != nil {
		log.Fatalf("Failed to initialize bot: %v", err)
	}

	// Запуск бота
	log.Println("Bot is running...")
	b.Start()
}
