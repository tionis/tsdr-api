package main

import (
	"context"
	"io"
	"log"
	"math/rand"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/bwmarrin/dgvoice"
	"github.com/bwmarrin/discordgo"
	_ "github.com/heroku/x/hmetrics/onload"
	"github.com/jonas747/dca"
	"github.com/rylio/ytdl"
)

// Global constants
const councilman = "706782033090707497"

// Discord ID of admin
var discordAdminID string

// A map of boolean channels that stop the playback indexed after guildIDs
var stopVoice map[string]chan bool

// A map of queue represented as ytdl.VideoInfo arrays indexed after guildIDs
var queueMap map[string][]*ytdl.VideoInfo

// Needed for onlyonce execution of random source
var onlyOnce sync.Once

// Represents a ten sided die, simplifies reroll handling
var dice = []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}

/*type glyphDiscordMsg struct {
	ChannelID string
	Message   string
}*/

// Main and Init
func glyphDiscordBot() {
	discordAdminID = "259076782408335360"
	stopVoice = make(map[string]chan bool)
	queueMap = make(map[string][]*ytdl.VideoInfo)

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
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, syscall.SIGQUIT, syscall.SIGHUP)
	<-sc

	// Cleanly close down the Discord session.
	for _, abort := range stopVoice {
		abort <- true
	}
	_ = dg.Close()
}

