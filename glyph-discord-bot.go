package main

import (
	"math/rand"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/bwmarrin/discordgo"
	_ "github.com/heroku/x/hmetrics/onload"
	"github.com/keybase/go-logging"
	"github.com/rylio/ytdl"
)

// Discord ID of admin
var discordAdminID string

// Read write lock for the voice update map
var voiceUpdateLock = sync.RWMutex{}

// A map of stream updates that modify the playback indexed after guildIDs
var voiceUpdate map[string]chan updateStream

// Read write lock for the queuemap
var queueMapLock sync.RWMutex

// A map of queue represented as ytdl.VideoInfo arrays indexed after guildIDs
var queueMap map[string][]*ytdl.VideoInfo

// Read write lock for the volume map
var volumeMapLock sync.RWMutex

// Volume Map
var volumeMap map[string]int

// Needed for onlyonce execution of random source
var onlyOnce sync.Once

var glyphDiscordLog = logging.MustGetLogger("glyphDiscord")

type updateStream struct {
	index int // 0 means stop the stream, 1 means skip, 2 means pause and 3 means volume update
}

// Main and Init
func glyphDiscordBot() {
	discordAdminID = "259076782408335360"
	voiceUpdate = make(map[string]chan updateStream)
	queueMap = make(map[string][]*ytdl.VideoInfo)
	volumeMap = make(map[string]int)

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
	dgStatus, err := getError("dgStatus")
	if err != nil {
		glyphDiscordLog.Warning("Error getting dgStatus from redis: ", err)
		dgStatus = "planning world domination"
	}
	_ = dg.UpdateStatus(0, dgStatus)

	// Wait here until CTRL-C or other term signal is received.
	glyphDiscordLog.Info("Glyph Discord Bot was started.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, syscall.SIGQUIT, syscall.SIGHUP)
	<-sc

	// Cleanly close down the Discord session.
	voiceUpdateLock.RLock()
	for _, abort := range voiceUpdate {
		abort <- updateStream{0}
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
			glyphDiscordLog.Info("New Command by " + m.Author.Username + ": " + m.Content)
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
			glyphDiscordLog.Info("New Command by " + m.Author.Username + ": " + m.Content)
			_, _ = s.ChannelMessageSend(m.ChannelID, "To roll construct dice just tell me how many I should roll and what Modifiers I shall apply.\nI can also roll custom dice like this: /roll 3d12")
		} else {
			rollHelper(s, m)
		}
	case "/r":
		rollHelper(s, m)

	// Help commands
	case "/help":
		glyphDiscordLog.Info(m.Author.Username + ": " + m.Content)
		_, _ = s.ChannelMessageSend(m.ChannelID, "Available Command Categories:\n - General Tasadar Network - /tn help\n - Music Bot - /music help\n - Uni Passau - /unip help\n - PnP Tools - /pnp help")
	case "/music":
		glyphDiscordLog.Info(m.Author.Username + ": " + m.Content)
		_, _ = s.ChannelMessageSend(m.ChannelID, "Available Music Bot Commands:\n - /play - Play a song with a given youtube URL, or add it to the queue if music is already playing. \n - /stop - Stop all music \n - /queue - Show current queue \n - /pause - Pause current playback \n - /remove - Remove song number x from queue\n - /volume - set the volume to specified value\n - /skip - Skip the currently running song")
	case "/unip":
		glyphDiscordLog.Info(m.Author.Username + ": " + m.Content)
		_, _ = s.ChannelMessageSend(m.ChannelID, "Available Commands:\n/food - Food for today\n/food tomorrow - Food for tomorrow")
	case "/pnp":
		glyphDiscordLog.Info(m.Author.Username + ": " + m.Content)
		_, _ = s.ChannelMessageSend(m.ChannelID, "Available Commands:\n - /roll - Roll Dice after construct rules\n - /save initmod - Save your init modifier\n - /construct or /co - get construct-specific help")
	case "/co", "/construct":
		if len(inputString) < 2 {
			glyphDiscordLog.Info(m.Author.Username + ": " + m.Content)
			_, _ = s.ChannelMessageSend(m.ChannelID, "Available Commands:\n - /co trait TRAITNAME - Get Description for specified trait")
		} else {
			switch inputString[1] {
			// Construct command here
			default:
				glyphDiscordLog.Info(m.Author.Username + ": " + m.Content)
				_, _ = s.ChannelMessageSend(m.ChannelID, "Invalid Command Syntax")
			}
		}

	// Food commands
	case "/food":
		glyphDiscordLog.Info(m.Author.Username + ": " + m.Content)
		_, _ = s.ChannelMessageSend(m.ChannelID, foodtoday())
	case "/food tomorrow":
		glyphDiscordLog.Info(m.Author.Username + ": " + m.Content)
		_, _ = s.ChannelMessageSend(m.ChannelID, foodtomorrow())

	// Music commands
	/*case "/play":
		glyphDiscordLog.Info(m.Author.Username + ": " + m.Content)
		parsePlayCommand(s, m)
	case "/stop":
		glyphDiscordLog.Info(m.Author.Username + ": " + m.Content)
		parseStopCommand(m)
	case "/queue":
		glyphDiscordLog.Info(m.Author.Username + ": " + m.Content)
		parseQueueCommand(s, m)
	case "/pause":
		glyphDiscordLog.Info(m.Author.Username + ": " + m.Content)
		parsePauseCommand(s, m)
	case "/remove":
		glyphDiscordLog.Info(m.Author.Username + ": " + m.Content)
		parseRemoveCommand(s, m)
	case "/volume":
		glyphDiscordLog.Info(m.Author.Username + ": " + m.Content)
		parseVolumeCommand(s, m)
	case "/skip":
		glyphDiscordLog.Info(m.Author.Username + ": " + m.Content)
		parseSkipCommand(s, m)
	case "/echo":
		glyphDiscordLog.Info(m.Author.Username + ": " + m.Content)
		echo(s, m)*/

	// Config commands
	case "/save":
		if len(inputString) < 2 {
			glyphDiscordLog.Info(m.Author.Username + ": " + m.Content)
			_, _ = s.ChannelMessageSend(m.ChannelID, "Save Data to the Bot. Currently available:\n - /save initmod x - Save you Init Modifier")
		} else {
			switch inputString[1] {
			case "initmod":
				if len(inputString) < 3 {
					err := del("glyph|discord:" + m.Author.ID + "|initmod")
					if err != nil {
						glyphDiscordLog.Info(m.Author.Username + ": " + m.Content)
						_, _ = s.ChannelMessageSend(m.ChannelID, "There was an internal error!")
					} else {
						glyphDiscordLog.Info(m.Author.Username + ": " + m.Content)
						_, _ = s.ChannelMessageSend(m.ChannelID, "Your init modifier was reset.")
					}
				} else {
					initMod, err := strconv.Atoi(inputString[2])
					if err != nil {
						glyphDiscordLog.Info(m.Author.Username + ": " + m.Content)
						_, _ = s.ChannelMessageSend(m.ChannelID, "There was an error in your command!")
					} else {
						err := setWithTimer("glyph|discord:"+m.Author.ID+"|initmod", strconv.Itoa(initMod), 2*24*time.Hour)
						if err != nil {
							glyphDiscordLog.Info(m.Author.Username + ": " + m.Content)
							_, _ = s.ChannelMessageSend(m.ChannelID, "There was an internal error!")
						} else {
							glyphDiscordLog.Info(m.Author.Username + ": " + m.Content)
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
		glyphDiscordLog.Info(m.Author.Username + ": " + m.Content)
		_, _ = s.ChannelMessageSend(m.ChannelID, "Pong!")
	case "/id":
		_, _ = s.ChannelMessageSend(m.ChannelID, "Your ID is:\n"+m.Author.ID)
		//default:
		//glyphDiscordLog.Info("Logged Unknown Command by " + m.Author.Username + ": "+ m.Content)
	case "/updateStatus":
		if m.Author.ID == discordAdminID {
			newStatus := strings.TrimPrefix(m.Content, "/updateStatus ")
			err := setWithTimer("dgStatus", newStatus, 7*24*time.Hour)
			if err != nil {
				glyphDiscordLog.Warning("Error setting dgStatus on Redis: ", err)
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
	}
}

// Check if a user has a given role in a given guild
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

// Parse roll command
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
		_, _ = s.ChannelMessageSend(m.ChannelID, "Simple 1D10 = "+strconv.Itoa(rollXSidedDie(1, 10)[0]))
		return
	case "chance":
		diceRollResult := rollXSidedDie(1, 10)[0]
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
			_, _ = s.ChannelMessageSend(m.ChannelID, "No init modifier saved, here's a simple D10 throw:\n1D10 = "+strconv.Itoa(rollXSidedDie(1, 10)[0]))
		} else {
			diceResult := rollXSidedDie(1, 10)[0]
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
		// Roll a chance die
		result := rollXSidedDie(1, 10)[0]
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
		// Assume that input was construct notation
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
					_, _ = s.ChannelMessageSend(m.ChannelID, "There was an error while parsing your input!")
					return
				}
			}
		}
		// Count Successes and Ones while parsing the return String
		// Parse Slice here
		var successes, ones int
		var output strings.Builder
		output.WriteString("Results: ")
		for i := range retSlice {
			output.WriteString("[")
			for j := range retSlice[i] {
				output.WriteString(strconv.Itoa(retSlice[i][j]))
				if j != len(retSlice[i])-1 {
					output.WriteString(" ⮞ ")
				}
				switch retSlice[i][j] {
				case 8, 9, 10:
					successes++
				case 1:
					ones++
				}
			}
			output.WriteString("] ")
			output.WriteString(" ")
		}
		if ones >= (throwCount/2 + 1) {
			if successes == 0 {
				output.WriteString("\nWell that's a **critical failure!**")
			} else {
				output.WriteString("\nThat was nearly a critical failure! But you had **" + strconv.Itoa(successes) + "** Successes!")
			}
		} else {
			if successes > 0 {
				if successes >= 5 {
					mentionString := m.Author.Mention()
					output.WriteString("\nThat were **" + strconv.Itoa(successes) + "** Successes!\n" + mentionString + " That was exceptional!")
				} else {
					if successes == 1 {
						output.WriteString("\nThat was **1** Success!")
					} else {
						output.WriteString("\nThat were **" + strconv.Itoa(successes) + "** Successes!")
					}
				}
			} else {
				output.WriteString("\nNo Success for you! That's bad, isn`t it?")
			}
		}
		_, _ = s.ChannelMessageSend(m.ChannelID, output.String())
	}
}

// Role x dice with given amount of sides
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

// Roll a construct roll without special modifiers
func normalConstructRoll(throwCount int) [][]int {
	retSlice := make([][]int, throwCount)
	for i := range retSlice {
		retSlice[i] = []int{}
		repeat := true
		for repeat {
			diceResult := rollXSidedDie(1, 10)[0]
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

// Roll a construct roll with 8 again
func constructRoll8(throwCount int) [][]int {
	retSlice := make([][]int, throwCount)
	for i := range retSlice {
		retSlice[i] = []int{}
		repeat := true
		for repeat {
			diceResult := rollXSidedDie(1, 10)[0]
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

// Roll a construct roll with 8 again and rote quality
func constructRoll8r(throwCount int) [][]int {
	retSlice := make([][]int, throwCount)
	isFirstReroll := true
	for i := range retSlice {
		retSlice[i] = []int{}
		repeat := true
		for repeat {
			diceResult := rollXSidedDie(1, 10)[0]
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

// Roll a construct roll with 9 again
func constructRoll9(throwCount int) [][]int {
	retSlice := make([][]int, throwCount)
	for i := range retSlice {
		retSlice[i] = []int{}
		repeat := true
		for repeat {
			diceResult := rollXSidedDie(1, 10)[0]
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

// Roll a construct roll with 9 again and rote quality
func constructRoll9r(throwCount int) [][]int {
	retSlice := make([][]int, throwCount)
	isFirstReroll := true
	for i := range retSlice {
		retSlice[i] = []int{}
		repeat := true
		for repeat {
			diceResult := rollXSidedDie(1, 10)[0]
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

// Roll a construct roll with no rerolls
func constructRolln(throwCount int) [][]int {
	retSlice := make([][]int, throwCount)
	for i := range retSlice {
		retSlice[i] = []int{}
		diceResult := rollXSidedDie(1, 10)[0]
		previousSlice := retSlice[i]
		if previousSlice == nil {
			previousSlice = []int{}
		}
		tmpSlice := append(previousSlice, diceResult)
		retSlice[i] = tmpSlice

	}
	return retSlice
}

// Roll a construct roll with rote quality
func constructRollRote(throwCount int) [][]int {
	retSlice := make([][]int, throwCount)
	isFirstReroll := true
	for i := range retSlice {
		retSlice[i] = []int{}
		repeat := true
		for repeat {
			diceResult := rollXSidedDie(1, 10)[0]
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

// Roll a construct roll with rote quality but no reroll on 10
func constructRollRoteNoReroll(throwCount int) [][]int {
	retSlice := make([][]int, throwCount)
	isFirstReroll := true
	for i := range retSlice {
		retSlice[i] = []int{}
		repeat := true
		for repeat {
			repeat = false
			diceResult := rollXSidedDie(1, 10)[0]
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

/*
// Takes inbound audio and sends it right back out.
func echo(s *discordgo.Session, m *discordgo.MessageCreate) {
	voiceChannel := ""
	g, err := s.State.Guild(m.GuildID)
	if err != nil {
		_, _ = s.ChannelMessageSend(m.ChannelID, "There was an error!")
		glyphDiscordLog.Warning("Error while getting guild states: ", err)
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
		glyphDiscordLog.Warning("Error while connecting to voice channel: ", err)
		_, _ = s.ChannelMessageSend(m.ChannelID, "An Error occurred!")
	}

	recv := make(chan *discordgo.Packet, 2)
	go dgvoice.ReceivePCM(s.VoiceConnections[m.GuildID], recv)

	send := make(chan []int16, 2)
	go dgvoice.SendPCM(s.VoiceConnections[m.GuildID], send)

	_ = s.VoiceConnections[m.GuildID].Speaking(true)
	defer s.VoiceConnections[m.GuildID].Speaking(false)

	for {
		if time.Until(stopTime) <= 0 {
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

// Parse the input string and return a youtube link
func getYouTubeURL(input string) string {
	if strings.HasPrefix(input, "https://www.youtube.com") || strings.HasPrefix(input, "http://www.youtube.com") ||
		strings.HasPrefix(input, "https://youtube.com") || strings.HasPrefix(input, "http://youtube.com") ||
		strings.HasPrefix(input, "https://youtu.be") || strings.HasPrefix(input, "http://youtu.be") {
		return input
	}
	if input == "" || input == " " || input == "/play" || input == "/play " {
		return "https://youtu.be/dQw4w9WgXcQ"
	}
	// Initiate search here
	return ""
}

// Parse the play command
func parsePlayCommand(s *discordgo.Session, m *discordgo.MessageCreate) {
	// Parse Youtube URL
	youtubeURL := getYouTubeURL(strings.TrimPrefix(m.Content, "/play "))
	if youtubeURL == "" {
		_, _ = s.ChannelMessageSend(m.ChannelID, "Error parsing your message!")
		return
	}

	// Get Videoinfo
	ctx := context.Background()
	ytdlClient := ytdl.DefaultClient
	videoInfo, err := ytdlClient.GetVideoInfo(ctx, youtubeURL)
	if err != nil {
		_, _ = s.ChannelMessageSend(m.ChannelID, "Could not get Videoinfo!")
		return
	}

	// Check whether music is playing, if not join voice Channel and create queue
	queueMapLock.RLock()
	queue := queueMap[m.GuildID]
	queueMapLock.RUnlock()
	if queue == nil {
		voiceChannel := ""
		g, err := s.State.Guild(m.GuildID)
		if err != nil {
			_, _ = s.ChannelMessageSend(m.ChannelID, "There was an error!")
			glyphDiscordLog.Warning("Error while getting guild states: ", err)
		}
		for _, state := range g.VoiceStates {
			if state.UserID == m.Author.ID {
				voiceChannel = state.ChannelID
				break
			}
		}

		if voiceChannel == "" {
			_, _ = s.ChannelMessageSend(m.ChannelID, "You must be in a voice channel to execute this!")
			return
		}

		// Join Voice Channel
		voiceConnection, err := s.ChannelVoiceJoin(m.GuildID, voiceChannel, false, true)
		if err != nil {
			_, _ = s.ChannelMessageSend(m.ChannelID, "Error joining your Voice Channel!")
			glyphDiscordLog.Warning("Error while joining voice channel: ", err)
		}

		// Create Queue
		queueMapLock.Lock()
		queueMap[m.GuildID] = make([]*ytdl.VideoInfo, 1)
		queueMap[m.GuildID][0] = videoInfo
		queueMapLock.Unlock()
		go streamMusic(voiceConnection)
	} else {
		queueMapLock.Lock()
		queueMap[m.GuildID] = append(queueMap[m.GuildID], videoInfo)
		queueMapLock.Unlock()
	}
}

func parseVolumeCommand(s *discordgo.Session, m *discordgo.MessageCreate) {
	queueMapLock.RLock()
	queue := queueMap[m.GuildID]
	queueMapLock.RUnlock()
	if queue == nil {
		_, _ = s.ChannelMessageSend(m.ChannelID, "Nothing playing right now!")
		return
	}
	volumeString := strings.TrimPrefix(m.Content, "/volume ")
	volume, err := strconv.Atoi(volumeString)
	if err != nil {
		_, _ = s.ChannelMessageSend(m.ChannelID, "Please specify a valid volume value")
		return
	}
	if volume == 0 {
		volume = -1
	}
	volumeMapLock.RLock()
	volumeMap[m.GuildID] = volume
	volumeMapLock.RUnlock()
	voiceUpdateLock.RLock()
	update := voiceUpdate[m.GuildID]
	voiceUpdateLock.RUnlock()
	update <- updateStream{3}
}

// Parse the queue command
func parseQueueCommand(s *discordgo.Session, m *discordgo.MessageCreate) {
	queueMapLock.RLock()
	queue := queueMap[m.GuildID]
	queueMapLock.RUnlock()
	if queue == nil {
		_, _ = s.ChannelMessageSend(m.ChannelID, "Nothing playing right now!")
		return
	}
	if len(queue) < 1 {
		_, _ = s.ChannelMessageSend(m.ChannelID, "Internal Error")
		glyphDiscordLog.Warning("queueMap videoInfo array is impossibly short!")
		return
	}
	var output strings.Builder
	output.WriteString("Current queue:\n")
	for i := 0; i < len(queue); i++ {
		output.WriteString("[" + strconv.Itoa(i) + "] " + queue[i].Title + "\n")
	}
	_, _ = s.ChannelMessageSend(m.ChannelID, output.String())
}

func parseSkipCommand(s *discordgo.Session, m *discordgo.MessageCreate) {
	queueMapLock.RLock()
	queue := queueMap[m.GuildID]
	queueMapLock.RUnlock()
	if queue == nil {
		_, _ = s.ChannelMessageSend(m.ChannelID, "Nothing playing right now!")
		return
	}
	voiceUpdateLock.RLock()
	update := voiceUpdate[m.GuildID]
	voiceUpdateLock.RUnlock()
	update <- updateStream{1}
}

// Parse the remove command
func parseRemoveCommand(s *discordgo.Session, m *discordgo.MessageCreate) {
	queueMapLock.RLock()
	queue := queueMap[m.GuildID]
	queueMapLock.RUnlock()
	if queue == nil {
		_, _ = s.ChannelMessageSend(m.ChannelID, "Nothing playing right now!")
		return
	}
	indexString := strings.TrimPrefix(m.Content, "/remove ")
	index, err := strconv.Atoi(indexString)
	if err != nil {
		_, _ = s.ChannelMessageSend(m.ChannelID, "Could not parse your Message, please specify the number of the song to remove!")
	}
	if index > len(queue)-1 || index < 0 {
		_, _ = s.ChannelMessageSend(m.ChannelID, "Please specify the valid number of the song to remove!")
	}
	if index == 0 {
		voiceUpdateLock.RLock()
		update := voiceUpdate[m.GuildID]
		voiceUpdateLock.RUnlock()
		update <- updateStream{1}
	} else {
		queueMapLock.Lock()
		queueMap[m.GuildID] = removeFromVideoInfoArray(queue, index)
		queueMapLock.Unlock()
	}
}

// Parse the stop command
func parseStopCommand(m *discordgo.MessageCreate) {
	voiceUpdateLock.RLock()
	update := voiceUpdate[m.GuildID]
	voiceUpdateLock.RUnlock()
	update <- updateStream{0}
}

// Parse the pause command
func parsePauseCommand(s *discordgo.Session, m *discordgo.MessageCreate) {
	voiceUpdateLock.RLock()
	update := voiceUpdate[m.GuildID]
	voiceUpdateLock.RUnlock()
	update <- updateStream{2}
}

// Stream music on voiceConnection
func streamMusic(voiceConnection *discordgo.VoiceConnection) {
	volumeMapLock.RLock()
	volume := volumeMap[voiceConnection.GuildID]
	volumeMapLock.RUnlock()

	switch {
	case volume == -1:
		volume = 0
	case volume == 0:
		volume = 100 // TODO guild default value here
	case volume > 512:
		volume = 512
	case volume < -1:
		volume = 0
	}

	options := dca.StdEncodeOptions
	options.RawOutput = true
	options.Bitrate = 96
	options.Application = "lowdelay"
	options.Volume = volume
	ctx := context.Background()
	ytdlClient := ytdl.DefaultClient

	queueMapLock.RLock()
	videoInfo := queueMap[voiceConnection.GuildID][0]
	queueMapLock.RUnlock()

	format := videoInfo.Formats.Extremes(ytdl.FormatAudioBitrateKey, true)[0]
	downloadURL, err := ytdlClient.GetDownloadURL(ctx, videoInfo, format)
	if err != nil {
		glyphDiscordLog.Warning("Error getting download URL for %s: %s\n", videoInfo.Title, err)
		voiceUpdateLock.RLock()
		update := voiceUpdate[voiceConnection.GuildID]
		voiceUpdateLock.RUnlock()
		update <- updateStream{1}
	}

	encodingSession, err := dca.EncodeFile(downloadURL.String(), options)
	if err != nil {
		glyphDiscordLog.Warning("Error creating encoding session for %s: %s\n", videoInfo.Title, err)
		voiceUpdateLock.RLock()
		update := voiceUpdate[voiceConnection.GuildID]
		voiceUpdateLock.RUnlock()
		update <- updateStream{1}
	}

	// Start Stream
	done := make(chan error)
	stream := dca.NewStream(encodingSession, voiceConnection, done)

	// Initialize stream control routine
	update := make(chan updateStream)
	voiceUpdateLock.Lock()
	voiceUpdate[voiceConnection.GuildID] = update
	voiceUpdateLock.Unlock()
	streamControl(encodingSession, voiceConnection, update, stream)

	// End stream
	err = <-done
	voiceUpdateLock.RLock()
	update = voiceUpdate[voiceConnection.GuildID]
	voiceUpdateLock.RUnlock()
	update <- updateStream{1}

	if err != nil && err != io.EOF {
		glyphDiscordLog.Warning("Error while ending Stream for %s: %s\n", videoInfo.Title, err)
		voiceUpdateLock.RLock()
		update := voiceUpdate[voiceConnection.GuildID]
		voiceUpdateLock.RUnlock()
		update <- updateStream{1}
	}
}

func streamControl(encodingSession *dca.EncodeSession, voiceConnection *discordgo.VoiceConnection, update chan updateStream, stream *dca.StreamingSession) {
	for {
		currentUpdate := <-update
		switch currentUpdate.index {
		case 0:
			// Total Stop
			_ = encodingSession.Stop()
			encodingSession.Cleanup()
			_ = voiceConnection.Disconnect()
			queueMapLock.Lock()
			queueMap[voiceConnection.GuildID] = nil
			queueMapLock.Unlock()
			return
		case 1:
			// Next song
			// Stop encoding session
			_ = encodingSession.Stop()
			encodingSession.Cleanup()

			queueMapLock.RLock()
			queue := queueMap[voiceConnection.GuildID]
			queueMapLock.RUnlock()

			// Check if there are any more songs in queue
			if len(queue) < 2 {
				queueMapLock.Lock()
				queueMap[voiceConnection.GuildID] = nil
				queueMapLock.Unlock()
				_ = voiceConnection.Disconnect()
				return
			}

			// Modify queue and handle next song
			queueMapLock.Lock()
			queueMap[voiceConnection.GuildID] = removeFromVideoInfoArray(queue, 0)
			queueMapLock.Unlock()
			go streamMusic(voiceConnection)
			return
		case 2:
			// Handle pausing event
			if stream.Paused() {
				stream.SetPaused(false)
			} else {
				stream.SetPaused(true)
			}
		case 3:
			// Restart current song with new volume
			// Stop encoding session
			_ = encodingSession.Stop()
			encodingSession.Cleanup()
			go streamMusic(voiceConnection)
			return
		}
	}
}

// Remove element x from videoInfo slice
func removeFromVideoInfoArray(slice []*ytdl.VideoInfo, x int) []*ytdl.VideoInfo {
	return append(slice[:x], slice[x+1:]...)
}
*/
