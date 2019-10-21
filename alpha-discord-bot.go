package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/bwmarrin/discordgo"
	_ "github.com/heroku/x/hmetrics/onload"
)

// Global Variables
var mcIP string
var mcOnline bool

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

	// Set some StartUp Stuff
	dg.UpdateStatus(0, "Manager of Tasadar Stuff")

	// Wait here until CTRL-C or other term signal is received.
	fmt.Println("[AlphaDiscordBot] Bot is now running.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc

	dg.UpdateStatus(0, "Manager of Tasadar Stuff")

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
		s.ChannelMessageSend(m.ChannelID, "If nobody says /mc cancel in the next 7 Minutes I will shut down the server!")
		//wait 7 Minutes, if no signal received
		// user rcon to trigger stop, which will trigger automatic stop
	case "/mc status":
		s.ChannelMessageSend(m.ChannelID, "I cant do that yet, please wait until the developer implements this!")
	case "/mc help":
		s.ChannelMessageSend(m.ChannelID, "Available Commands:\n/mc start - Starts the Minecraft Server\n/mc status - Get the current status of the Minecraft Server\n/mc stop - Stop the Minecraft Server")
	}
}

func mcStart() {
	// Make call to hetzner api to create vm from snapshot or file
	// wait and check regularily for global variable
	// if mcOnline gets true
	// check mcIP and set it on digitalocean dns with a ttl of 60
	// wait till server is started
	// init online checker in goroutine
}
