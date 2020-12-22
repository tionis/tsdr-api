package main

import (
	"errors"
	"os"
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
	bot, err := tgbotapi.NewBotAPI(os.Getenv("TELEGRAM_TOKEN"))
	if err != nil {
		glyphTelegramLog.Fatal(err)
	}

	glyphTelegramLog.Info("Glyph Telegram Bot was started.")

	bot.Debug = debug

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates, err := bot.GetUpdatesChan(u)

	telegramGlyphBot := &glyph.Bot{
		QuoteDBHandler: &glyph.QuoteDBHandler{
			AddQuote:       data.AddQuote,
			GetRandomQuote: data.GetRandomQuote,
		},
		SetContext:           getTelegramSetContext(),
		GetContext:           getTelegramGetContext(),
		GetUserData:          getTelegramGetUserData(),
		SetUserData:          getTelegramSetUserData(),
		SendMessageToChannel: getTelegramSendMessage(bot),
		GetMention:           getTelegramGetMention(),
		Logger:               glyphTelegramLog,
		Prefix:               "/",
	}

	for update := range updates {
		if update.Message == nil { // ignore any non-Message Updates
			continue
		}

		// Save Username to cache
		userID := strconv.Itoa(update.Message.From.ID)
		go data.SetTmp("glyph:tg:nameCache", userID, update.Message.From.FirstName+" "+update.Message.From.LastName, 30*time.Minute)

		// Trim prefix
		content := strings.TrimPrefix(update.Message.Text, telegramGlyphBot.Prefix)

		message := glyph.MessageData{
			Content:          content,
			AuthorID:         userID,
			IsDM:             !update.Message.Chat.IsGroup(),
			SupportsMarkdown: true,
			ChannelID:        strconv.FormatInt(update.Message.Chat.ID, 10),
			IsCommand:        update.Message.IsCommand(),
		}

		go telegramGlyphBot.HandleAll(message)
	}
}

// Init callback functions
func getTelegramSendMessage(bot *tgbotapi.BotAPI) func(channelID, message string) error {
	return func(channelID, message string) error {
		id, err := strconv.ParseInt(channelID, 10, 64)
		if err != nil {
			return errors.New("Error parsing channelID " + channelID + ": " + err.Error())
		}
		msg := tgbotapi.NewMessage(id, message)
		msg.ParseMode = "Markdown"
		_, err = bot.Send(msg)
		if err != nil {
			return errors.New("error sending message to " + channelID + ": " + err.Error())
		}
		return nil
	}
}

func getTelegramSetUserData() func(discordUserID, key string, value interface{}) error {
	return func(discordUserID, key string, value interface{}) error {
		userID, err := data.GetUserIDFromTelegramID(discordUserID)
		if err != nil {
			return err
		}
		err = data.SetUserData(userID, "glyph", key, value)
		if err != nil {
			return err
		}
		return nil
	}
}

func getTelegramGetUserData() func(discordUserID, key string) (interface{}, error) {
	return func(discordUserID, key string) (interface{}, error) {
		userID, err := data.GetUserIDFromTelegramID(discordUserID)
		if err != nil {
			return nil, err
		}
		value, err := data.GetUserData(userID, "glyph", key)
		if err != nil {
			return nil, err
		}
		return value, nil
	}
}
func getTelegramSetContext() func(userID, channelID, key, value string, ttl time.Duration) error {
	return func(userID, channelID, key, value string, ttl time.Duration) error {
		data.SetTmp("glyph:tg:"+channelID+":"+userID, key, value, ttl)
		return nil
	}
}

func getTelegramGetContext() func(userID, channelID, key string) (string, error) {
	return func(userID, channelID, key string) (string, error) {
		return data.GetTmp("glyph:tg:"+channelID+":"+userID, key), nil
	}
}

func getTelegramGetMention() func(userID string) (string, error) {
	return func(userID string) (string, error) {
		friendlyName := data.GetTmp("glyph:tg:nameCache", userID)
		if friendlyName == "" {
			friendlyName = userID
		}
		return "[" + friendlyName + "](tg://user?id=" + userID + ")", nil
	}
}
