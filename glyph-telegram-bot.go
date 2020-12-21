package main

import (
	"strconv"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	_ "github.com/heroku/x/hmetrics/onload" // Heroku advanced go metrics
	"github.com/keybase/go-logging"
	"github.com/tionis/tsdr-api/data"
	"github.com/tionis/tsdr-api/glyph"
)

var glyphTelegramLog = logging.MustGetLogger("glyphTelegram")

const glyphTelegramContextDelay = time.Hour * 24

var msgGlyph chan string

// GlyphTelegramBot handles all the glyph bot interfacing between the glyph logic and telegram
func glyphTelegramBot(debug bool) {
	bot, err := tgbotapi.NewBotAPI("MyAwesomeBotToken")
	if err != nil {
		glyphTelegramLog.Fatal(err)
	}

	glyphTelegramLog.Info("Glyph Telegram Bot was started.")

	bot.Debug = debug

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates, err := bot.GetUpdatesChan(u)

	// Init callback functions
	var telegramSendMessage = func(channelID, message string) {
		id, err := strconv.ParseInt(channelID, 10, 64)
		if err != nil {
			glyphDiscordLog.Errorf("Error parsing channelID %v: %v", channelID, err)
		}
		msg := tgbotapi.NewMessage(id, message)
		_, err = bot.Send(msg)
		if err != nil {
			glyphDiscordLog.Errorf("Error sending message to %v: %v", channelID, err)
		}
	}
	var telegramSetUserData = func(discordUserID, key string, value interface{}) {
		userID, err := data.GetUserIDFromTelegramID(discordUserID)
		if err != nil {
			glyphDiscordLog.Errorf("error setting user data: %v", err)
			return
		}
		err = data.SetUserData(userID, "glyph", key, value)
		if err != nil {
			glyphDiscordLog.Errorf("error setting user data: %v", err)
			return
		}
	}
	var telegramGetUserData = func(discordUserID, key string) interface{} {
		userID, err := data.GetUserIDFromTelegramID(discordUserID)
		if err != nil {
			glyphDiscordLog.Errorf("error setting user data: %v", err)
			return ""
		}
		value, err := data.GetUserData(userID, "glyph", key)
		if err != nil {
			glyphDiscordLog.Errorf("error setting user data: %v", err)
			return ""
		}
		return value
	}
	var telegramSetContext = func(userID, channelID, key, value string, ttl time.Duration) {
		data.SetTmp("glyph:tg:"+channelID+":"+userID, key, value, ttl)
	}
	var telegramGetContext = func(userID, channelID, key string) string {
		return data.GetTmp("glyph:tg:"+channelID+":"+userID, key)
	}
	var telegramGetMention = func(userID string) string {
		return "[inline mention of a user](tg://user?id=" + userID + ")"
	}

	telegramGlyphBot := &glyph.Bot{
		AddQuote:             data.AddQuote,
		GetRandomQuote:       data.GetRandomQuote,
		SetContext:           telegramSetContext,
		GetContext:           telegramGetContext,
		GetUserData:          telegramGetUserData,
		SetUserData:          telegramSetUserData,
		SendMessageToChannel: telegramSendMessage,
		GetMention:           telegramGetMention,
		Prefix:               "/",
	}

	for update := range updates {
		if update.Message == nil { // ignore any non-Message Updates
			continue
		}

		// Trim prefix
		content := strings.TrimPrefix(update.Message.Text, telegramGlyphBot.Prefix)

		message := glyph.MessageData{
			Content:          content,
			AuthorID:         strconv.Itoa(update.Message.From.ID),
			IsDM:             !update.Message.Chat.IsGroup(),
			SupportsMarkdown: true,
			ChannelID:        strconv.FormatInt(update.Message.Chat.ID, 10),
			IsCommand:        update.Message.IsCommand(),
		}

		go telegramGlyphBot.HandleAll(message)
	}
}
