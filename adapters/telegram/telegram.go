package telegram

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	_ "github.com/heroku/x/hmetrics/onload" // Heroku advanced go metrics
	"github.com/keybase/go-logging"
	"github.com/tionis/tsdr-api/data"
	"github.com/tionis/tsdr-api/glyph"
)

const adapterID = "telegram"

//const glyphTelegramContextDelay = time.Hour * 24

// Bot represents a config of the bot
type Bot struct {
	logger           *logging.Logger
	dataBackend      *data.GlyphData
	updates          tgbotapi.UpdatesChannel
	telegramGlyphBot *glyph.Bot
	bot              *tgbotapi.BotAPI
}

// Init initializes the bot adapter with the given data backend and token and returns a bot config
func Init(data *data.GlyphData, telegramToken string) Bot {
	out := Bot{logging.MustGetLogger("glyphTelegram"), data, nil, nil, nil}
	var err error
	out.bot, err = tgbotapi.NewBotAPI(telegramToken)
	if err != nil {
		out.logger.Fatal(err)
	}

	out.logger.Info("Glyph Telegram Bot was started.")

	out.bot.Debug = false // Not really needed, not even for development

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	out.updates, err = out.bot.GetUpdatesChan(u)
	if err != nil {
		out.logger.Error("Could not create update channel to listen to telegram updates:", err)
	}

	out.telegramGlyphBot = &glyph.Bot{
		QuoteDBHandler: &glyph.QuoteDB{
			AddQuote:         data.AddQuote,
			GetRandomQuote:   data.GetRandomQuote,
			GetQuoteOfTheDay: out.getTelegramGetQuoteOfTheDay(),
			SetQuoteOfTheDay: out.getTelegramSetQuoteOfTheDay(),
		},
		UserDBHandler: &glyph.UserDB{
			GetUserData:           out.getTelegramGetUserData(),
			SetUserData:           out.getTelegramSetUserData(),
			DeleteUserData:        out.getTelegramDeleteUserData(),
			GetMatrixUserID:       out.getTelegramGetMatrixUserID(),
			DoesMatrixUserIDExist: data.DoesUserIDExist,
			AddAuthSession:        data.AddAuthSession,
			GetAuthSessionStatus:  data.GetAuthSessionStatus,
			AuthenticateSession:   data.AuthenticateSession,
			DeleteSession:         data.DeleteSession,
			GetAuthSessions:       data.GetAuthSessions,
		},
		SetContext:            out.getTelegramSetContext(),
		GetContext:            out.getTelegramGetContext(),
		SendMessageToChannel:  out.getTelegramSendMessage(),
		GetMention:            out.getTelegramGetMention(),
		SendMessageViaAdapter: out.getSendMessageViaAdapter(),
		CurrentAdapter:        adapterID,
		Logger:                out.logger,
		Prefix:                "/",
	}
	return out
}

// Start starts the bot and aborts its execution when a value on the stop signal is received
func (b Bot) Start(stop chan bool, syncGroup *sync.WaitGroup) {
	b.logger.Info("Glyph Discord Bot was started.")

	// Start message send Service
	go b.startMessageSendService()

	// Start listening for updates
	for {
		select {
		case update := <-b.updates:
			if update.Message == nil { // ignore any non-Message Updates
				continue
			}

			// Save Username to cache
			userID := strconv.Itoa(update.Message.From.ID)
			go b.dataBackend.SetTmp("glyph:tg:nameCache", userID, update.Message.From.FirstName+" "+update.Message.From.LastName, 30*time.Minute)
			b.updateChatIDFromTelegramID(userID, update.Message.Chat.ID)

			// Trim prefix
			content := strings.TrimPrefix(update.Message.Text, b.telegramGlyphBot.Prefix)

			message := glyph.MessageData{
				Content:          content,
				AuthorID:         userID,
				IsDM:             !update.Message.Chat.IsGroup(),
				SupportsMarkdown: true,
				ChannelID:        strconv.FormatInt(update.Message.Chat.ID, 10),
				IsCommand:        update.Message.IsCommand(),
			}

			go b.telegramGlyphBot.HandleAll(message)

		case <-stop:
			b.bot.StopReceivingUpdates()
			syncGroup.Done()
			b.logger.Info("Glyph Telegram Bot was stopped.")
		}
	}
}

func (b Bot) startMessageSendService() {
	// Create channel to receive messages on
	messageChannel := make(chan data.AdapterMessage, 10)
	b.dataBackend.RegisterAdapterChannel(adapterID, messageChannel)

	for {
		message := <-messageChannel
		id, err := b.getChatIDFromMatrixUserID(message.UserID)
		if err != nil {
			b.logger.Warningf("failed to get chat id of user %v: %v", message.UserID, err)
			continue
		}
		msg := tgbotapi.NewMessage(id, message.Message)
		msg.ParseMode = "Markdown"
		_, err = b.bot.Send(msg)
		if err != nil {
			b.logger.Warningf("failed to send telegram message to %v on channel %v:", message.UserID, id, err)
			continue
		}
	}
}

func (b Bot) updateChatIDFromTelegramID(telegramID string, chatID int64) {
	gottenChatID := b.dataBackend.GetTmp("glyph", "tg:"+telegramID+"|DM-ChatID")
	if gottenChatID == "" {
		b.dataBackend.SetTmp("glyph", "tg:"+telegramID+"|DM-ChatID", fmt.Sprint(chatID), 24*time.Hour)
		matrixID, err := b.dataBackend.GetUserIDFromValueOfKey("telegramID", telegramID)
		if err != nil {
			b.logger.Debugf("Could not get matrix id for telegram user %v: %v", telegramID, err)
			return
		}
		err = b.dataBackend.SetUserData(matrixID, "telegram-DM-ChatID", fmt.Sprint(chatID))
		if err != nil {
			b.logger.Warningf("Could not set user data for telegram dm chat id: %v", err)
			return
		}
	}
}

