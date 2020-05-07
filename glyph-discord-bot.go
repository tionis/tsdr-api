package main

import (
	"io/ioutil"
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

	"github.com/bwmarrin/discordgo"
	_ "github.com/heroku/x/hmetrics/onload"
)

// Global constants
const councilman = "706782033090707497"

// Global Variables
var mcStopping bool
var mcRunning bool
var mainChannelID string
var msgDiscord chan glyphDiscordMsg
var lastPlayerOnline time.Time
var discordAdminID string
var onlyOnce sync.Once
var dice = []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}

type glyphDiscordMsg struct {
	ChannelID string
	Message   string
}

// Some Constants
const lastPlayerOnlineLayout = "2006-01-02T15:04:05.000Z"
const tnGatewayAddress = "https://tn.tasadar.net"

// Main and Init
func glyphDiscordBot() {
	mainChannelID = "574959338754670602"
	discordAdminID = "259076782408335360"

	dg, err := discordgo.New("Bot " + os.Getenv("DISCORD_TOKEN"))
	if err != nil {
		log.Println("[GlyphDiscordBot] Error creating Discord session,", err)
	}

	// Register the messageCreate func as a callback for MessageCreate events.
	dg.AddHandler(messageCreate)

	// Open a websocket connection to Discord and begin listening.
	err = dg.Open()
	if err != nil {
		log.Println("[GlyphDiscordBot] Error opening connection,", err)
		return
	}

	// Init Variables from redis
	lastPlayerOnlineString, err := getError("mc|lastPlayerOnline")
	if err != nil {
		log.Println("Error reading mc|lastPlayerOnline from Redis: ", err)
		lastPlayerOnline = time.Now()
	}
	lastPlayerOnline, err = time.Parse(lastPlayerOnlineLayout, lastPlayerOnlineString)
	if err != nil {
		log.Println("Error transforming mc|lastPlayerOnline to time object: ", err)
		lastPlayerOnline = time.Now()
	}

	// Define Input Channel for discord messages outside from normal responses
	go func(s *discordgo.Session) {
		msgDiscord = make(chan glyphDiscordMsg)
		for {
			toSend := <-msgDiscord
			_, _ = s.ChannelMessageSend(toSend.ChannelID, toSend.Message)
		}
	}(dg)

	// Init mcRunning and mcStopping
	mcRunningString, err := getError("mc|IsRunning")
	if err != nil {
		log.Println("[GlyphDiscordBot] Error getting Redis value for mc|IsRunning", err)
	} else if mcRunningString == "true" {
		mcRunning = true
	} else if mcRunningString == "false" {
		mcRunning = false
	} else {
		mcRunning = false
		log.Println("[GlyphDiscordBot] Error converting Redis value for mc|IsRunning, expected true or false but got " + mcRunningString)
	}
	mcStoppingString, err := getError("mc|IsStopping")
	if err != nil {
		log.Println("[GlyphDiscordBot] Error getting Redis value for mc|IsStopping", err)
	} else if mcStoppingString == "true" {
		// Check if it maybe has already been stopped
		mcStopping = true
		var message glyphDiscordMsg
		message.Message = "Restarting shutdown sequence...\nYou habe 5 Minutes!"
		message.ChannelID = mainChannelID
		msgDiscord <- message
		go stopMCServerIn(5)
	} else if mcStoppingString == "false" {
		mcStopping = false
	} else {
		mcStopping = false
		log.Println("[GlyphDiscordBot] Error converting Redis value for mc|IsStopping, expected true or false but got " + mcRunningString)
	}

	// Get Server State
	go pingMC()

	// Set some StartUp Stuff
	dgStatus, err := getError("dgStatus")
	if err != nil {
		log.Println("[GlyphDiscordBot] Error getting dgStatus from redis: ", err)
		dgStatus = "planning world domination"
	}
	_ = dg.UpdateStatus(0, dgStatus)

	// Wait here until CTRL-C or other term signal is received.
	log.Println("[GlyphDiscordBot] Glyph Discord Bot was started.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc

	// Cleanly close down the Discord session.
	_ = dg.Close()
}

func getMCMainChannelID(guildID string) string {
	// TODO add setting thingie
	return "574959338754670602"
	//return kvget("mc|" + guildID + "|mainchannel")
}

// This function will be called (due to AddHandler above) every time a new
// message is created on any channel that the autenticated bot has access to.
func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {

	// Ignore all messages created by the bot itself
	if m.Author.ID == s.State.User.ID {
		return
	}

	// Check if glyph is currently in a conversation with user
	context := get("glyph|discord:" + m.Author.ID + "|context")
	if context != "" {
		switch context {
		case "construct-character-creation":
			// TODO: Character creation dialog
		default:
			log.Println("[GlyphDiscordBot] New Command by " + m.Author.Username + ": " + m.Content)
			_, _ = s.ChannelMessageSend(m.ChannelID, "I encountered an interal error, please contact the administrator.")
			return
		}
	}

	// Check if a known command was written
	inputString := strings.Split(m.Content, " ")
	switch inputString[0] {
	case "/roll":
		if len(inputString) < 2 {
			log.Println("[GlyphDiscordBot] New Command by " + m.Author.Username + ": " + m.Content)
			_, _ = s.ChannelMessageSend(m.ChannelID, "To roll construct dice just tell me how many I should roll and what Modifiers I shall apply.\nI can also roll custom dice like this: /roll 3d12")
		} else {
			rollHelper(s, m)
		}
	case "/r":
		rollHelper(s, m)
	case "/help":
		log.Println("[GlyphDiscordBot] New Command by " + m.Author.Username + ": " + m.Content)
		_, _ = s.ChannelMessageSend(m.ChannelID, "Available Command Categories:\n - General Tasadar Network - /tn help\n - Minecraft Server - /mc help\n - Uni Passau - /unip help\n - PnP Tools - /pnp help")
	case "/unip":
		log.Println("[GlyphDiscordBot] New Command by " + m.Author.Username + ": " + m.Content)
		_, _ = s.ChannelMessageSend(m.ChannelID, "Available Commands:\n/food - Food for today\n/food tomorrow - Food for tomorrow")
	case "/tn":
		if len(inputString) < 2 {
			log.Println("[GlyphDiscordBot] New Command by " + m.Author.Username + ": " + m.Content)
			_, _ = s.ChannelMessageSend(m.ChannelID, "Available Commands:\nNone!")
		} else {
			switch inputString[1] {
			case "pic":
				tnPicHandler(s, m)
			default:
				log.Println("[GlyphDiscordBot] New Command by " + m.Author.Username + ": " + m.Content)
				_, _ = s.ChannelMessageSend(m.ChannelID, "Available Commands:\nNone!")
			}
		}
	case "/pnp":
		log.Println("[GlyphDiscordBot] New Command by " + m.Author.Username + ": " + m.Content)
		_, _ = s.ChannelMessageSend(m.ChannelID, "Available Commands:\n - /roll - Roll Dice after construct rules\n - /save initmod - Save your init modifier\n - /construct or /co - get construct-specific help")
	case "/co", "/construct":
		if len(inputString) < 2 {
			log.Println("[GlyphDiscordBot] New Command by " + m.Author.Username + ": " + m.Content)
			_, _ = s.ChannelMessageSend(m.ChannelID, "Available Commands:\n - /co trait TRAITNAME - Get Description for specified trait")
		} else {
			switch inputString[1] {
			// Construct command here
			default:
				log.Println("[GlyphDiscordBot] New Command by " + m.Author.Username + ": " + m.Content)
				_, _ = s.ChannelMessageSend(m.ChannelID, "Invalid Command Syntax")
			}
		}
	case "/food":
		log.Println("[GlyphDiscordBot] New Command by " + m.Author.Username + ": " + m.Content)
		_, _ = s.ChannelMessageSend(m.ChannelID, foodtoday())
	case "/food tomorrow":
		log.Println("[GlyphDiscordBot] New Command by " + m.Author.Username + ": " + m.Content)
		_, _ = s.ChannelMessageSend(m.ChannelID, foodtomorrow())
	case "/ping":
		log.Println("[GlyphDiscordBot] New Command by " + m.Author.Username + ": " + m.Content)
		_, _ = s.ChannelMessageSend(m.ChannelID, "Pong!")
	case "/save":
		if len(inputString) < 2 {
			log.Println("[GlyphDiscordBot] New Command by " + m.Author.Username + ": " + m.Content)
			_, _ = s.ChannelMessageSend(m.ChannelID, "Save Data to the Bot. Currently available:\n - /save initmod x - Save you Init Modifier")
		} else {
			switch inputString[1] {
			case "initmod":
				if len(inputString) < 3 {
					err := del("glyph:udata|discord:" + m.Author.ID + "|initmod")
					if err != nil {
						log.Println("[GlyphDiscordBot] New Command by " + m.Author.Username + ": " + m.Content)
						_, _ = s.ChannelMessageSend(m.ChannelID, "There was an internal error!")
					} else {
						log.Println("[GlyphDiscordBot] New Command by " + m.Author.Username + ": " + m.Content)
						_, _ = s.ChannelMessageSend(m.ChannelID, "Your init modifiert was reset.")
					}
				} else {
					initMod, err := strconv.Atoi(inputString[2])
					if err != nil {
						log.Println("[GlyphDiscordBot] New Command by " + m.Author.Username + ": " + m.Content)
						_, _ = s.ChannelMessageSend(m.ChannelID, "There was an error in your command!")
					} else {
						err := set("glyph:udata|discord:"+m.Author.ID+"|initmod", strconv.Itoa(initMod))
						if err != nil {
							log.Println("[GlyphDiscordBot] New Command by " + m.Author.Username + ": " + m.Content)
							_, _ = s.ChannelMessageSend(m.ChannelID, "There was an internal error!")
						} else {
							log.Println("[GlyphDiscordBot] New Command by " + m.Author.Username + ": " + m.Content)
							_, _ = s.ChannelMessageSend(m.ChannelID, "Your init modifiert was set to "+strconv.Itoa(initMod)+".")
						}
					}
				}
			default:
				_, _ = s.ChannelMessageSend(m.ChannelID, "Sorry, I dont know what to save here!")
			}
		}
	case "/mc":
		if len(inputString) < 2 {
			log.Println("[GlyphDiscordBot] New Command by " + m.Author.Username + ": " + m.Content)
			_, _ = s.ChannelMessageSend(m.ChannelID, "Available Commands:\n/mc start - Starts the Minecraft Server\n/mc status - Get the current status of the Minecraft Server\n/mc stop - Stop the Minecraft Server")

		} else {
			switch inputString[1] {
			case "start":
				log.Println("[GlyphDiscordBot] New Command by " + m.Author.Username + ": " + m.Content)
				if m.ChannelID == mainChannelID {
					if mcRunning {
						_, _ = s.ChannelMessageSend(m.ChannelID, "Server already running!")
					} else {
						_, _ = s.ChannelMessageSend(m.ChannelID, "Starting MC-Server...")
						success := mcStart()
						if !success {
							_, _ = s.ChannelMessageSend(m.ChannelID, "An Error occurred!")
							log.Println("[GlyphDiscordBot] Error starting mc-server")
						}
						pingInMinutes(2)
					}
				} else {
					_, _ = s.ChannelMessageSend(m.ChannelID, "Please use this command in the minecraft channel!")
				}
			case "stop":
				log.Println("[GlyphDiscordBot] New Command by " + m.Author.Username + ": " + m.Content)
				go mcShutdownDiscord(s, m, 3)
			case "cancel":
				log.Println("[GlyphDiscordBot] New Command by " + m.Author.Username + ": " + m.Content)
				if mcStopping {
					mcStopping = false
					_, _ = s.ChannelMessageSend(m.ChannelID, "Server shutdown stopped.")
				} else if mcRunning {
					_, _ = s.ChannelMessageSend(m.ChannelID, "There is currently no Server Shutdown scheduled!")
				} else {
					_, _ = s.ChannelMessageSend(m.ChannelID, "Server is currently not running!")
				}
			case "status":
				log.Println("[GlyphDiscordBot] New Command by " + m.Author.Username + ": " + m.Content)
				pingMC()
				if mcRunning {
					if !mcRunning {
						_, _ = s.ChannelMessageSend(m.ChannelID, "Warning! - Server currently not running!")
						return
					}
					client := &http.Client{}
					req, _ := http.NewRequest("GET", "https://mcapi.tasadar.net/mc/creepercount", nil)
					mcAPIToken := get("mcapi|token")
					req.Header.Set("TASADAR_SECRET", mcAPIToken)
					res, err := client.Do(req)
					if res == nil {
						log.Println("Error connecting to mcAPI in status-1")
						s.ChannelMessageSend(m.ChannelID, "I'm having problems reaching the Server, please try again later.\nIf this problem persists contact the admin.")
						return
					}
					defer res.Body.Close()
					if res.StatusCode != 200 {
						log.Println("Error connecting to mcAPI in status-2")
						s.ChannelMessageSend(m.ChannelID, "I'm having problems reaching the Server, please try again later.\nIf this problem persists contact the admin.")
						return
					}
					bodyBytes, err := ioutil.ReadAll(res.Body)
					if err != nil {
						log.Println("Error connecting to mcAPI in status-3")
					}
					creeperCountString := string(bodyBytes)
					req, _ = http.NewRequest("GET", "https://mcapi.tasadar.net/mc/playercount", nil)
					req.Header.Set("TASADAR_SECRET", mcAPIToken)
					res, err = client.Do(req)
					if res == nil {
						log.Println("Error connecting to mcAPI in status-4")
						s.ChannelMessageSend(m.ChannelID, "I'm having problems reaching the Server, please try again later.\nIf this problem persists contact the admin.")
						return
					}
					defer res.Body.Close()
					if res.StatusCode != 200 {
						log.Println("Error connecting to mcAPI in status-5")
						s.ChannelMessageSend(m.ChannelID, "I'm having problems reaching the Server, please try again later.\nIf this problem persists contact the admin.")
						return
					}
					bodyBytes, err = ioutil.ReadAll(res.Body)
					if err != nil {
						log.Println("Error connecting to mcAPI in status-6")
					}
					playerCountString := string(bodyBytes)
					_, _ = s.ChannelMessageSend(m.ChannelID, "Server currently online\nAt the moment there are "+playerCountString+" players on the server and there are "+creeperCountString+" Creepers loaded.")
				} else {
					_, _ = s.ChannelMessageSend(m.ChannelID, "Server currently offline!\nTo start it use /mc start")
				}

			default:
				log.Println("[GlyphDiscordBot] New Command by " + m.Author.Username + ": " + m.Content)
				_, _ = s.ChannelMessageSend(m.ChannelID, "Available Commands:\n/mc start - Starts the Minecraft Server\n/mc status - Get the current status of the Minecraft Server\n/mc stop - Stop the Minecraft Server")
			}
		}
	case "/id":
		_, _ = s.ChannelMessageSend(m.ChannelID, "Your ID is:\n"+m.Author.ID)
		//default:
		//log.Println("[GlyphDiscordBot] Logged Unknown Command by " + m.Author.Username + ": "+ m.Content)
	case "/updateStatus":
		if m.Author.ID == discordAdminID {
			newStatus := strings.TrimPrefix(m.Content, "/updateStatus ")
			err := set("dgStatus", newStatus)
			if err != nil {
				log.Println("Error setting dgStatus on Redis: ", err)
				_, _ = s.ChannelMessageSend(m.ChannelID, "Error sending Status to Safe!")
			}
			_ = s.UpdateStatus(0, newStatus)
			_, _ = s.ChannelMessageSend(m.ChannelID, "New Status set!")
		} else {
			_, _ = s.ChannelMessageSend(m.ChannelID, "You are not authorized to execute this command!\nThis incident will be reported.\nhttps://imgs.xkcd.com/comics/incident.png")
		}
	case "/whoami":
		s.ChannelMessageSend(m.ChannelID, m.Author.String())
	case "/todo":
		s.ChannelMessageSend(m.ChannelID, "Feature still in Development")
	case "/amiadmin":
		hasRole, err := memberHasRole(s, m.GuildID, m.Author.ID, councilman)
		if err != nil {
			log.Println("[GlyphDiscordBot] Error while checking if member has role: ", err)
			s.ChannelMessageSend(m.ChannelID, "An error occurred!")
		}
		if hasRole {
			s.ChannelMessageSend(m.ChannelID, "TRUE")
		} else {
			s.ChannelMessageSend(m.ChannelID, "FALSE")
		}
	case "/kick":
		hasRole, err := memberHasRole(s, m.GuildID, m.Author.ID, councilman)
		if err != nil {
			log.Println("[GlyphDiscordBot] Error while checking if member has role: ", err)
			s.ChannelMessageSend(m.ChannelID, "An error occurred!")
		}
		if hasRole {
			s.ChannelMessageSend(m.ChannelID, "Not implemented yet!")
		} else {
			s.ChannelMessageSend(m.ChannelID, "You are not authorized to execute this command!")
		}
	}
}

func memberHasRole(s *discordgo.Session, guildID string, userID string, roleID string) (bool, error) {
	member, err := s.State.Member(guildID, userID)
	if err != nil {
		if member, err = s.GuildMember(guildID, userID); err != nil {
			return false, err
		}
	}

	// Iterate through the role IDs stored in member.Roles
	// to check permissions
	for _, roleID := range member.Roles {
		role, err := s.State.Role(guildID, roleID)
		if err != nil {
			return false, err
		}
		if role.ID == roleID {
			return true, nil
		}
	}

	return false, nil
}

func mcShutdownDiscord(s *discordgo.Session, m *discordgo.MessageCreate, minutes int) {
	mcStopping = true
	minutesString := strconv.Itoa(minutes)
	if !mcRunning {
		_, _ = s.ChannelMessageSend(m.ChannelID, "The Server is currently not running!\nYou must start the Server to stop it!")
		return
	}
	_, _ = s.ChannelMessageSend(m.ChannelID, "If nobody says /mc cancel in the next "+minutesString+" Minutes I will shut down the server!")
	if MCMainChannelID := getMCMainChannelID(m.GuildID); m.ChannelID != MCMainChannelID {
		_, _ = s.ChannelMessageSend(MCMainChannelID, "If nobody says /mc cancel in the next "+minutesString+" Minutes I will shut down the server!")
	}
	time.Sleep(time.Duration(minutes) * time.Minute)
	if mcStopping {
		client := &http.Client{}
		req, _ := http.NewRequest("GET", "https://mcapi.tasadar.net/mc/stop", nil)
		mcAPIToken := get("mcapi|token")
		req.Header.Set("TASADAR_SECRET", mcAPIToken)
		res, err := client.Do(req)
		if res == nil || err != nil {
			log.Println("Error connecting to mcAPI in mcShutdownDiscord")
			s.ChannelMessageSend(m.ChannelID, "I'm having problems reaching the Server, please try again later.\nIf this problem persists contact the admin.")
		}
		if res.StatusCode == 200 {
			mcStopping = false
			mcRunning = false
			set("mc|IsStopping", "false")
			s.ChannelMessageSend(m.ChannelID, "Server shutting down...")
		}
	}
}

func mcStart() bool {
	client := &http.Client{}
	req, _ := http.NewRequest("GET", "https://mcapi.tasadar.net/mc/start", nil)
	mcAPIToken := get("mcapi|token")
	req.Header.Set("TASADAR_SECRET", mcAPIToken)
	res, err := client.Do(req)
	if res == nil {
		log.Println("Error connecting to mcAPI in mcStart")
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
			log.Println("[GlyphDiscordBot] Error setting mc|IsRunning on Redis: ", err)
		}
		lastPlayerOnline = time.Now()
		_ = set("mc|lastPlayerOnline", lastPlayerOnline.Format(lastPlayerOnlineLayout))
		if err != nil {
			log.Println("[GlyphDiscordBot] Error setting mc|lastPlayerOnline on Redis: ", err)
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
		client := &http.Client{}
		req, _ := http.NewRequest("GET", "https://mcapi.tasadar.net/mc/ping", nil)
		mcAPIToken := get("mcapi|token")
		req.Header.Set("TASADAR_SECRET", mcAPIToken)
		res, err := client.Do(req)
		if res == nil {
			log.Println("[GlyphDiscordBot] Error in request!")
			return
		}
		defer res.Body.Close()
		bodyBytes, err := ioutil.ReadAll(res.Body)
		if err != nil {
			log.Println("Error connecting to mcAPI in updateMC-1")
		}
		playerCount := 0
		bodyString := string(bodyBytes)
		if res.StatusCode == 200 {
			if bodyString == "online" {
				mcRunning = true
				client := &http.Client{}
				req, _ := http.NewRequest("GET", "https://mcapi.tasadar.net/mc/playercount", nil)
				mcAPIToken := get("mcapi|token")
				req.Header.Set("TASADAR_SECRET", mcAPIToken)
				res, err := client.Do(req)
				if res == nil {
					log.Println("[GlyphDiscordBot] Error in request!")
					return
				}
				defer res.Body.Close()
				bodyBytes, err := ioutil.ReadAll(res.Body)
				if err != nil {
					log.Println("Error connecting to mcAPI in updateMC-2")
				}
				bodyString := string(bodyBytes)
				playerCount, err = strconv.Atoi(bodyString)
				if err != nil {
					log.Println("Error reading response from mcAPI")
				}
			} else if bodyString == "offline" {
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
				return
			} else {
				log.Println("Error interpreting mcAPI response")
			}
		}
		if playerCount > 0 {
			lastPlayerOnline = time.Now()
			_ = set("mc|lastPlayerOnline", lastPlayerOnline.Format(lastPlayerOnlineLayout))
			if err != nil {
				log.Println("[GlyphDiscordBot] Error setting mc|lastPlayerOnline on Redis: ", err)
			}
		} else {
			if time.Since(lastPlayerOnline).Minutes() > 30 {
				lastPlayerOnline = time.Now()
				_ = set("mc|lastPlayerOnline", lastPlayerOnline.Format(lastPlayerOnlineLayout))
				if err != nil {
					log.Println("[GlyphDiscordBot] Error setting mc|lastPlayerOnline on Redis: ", err)
				}
				mcStopPlayerOffline()
			}
		}
	}
}

func mcStopPlayerOffline() {
	mcStopping = true
	err := set("mc|IsStopping", "true")
	if err != nil {
		log.Println("[GlyphDiscordBot] Error saving mc|IsStopping to database")
	}
	var message glyphDiscordMsg
	message.Message = "There were no players on the Server for quite some time.\nIf nobody says /mc cancel in the next 5 Minutes I will shut down the server!"
	message.ChannelID = mainChannelID // TODO: Move this code to mcapi
	msgDiscord <- message
	stopMCServerIn(5)
}

func stopMCServerIn(minutesToShutdown int) {
	time.Sleep(time.Duration(minutesToShutdown) * time.Minute)
	if mcStopping {
		client := &http.Client{}
		req, _ := http.NewRequest("GET", "https://mcapi.tasadar.net/mc/stop", nil)
		mcAPIToken := get("mcapi|token")
		req.Header.Set("TASADAR_SECRET", mcAPIToken)
		res, err := client.Do(req)
		if res == nil || err != nil {
			log.Println("Error connecting to mcAPI in stopMcServerIn-1")
			var message glyphDiscordMsg
			message.Message = "Error stopping Server"
			message.ChannelID = mainChannelID
			msgDiscord <- message
			return
		}
		if res.StatusCode == 200 {
			mcRunning = false
			err = set("mc|IsRunning", "false")
			if err != nil {
				log.Println("[GlyphDiscordBot] Error setting mc|IsRunning on Redis: ", err)
			}
			err = set("mc|IsStopping", "false")
			if err != nil {
				log.Println("[GlyphDiscordBot] Error setting mc|IsStopping on Redis: ", err)
			}
			var message glyphDiscordMsg
			message.Message = "Shutting down Server..."
			message.ChannelID = mainChannelID
			msgDiscord <- message
		} else {
			log.Println("Error connecting to mcAPI in stopMcServerIn-2")
			var message glyphDiscordMsg
			message.Message = "Error stopping Server! Retrying in " + strconv.Atoi(minutesToShutdown) + " Minutes!"
			message.ChannelID = mainChannelID
			msgDiscord <- message
			if minutesToShutdown < 60 {
				stopMCServerIn(minutesToShutdown*2)
			}else{
				message.Message = "Error stopping Server! Maximum retries reached. I will stop trying now."
				message.ChannelID = mainChannelID
				msgDiscord <- message
			}
			mcStopping = false
			err = set("mc|IsStopping", "false")
			if err != nil {
				log.Println("[GlyphDiscordBot] Error setting mc|IsRunning on Redis: ", err)
			}
			return
		}
	}
}

func tnPicHandler(s *discordgo.Session, m *discordgo.MessageCreate) {
	inputString := strings.Split(m.Content, " ")
	tnAddress := ""
	if len(inputString) != 3 {
		log.Println("[GlyphDiscordBot] New Command by " + m.Author.Username + ": " + m.Content)
		_, _ = s.ChannelMessageSend(m.ChannelID, "Please Specify a valid TN-Address.")
		return
	}
	if strings.HasPrefix(inputString[2], "/ipfs/") {
		tnAddress = inputString[2]
	} else if strings.HasPrefix(inputString[2], "Qm") {
		tnAddress = "/ipfs/" + inputString[2]
	} else if strings.HasPrefix(inputString[2], "/ipns/") {
		tnAddress = inputString[2]
	} else if strings.HasPrefix(inputString[2], "/hash/") {
		tnAddress = "/ipfs/" + strings.TrimPrefix(inputString[2], "/hash/")
	} else if strings.HasPrefix(inputString[2], "/name/") {
		tnAddress = "/ipns/" + strings.TrimPrefix(inputString[2], "/hash/")
	} else {
		log.Println("[GlyphDiscordBot] New Command by " + m.Author.Username + ": " + m.Content)
		_, _ = s.ChannelMessageSend(m.ChannelID, "Sorry, I couldn't parse your Address.\nPlease Specify a valid TN-Address.")
		return
	}
	embed := &discordgo.MessageEmbed{
		Author:      &discordgo.MessageEmbedAuthor{},
		Color:       0x00ff00, // Green
		Description: "Heres a picture from the Tasadar Network:",
		Image: &discordgo.MessageEmbedImage{
			URL: tnGatewayAddress + tnAddress,
		},
		Thumbnail: &discordgo.MessageEmbedThumbnail{
			URL: tnGatewayAddress + tnAddress,
		},
		Timestamp: time.Now().Format(time.RFC3339), // Discord wants ISO8601; RFC3339 is an extension of ISO8601 and should be completely compatible.
		Title:     "Tasadar Picture",
	}
	s.ChannelMessageSendEmbed(m.ChannelID, embed)
}

func rollHelper(s *discordgo.Session, m *discordgo.MessageCreate) {
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
		diceRollResult := roll1D10()
		switch diceRollResult {
		case 10:
			s.ChannelMessageSend(m.ChannelID, "**Success!** Your Chance Die showed a 10!")
		case 1:
			s.ChannelMessageSend(m.ChannelID, "**Fail!** Your Chance Die failed spectaculary!")
		default:
			s.ChannelMessageSend(m.ChannelID, "Fail! You rolled a **"+strconv.Itoa(diceRollResult)+"** on your Chance die!")
		}
		return
	case "init":
		initModString := get("glyph:udata|discord:" + m.Author.ID + "|initmod")
		initMod, err := strconv.Atoi(initModString)
		if err != nil {
			initModString = ""
		}
		if initModString == "" {
			s.ChannelMessageSend(m.ChannelID, "No init modifier saved, heres a simple D10 throw:\n1D10 = "+strconv.Itoa(roll1D10()))
		} else {
			diceResult := roll1D10()
			endResult := diceResult + initMod
			s.ChannelMessageSend(m.ChannelID, "Your Initiative is: **"+strconv.Itoa(endResult)+"**\n"+strconv.Itoa(diceResult)+" + "+strconv.Itoa(initMod)+" = "+strconv.Itoa(endResult))
		}
		return
	}

	// Check which dice designation is used
	if strings.Contains(inputString[1], "d") {
		// Catch error in dice designation [/roll 1*s*10 ]
		diceIndex := strings.Split(inputString[1], "d")
		if len(diceIndex) < 2 {
			s.ChannelMessageSend(m.ChannelID, "There was an error in your command!")
			return
		}
		sides, err := strconv.Atoi(diceIndex[1])
		if err != nil {
			s.ChannelMessageSend(m.ChannelID, "There was an error in your command!")
			return
		}
		amount, err := strconv.Atoi(diceIndex[0])
		if err != nil {
			s.ChannelMessageSend(m.ChannelID, "There was an error in your command!")
			return
		}

		// Catch d-notation and read modifiers
		if amount < 1 {
			s.ChannelMessageSend(m.ChannelID, "Nice try!")
			return
		}
		switch {
		case sides < 1:
			s.ChannelMessageSend(m.ChannelID, "Nice try!")
			return
		case sides == 1:
			s.ChannelMessageSend(m.ChannelID, "Really? Thats one times "+diceIndex[0]+". I think you can do the math yourself!")
			return
		default:
			if amount > 1000 {
				s.ChannelMessageSend(m.ChannelID, "Maybe try a few less dice. We're not playing Warhammer Ultimate here.")
				return
			}
			retSlice := rollXSidedDie(amount, sides)
			var retString strings.Builder
			endresult := 0
			retString.Write([]byte(diceIndex[0] + "d" + diceIndex[1] + ": "))
			for i := range retSlice {
				endresult += retSlice[i]
				if i != len(retSlice)-1 {
					retString.Write([]byte(strconv.Itoa(retSlice[i]) + " + "))
				} else {
					retString.Write([]byte(strconv.Itoa(retSlice[i]) + " = " + strconv.Itoa(endresult)))
				}
			}
			s.ChannelMessageSend(m.ChannelID, retString.String())
			return
		}
	} else if inputString[1] == "chance" {
		result := roll1D10()
		retString := ""
		switch result {
		case 10:
			retString = "Success! Your rolled a 10!"
		case 1:
			retString = "Critical Fail! That was a 1!"
		default:
			retString = "Fail! Your rolled a " + strconv.Itoa(result) + "!"
		}
		s.ChannelMessageSend(m.ChannelID, retString)
		return
	} else {
		var roteQuality, noReroll, eightAgain, nineAgain bool
		if len(inputString) > 2 {
			roteQuality = strings.Contains(inputString[2], "r")
			noReroll = strings.Contains(inputString[2], "n")
			eightAgain = strings.Contains(inputString[2], "8")
			nineAgain = strings.Contains(inputString[2], "9")
		}

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
		var retSlice [][]int
		// Use correct Method
		if len(inputString) < 3 {
			// Check if special parameters were used
			retSlice = normalConstructRoll(throwCount)
		} else {
			if eightAgain {
				// Check if 8again -> infers no 9Again -> only check for n and r
				if roteQuality {
					retSlice = constructRoll8r(throwCount)
				} else {
					retSlice = constructRoll8(throwCount)
				}
			} else if nineAgain {
				if roteQuality {
					retSlice = constructRoll9r(throwCount)
				} else {
					retSlice = constructRoll9(throwCount)
				}
			} else if roteQuality {
				if noReroll {
					retSlice = constructRollrn(throwCount)
				} else {
					retSlice = constructRollr(throwCount)
				}
			} else {
				if noReroll {
					retSlice = constructRolln(throwCount)
				} else {
					log.Println("[Glyph Discord Bot] This should not have been executed! Perhaps an error in syntax?")
					s.ChannelMessageSend(m.ChannelID, "There was an error while parsing your input!")
					return
				}
			}
		}
		// Count Successes and Ones while parsing the return String
		// Parse Slice here
		var successes, ones int
		retString := "Results: "
		for i := range retSlice {
			//retString += "["
			for j := range retSlice[i] {
				retString += strconv.Itoa(retSlice[i][j])
				if j != len(retSlice[i])-1 {
					retString += " ⮞ "
				}
				switch retSlice[i][j] {
				case 8, 9, 10:
					successes++
				case 1:
					ones++
				}
			}
			//retString += "] "
			retString += " "
		}
		if ones >= (throwCount/2 + 1) {
			if successes == 0 {
				retString += "\nWell thats a **critical failure!**"
			} else {
				retString += "\nThat was nearly a critical failure! But you had **" + strconv.Itoa(successes) + "** Successes!"
			}
		} else {
			if successes > 0 {
				if successes >= 5 {
					mentionString := m.Author.Mention()
					retString += "\nThat were **" + strconv.Itoa(successes) + "** Successes!\n" + mentionString + " That was exceptional!"
				} else {
					retString += "\nThat were **" + strconv.Itoa(successes) + "** Successes!"
				}
			} else {
				retString += "\nNo Success for you! Thats bad, isn`t it?"
			}
		}
		s.ChannelMessageSend(m.ChannelID, retString)
	}
}

