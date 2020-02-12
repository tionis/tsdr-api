package main

import (
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
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
var lastPlayerOnline time.Time
var discordAdminID string
var rconPassword string
var onlyOnce sync.Once
var dice = []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}

// Some Constants
const lastPlayerOnlineLayout = "2006-01-02T15:04:05.000Z"
const rconAddress = "mc.tasadar.net"

// Main and Init
func alphaDiscordBot() {
	mainChannelID = "574959338754670602"
	discordAdminID = "259076782408335360"
	rconPassword = os.Getenv("RCON_PASS")
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

	//Init Variables from redis
	lastPlayerOnlineString, err := redclient.Get("mc|lastPlayerOnline").Result()
	if err != nil {
		log.Println("Error reading mc|lastPlayerOnline from Redis: ", err)
		lastPlayerOnline = time.Now()
	}
	lastPlayerOnline, err = time.Parse(lastPlayerOnlineLayout, lastPlayerOnlineString)
	if err != nil {
		log.Println("Error transforming mc|lastPlayerOnline to time object: ", err)
		lastPlayerOnline = time.Now()
	}
	// Init mcRunning and mcStopping
	mcRunningString, err := redclient.Get("mc|IsRunning").Result()
	if err != nil {
		log.Println("[AlphaDiscordBot] Error getting Redis value for mc|IsRunning", err)
	} else if mcRunningString == "true" {
		mcRunning = true
	} else if mcRunningString == "false" {
		mcRunning = false
	} else {
		mcRunning = false
		log.Println("[AlphaDiscordBot] Error converting Redis value for mc|IsRunning, expected true or false but got " + mcRunningString)
	}
	mcStopping = false // Redis save not necessary - edge cases not important enough

	// Get Server State
	go pingMC()

	// Set some StartUp Stuff
	dgStatus, err := redclient.Get("dg|status").Result()
	if err != nil {
		log.Println("[AlphaDiscordBot] Error getting dgStatus from redis: ", err)
		dgStatus = "Mass Effect 5"
	}
	_ = dg.UpdateStatus(0, dgStatus)

	// Define input channel
	go func(s *discordgo.Session) {
		msgDiscordMC = make(chan string)
		for {
			toSend := <-msgDiscordMC
			_, _ = s.ChannelMessageSend(mainChannelID, toSend)
		}
	}(dg)

	// Wait here until CTRL-C or other term signal is received.
	fmt.Println("[AlphaDiscordBot] Bot is now running.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc

	// Cleanly close down the Discord session.
	_ = dg.Close()
}

// This function will be called (due to AddHandler above) every time a new
// message is created on any channel that the autenticated bot has access to.
func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {

	// Ignore all messages created by the bot itself
	// This isn't required in this specific example but it's a good practice.
	if m.Author.ID == s.State.User.ID {
		return
	}

	inputString := strings.Split(m.Content, " ")
	switch inputString[0] {
	case "/roll":
		rollHelper(s, m)
	case "/r":
		rollHelper(s, m)
	case "/help":
		log.Println("[AlphaDiscordBot] New Command by " + m.Author.Username + "\n[AlphaDiscordBot] " + m.Content)
		_, _ = s.ChannelMessageSend(m.ChannelID, "Available Command Categories:\n - Minecraft Server - /mc help\n - Uni Passau - /unip help\n - General Tasadar Network - /tn help")
	case "/unip":
		log.Println("[AlphaDiscordBot] New Command by " + m.Author.Username + "\n[AlphaDiscordBot] " + m.Content)
		_, _ = s.ChannelMessageSend(m.ChannelID, "Available Commands:\n/food - Food for today\n/food tomorrow - Food for tomorrow")
	case "/tn":
		log.Println("[AlphaDiscordBot] New Command by " + m.Author.Username + "\n[AlphaDiscordBot] " + m.Content)
		_, _ = s.ChannelMessageSend(m.ChannelID, "Available Commands:\n/linkAccount username password - Link your Tasadar Account to your Discord Account (use only in DM!)")
	case "/food":
		log.Println("[AlphaDiscordBot] New Command by " + m.Author.Username + "\n[AlphaDiscordBot] " + m.Content)
		_, _ = s.ChannelMessageSend(m.ChannelID, foodtoday())
	case "/food tomorrow":
		log.Println("[AlphaDiscordBot] New Command by " + m.Author.Username + "\n[AlphaDiscordBot] " + m.Content)
		_, _ = s.ChannelMessageSend(m.ChannelID, foodtomorrow())
	case "/ping":
		log.Println("[AlphaDiscordBot] New Command by " + m.Author.Username + "\n[AlphaDiscordBot] " + m.Content)
		_, _ = s.ChannelMessageSend(m.ChannelID, "Pong!")
	case "/mc":
		switch inputString[1] {
		case "start":
			log.Println("[AlphaDiscordBot] New Command by " + m.Author.Username + "\n[AlphaDiscordBot] " + m.Content)
			if mcRunning {
				_, _ = s.ChannelMessageSend(m.ChannelID, "Server already running!")
			} else {
				_, _ = s.ChannelMessageSend(m.ChannelID, "Starting MC-Server...")
				success := mcStart()
				if !success {
					_, _ = s.ChannelMessageSend(m.ChannelID, "An Error occurred!")
					log.Println("[AlphaDiscordBot] Error starting mc-server")
				}
			}
			pingInMinutes(2)
		case "stop":
			log.Println("[AlphaDiscordBot] New Command by " + m.Author.Username + "\n[AlphaDiscordBot] " + m.Content)
			go mcShutdownDiscord(s, m, 7)
		case "cancel":
			log.Println("[AlphaDiscordBot] New Command by " + m.Author.Username + "\n[AlphaDiscordBot] " + m.Content)
			if mcStopping {
				mcStopping = false
				_, _ = s.ChannelMessageSend(m.ChannelID, "Server shutdown stopped!")
				client, err := newClient(rconAddress, 25575, rconPassword)
				if err != nil {
					_, _ = s.ChannelMessageSend(m.ChannelID, "Internal Error, please ask an admin to check the logs.")
					log.Println("[AlphaDiscordBot] Error while trying to create RCON-Client-Object")
					return
				}
				_, err = client.sendCommand("tellraw @a [{\"text\":\"Server shutdown was aborted!\",\"bold\":false,\"italic\":true,\"underlined\":false,\"striketrough\":false,\"obfuscated\":false,\"color\":\"gray\"}]")
				if err != nil {
					log.Println("[AlphaDiscordBot] RCON server command connection failed: ", err)
					return
				}
			} else if mcRunning {
				_, _ = s.ChannelMessageSend(m.ChannelID, "There is currently no Server Shutdown scheduled!")
			} else {
				_, _ = s.ChannelMessageSend(m.ChannelID, "Server is currently not running!")
			}
		case "status":
			log.Println("[AlphaDiscordBot] New Command by " + m.Author.Username + "\n[AlphaDiscordBot] " + m.Content)
			pingMC()
			if mcRunning {
				client, err := newClient(rconAddress, 25575, rconPassword)
				if err != nil {
					_, _ = s.ChannelMessage(m.ChannelID, "An Error occurred, please contact the administrator!")
					log.Println("[AlphaDiscordBot] Error while creating rcon client: ", err)
					return
				}
				if !mcRunning {
					_, _ = s.ChannelMessageSend(m.ChannelID, "Warning! - Server currently not running!")
					return
				}
				resp, err := client.sendCommand("execute if entity @a")
				if err != nil {
					log.Println("[AlphaDiscordBot] RCON server command connection failed: : ", err)
					_, _ = s.ChannelMessageSend(m.ChannelID, "An error occurred while trying to get the status, please contact the administrator.")
					return
				}
				var creeperCountString string
				res, err := client.sendCommand("execute if entity @e[type=creeper]")
				if err != nil {
					log.Println("[AlphaDiscordBot] RCON server command connection failed: : ", err)
					_, _ = s.ChannelMessageSend(m.ChannelID, "An error occurred while trying to get the status, please contact the administrator.")
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
				_, _ = s.ChannelMessageSend(m.ChannelID, "Server currently online\nAt the moment there are "+playerCountString+" players on the server and there are "+creeperCountString+" Creepers loaded.")
			} else {
				_, _ = s.ChannelMessageSend(m.ChannelID, "Server currently offline!\nTo start it use /mc start")
			}

		default:
			log.Println("[AlphaDiscordBot] New Command by " + m.Author.Username + "\n[AlphaDiscordBot] " + m.Content)
			_, _ = s.ChannelMessageSend(m.ChannelID, "Available Commands:\n/mc start - Starts the Minecraft Server\n/mc status - Get the current status of the Minecraft Server\n/mc stop - Stop the Minecraft Server")
		}
	case "/id":
		_, _ = s.ChannelMessageSend(m.ChannelID, "Your ID is:\n"+m.Author.ID)
		//default:
		//log.Println("[AlphaDiscordBot] Logged Unknown Command by " + m.Author.Username + "\n[AlphaDiscordBot] " + m.Content)
	case "/updateStatus":
		if m.Author.ID == discordAdminID {
			newStatus := strings.TrimPrefix(m.Content, "/updateStatus ")
			err := set("dg|status", newStatus)
			if err != nil {
				log.Println("Error setting dg|status on Redis: ", err)
				_, _ = s.ChannelMessageSend(m.ChannelID, "Error sending Status to Safe!")
			}
			_ = s.UpdateStatus(0, newStatus)
			_, _ = s.ChannelMessageSend(m.ChannelID, "New Status set!")
		} else {
			_, _ = s.ChannelMessageSend(m.ChannelID, "You are not authorized to execute this command!\nThis incident will be reported.\nhttps://imgs.xkcd.com/comics/incident.png")
		}
	case "/linkAccount":
		tokens := strings.Split(strings.TrimPrefix(m.Content, "/linkAccount "), " ")
		if len(tokens) < 2 {
			_, _ = s.ChannelMessageSend(m.ChannelID, "I couldn't parse your Message, please check your Syntax!")
		} else {
			if authUser(tokens[0], tokens[1]) {
				err := set("dg|"+m.Author.ID+"|username", tokens[0])
				if err != nil {
					_, _ = s.ChannelMessageSend(m.ChannelID, "Error saving your link")
				} else {
					_, _ = s.ChannelMessageSend(m.ChannelID, "I established your new Link successfully")
				}
			} else {
				_, _ = s.ChannelMessageSend(m.ChannelID, "Authentication failed, please double-check your password and username!")
			}
		}
		_ = s.ChannelMessageDelete(m.ChannelID, m.ID)
	}
}

func mcShutdownDiscord(s *discordgo.Session, m *discordgo.MessageCreate, minutes int) {
	minutesString := strconv.Itoa(minutes)
	if !mcRunning {
		_, _ = s.ChannelMessageSend(m.ChannelID, "The Server is currently not running!\nYou must start the Server to stop it!")
		return
	}
	client, err := newClient(rconAddress, 25575, rconPassword)
	mcStopping = true
	if err != nil {
		_, _ = s.ChannelMessageSend(m.ChannelID, "Internal Error, please ask an admin to check the logs.")
		log.Println("[AlphaDiscordBot] Error while trying to create RCON-Client-Object")
		return
	}
	_, err = client.sendCommand("tellraw @a [{\"text\":\"Server shutdown commencing in \",\"bold\":false,\"italic\":true,\"underlined\":false,\"striketrough\":false,\"obfuscated\":false,\"color\":\"gray\"},{\"text\":\"" + minutesString + " Minutes!\",\"bold\":false,\"italic\":true,\"underlined\":false,\"striketrough\":false,\"obfuscated\":false,\"color\":\"dark_aqua\"}]")
	if err != nil {
		log.Println("[AlphaDiscordBot] RCON server command connection failed: ", err)
		return
	}
	_, err = client.sendCommand("tellraw @a [{\"text\":\"Type /mc cancel in the Discord Chat to cancel the shutdown! \",\"bold\":false,\"italic\":true,\"underlined\":false,\"striketrough\":false,\"obfuscated\":false,\"color\":\"gray\"}]")
	if err != nil {
		log.Println("[AlphaDiscordBot] RCON server command connection failed: ", err)
	}
	_, _ = s.ChannelMessageSend(m.ChannelID, "If nobody says /mc cancel in the next "+minutesString+" Minutes I will shut down the server!")
	if m.ChannelID != mainChannelID {
		_, _ = s.ChannelMessageSend(mainChannelID, "If nobody says /mc cancel in the next "+minutesString+" Minutes I will shut down the server!")
	}
	time.Sleep(time.Duration(minutes) * time.Minute)
	if mcStopping {
		_, _ = s.ChannelMessageSend(m.ChannelID, "Shutting down Server...")
		if err != nil {
			log.Println("[AlphaDiscordBot] RCON server connection failed", err)
		}
		_, err = client.sendCommand("title @a title {\"text\":\"Warning!\",\"bold\":false,\"italic\":false,\"underlined\":false,\"striketrough\":false,\"obfuscated\":false,\"color\":\"red\"}")
		if err != nil {
			log.Println("[AlphaDiscordBot] RCON server command connection failed: ", err)
		}
		_, err = client.sendCommand("tellraw @a [{\"text\":\"Server shutdown commencing in \",\"bold\":false,\"italic\":true,\"underlined\":false,\"striketrough\":false,\"obfuscated\":false,\"color\":\"gray\"},{\"text\":\"10 Seconds!\",\"bold\":false,\"italic\":true,\"underlined\":false,\"striketrough\":false,\"obfuscated\":false,\"color\":\"dark_aqua\"}]")
		if err != nil {
			log.Println("[AlphaDiscordBot] RCON server command connection failed: ", err)
		}
		time.Sleep(3 * time.Second)
		for i := 10; i >= 0; i-- {
			time.Sleep(1 * time.Second)
			_, err = client.sendCommand("title @a title {\"text\":\"" + strconv.Itoa(i) + "\",\"bold\":false,\"italic\":false,\"underlined\":false,\"striketrough\":false,\"obfuscated\":false,\"color\":\"red\"}")
			if err != nil {
				log.Println("[AlphaDiscordBot] RCON server command connection failed: ", err)
			}
		}
		_, err = client.sendCommand("stop")
		if err != nil {
			log.Println("[AlphaDiscordBot] RCON server command connection failed: ", err)
		}
	}
}

func mcStart() bool {
	client := &http.Client{}
	req, _ := http.NewRequest("GET", "https://mcapi.tasadar.net/mc/start", nil)
	req.Header.Set("TASADAR_SECRET", "JFyMdGUgx3Re2r2VefLYFJeGNosscB98")
	res, err := client.Do(req)
	if res == nil {
		log.Println("Error connecting to mcAPI")
		return false
	}
	if res.StatusCode == 200 {
		mcRunning = true
		var mcRunningString string
		if mcRunning {
			mcRunningString = "true"
		} else {
			mcRunningString = "false"
		}
		_ = set("mc|IsRunning", mcRunningString)
		if err != nil {
			log.Println("[AlphaDiscordBot] Error setting mc|IsRunning on Redis: ", err)
		}
		lastPlayerOnline = time.Now()
		_ = set("mc|lastPlayerOnline", lastPlayerOnline.Format(lastPlayerOnlineLayout))
		if err != nil {
			log.Println("[AlphaDiscordBot] Error setting mc|lastPlayerOnline on Redis: ", err)
		}
		return true
	}
	log.Println(res, err)
	return false
}

func pingInMinutes(minutes int) {
	time.Sleep(time.Duration(minutes) * time.Minute)
	pingMC()
}

func updateMC() {
	if mcRunning && !mcStopping {
		client, err := newClient(rconAddress, 25575, rconPassword)
		if err != nil {
			mcRunning = false
			var mcRunningString string
			if mcRunning {
				mcRunningString = "true"
			} else {
				mcRunningString = "false"
			}
			err = set("mc|IsRunning", mcRunningString)
			if err != nil {
				log.Println("Error setting mc|IsRunning on Redis: ", err)
			}
			log.Println("[AlphaDiscordBot] Error while creating rcon client: ", err)
			return
		}
		resp, err := client.sendCommand("execute if entity @a")
		if err != nil {
			log.Println("[AlphaDiscordBot] RCON server command connection failed: ", err)
			return
		}
		var playerCountString string
		if strings.Contains(resp, "Test failed") {
			playerCountString = "0"
		} else {
			playerCountString = strings.TrimPrefix(resp, "Test passed, count: ")
		}
		playerCount, err := strconv.Atoi(playerCountString)
		if err != nil {
			log.Println("[AlphaDiscordBot] Error converting PlayerCountString to int: ", err)
		}
		if playerCount > 0 {
			lastPlayerOnline = time.Now()
			_ = set("mc|lastPlayerOnline", lastPlayerOnline.Format(lastPlayerOnlineLayout))
			if err != nil {
				log.Println("[AlphaDiscordBot] Error setting mc|lastPlayerOnline on Redis: ", err)
			}
		} else {
			if time.Now().Sub(lastPlayerOnline).Minutes() > 30 {
				lastPlayerOnline = time.Now()
				_ = set("mc|lastPlayerOnline", lastPlayerOnline.Format(lastPlayerOnlineLayout))
				if err != nil {
					log.Println("[AlphaDiscordBot] Error setting mc|lastPlayerOnline on Redis: ", err)
				}
				mcStopPlayerOffline()
			}
		}
	}
}

func mcStopPlayerOffline() {
	client, err := newClient(rconAddress, 25575, rconPassword)
	if err != nil {
		log.Println("[AlphaDiscordBot] RCON server command connection failed: ", err)
		return
	}
	mcStopping = true
	_, err = client.sendCommand("tellraw @a [{\"text\":\"Server shutdown commencing in \",\"bold\":false,\"italic\":true,\"underlined\":false,\"striketrough\":false,\"obfuscated\":false,\"color\":\"gray\"},{\"text\":\" 5 Minutes!\",\"bold\":false,\"italic\":true,\"underlined\":false,\"striketrough\":false,\"obfuscated\":false,\"color\":\"dark_aqua\"}]")
	if err != nil {
		log.Println("[AlphaDiscordBot] RCON server command connection failed: ", err)
		return
	}
	_, err = client.sendCommand("tellraw @a [{\"text\":\"Type /mc cancel in the Discord Chat to cancel the shutdown! \",\"bold\":false,\"italic\":true,\"underlined\":false,\"striketrough\":false,\"obfuscated\":false,\"color\":\"gray\"}]")
	if err != nil {
		log.Println("[AlphaDiscordBot] RCON server command connection failed: ", err)
		return
	}
	msgDiscordMC <- "There were no players on the Server for quite some time.\nIf nobody says /mc cancel in the next 5 Minutes I will shut down the server!"
	time.Sleep(5 * time.Minute)
	if mcStopping {
		err = client.reconnect()
		msgDiscordMC <- "Shutting down Server..."
		if err != nil {
			log.Println("[AlphaDiscordBot] RCON server connection failed", err)
		}
		_, err = client.sendCommand("title @a title {\"text\":\"Warning!\",\"bold\":false,\"italic\":false,\"underlined\":false,\"striketrough\":false,\"obfuscated\":false,\"color\":\"red\"}")
		if err != nil {
			log.Println("[AlphaDiscordBot] RCON server command connection failed: ", err)
		}
		_, err = client.sendCommand("tellraw @a [{\"text\":\"Server shutdown commencing in \",\"bold\":false,\"italic\":true,\"underlined\":false,\"striketrough\":false,\"obfuscated\":false,\"color\":\"gray\"},{\"text\":\"10 Seconds!\",\"bold\":false,\"italic\":true,\"underlined\":false,\"striketrough\":false,\"obfuscated\":false,\"color\":\"dark_aqua\"}]")
		if err != nil {
			log.Println("[AlphaDiscordBot] RCON server command connection failed: ", err)
		}
		time.Sleep(3 * time.Second)
		for i := 10; i >= 0; i-- {
			time.Sleep(1 * time.Second)
			_, err = client.sendCommand("title @a title {\"text\":\"" + strconv.Itoa(i) + "\",\"bold\":false,\"italic\":false,\"underlined\":false,\"striketrough\":false,\"obfuscated\":false,\"color\":\"red\"}")
			if err != nil {
				log.Println("[AlphaDiscordBot] RCON server command connection failed: ", err)
			}
		}
		_, err = client.sendCommand("stop")
		if err != nil {
			log.Println("[AlphaDiscordBot] RCON server command connection failed - trying again: ", err)
			_ = client.reconnect()
			_, err = client.sendCommand("stop")
			if err != nil {
				log.Println("[AlphaDiscordBot] RCON server reconnect failed finally: ", err)
				msgDiscordMC <- "Error while trying to stop Server"
				msgAlpha <- "Error sending stop command to MC-Server"
				mcStopping = false
			}
		}
	}
}

func rollHelper(s *discordgo.Session, m *discordgo.MessageCreate) {
	// ToDO: For this take Inspiration here: https://github.com/further-reading/Dicecord-Chatbot
	// and here: https://github.com/Celeo/CoD_dice_roller
	// Should look like this 14 rolls:
	// 5 10(10(6)) 8 4 3 9 3 1 3 2 2 9 8 10(10))
	// Or with rote:
	// 5 10(10(6)) 8 4 3 9 3 1 3 2 2 9 8 10(10)) | Rote: 8 5 3 5 6 8 7 4
	// x Successes

	// Catch errors in command
	inputString := strings.Split(m.Content, " ")
	if len(inputString) < 2 {
		s.ChannelMessageSend(m.ChannelID, "There was an error in your command!")
		return
	}

	// Catch simple commands
	switch inputString[1] {
	case "one":
		s.ChannelMessageSend(m.ChannelID, "Simple 1D10 = "+strconv.Itoa(roll1D10()))
		return
	case "chance":
		var retString strings.Builder
		retString.Write([]byte("Chance-Die: "))
		// ToDo: Code to be added...
	}

	// Check which dice designation is used
	if strings.Contains(inputString[1], "d") {
		// Catch error in dice designation [/roll 1*s*10 ]
		diceIndex := strings.Split(inputString[1], "d")
		if len(diceIndex) < 2 {
			s.ChannelMessageSend(m.ChannelID, "There was an error in your command!")
			return
		}

		// Catch d-notation and read modifiers
		switch diceIndex[1] {
		// Catch error created by using a wrong dice side number: [/roll 1d*34*] | [/roll 1d*4f*]
		default:
			s.ChannelMessageSend(m.ChannelID, "Warning! "+diceIndex[1]+"-sided dice are not supported yet!")
			return
		}
	} else if inputString[1] == "chance" {
		s.ChannelMessageSend(m.ChannelID, "Not implemented yet!")
		return
	} else {
		// Start Dice Rolling in CoD Mode: Parse 9-again 8-again Rote-quality
		// Opperate by using Modes like this:
		// 9 = 9-again
		// 8 = 8,9-again
		// r = rote-quality
		// r9 = rote and 9-again
		// r8 = rote and 8,9 again
		// n = roll but nothing is rolled again
		// nr = roll with rote quality but reroll nothing

		// Init needed variables
		//var retString strings.Builder
		//successes := 0

		// Catch invalid number of dice to throw
		throwCount, err := strconv.Atoi(inputString[1])
		if err != nil {
			s.ChannelMessageSend(m.ChannelID, "There was an error in your command!")
			return
		}
		if throwCount > 1000 {
			s.ChannelMessageSend(m.ChannelID, "Don't you think that are a few to many dice to throw?")
			return
		}

		// Rest of Code here
	}
}

func roll1D10() int {

	onlyOnce.Do(func() {
		rand.Seed(time.Now().UnixNano()) // only run once
	})

	return dice[rand.Intn(len(dice))]
}

func pingMC() {
	// To be edited with a true server ping - finished (more or less) - can still be improved!
	_, _, err := bot.PingAndList("mc.tasadar.net", 25565)
	if mcRunning == false && err == nil { // Resets Counter if server online trough other means
		lastPlayerOnline = time.Now()
		err = set("mc|lastPlayerOnline", lastPlayerOnline.Format(lastPlayerOnlineLayout))
		if err != nil {
			log.Println("[AlphaDiscordBot] Error setting mc|lastPlayerOnline on Redis: ", err)
		}
	}
	mcRunning = err == nil
	var mcRunningString string
	if mcRunning {
		mcRunningString = "true"
	} else {
		mcRunningString = "false"
	}
	err = set("mc|IsRunning", mcRunningString)
	if err != nil {
		log.Println("[AlphaDiscordBot] Error setting mc|IsRunning on Redis: ", err)
	}
}
