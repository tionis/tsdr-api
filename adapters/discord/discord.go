package discord

import (
	"strings"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"         // This provides the connection to the discord bot api
	_ "github.com/heroku/x/hmetrics/onload" // Heroku advanced go metrics
	"github.com/keybase/go-logging"         // This unifies logging across components of the application
	"github.com/tionis/tsdr-api/data"       // This implements the glyph specific data layer
	"github.com/tionis/tsdr-api/glyph"      // This implements the glyph bot logic and dictates the adapters design
)

/*type glyphDiscordMsgObject struct {
	ChannelID string `form:"channelid" json:"channelid" binding:"required"`
	Message   string `form:"message" json:"message" binding:"required"`
}*/

const adapterID = "discord"

// Bot represents a configuration of the bot and exposes functions to interact with it
type Bot struct {
	dataBackend       *data.GlyphData
	discordGlyphBot   *glyph.Bot
	logger            *logging.Logger
	discordBotMention string
	dg                *discordgo.Session
	//var glyphSend chan glyphDiscordMsgObject
}

// Init initializes the bot adapter with the given data backend and token
// and returns a bot config that can then be started
func Init(data *data.GlyphData, discordToken string) Bot {
	// Init logging and other required objects
	logger := logging.MustGetLogger("glyphDiscord")
	//glyphSend = make(chan glyphDiscordMsgObject, 2)

	// Init Sessions
	dg, err := discordgo.New("Bot " + discordToken)
	if err != nil {
		logger.Error("Error creating Discord session,", err)
		return Bot{}
	}

	// Open a websocket connection to Discord and begin listening.
	err = dg.Open()
	if err != nil {
		logger.Error("Error opening connection,", err)
		return Bot{}
	}
	_ = dg.UpdateStatus(0, "/help for help")

	bot := Bot{
		dataBackend:       data,
		discordGlyphBot:   nil,
		discordBotMention: "<@!" + dg.State.User.ID + ">",
		dg:                dg,
		logger:            logger,
	}

	bot.discordGlyphBot = &glyph.Bot{
		QuoteDBHandler: &glyph.QuoteDB{
			AddQuote:         data.AddQuote,
			GetRandomQuote:   data.GetRandomQuote,
			GetQuoteOfTheDay: bot.getDiscordGetQuoteOfTheDay(),
			SetQuoteOfTheDay: bot.getDiscordSetQuoteOfTheDay(),
		},
		UserDBHandler: &glyph.UserDB{
			GetUserData:                  bot.getDiscordGetUserData(),
			SetUserData:                  bot.getDiscordSetUserData(),
			DeleteUserData:               bot.getDiscordDeleteUserData(),
			GetMatrixUserID:              bot.getDiscordGetMatrixUserID(),
			DoesMatrixUserIDExist:        data.DoesUserIDExist,
			AddAuthSession:               data.AddAuthSession,
			AddAuthSessionWithAdapterAdd: data.AddAuthSessionWithAdapterAdd,
			AuthenticateSession:          data.AuthenticateSession,
			DeleteSession:                data.DeleteSession,
			GetAuthSessions:              data.GetAuthSessions,
			RegisterNewUser:              data.UserAdd,
		},
		SetContext:            bot.getDiscordSetContext(),
		GetContext:            bot.getDiscordGetContext(),
		SendMessageToChannel:  bot.getDiscordSendMessage(dg),
		GetMention:            bot.getDiscordGetMention(),
		SendMessageViaAdapter: bot.getSendMessageViaAdapter(),
		CurrentAdapter:        adapterID,
		Logger:                logger,
		Prefix:                "/",
	}

	// Register the messageCreate func as a callback for MessageCreate events.
	dg.AddHandler(bot.messageCreate)
	return bot
}

// Start starts the bot in its own goroutine and stops it if it receives an value on the stop channel,
// when stopped it decrements the counter of the waitgroup. Please note that it increments it on its own
// on startup.
func (b Bot) Start(stop chan bool, syncGroup *sync.WaitGroup) {
	syncGroup.Add(1)

	// Write start message into log
	b.logger.Info("Glyph Discord Bot was started.")

	go b.startMessageSendService()

	<-stop // Wait here until stop signal received

	// Close the Discord session.
	_ = b.dg.Close()
	syncGroup.Done()
	b.logger.Info("Glyph Discord Bot was stopped.")
}

// This function will be called (due to AddHandler above) every time a new
// message is created on any channel that the authenticated bot has access to.
func (b Bot) messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	// Build callback functions

	// Ignore all messages created by the bot itself
	if m.Author.ID == s.State.User.ID {
		return
	}

	// Check if message was sent in an DM
	AuthorDMChannelID := b.dataBackend.GetTmp("glyph", "dg:"+m.Author.ID+"|DM-Channel")
	if AuthorDMChannelID == "" {
		AuthorDMChannel, err := s.UserChannelCreate(m.Author.ID)
		if err != nil {
			return
		}
		AuthorDMChannelID = AuthorDMChannel.ID
		b.dataBackend.SetTmp("glyph", "dg:"+m.Author.ID+"|DM-Channel", AuthorDMChannelID, time.Hour*24)
	}

	// Build message object
	message := glyph.MessageData{
		Content:          m.Content,
		AuthorID:         m.Author.ID,
		ChannelID:        m.ChannelID,
		IsDM:             m.ChannelID == AuthorDMChannelID,
		SupportsMarkdown: true,
		IsCommand:        false,
	}

	// Split message into chunks
	tokens := strings.Split(message.Content, " ")

	if strings.HasPrefix(tokens[0], b.discordGlyphBot.Prefix) {
		message.Content = strings.TrimPrefix(message.Content, "/")
		message.IsCommand = true
	} else if tokens[0] == b.discordBotMention {
		message.Content = strings.TrimPrefix(message.Content, b.discordBotMention)
		message.IsCommand = true
	}

	message.Content = strings.TrimLeft(message.Content, "\t\r\n\v\f ")

	// Pass message object to glyph bot logic
	go b.discordGlyphBot.HandleAll(message)
}

