package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/Tnze/go-mc/bot"
	"github.com/bwmarrin/discordgo"
	_ "github.com/heroku/x/hmetrics/onload"
)

// Global Variables
var mcStopping bool
var mcRunning bool
var mainChannelID string
var msgDiscordMC chan string

// Main and Init
func alphaDiscordBot() {
	mainChannelID = os.Getenv("MC_CHANNEL_ID")
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
	mcRunning, mcStopping = false, false
	go pingMC()

	// Set some StartUp Stuff
	dg.UpdateStatus(0, "Manager of Tasadar Stuff")

	// Define input channel
	go func(s *discordgo.Session) {
		msgDiscordMC = make(chan string)
		for {
			toSend := <-msgDiscordMC
			s.ChannelMessageSend(mainChannelID, toSend)
		}
	}(dg)

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
		if mcRunning {
			s.ChannelMessageSend(m.ChannelID, "Server already running!")
		} else {
			s.ChannelMessageSend(m.ChannelID, "Starting MC-Server...")
			success := mcStart()
			if !success {
				s.ChannelMessageSend(m.ChannelID, "An Error occurred!")
				log.Println("[AlphaDiscordBot] Error starting mc-server")
			}
		}
		pingInMinutes(2)
	case "/mc stop":
		go mcShutdownDiscord(s, m, 7)
	case "/mc cancel":
		if mcStopping {
			mcStopping = false
			s.ChannelMessageSend(m.ChannelID, "Server shutdown stopped!")
			client, err := newClient(os.Getenv("RCON_ADDRESS"), 25575, os.Getenv("RCON_PASS"))
			_, err = client.sendCommand("tellraw @a [{\"text\":\"Server shutdown was aborted!\",\"bold\":false,\"italic\":true,\"underlined\":false,\"striketrough\":false,\"obfuscated\":false,\"color\":\"gray\"}]")
			if err != nil {
				log.Println("[AlphaDiscordBot] RCON server command connection failed")
			}
		} else if mcRunning {
			s.ChannelMessageSend(m.ChannelID, "There is currently no Server Shutdown scheduled!")
		} else {
			s.ChannelMessageSend(m.ChannelID, "Server is currently not running!")
		}
	case "/mc status":
		pingMC()
		client, err := newClient(os.Getenv("RCON_ADDRESS"), 25575, os.Getenv("RCON_PASS"))
		if err != nil {
			mcRunning = false
			s.ChannelMessage(m.ChannelID, "An Error occurred, please contact the administrator!")
			log.Println("[AlphaDiscordBot] Error while creating rcon client: ", err)
			return
		}
		if !mcRunning {
			s.ChannelMessageSend(m.ChannelID, "Warning! - Server currently not running!")
			return
		}
		resp, err := client.sendCommand("execute if entity @a")
		if err != nil {
			log.Println("[AlphaDiscordBot] RCON server command connection failed: ", err)
			s.ChannelMessageSend(m.ChannelID, "An error occurred while trying to get the status, please contact the administrator.")
			return
		}
		var creeperCountString string
		res, err := client.sendCommand("execute if entity @e[type=creeper]")
		if err != nil {
			log.Println("[AlphaDiscordBot] RCON server command connection failed: ", err)
			s.ChannelMessageSend(m.ChannelID, "An error occurred while trying to get the status, please contact the administrator.")
			return
		}
		if strings.Contains(res, "Test failed") {
			creeperCountString = "0"
		} else {
			creeperCountString = strings.TrimPrefix(res, "Test passed, count: ")
		}
		var playerCountString string
		if strings.Contains(resp, "Test failed") {
			playerCountString = "0"
		} else {
			playerCountString = strings.TrimPrefix(resp, "Test passed, count: ")
		}
		if err != nil {
			log.Println("[AlphaDiscordBot] Player counting failed: ", err)
			s.ChannelMessageSend(m.ChannelID, "An error occurred while counting players, please contact the administrator.")
			log.Println(err)
		} else {
			s.ChannelMessageSend(m.ChannelID, "Server currently online\nAt the moment there are "+playerCountString+" players on the server and there are "+creeperCountString+" Creepers loaded.")
		}
	case "/mc help":
		s.ChannelMessageSend(m.ChannelID, "Available Commands:\n/mc start - Starts the Minecraft Server\n/mc status - Get the current status of the Minecraft Server\n/mc stop - Stop the Minecraft Server")
	}
}