// This function will be called (due to AddHandler above) every time a new
// message is created on any channel that the authenticated bot has access to.
func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {

	// Ignore all messages created by the bot itself
	if m.Author.ID == s.State.User.ID {
		return
	}

	// Check if glyph is currently in a conversation with user
	messageContext := get("glyph|discord:" + m.Author.ID + "|messageContext")
	if messageContext != "" {
		switch messageContext {
		case "construct-character-creation":
			// TODO: Character creation dialog
		default:
			log.Println("[GlyphDiscordBot] New Command by " + m.Author.Username + ": " + m.Content)
			_, _ = s.ChannelMessageSend(m.ChannelID, "I encountered an internal error, please contact the administrator.")
			return
		}
	}

	// Check if a known command was written
	inputString := strings.Split(m.Content, " ")
	switch inputString[0] {
	// Dice commands
	case "/roll":
		if len(inputString) < 2 {
			log.Println("[GlyphDiscordBot] New Command by " + m.Author.Username + ": " + m.Content)
			_, _ = s.ChannelMessageSend(m.ChannelID, "To roll construct dice just tell me how many I should roll and what Modifiers I shall apply.\nI can also roll custom dice like this: /roll 3d12")
		} else {
			rollHelper(s, m)
		}
	case "/r":
		rollHelper(s, m)

	// Help commands
	case "/help":
		log.Println("[GlyphDiscordBot] New Command by " + m.Author.Username + ": " + m.Content)
		_, _ = s.ChannelMessageSend(m.ChannelID, "Available Command Categories:\n - General Tasadar Network - /tn help\n - Music Bot - /music help\n - Uni Passau - /unip help\n - PnP Tools - /pnp help")
	case "/music":
		log.Println("[GlyphDiscordBot] New Command by " + m.Author.Username + ": " + m.Content)
		_, _ = s.ChannelMessageSend(m.ChannelID, "Available Music Bot Commands:\n - /play - Play a song with a given youtube URL, or add it to the queue if music is already playing. \n - /stop - Stop all music \n - /queue - Show current queue \n - /pause - Pause current playback \n - /remove - Remove song number x from queue\n - /volume - set the volume to specified value")
	case "/unip":
		log.Println("[GlyphDiscordBot] New Command by " + m.Author.Username + ": " + m.Content)
		_, _ = s.ChannelMessageSend(m.ChannelID, "Available Commands:\n/food - Food for today\n/food tomorrow - Food for tomorrow")
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

	// Food commands
	case "/food":
		log.Println("[GlyphDiscordBot] New Command by " + m.Author.Username + ": " + m.Content)
		_, _ = s.ChannelMessageSend(m.ChannelID, foodtoday())
	case "/food tomorrow":
		log.Println("[GlyphDiscordBot] New Command by " + m.Author.Username + ": " + m.Content)
		_, _ = s.ChannelMessageSend(m.ChannelID, foodtomorrow())

	// Music commands
	case "/play":
		log.Println("[GlyphDiscordBot] New Command by " + m.Author.Username + ": " + m.Content)
		parsePlayCommand(s, m)
	case "/stop":
		log.Println("[GlyphDiscordBot] New Command by " + m.Author.Username + ": " + m.Content)
		parseStopCommand(s, m)
	case "/queue":
		log.Println("[GlyphDiscordBot] New Command by " + m.Author.Username + ": " + m.Content)
		parseQueueCommand(s, m)
	case "/pause":
		log.Println("[GlyphDiscordBot] New Command by " + m.Author.Username + ": " + m.Content)
		_, _ = s.ChannelMessageSend(m.ChannelID, "Not implemented yet!")
	case "/remove":
		log.Println("[GlyphDiscordBot] New Command by " + m.Author.Username + ": " + m.Content)
		parseRemoveCommand(s, m)
	case "/volume":
		log.Println("[GlyphDiscordBot] New Command by " + m.Author.Username + ": " + m.Content)
		_, _ = s.ChannelMessageSend(m.ChannelID, "Not implemented yet!")
	case "/echo":
		log.Println("[GlyphDiscordBot] New Command by " + m.Author.Username + ": " + m.Content)
		echo(s, m)

	// Config commands
	case "/save":
		if len(inputString) < 2 {
			log.Println("[GlyphDiscordBot] New Command by " + m.Author.Username + ": " + m.Content)
			_, _ = s.ChannelMessageSend(m.ChannelID, "Save Data to the Bot. Currently available:\n - /save initmod x - Save you Init Modifier")
		} else {
			switch inputString[1] {
			case "initmod":
				if len(inputString) < 3 {
					err := del("glyph|discord:" + m.Author.ID + "|initmod")
					if err != nil {
						log.Println("[GlyphDiscordBot] New Command by " + m.Author.Username + ": " + m.Content)
						_, _ = s.ChannelMessageSend(m.ChannelID, "There was an internal error!")
					} else {
						log.Println("[GlyphDiscordBot] New Command by " + m.Author.Username + ": " + m.Content)
						_, _ = s.ChannelMessageSend(m.ChannelID, "Your init modifier was reset.")
					}
				} else {
					initMod, err := strconv.Atoi(inputString[2])
					if err != nil {
						log.Println("[GlyphDiscordBot] New Command by " + m.Author.Username + ": " + m.Content)
						_, _ = s.ChannelMessageSend(m.ChannelID, "There was an error in your command!")
					} else {
						err := setWithTimer("glyph|discord:"+m.Author.ID+"|initmod", strconv.Itoa(initMod), 2*24*time.Hour)
						if err != nil {
							log.Println("[GlyphDiscordBot] New Command by " + m.Author.Username + ": " + m.Content)
							_, _ = s.ChannelMessageSend(m.ChannelID, "There was an internal error!")
						} else {
							log.Println("[GlyphDiscordBot] New Command by " + m.Author.Username + ": " + m.Content)
							_, _ = s.ChannelMessageSend(m.ChannelID, "Your init modifier was set to "+strconv.Itoa(initMod)+".")
						}
					}
				}
			default:
				_, _ = s.ChannelMessageSend(m.ChannelID, "Sorry, I dont know what to save here!")
			}
		}

	// MISC
	case "/ping":
		log.Println("[GlyphDiscordBot] New Command by " + m.Author.Username + ": " + m.Content)
		_, _ = s.ChannelMessageSend(m.ChannelID, "Pong!")
	case "/id":
		_, _ = s.ChannelMessageSend(m.ChannelID, "Your ID is:\n"+m.Author.ID)
		//default:
		//log.Println("[GlyphDiscordBot] Logged Unknown Command by " + m.Author.Username + ": "+ m.Content)
	case "/updateStatus":
		if m.Author.ID == discordAdminID {
			newStatus := strings.TrimPrefix(m.Content, "/updateStatus ")
			err := setWithTimer("dgStatus", newStatus, 7*24*time.Hour)
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
		_, _ = s.ChannelMessageSend(m.ChannelID, m.Author.String())
	case "/todo":
		_, _ = s.ChannelMessageSend(m.ChannelID, "Feature still in Development")
	case "/amiadmin":
		hasRole, err := memberHasRole(s, m.GuildID, m.Author.ID, councilman)
		if err != nil {
			log.Println("[GlyphDiscordBot] Error while checking if member has role: ", err)
			_, _ = s.ChannelMessageSend(m.ChannelID, "An error occurred!")
		}
		if hasRole {
			_, _ = s.ChannelMessageSend(m.ChannelID, "TRUE")
		} else {
			_, _ = s.ChannelMessageSend(m.ChannelID, "FALSE")
		}
	case "/kick":
		hasRole, err := memberHasRole(s, m.GuildID, m.Author.ID, councilman)
		if err != nil {
			log.Println("[GlyphDiscordBot] Error while checking if member has role: ", err)
			_, _ = s.ChannelMessageSend(m.ChannelID, "An error occurred!")
		}
		if hasRole {
			_, _ = s.ChannelMessageSend(m.ChannelID, "Not implemented yet!")
		} else {
			_, _ = s.ChannelMessageSend(m.ChannelID, "You are not authorized to execute this command!")
		}
	case "onlinecheck":
		_, _ = s.ChannelMessageSend(m.ChannelID, "I'm online")
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
}

func rollHelper(s *discordgo.Session, m *discordgo.MessageCreate) {
	// Catch errors in command
	inputString := strings.Split(m.Content, " ")
	if len(inputString) < 2 {
		_, _ = s.ChannelMessageSend(m.ChannelID, "There was an error in your command!")
		return
	}

	// Catch simple commands
	switch inputString[1] {
	case "one":
		_, _ = s.ChannelMessageSend(m.ChannelID, "Simple 1D10 = "+strconv.Itoa(roll1D10()))
		return
	case "chance":
		diceRollResult := roll1D10()
		switch diceRollResult {
		case 10:
			_, _ = s.ChannelMessageSend(m.ChannelID, "**Success!** Your Chance Die showed a 10!")
		case 1:
			_, _ = s.ChannelMessageSend(m.ChannelID, "**Fail!** Your Chance Die failed spectacularly!")
		default:
			_, _ = s.ChannelMessageSend(m.ChannelID, "Fail! You rolled a **"+strconv.Itoa(diceRollResult)+"** on your Chance die!")
		}
		return
	case "init":
		initModString := get("glyph|discord:" + m.Author.ID + "|initmod")
		initMod, err := strconv.Atoi(initModString)
		if err != nil {
			initModString = ""
		}
		if initModString == "" {
			_, _ = s.ChannelMessageSend(m.ChannelID, "No init modifier saved, here's a simple D10 throw:\n1D10 = "+strconv.Itoa(roll1D10()))
		} else {
			diceResult := roll1D10()
			endResult := diceResult + initMod
			_, _ = s.ChannelMessageSend(m.ChannelID, "Your Initiative is: **"+strconv.Itoa(endResult)+"**\n"+strconv.Itoa(diceResult)+" + "+strconv.Itoa(initMod)+" = "+strconv.Itoa(endResult))
		}
		return
	}

	// Check which dice designation is used
	if strings.Contains(inputString[1], "d") {
		// Catch error in dice designation [/roll 1*s*10 ]
		diceIndex := strings.Split(inputString[1], "d")
		if len(diceIndex) < 2 {
			_, _ = s.ChannelMessageSend(m.ChannelID, "There was an error in your command!")
			return
		}
		sides, err := strconv.Atoi(diceIndex[1])
		if err != nil {
			_, _ = s.ChannelMessageSend(m.ChannelID, "There was an error in your command!")
			return
		}
		amount, err := strconv.Atoi(diceIndex[0])
		if err != nil {
			_, _ = s.ChannelMessageSend(m.ChannelID, "There was an error in your command!")
			return
		}

		// Catch d-notation and read modifiers
		if amount < 1 {
			_, _ = s.ChannelMessageSend(m.ChannelID, "Nice try!")
			return
		}
		switch {
		case sides < 1:
			_, _ = s.ChannelMessageSend(m.ChannelID, "Nice try!")
			return
		case sides == 1:
			_, _ = s.ChannelMessageSend(m.ChannelID, "Really? That's one times "+diceIndex[0]+". I think you can do the math yourself!")
			return
		default:
			if amount > 1000 {
				_, _ = s.ChannelMessageSend(m.ChannelID, "Maybe try a few less dice. We're not playing Warhammer Ultimate here.")
				return
			}
			retSlice := rollXSidedDie(amount, sides)
			var retString strings.Builder
			endResult := 0
			retString.Write([]byte(diceIndex[0] + "d" + diceIndex[1] + ": "))
			for i := range retSlice {
				endResult += retSlice[i]
				if i != len(retSlice)-1 {
					retString.Write([]byte(strconv.Itoa(retSlice[i]) + " + "))
				} else {
					retString.Write([]byte(strconv.Itoa(retSlice[i]) + " = " + strconv.Itoa(endResult)))
				}
			}
			_, _ = s.ChannelMessageSend(m.ChannelID, retString.String())
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
		_, _ = s.ChannelMessageSend(m.ChannelID, retString)
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
			_, _ = s.ChannelMessageSend(m.ChannelID, "There was an error in your command!")
			return
		}
		if throwCount > 1000 {
			_, _ = s.ChannelMessageSend(m.ChannelID, "Don't you think that are a few to many dice to throw?")
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
					retSlice = constructRollRoteNoReroll(throwCount)
				} else {
					retSlice = constructRollRote(throwCount)
				}
			} else {
				if noReroll {
					retSlice = constructRolln(throwCount)
				} else {
					log.Println("[Glyph Discord Bot] This should not have been executed! Perhaps an error in syntax?")
					_, _ = s.ChannelMessageSend(m.ChannelID, "There was an error while parsing your input!")
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
				retString += "\nWell that's a **critical failure!**"
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
				retString += "\nNo Success for you! That's bad, isn`t it?"
			}
		}
		_, _ = s.ChannelMessageSend(m.ChannelID, retString)
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

func constructRollRote(throwCount int) [][]int {
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

func constructRollRoteNoReroll(throwCount int) [][]int {
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

// Takes inbound audio and sends it right back out.
func echo(s *discordgo.Session, m *discordgo.MessageCreate) {
	voiceChannel := ""
	g, err := s.State.Guild(m.GuildID)
	if err != nil {
		_, _ = s.ChannelMessageSend(m.ChannelID, "There was an error!")
		log.Println("[GlyphDiscordBot] Error while getting guild states: ", err)
	}
	for _, state := range g.VoiceStates {
		if state.UserID == m.Author.ID {
			voiceChannel = state.ChannelID
			break
		}
	}

	if voiceChannel == "" {
		_, _ = s.ChannelMessageSend(m.ChannelID, "This command only works when you are in a voice channel!")
	}

	// Parse the echo length
	minutesString := strings.TrimPrefix(m.Content, "/echo ")
	minutes, err := strconv.Atoi(minutesString)
	if err != nil {
		minutes = 1
	}
	stopTime := time.Now()
	stopTime = time.Now().Add(time.Duration(minutes) * time.Minute)

	voiceConnection, err := s.ChannelVoiceJoin(m.GuildID, voiceChannel, false, false)
	if err != nil {
		log.Println("[GlyphDiscordBot] Error while connecting to voice channel: ", err)
		_, _ = s.ChannelMessageSend(m.ChannelID, "An Error occurred!")
	}

	recv := make(chan *discordgo.Packet, 2)
	go dgvoice.ReceivePCM(s.VoiceConnections[m.GuildID], recv)

	send := make(chan []int16, 2)
	go dgvoice.SendPCM(s.VoiceConnections[m.GuildID], send)

	_ = s.VoiceConnections[m.GuildID].Speaking(true)
	//defer s.VoiceConnections[m.GuildID].Speaking(false)

	abort := make(chan bool)
	stopVoice[voiceConnection.GuildID] = abort

	for {
		if time.Now().Sub(stopTime) > 0 {
			_ = voiceConnection.Disconnect()
			return
		}
		p, ok := <-recv
		if !ok {
			_ = voiceConnection.Disconnect()
			return
		}

		send <- p.PCM

	}
}

func getYouTubeURL(input string) string {
	if strings.HasPrefix(input, "https://") || strings.HasPrefix(input, "http://") {
		return input
	}
	if input == "" || input == " " || input == "/play" || input == "/play " {
		return "https://youtu.be/dQw4w9WgXcQ"
	}
	// Initiate search here
	return ""
}

func parsePlayCommand(s *discordgo.Session, m *discordgo.MessageCreate) {
	// Parse Youtube URL
	youtubeURL := getYouTubeURL(strings.TrimPrefix(m.Content, "/play "))
	if youtubeURL == "" {
		_, _ = s.ChannelMessageSend(m.ChannelID, "Error parsing your message!")
	}

	// Get Videoinfo
	ctx := context.Background()
	ytdlClient := ytdl.DefaultClient
	videoInfo, err := ytdlClient.GetVideoInfo(ctx, youtubeURL)
	if err != nil {
		_, _ = s.ChannelMessageSend(m.ChannelID, "Could not get Videoinfo!")
	}

	// Check whether music is playing, if not join voice Channel and create queue
	if queueMap[m.GuildID] == nil {
		voiceChannel := ""
		g, err := s.State.Guild(m.GuildID)
		if err != nil {
			_, _ = s.ChannelMessageSend(m.ChannelID, "There was an error!")
			log.Println("[GlyphDiscordBot] Error while getting guild states: ", err)
		}
		for _, state := range g.VoiceStates {
			if state.UserID == m.Author.ID {
				voiceChannel = state.ChannelID
				break
			}
		}

		// Join Voice Channel
		voiceConnection, err := s.ChannelVoiceJoin(m.GuildID, voiceChannel, false, true)
		if err != nil {
			_, _ = s.ChannelMessageSend(m.ChannelID, "Error joining your Voice Channel!")
			log.Println("[GlyphDiscordBot] Error while joining voice channel: ", err)
		}

		// Create Queue
		queueMap[m.GuildID] = make([]*ytdl.VideoInfo, 1)
		queueMap[m.GuildID][0] = videoInfo
		go streamMusic(voiceConnection)
	} else {
		queueMap[m.GuildID] = append(queueMap[m.GuildID], videoInfo)
	}
}

func parseQueueCommand(s *discordgo.Session, m *discordgo.MessageCreate) {
	if queueMap[m.GuildID] == nil {
		_, _ = s.ChannelMessageSend(m.ChannelID, "Nothing playing right now!")
		return
	}
	if len(queueMap[m.GuildID]) < 1 {
		_, _ = s.ChannelMessageSend(m.ChannelID, "Internal Error")
		log.Println("[GlyphDiscordBot] queueMap videoInfo array is impossibly short!")
		return
	}
	var output strings.Builder
	output.WriteString("Current queue:\n")
	for i := 0; i < len(queueMap[m.GuildID]); i++ {
		output.WriteString("[" + strconv.Itoa(i) + "] " + queueMap[m.GuildID][i].Title + "\n")
	}
	_, _ = s.ChannelMessageSend(m.ChannelID, output.String())
}

func parseRemoveCommand(s *discordgo.Session, m *discordgo.MessageCreate) {
	indexString := strings.TrimPrefix(m.Content, "/remove ")
	index, err := strconv.Atoi(indexString)
	if err != nil {
		_, _ = s.ChannelMessageSend(m.ChannelID, "Could not parse your Message, please specify the number of the song to remove!")
	}
	if index > len(queueMap[m.GuildID])-1 || index < 0 {
		_, _ = s.ChannelMessageSend(m.ChannelID, "Please specify the valid number of the song to remove!")
	}
	if index == 0 {
		stopVoice[m.GuildID] <- false
	} else {
		queueMap[m.GuildID] = remove(queueMap[m.GuildID], index)
	}
}

func parseStopCommand(s *discordgo.Session, m *discordgo.MessageCreate) {
	stopVoice[m.GuildID] <- true
	_, _ = s.ChannelMessageSend(m.ChannelID, "Playback stopped")
}

func streamMusic(voiceConnection *discordgo.VoiceConnection) {
	options := dca.StdEncodeOptions
	options.RawOutput = true
	options.Bitrate = 96
	options.Application = "lowdelay"
	ctx := context.Background()
	ytdlClient := ytdl.DefaultClient

	videoInfo := queueMap[voiceConnection.GuildID][0]

	format := videoInfo.Formats.Extremes(ytdl.FormatAudioBitrateKey, true)[0]
	downloadURL, err := ytdlClient.GetDownloadURL(ctx, videoInfo, format)
	if err != nil {
		log.Printf("[GlyphDiscordBot] Error getting download URL for %s: %s\n", videoInfo.Title, err)
		stopVoice[voiceConnection.GuildID] <- false
	}

	encodingSession, err := dca.EncodeFile(downloadURL.String(), options)
	if err != nil {
		log.Printf("[GlyphDiscordBot] Error creating encoding session for %s: %s\n", videoInfo.Title, err)
		stopVoice[voiceConnection.GuildID] <- false
	}

	abort := make(chan bool)
	stopVoice[voiceConnection.GuildID] = abort
	go func(encodingSession *dca.EncodeSession, voiceConnection *discordgo.VoiceConnection, abort chan bool) {
		totalStop := <-abort
		if totalStop {
			_ = encodingSession.Stop()
			encodingSession.Cleanup()
			_ = voiceConnection.Disconnect()
			queueMap[voiceConnection.GuildID] = nil
			return
		} else {
			_ = encodingSession.Stop()
			encodingSession.Cleanup()
			// Check if there are any more songs in queue
			if len(queueMap[voiceConnection.GuildID]) < 2 {
				queueMap[voiceConnection.GuildID] = nil
				_ = voiceConnection.Disconnect()
				return
			}

			queueMap[voiceConnection.GuildID] = remove(queueMap[voiceConnection.GuildID], 0)
			go streamMusic(voiceConnection)
		}
	}(encodingSession, voiceConnection, abort)

	done := make(chan error)
	queueMap[voiceConnection.GuildID][0] = videoInfo
	dca.NewStream(encodingSession, voiceConnection, done)
	err = <-done
	stopVoice[voiceConnection.GuildID] <- false
	if err != nil && err != io.EOF {
		log.Printf("[GlyphDiscordBot] Error while ending Stream for %s: %s\n", videoInfo.Title, err)
		stopVoice[voiceConnection.GuildID] <- false
	}
}

// Remove element x from videoInfo slice
func remove(slice []*ytdl.VideoInfo, x int) []*ytdl.VideoInfo {
	return append(slice[:x], slice[x+1:]...)
}