func (b Bot) startMessageSendService() {
	messageChannel := make(chan data.AdapterMessage, 10)
	b.dataBackend.RegisterAdapterChannel(adapterID, messageChannel)

	for {
		message := <-messageChannel
		discordID, err := b.dataBackend.GetUserData(message.UserID, adapterID+"ID")
		if err != nil {
			b.logger.Warningf("failed to get discordID of user %v: %v", message.UserID, err)
			continue
		}
		AuthorDMChannelID := b.dataBackend.GetTmp("glyph", "dg:"+discordID+"|DM-Channel")
		if AuthorDMChannelID == "" {
			AuthorDMChannel, err := b.dg.UserChannelCreate(discordID)
			if err != nil {
				return
			}
			AuthorDMChannelID = AuthorDMChannel.ID
			b.dataBackend.SetTmp("glyph", "dg:"+discordID+"|DM-Channel", AuthorDMChannelID, time.Hour*24)
		}
		_, err = b.dg.ChannelMessageSend(AuthorDMChannelID, message.Message)
		if err != nil {
			b.logger.Warningf("failed to send discord message to %v on channel %v:", message.UserID, AuthorDMChannelID, err)
			continue
		}
	}
}

// Check if a user has a given role in a given guild
/*func memberHasRole(s *discordgo.Session, guildID string, userID string, roleID string) (bool, error) {
	member, err := s.State.Member(guildID, userID)
	if err != nil {
		if member, err = s.GuildMember(guildID, userID); err != nil {
			return false, err
		}
	}

	// Iterate through the role IDs stored in member.Roles
	// to check permissions
	for _, userRoleID := range member.Roles {
		role, err := s.State.Role(guildID, userRoleID)
		if err != nil {
			return false, err
		}
		if role.ID == roleID {
			return true, nil
		}
	}

	return false, nil
}*/

func (b Bot) getDiscordGetContext() func(userID, channelID, key string) (string, error) {
	return func(userID, channelID, key string) (string, error) {
		return b.dataBackend.GetTmp("glyph:dc:"+channelID+":"+userID, key), nil
	}
}

func (b Bot) getDiscordSetContext() func(userID, channelID, key, value string, ttl time.Duration) error {
	return func(userID, channelID, key, value string, ttl time.Duration) error {
		b.dataBackend.SetTmp("glyph:dc:"+channelID+":"+userID, key, value, ttl)
		return nil
	}
}

func (b Bot) getDiscordSendMessage(dg *discordgo.Session) func(channelID, message string) error {
	return func(channelID, message string) error {
		_, err := dg.ChannelMessageSend(channelID, message)
		if err != nil {
			return err
		}
		return nil
	}
}

func (b Bot) getDiscordGetMention() func(userID string) (string, error) {
	return func(userID string) (string, error) {
		return "<@" + userID + ">", nil
	}
}

func (b Bot) getDiscordSetUserData() func(discordUserID, key string, value string) error {
	return func(discordUserID, key string, value string) error {
		userID, err := b.dataBackend.GetUserIDFromValueOfKey(adapterID+"ID", discordUserID)
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

func (b Bot) getDiscordGetUserData() func(discordUserID, key string) (string, error) {
	return func(discordUserID, key string) (string, error) {
		userID, err := b.dataBackend.GetUserIDFromValueOfKey(adapterID+"ID", discordUserID)
		if err != nil {
			return "", err
		}
		value, err := b.dataBackend.GetUserData(userID, key)
		if err != nil {
			b.logger.Errorf("error setting user data: %v", err)
			return "", err
		}
		return value, err
	}
}

func (b Bot) getDiscordGetQuoteOfTheDay() func(discordUserID string) (glyph.QuoteOfTheDay, error) {
	return func(discordUserID string) (glyph.QuoteOfTheDay, error) {
		userID, err := b.dataBackend.GetUserIDFromValueOfKey(adapterID+"ID", discordUserID)
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

func (b Bot) getDiscordSetQuoteOfTheDay() func(discordUserID string, quoteOfTheDay glyph.QuoteOfTheDay) error {
	return func(discordUserID string, quoteOfTheDay glyph.QuoteOfTheDay) error {
		userID, err := b.dataBackend.GetUserIDFromValueOfKey(adapterID+"ID", discordUserID)
		if err != nil {
			return err
		}
		return b.dataBackend.SetQuoteOfTheDayOfUser(userID, quoteOfTheDay)
	}
}

func (b Bot) getDiscordGetMatrixUserID() func(discordUserID string) (string, error) {
	return func(discordUserID string) (string, error) {
		return b.dataBackend.GetUserIDFromValueOfKey(adapterID+"ID", discordUserID)
	}
}

func (b Bot) getDiscordDeleteUserData() func(discordUserID, key string) error {
	return func(discordUserID, key string) error {
		userID, err := b.dataBackend.GetUserIDFromValueOfKey(adapterID+"ID", discordUserID)
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