func mcShutdownDiscord(s *discordgo.Session, m *discordgo.MessageCreate, minutes int) {
	minutesString := strconv.Itoa(minutes)
	client, err := newClient(os.Getenv("RCON_ADDRESS"), 25575, os.Getenv("RCON_PASS"))
	if !mcRunning {
		s.ChannelMessageSend(m.ChannelID, "The Server is currently not running!")
		return
	}
	mcStopping = true
	_, err = client.sendCommand("tellraw @a [{\"text\":\"Server shutdown commencing in \",\"bold\":false,\"italic\":true,\"underlined\":false,\"striketrough\":false,\"obfuscated\":false,\"color\":\"gray\"},{\"text\":\"" + minutesString + " Minutes!\",\"bold\":false,\"italic\":true,\"underlined\":false,\"striketrough\":false,\"obfuscated\":false,\"color\":\"dark_aqua\"}]")
	if err != nil {
		log.Println("[AlphaDiscordBot] RCON server command connection failed")
	}
	_, err = client.sendCommand("tellraw @a [{\"text\":\"Type /mc cancel in the Discord Chat to cancel the shutdown! \",\"bold\":false,\"italic\":true,\"underlined\":false,\"striketrough\":false,\"obfuscated\":false,\"color\":\"gray\"}]")
	if err != nil {
		log.Println("[AlphaDiscordBot] RCON server command connection failed")
	}
	s.ChannelMessageSend(m.ChannelID, "If nobody says /mc cancel in the next "+minutesString+" Minutes I will shut down the server!")
	if m.ChannelID != mainChannelID {
		s.ChannelMessageSend(mainChannelID, "If nobody says /mc cancel in the next "+minutesString+" Minutes I will shut down the server!")
	}
	time.Sleep(time.Duration(minutes) * time.Minute)
	if mcStopping {
		s.ChannelMessageSend(m.ChannelID, "Shutting down Server...")
		if err != nil {
			log.Println("[AlphaDiscordBot] RCON server connection failed")
		}
		_, err = client.sendCommand("title @a title {\"text\":\"Warning!\",\"bold\":false,\"italic\":false,\"underlined\":false,\"striketrough\":false,\"obfuscated\":false,\"color\":\"red\"}")
		if err != nil {
			log.Println("[AlphaDiscordBot] RCON server command connection failed")
		}
		_, err = client.sendCommand("tellraw @a [{\"text\":\"Server shutdown commencing in \",\"bold\":false,\"italic\":true,\"underlined\":false,\"striketrough\":false,\"obfuscated\":false,\"color\":\"gray\"},{\"text\":\"10 Seconds!\",\"bold\":false,\"italic\":true,\"underlined\":false,\"striketrough\":false,\"obfuscated\":false,\"color\":\"dark_aqua\"}]")
		if err != nil {
			log.Println("[AlphaDiscordBot] RCON server command connection failed")
		}
		time.Sleep(3 * time.Second)
		for i := 10; i >= 0; i-- {
			time.Sleep(1 * time.Second)
			_, err = client.sendCommand("title @a title {\"text\":\"" + strconv.Itoa(i) + "\",\"bold\":false,\"italic\":false,\"underlined\":false,\"striketrough\":false,\"obfuscated\":false,\"color\":\"red\"}")
			if err != nil {
				log.Println("[AlphaDiscordBot] RCON server command connection failed")
			}
		}
		_, err = client.sendCommand("stop")
		if err != nil {
			log.Println("[AlphaDiscordBot] RCON server command connection failed")
		}
	}
}

func mcStart() bool {
	client := &http.Client{}
	req, _ := http.NewRequest("GET", "https://mcapi.tasadar.net/mc/start", nil)
	req.Header.Set("TASADAR_SECRET", "JFyMdGUgx3Re2r2VefLYFJeGNosscB98")
	res, err := client.Do(req)
	if res.StatusCode == 200 {
		mcRunning = true
		return true
	}
	log.Println(res, err)
	return false
}

func pingInMinutes(minutes int) {
	time.Sleep(time.Duration(minutes) * time.Minute)
	pingMC()
}

func pingMC() {
	// To be edited with a true server ping - finished (more or less) - can still be improved!
	_, _, err := bot.PingAndList("mc.tasadar.net", 25565)
	mcRunning = err == nil
}
