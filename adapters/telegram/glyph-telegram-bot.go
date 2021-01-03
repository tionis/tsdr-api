package telegram

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

var dataBackend *data.GlyphData

// InitBot initializes and starts the bot adapter with the given data backend
func InitBot(data *data.GlyphData, debug bool) {
	dataBackend = data
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
		QuoteDBHandler: &glyph.QuoteDB{
			AddQuote:         dataBackend.AddQuote,
			GetRandomQuote:   dataBackend.GetRandomQuote,
			GetQuoteOfTheDay: getTelegramGetQuoteOfTheDay(),
			SetQuoteOfTheDay: getTelegramSetQuoteOfTheDay(),
		},
		UserDBHandler: &glyph.UserDB{
			GetUserData:           getTelegramGetUserData(),
			SetUserData:           getTelegramSetUserData(),
			DeleteUserData:        getTelegramDeleteUserData(),
			GetMatrixUserID:       getTelegramGetMatrixUserID(),
			DoesMatrixUserIDExist: dataBackend.DoesUserIDExist,
			AddAuthSession:        dataBackend.AddAuthSession,
			GetAuthSessionStatus:  dataBackend.GetAuthSessionStatus,
			AuthenticateSession:   dataBackend.AuthenticateSession,
			DeleteSession:         dataBackend.DeleteSession,
			GetAuthSessions:       dataBackend.GetAuthSessions,
		},
		SetContext:           getTelegramSetContext(),
		GetContext:           getTelegramGetContext(),
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
		go dataBackend.SetTmp("glyph:tg:nameCache", userID, update.Message.From.FirstName+" "+update.Message.From.LastName, 30*time.Minute)

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

func getTelegramSetUserData() func(telegramUserID, key string, value string) error {
	return func(telegramUserID, key string, value string) error {
		userID, err := dataBackend.GetUserIDFromValueOfKey("telegramID", telegramUserID)
		if err != nil {
			return err
		}
		err = dataBackend.SetUserData(userID, key, value)
		if err != nil {
			return err
		}
		return nil
	}
}

func getTelegramGetUserData() func(telegramUserID, key string) (string, error) {
	return func(telegramUserID, key string) (string, error) {
		userID, err := dataBackend.GetUserIDFromValueOfKey("telegramID", telegramUserID)
		if err != nil {
			return "", err
		}
		value, err := dataBackend.GetUserData(userID, key)
		if err != nil {
			return "", err
		}
		return value, nil
	}
}
func getTelegramSetContext() func(userID, channelID, key, value string, ttl time.Duration) error {
	return func(userID, channelID, key, value string, ttl time.Duration) error {
		dataBackend.SetTmp("glyph:tg:"+channelID+":"+userID, key, value, ttl)
		return nil
	}
}

func getTelegramGetContext() func(userID, channelID, key string) (string, error) {
	return func(userID, channelID, key string) (string, error) {
		return dataBackend.GetTmp("glyph:tg:"+channelID+":"+userID, key), nil
	}
}

func getTelegramGetMention() func(userID string) (string, error) {
	return func(userID string) (string, error) {
		friendlyName := dataBackend.GetTmp("glyph:tg:nameCache", userID)
		if friendlyName == "" {
			friendlyName = userID
		}
		return "[" + friendlyName + "](tg://user?id=" + userID + ")", nil
	}
}

func getTelegramGetQuoteOfTheDay() func(telegramUserID string) (glyph.QuoteOfTheDay, error) {
	return func(telegramUserID string) (glyph.QuoteOfTheDay, error) {
		userID, err := dataBackend.GetUserIDFromValueOfKey("telegramID", telegramUserID)
		if err != nil {
			return glyph.QuoteOfTheDay{}, err
		}
		qotd, err := dataBackend.GetQuoteOfTheDayOfUser(userID)
		if err != nil {
			return glyph.QuoteOfTheDay{}, err
		}
		return qotd, nil
	}
}

func getTelegramSetQuoteOfTheDay() func(telegramUserID string, quoteOfTheDay glyph.QuoteOfTheDay) error {
	return func(telegramUserID string, quoteOfTheDay glyph.QuoteOfTheDay) error {
		userID, err := dataBackend.GetUserIDFromValueOfKey("telegramID", telegramUserID)
		if err != nil {
			return err
		}
		return dataBackend.SetQuoteOfTheDayOfUser(userID, quoteOfTheDay)
	}
}

func getTelegramGetMatrixUserID() func(telegramUserID string) (string, error) {
	return func(telegramUserID string) (string, error) {
		return dataBackend.GetUserIDFromValueOfKey("telegramID", telegramUserID)
	}
}

func getTelegramDeleteUserData() func(telegramUserID, key string) error {
	return func(telegramUserID, key string) error {
		userID, err := dataBackend.GetUserIDFromValueOfKey("telegramID", telegramUserID)
		if err != nil {
			return err
		}
		return dataBackend.DeleteUserData(userID, key)
	}
}
