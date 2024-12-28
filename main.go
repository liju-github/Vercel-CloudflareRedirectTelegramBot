package main

import (
	"log"
	"os"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/joho/godotenv"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	telegramToken := os.Getenv("TELEGRAM_TOKEN")
	if telegramToken == "" {
		log.Fatal("TELEGRAM_TOKEN environment variable is not set")
	}

	secretCode = os.Getenv("SECRET_CODE")
	if secretCode == "" {
		panic("SECRET_CODE environment variable is not set")
	}

	bot, err := tgbotapi.NewBotAPI(telegramToken)
	if err != nil {
		log.Fatal("Failed to create Telegram bot. Please check your TELEGRAM_TOKEN.")
	}

	bot.Debug = false

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := bot.GetUpdatesChan(u)

	for update := range updates {
		if update.Message == nil {
			continue
		}

		if update.Message.IsCommand() {
			handleTelegramCommand(bot, update)
		}
	}
}