func rollXSidedDie(throwCount, sides int) []int {
	onlyOnce.Do(func() {
		rand.Seed(time.Now().UnixNano()) // only run once
	})
	retSlice := make([]int, throwCount)
	for i := 0; i < throwCount; i++ {
		retSlice[i] = rand.Intn(sides) + 1
	}
	return retSlice
}

func normalConstructRoll(throwCount int) [][]int {
	retSlice := make([][]int, throwCount)
	for i := range retSlice {
		retSlice[i] = []int{}
		repeat := true
		for repeat {
			diceResult := roll1D10()
			if diceResult != 10 {
				repeat = false
			}
			previousSlice := retSlice[i]
			if previousSlice == nil {
				previousSlice = []int{}
			}
			tmpSlice := append(previousSlice, diceResult)
			retSlice[i] = tmpSlice
		}
	}
	return retSlice
}

func constructRoll8(throwCount int) [][]int {
	retSlice := make([][]int, throwCount)
	for i := range retSlice {
		retSlice[i] = []int{}
		repeat := true
		for repeat {
			diceResult := roll1D10()
			if diceResult < 8 {
				repeat = false
			}
			previousSlice := retSlice[i]
			if previousSlice == nil {
				previousSlice = []int{}
			}
			tmpSlice := append(previousSlice, diceResult)
			retSlice[i] = tmpSlice
		}
	}
	return retSlice
}

