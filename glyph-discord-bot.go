package main

import (
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/bwmarrin/discordgo"
	_ "github.com/heroku/x/hmetrics/onload"
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

var discordBotMention1 string
var discordBotMention2 string

// Needed for onlyonce execution of random source
var onlyOnce sync.Once

// Logger
var glyphDiscordLog = logging.MustGetLogger("glyphDiscord")

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

	// Init both mention strings

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
	var discordSend = func(channelID, message string) {
		_, err := s.ChannelMessageSend(channelID, message)
		if err != nil {
			glyphDiscordLog.Errorf("Error sending message to %v: %v", channelID, err)
		}
	}
	var discordGetMention = func(userID string) string {
		return "<@" + userID + ">"
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
	var discordSetContext = func(userID, key, value string, ttl time.Duration) {
		data.SetTmp("glyph:dc:ctx", key, value, ttl)
	}
	var discordGetContext = func(userID, key string) string {
		return data.GetTmp("glyph:dc:ctx", key)
	}

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
		Content:              m.Content,
		AuthorID:             m.Author.ID,
		ChannelID:            m.ChannelID,
		SendMessageToChannel: discordSend,
		IsDM:                 m.ChannelID == AuthorDMChannelID,
		SupportsMarkdown:     true,
		GetMention:           discordGetMention,
		SetUserData:          discordSetUserData,
		GetUserData:          discordGetUserData,
		GetContext:           discordGetContext,
		SetContext:           discordSetContext,
	}

	// Split message into chunks
	tokens := strings.Split(message.Content, " ")

	if strings.HasPrefix(tokens[0], "/") {
		message.Content = strings.TrimPrefix(message.Content, "/")
	} else if tokens[0] == discordBotMention1 {
		message.Content = strings.TrimPrefix(message.Content, discordBotMention1+" ")
	} else if tokens[0] == discordBotMention2 {
		message.Content = strings.TrimPrefix(message.Content, discordBotMention2+" ")
	}

	// Pass message object to glyph bot logic
	go glyph.HandleAll(message)
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