func (b Bot) getChatIDFromMatrixUserID(userID string) (int64, error) {
	telegramID, err := b.dataBackend.GetUserData(userID, "telegramID")
	if err != nil {
		return 0, err
	}
	chatID := b.dataBackend.GetTmp("glyph", "tg:"+telegramID+"|DM-ChatID")
	if chatID == "" {
		chatID, err = b.dataBackend.GetUserData(userID, "telegram-DM-ChatID")
		if err != nil {
			return 0, err
		}
		b.dataBackend.SetTmp("glyph", "tg:"+telegramID+"|DM-ChatID", chatID, 24*time.Hour)
	}
	return strconv.ParseInt(chatID, 10, 64)
}

// Init callback functions
func (b Bot) getTelegramSendMessage() func(channelID, message string) error {
	return func(channelID, message string) error {
		id, err := strconv.ParseInt(channelID, 10, 64)
		if err != nil {
			return errors.New("Error parsing channelID " + channelID + ": " + err.Error())
		}
		msg := tgbotapi.NewMessage(id, message)
		msg.ParseMode = "Markdown"
		_, err = b.bot.Send(msg)
		if err != nil {
			return errors.New("error sending message to " + channelID + ": " + err.Error())
		}
		return nil
	}
}

func (b Bot) getTelegramSetUserData() func(telegramUserID, key string, value string) error {
	return func(telegramUserID, key string, value string) error {
		userID, err := b.dataBackend.GetUserIDFromValueOfKey("telegramID", telegramUserID)
		if err != nil {
			return err
		}
		err = b.dataBackend.SetUserData(userID, key, value)
		if err != nil {
			return err
		}
		return nil
	}
}

func (b Bot) getTelegramGetUserData() func(telegramUserID, key string) (string, error) {
	return func(telegramUserID, key string) (string, error) {
		userID, err := b.dataBackend.GetUserIDFromValueOfKey("telegramID", telegramUserID)
		if err != nil {
			return "", err
		}
		value, err := b.dataBackend.GetUserData(userID, key)
		if err != nil {
			return "", err
		}
		return value, nil
	}
}
func (b Bot) getTelegramSetContext() func(userID, channelID, key, value string, ttl time.Duration) error {
	return func(userID, channelID, key, value string, ttl time.Duration) error {
		b.dataBackend.SetTmp("glyph:tg:"+channelID+":"+userID, key, value, ttl)
		return nil
	}
}

func (b Bot) getTelegramGetContext() func(userID, channelID, key string) (string, error) {
	return func(userID, channelID, key string) (string, error) {
		return b.dataBackend.GetTmp("glyph:tg:"+channelID+":"+userID, key), nil
	}
}

func (b Bot) getTelegramGetMention() func(userID string) (string, error) {
	return func(userID string) (string, error) {
		friendlyName := b.dataBackend.GetTmp("glyph:tg:nameCache", userID)
		if friendlyName == "" {
			friendlyName = userID
		}
		return "[" + friendlyName + "](tg://user?id=" + userID + ")", nil
	}
}

func (b Bot) getTelegramGetQuoteOfTheDay() func(telegramUserID string) (glyph.QuoteOfTheDay, error) {
	return func(telegramUserID string) (glyph.QuoteOfTheDay, error) {
		userID, err := b.dataBackend.GetUserIDFromValueOfKey("telegramID", telegramUserID)
		if err != nil {
			return glyph.QuoteOfTheDay{}, err
		}
		qotd, err := b.dataBackend.GetQuoteOfTheDayOfUser(userID)
		if err != nil {
			return glyph.QuoteOfTheDay{}, err
		}
		return qotd, nil
	}
}

func (b Bot) getTelegramSetQuoteOfTheDay() func(telegramUserID string, quoteOfTheDay glyph.QuoteOfTheDay) error {
	return func(telegramUserID string, quoteOfTheDay glyph.QuoteOfTheDay) error {
		userID, err := b.dataBackend.GetUserIDFromValueOfKey("telegramID", telegramUserID)
		if err != nil {
			return err
		}
		return b.dataBackend.SetQuoteOfTheDayOfUser(userID, quoteOfTheDay)
	}
}

func (b Bot) getTelegramGetMatrixUserID() func(telegramUserID string) (string, error) {
	return func(telegramUserID string) (string, error) {
		return b.dataBackend.GetUserIDFromValueOfKey("telegramID", telegramUserID)
	}
}

func (b Bot) getTelegramDeleteUserData() func(telegramUserID, key string) error {
	return func(telegramUserID, key string) error {
		userID, err := b.dataBackend.GetUserIDFromValueOfKey("telegramID", telegramUserID)
		if err != nil {
			return err
		}
		return b.dataBackend.DeleteUserData(userID, key)
	}
}

// ToDo this is duplicate code and may be removable in the future
func (b Bot) getSendMessageViaAdapter() func(matrixUserID, message string) error {
	return func(matrixUserID, message string) error {
		adapterIDs, err := b.dataBackend.UserGetPreferredAdapters(matrixUserID)
		if err != nil {
			return err
		}
		for _, item := range adapterIDs {
			channel, err := b.dataBackend.GetAdapterChannel(item)
			if err != nil {
				return err
			}
			channel <- data.AdapterMessage{UserID: matrixUserID, Message: message}
		}
		return nil
	}
}
