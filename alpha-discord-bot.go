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
	case "/food":
		s.ChannelMessageSend(m.ChannelID, foodtoday())
	case "/food tomorrow":
		s.ChannelMessageSend(m.ChannelID, foodtomorrow())
	case "/ping":
		s.ChannelMessageSend(m.ChannelID, "Pong!")
	}
}
