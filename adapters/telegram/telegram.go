package telegram

import (
	"errors"
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
const glyphTelegramContextDelay = time.Hour * 24

// Bot represents a config of the bot
type Bot struct {
	logger           *logging.Logger
	dataBackend      *data.GlyphData
	updates          tgbotapi.UpdatesChannel
	telegramGlyphBot *glyph.Bot
	//var msgGlyph chan string
}

// TODO message send channel with transformation from matrixID to telegramID

// Init initializes the bot adapter with the given data backend and token and returns a bot config
func Init(data *data.GlyphData, telegramToken string) Bot {
	out := Bot{logging.MustGetLogger("glyphTelegram"), data, nil, nil}
	bot, err := tgbotapi.NewBotAPI(telegramToken)
	if err != nil {
		out.logger.Fatal(err)
	}

	out.logger.Info("Glyph Telegram Bot was started.")

	bot.Debug = false // Not really needed, not even for development

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	out.updates, err = bot.GetUpdatesChan(u)

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
		SendMessageToChannel:  out.getTelegramSendMessage(bot),
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
	// TODO implement graceful shutdown via stop channel
	//defer not working as above not implemented
	syncGroup.Done()

	// Start message send Service
	go b.startMessageSendService()

	// Start listening for updates
	for update := range b.updates {
		if update.Message == nil { // ignore any non-Message Updates
			continue
		}

		// Save Username to cache
		userID := strconv.Itoa(update.Message.From.ID)
		go b.dataBackend.SetTmp("glyph:tg:nameCache", userID, update.Message.From.FirstName+" "+update.Message.From.LastName, 30*time.Minute)

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
	}
}

func (b Bot) startMessageSendService() {
	// Create channel to receive messages on

	// TODO register channel at data layer

}

// Init callback functions
func (b Bot) getTelegramSendMessage(bot *tgbotapi.BotAPI) func(channelID, message string) error {
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
