package main

import (
	"log"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func SendTelegramMessage(message string) {
	bot, err := tgbotapi.NewBotAPI("8487412493:AAGiIJWP_VKEfkJ6cbanAmfE_szoLDfNZE4")
	if err != nil {
		log.Fatal(err)
	}

	msg := tgbotapi.NewMessage(1784293951, message)
	if _, err := bot.Send(msg); err != nil {
		log.Fatal(err)
	}
}
