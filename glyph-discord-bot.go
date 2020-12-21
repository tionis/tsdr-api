package main

import (
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/bwmarrin/discordgo"
	_ "github.com/heroku/x/hmetrics/onload" // Heroku advanced go metrics
	"github.com/keybase/go-logging"
	"github.com/tionis/tsdr-api/data"
	"github.com/tionis/tsdr-api/glyph"
)

type glyphDiscordMsgObject struct {
	ChannelID string `form:"channelid" json:"channelid" binding:"required"`
	Message   string `form:"message" json:"message" binding:"required"`
}

// Discord ID of admin
//var discordAdminID string = "259076782408335360"
var discordServerID string = "695330213953011733"

var glyphSend chan glyphDiscordMsgObject

var discordBotMention string

// Needed for onlyonce execution of random source
var onlyOnce sync.Once

// Logger
var glyphDiscordLog = logging.MustGetLogger("glyphDiscord")

var discordGlyphBot *glyph.Bot

// Main and Init
func glyphDiscordBot() {
	glyphSend = make(chan glyphDiscordMsgObject, 2)
	dg, err := discordgo.New("Bot " + os.Getenv("DISCORD_TOKEN"))
	if err != nil {
		glyphDiscordLog.Error("Error creating Discord session,", err)
	}

	// Register the messageCreate func as a callback for MessageCreate events.
	dg.AddHandler(messageCreate)

	// Open a websocket connection to Discord and begin listening.
	err = dg.Open()
	if err != nil {
		glyphDiscordLog.Error("Error opening connection,", err)
		return
	}

	// Set some StartUp Stuff
	/*dgStatus, err := getError("dgStatus")
	  if err != nil {
	      glyphDiscordLog.Warning("Error getting dgStatus from redis: ", err)
	      dgStatus = "/help for help"
	  }*/
	_ = dg.UpdateStatus(0, "/help for help")

	// Init mention string
	discordBotMention = "<@!" + dg.State.User.ID + ">"

	// Init callback functions
	var discordSendMessage = func(channelID, message string) {
		_, err := dg.ChannelMessageSend(channelID, message)
		if err != nil {
			glyphDiscordLog.Errorf("Error sending message to %v: %v", channelID, err)
		}
	}
	var discordSetUserData = func(discordUserID, key string, value interface{}) {
		userID, err := data.GetUserIDFromDiscordID(discordUserID)
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
	var discordGetUserData = func(discordUserID, key string) interface{} {
		userID, err := data.GetUserIDFromDiscordID(discordUserID)
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
	var discordSetContext = func(userID, channelID, key, value string, ttl time.Duration) {
		data.SetTmp("glyph:dc:"+channelID+":"+userID, key, value, ttl)
	}
	var discordGetContext = func(userID, channelID, key string) string {
		return data.GetTmp("glyph:dc:"+channelID+":"+userID, key)
	}
	discordGlyphBot = &glyph.Bot{
		AddQuote:             data.AddQuote,
		GetRandomQuote:       data.GetRandomQuote,
		SetContext:           discordSetContext,
		GetContext:           discordGetContext,
		GetUserData:          discordGetUserData,
		SetUserData:          discordSetUserData,
		SendMessageToChannel: discordSendMessage,
		GetMention:           func(userID string) string { return "<@" + userID + ">" },
		Prefix:               "/",
	}

	go func(dg *discordgo.Session) {
		for {
			sig := <-glyphSend
			dg.ChannelMessageSend(sig.ChannelID, sig.Message)
		}
	}(dg)

	// Wait here until CTRL-C or other term signal is received.
	glyphDiscordLog.Info("Glyph Discord Bot was started.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, syscall.SIGQUIT, syscall.SIGHUP)
	<-sc

	// Cleanly close down the Discord session.
	_ = dg.Close()
}

// This function will be called (due to AddHandler above) every time a new
// message is created on any channel that the authenticated bot has access to.
func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	// Build callback functions

	// Ignore all messages created by the bot itself
	if m.Author.ID == s.State.User.ID {
		return
	}

	// Check if message was sent in an DM
	AuthorDMChannelID := data.GetTmp("glyph", "dg:"+m.Author.ID+"|DM-Channel")
	if AuthorDMChannelID == "" {
		AuthorDMChannel, err := s.UserChannelCreate(m.Author.ID)
		if err != nil {
			return
		}
		AuthorDMChannelID = AuthorDMChannel.ID
		data.SetTmp("glyph", "dg:"+m.Author.ID+"|DM-Channel", AuthorDMChannelID, time.Hour*24)
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

	if strings.HasPrefix(tokens[0], discordGlyphBot.Prefix) {
		message.Content = strings.TrimPrefix(message.Content, "/")
		message.IsCommand = true
	} else if tokens[0] == discordBotMention {
		message.Content = strings.TrimPrefix(message.Content, discordBotMention)
		message.IsCommand = true
	}

	message.Content = strings.TrimLeft(message.Content, "\t \r \n \v \f ")

	// Pass message object to glyph bot logic
	go discordGlyphBot.HandleAll(message)
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