func constructRoll8r(throwCount int) [][]int {
	retSlice := make([][]int, throwCount)
	isFirstReroll := true
	for i := range retSlice {
		retSlice[i] = []int{}
		repeat := true
		for repeat {
			diceResult := roll1D10()
			if isFirstReroll && diceResult < 8 {
				repeat = true
				isFirstReroll = false
			} else {
				if diceResult < 8 {
					repeat = false
				} else {
					isFirstReroll = false
				}
			}
			previousSlice := retSlice[i]
			if previousSlice == nil {
				previousSlice = []int{}
			}
			tmpSlice := append(previousSlice, diceResult)
			retSlice[i] = tmpSlice
		}
		isFirstReroll = true
	}
	return retSlice
}

func constructRoll9(throwCount int) [][]int {
	retSlice := make([][]int, throwCount)
	for i := range retSlice {
		retSlice[i] = []int{}
		repeat := true
		for repeat {
			diceResult := roll1D10()
			if diceResult < 9 {
				repeat = false
			}
			previousSlice := retSlice[i]
			if previousSlice == nil {
				previousSlice = []int{}
			}
			tmpSlice := append(previousSlice, diceResult)
			retSlice[i] = tmpSlice
		}
	}
	return retSlice
}

func constructRoll9r(throwCount int) [][]int {
	retSlice := make([][]int, throwCount)
	isFirstReroll := true
	for i := range retSlice {
		retSlice[i] = []int{}
		repeat := true
		for repeat {
			diceResult := roll1D10()
			if isFirstReroll && diceResult < 8 {
				repeat = true
				isFirstReroll = false
			} else {
				if diceResult < 9 {
					repeat = false
				} else {
					isFirstReroll = false
				}
			}
			previousSlice := retSlice[i]
			if previousSlice == nil {
				previousSlice = []int{}
			}
			tmpSlice := append(previousSlice, diceResult)
			retSlice[i] = tmpSlice
		}
		isFirstReroll = true
	}
	return retSlice
}

