package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
	"github.com/PaulSonOfLars/gotgbot/v2/ext/handlers"
	"github.com/tmc/langchaingo/llms"
)

type controller struct {
	bot        *gotgbot.Bot
	updater    *ext.Updater
	dispatcher *ext.Dispatcher
	state      StateStorage
	chats      ChatStorage
}

type Controller interface {
	Start() error
	Stop() error
}

func NewController(token string, llm llms.Model) Controller {
	b, err := gotgbot.NewBot(token, nil)
	if err != nil {
		log.Fatalf("failed to create bot: %v", err)
	}

	dispatcher := ext.NewDispatcher(&ext.DispatcherOpts{
		Error: func(b *gotgbot.Bot, ctx *ext.Context, err error) ext.DispatcherAction {
			log.Println("an error occurred while handling update:", err.Error())
			return ext.DispatcherActionNoop
		},
		MaxRoutines: ext.DefaultMaxRoutines,
	})

	updater := ext.NewUpdater(dispatcher, nil)

	chatStorage := NewChatStorage(llm)
	stateStorage := NewStateStorage()

	return &controller{
		bot:        b,
		updater:    updater,
		dispatcher: dispatcher,
		state:      stateStorage,
		chats:      chatStorage,
	}
}

func (c *controller) Start() error {
	c.dispatcher.AddHandler(handlers.NewCommand("start", c.startHandler))
	c.dispatcher.AddHandler(handlers.NewCommand("setrole", c.setRoleHandler))
	c.dispatcher.AddHandler(handlers.NewMessage(c.messageFilter, c.messageHandler))

	log.Printf("Starting bot, username: %s", c.bot.User.Username)
	return c.updater.StartPolling(c.bot, &ext.PollingOpts{
		DropPendingUpdates: true,
		GetUpdatesOpts: &gotgbot.GetUpdatesOpts{
			Timeout: 9,
			RequestOpts: &gotgbot.RequestOpts{
				Timeout: time.Second * 10,
			},
		},
	})
}

func (c *controller) Stop() error {
	return c.updater.Stop()
}

func (c *controller) messageFilter(msg *gotgbot.Message) bool {
	chatID := msg.Chat.Id
	if c.state.IsWaitingForRole(chatID) {
		return true
	}

	if msg.Chat.Type == "private" {
		return true
	}

	if (msg.Chat.Type == "group" || msg.Chat.Type == "supergroup") && msg.ReplyToMessage != nil {
		for _, entity := range msg.Entities {
			if entity.Type == "mention" {
				mention := msg.Text[entity.Offset : entity.Offset+entity.Length]
				if mention == "@"+c.bot.User.Username {
					return true
				}
			}
		}
	}

	return false
}

func (c *controller) startHandler(b *gotgbot.Bot, ctx *ext.Context) error {
	_, err := ctx.EffectiveMessage.Reply(b, "Привет! Я бот, который будет тем кем тебе захочется. Используй /setrole для задания роли.", nil)
	return err
}

func (c *controller) setRoleHandler(b *gotgbot.Bot, ctx *ext.Context) error {
	chatId := ctx.EffectiveChat.Id
	c.state.SetWaitingForRole(chatId, true)

	_, err := ctx.EffectiveMessage.Reply(b, "Напишите кем мне быть", nil)
	return err
}

func (c *controller) messageHandler(b *gotgbot.Bot, ctx *ext.Context) error {
	msg := ctx.EffectiveMessage
	chat := ctx.EffectiveChat

	aiChat := c.chats.GetOrCreateChat(chat.Id)

	// Задание роли
	if c.state.IsWaitingForRole(chat.Id) {
		c.state.SetWaitingForRole(chat.Id, false)
		aiChat.SetRole(context.TODO(), msg.Text)
		_, err := ctx.EffectiveMessage.Reply(b, "Роль установлена!", nil)
		return err
	}

	// Личное сообщение боту
	if chat.Type == "private" {
		resp, err := aiChat.GenerateContentForUser(context.TODO(), msg.Text)
		if err != nil {
			return fmt.Errorf("failed to generate response: %w", err)
		}
		_, err = ctx.EffectiveMessage.Reply(b, resp, &gotgbot.SendMessageOpts{
			ParseMode: "Markdown",
		})
		return err
	}

	// В группе: бот упомянут и это ответ на сообщение
	if chat.Type == "group" || chat.Type == "supergroup" {
		resp, err := aiChat.GenerateContentForUser(context.TODO(), msg.ReplyToMessage.Text)
		if err != nil {
			return fmt.Errorf("failed to generate response: %w", err)
		}
		_, err = ctx.EffectiveMessage.Reply(b, resp, nil)
		return err
	}

	return nil
}
