package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/bwmarrin/discordgo"
	_ "github.com/heroku/x/hmetrics/onload"
	"github.com/seeruk/minecraft-rcon/rcon"
)

// Global Variables
var mcStopping bool
var mcRunning bool

// Main and Init
func alphaDiscordBot() {
	dg, err := discordgo.New("Bot " + os.Getenv("AlphaDiscordBot"))
	if err != nil {
		log.Println("[AlphaDiscordBot] Error creating Discord session,", err)
	}
	// Register the messageCreate func as a callback for MessageCreate events.
	dg.AddHandler(messageCreate)

	// Open a websocket connection to Discord and begin listening.
	err = dg.Open()
	if err != nil {
		fmt.Println("[AlphaDiscordBot] Error opening connection,", err)
		return
	}

	// Init Server Stuff
	// Get Server State
	mcRunning, mcStopping = true, false

	// Set some StartUp Stuff
	dg.UpdateStatus(0, "Manager of Tasadar Stuff")

	// Wait here until CTRL-C or other term signal is received.
	fmt.Println("[AlphaDiscordBot] Bot is now running.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc

	// Cleanly close down the Discord session.
	dg.Close()
}

// This function will be called (due to AddHandler above) every time a new
// message is created on any channel that the autenticated bot has access to.
func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {

	// Ignore all messages created by the bot itself
	// This isn't required in this specific example but it's a good practice.
	if m.Author.ID == s.State.User.ID {
		return
	}

	switch m.Content {
	case "/help":
		s.ChannelMessageSend(m.ChannelID, "Available Command Categories:\n - Minecraft Server - /mc help\n - Uni Passau - /unip help\n - General Tasadar Network - /tn help")
	case "/unip help":
		s.ChannelMessageSend(m.ChannelID, "Available Commands:\n/food - Food for today\n/food tomorrow - Food for tomorrow")
	case "/tn help":
		s.ChannelMessageSend(m.ChannelID, "Not implemented yet!")
	case "/food":
		s.ChannelMessageSend(m.ChannelID, foodtoday())
	case "/food tomorrow":
		s.ChannelMessageSend(m.ChannelID, foodtomorrow())
	case "/ping":
		s.ChannelMessageSend(m.ChannelID, "Pong!")
	case "/mc start":
		s.ChannelMessageSend(m.ChannelID, "I cant do that yet, please wait until the developer implements this!")
		//mcStart()
	case "/mc stop":
		go mcShutdownDiscord(s, m)
	case "/mc cancel":
		if mcStopping {
			mcStopping = false
			s.ChannelMessageSend(m.ChannelID, "Server shutdown stopped!")
			client, err := rcon.NewClient(os.Getenv("RCON_ADDRESS"), 25575, os.Getenv("RCON_PASS"))
			_, err = client.SendCommand("tellraw @a [{\"text\":\"Server shutdown was aborted!\",\"bold\":false,\"italic\":true,\"underlined\":false,\"striketrough\":false,\"obfuscated\":false,\"color\":\"gray\"}]")
			if err != nil {
				log.Println("[AlphaDiscordBot] RCON server command connection failed")
			}
		} else if mcRunning {
			s.ChannelMessageSend(m.ChannelID, "There is currently no Server Shutdown scheduled!")
		} else {
			s.ChannelMessageSend(m.ChannelID, "Server is currently not running!")
		}
	case "/mc status":
		// Send query
		// Answer
		s.ChannelMessageSend(m.ChannelID, "I cant do that yet, please wait until the developer implements this!")
	case "/mc help":
		s.ChannelMessageSend(m.ChannelID, "Available Commands:\n/mc start - Starts the Minecraft Server\n/mc status - Get the current status of the Minecraft Server\n/mc stop - Stop the Minecraft Server")
	}
}

func mcShutdownDiscord(s *discordgo.Session, m *discordgo.MessageCreate) {
	client, err := rcon.NewClient(os.Getenv("RCON_ADDRESS"), 25575, os.Getenv("RCON_PASS"))
	if !mcRunning {
		s.ChannelMessageSend(m.ChannelID, "The Server is currently not running!")
		return
	}
	mcStopping = true
	_, err = client.SendCommand("tellraw @a [{\"text\":\"Server shutdown commencing in \",\"bold\":false,\"italic\":true,\"underlined\":false,\"striketrough\":false,\"obfuscated\":false,\"color\":\"gray\"},{\"text\":\"7 Minutes!\",\"bold\":false,\"italic\":true,\"underlined\":false,\"striketrough\":false,\"obfuscated\":false,\"color\":\"dark_aqua\"}]")
	if err != nil {
		log.Println("[AlphaDiscordBot] RCON server command connection failed")
	}
	_, err = client.SendCommand("tellraw @a [{\"text\":\"Type /mc cancel in the Discord Chat to cancel the shutdown! \",\"bold\":false,\"italic\":true,\"underlined\":false,\"striketrough\":false,\"obfuscated\":false,\"color\":\"gray\"}]")
	if err != nil {
		log.Println("[AlphaDiscordBot] RCON server command connection failed")
	}
	s.ChannelMessageSend(m.ChannelID, "If nobody says /mc cancel in the next 7 Minutes I will shut down the server!")
	time.Sleep(7 * time.Minute)
	if mcStopping {
		//Send rcon command
		//if no error send message
		s.ChannelMessageSend(m.ChannelID, "Shutting down Server...")
		if err != nil {
			log.Println("[AlphaDiscordBot] RCON server connection failed")
		}
		_, err = client.SendCommand("title @a title {\"text\":\"Warning!\",\"bold\":false,\"italic\":false,\"underlined\":false,\"striketrough\":false,\"obfuscated\":false,\"color\":\"red\"}")
		if err != nil {
			log.Println("[AlphaDiscordBot] RCON server command connection failed")
		}
		_, err = client.SendCommand("tellraw @a [{\"text\":\"Server shutdown commencing in \",\"bold\":false,\"italic\":true,\"underlined\":false,\"striketrough\":false,\"obfuscated\":false,\"color\":\"gray\"},{\"text\":\"10 Seconds!\",\"bold\":false,\"italic\":true,\"underlined\":false,\"striketrough\":false,\"obfuscated\":false,\"color\":\"dark_aqua\"}]")
		if err != nil {
			log.Println("[AlphaDiscordBot] RCON server command connection failed")
		}
		time.Sleep(3 * time.Second)
		for i := 10; i >= 0; i-- {
			time.Sleep(1 * time.Second)
			_, err = client.SendCommand("title @a title {\"text\":\"" + strconv.Itoa(i) + "\",\"bold\":false,\"italic\":false,\"underlined\":false,\"striketrough\":false,\"obfuscated\":false,\"color\":\"red\"}")
			if err != nil {
				log.Println("[AlphaDiscordBot] RCON server command connection failed")
			}
		}
		_, err = client.SendCommand("stop")
		if err != nil {
			log.Println("[AlphaDiscordBot] RCON server command connection failed")
		}
	}
}