func constructRolln(throwCount int) [][]int {
	retSlice := make([][]int, throwCount)
	for i := range retSlice {
		retSlice[i] = []int{}
		diceResult := roll1D10()
		previousSlice := retSlice[i]
		if previousSlice == nil {
			previousSlice = []int{}
		}
		tmpSlice := append(previousSlice, diceResult)
		retSlice[i] = tmpSlice

	}
	return retSlice
}

func constructRollr(throwCount int) [][]int {
	retSlice := make([][]int, throwCount)
	isFirstReroll := true
	for i := range retSlice {
		retSlice[i] = []int{}
		repeat := true
		for repeat {
			diceResult := roll1D10()
			if isFirstReroll && diceResult < 8 {
				repeat = true
				isFirstReroll = false
			} else {
				if diceResult < 10 {
					repeat = false
				} else {
					isFirstReroll = false
				}
			}
			previousSlice := retSlice[i]
			if previousSlice == nil {
				previousSlice = []int{}
			}
			tmpSlice := append(previousSlice, diceResult)
			retSlice[i] = tmpSlice
		}
		isFirstReroll = true
	}
	return retSlice
}

func constructRollrn(throwCount int) [][]int {
	retSlice := make([][]int, throwCount)
	isFirstReroll := true
	for i := range retSlice {
		retSlice[i] = []int{}
		repeat := true
		for repeat {
			repeat = false
			diceResult := roll1D10()
			if isFirstReroll && diceResult < 8 {
				repeat = true
				isFirstReroll = false
			}
			previousSlice := retSlice[i]
			if previousSlice == nil {
				previousSlice = []int{}
			}
			tmpSlice := append(previousSlice, diceResult)
			retSlice[i] = tmpSlice
		}
		isFirstReroll = true
	}
	return retSlice
}

