package tgbot

import (
	"log"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type Bot struct {
	tg *tgbotapi.BotAPI
}

func New(token string) *Bot {
	bot, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		log.Println(err)
		return nil
	}
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 300
	updates := bot.GetUpdatesChan(u)
	_bot := &Bot{bot}
	go func() {
		for update := range updates {
			if update.Message != nil {
				_bot.executor(update)
			}
		}
	}()
	return _bot
}

func (b *Bot) executor(update *tgbotapi.Update) {

}

func (b *Bot) GetInlineMessage() {

}