func roll1D10() int {

	onlyOnce.Do(func() {
		rand.Seed(time.Now().UnixNano()) // only run once
	})

	return dice[rand.Intn(len(dice))]
}

func pingMC() {
	client := &http.Client{}
	req, _ := http.NewRequest("GET", "https://mcapi.tasadar.net/mc/ping", nil)
	mcAPIToken := get("mcapi|token")
	req.Header.Set("TASADAR_SECRET", mcAPIToken)
	res, err := client.Do(req)
	if res == nil {
		log.Println("[GlyphDiscordBot] Error in request!")
		return
	}
	defer res.Body.Close()
	var onlineCheck bool
	if res.StatusCode == 200 {
		bodyBytes, err := ioutil.ReadAll(res.Body)
		if err != nil {
			log.Println("Error connecting to mcAPI in pingMC-1")
		}
		bodyString := string(bodyBytes)
		onlineCheck = bodyString == "online"
	} else {
		log.Println("[GlyphDiscordBot] Error contacting the MC API")
		return
	}
	if !mcRunning && onlineCheck { // Resets Counter if server online trough other means
		lastPlayerOnline = time.Now()
		err = set("mc|lastPlayerOnline", lastPlayerOnline.Format(lastPlayerOnlineLayout))
		if err != nil {
			log.Println("[GlyphDiscordBot] Error setting mc|lastPlayerOnline on Redis: ", err)
		}
	}
	mcRunning = onlineCheck
	var mcRunningString string
	if mcRunning {
		mcRunningString = "true"
	} else {
		mcRunningString = "false"
	}
	err = set("mc|IsRunning", mcRunningString)
	if err != nil {
		log.Println("[GlyphDiscordBot] Error setting mc|IsRunning on Redis: ", err)
	}
}
