package glyph

import (
	"errors"
	"math"
	"math/rand"
	"strconv"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	UniPassauBot "github.com/tionis/uni-passau-bot/api"
)

// ErrNoCommandMatched represents the state in which no command could be matched
var ErrNoCommandMatched = errors.New("no command was matched")
var standardContextDelay = time.Minute * 5

// MessageData represents an message the bot can act on with callback functions
type MessageData struct {
	Content              string                                                   `json:"content,required"`
	AuthorID             string                                                   `json:"authorID,omitempty"`
	IsDM                 bool                                                     `json:"isDM,omitempty"`
	SupportsMarkdown     bool                                                     `json:"supportsMarkdown,omitempty"`
	ChannelID            string                                                   `json:"channelID,omitempty"`
	GetMention           func(userID string) string                               `json:"getMention,omitempty"`          // A function that when passed an userID returns an string mentioning the user
	GetContext           func(userID, channelID string) string                    `json:"getContext,omitempty"`          // A function that when passed an channelID and UserID returns the current chat context
	SetContext           func(userID, channelID, value string, ttl time.Duration) `json:"setContext,omitempty"`          // A function that allows setting the current channelID+UserID context
	SetUserData          func(userID, key string, value interface{})              `json:"saveUserData,omitempty"`        // A function that saves data to a specific user by key
	GetUserData          func(userID, key string) interface{}                     `json:"getUserData,omitempty"`         // A function that gets data to a specific user by key
	SendMessageToChannel func(channelID, message string)                          `json:"sendMessageToChannel,required"` // Send a simple text message to specified channel
}

// HandleAll takes a MessageData object and parses it for the glyph bot, calling callback functions as needed
func HandleAll(message MessageData) error {
	tokens := strings.Split(message.Content, " ")
	switch tokens[0] {
	case "help":
		go handleHelp(message)
	case "unip":
		go handleUnip(message)
	case "/help":
		go handleHelp(message)
	case "/unip":
		go handleUnip(message)
	case "/pnp":
		go handlePnPHelp(message)

	// Food commands
	case "/food":
		go handleFoodToday(message)
	case "/food tomorrow":
		go handleFoodTomorrow(message)

	// Config commands
	case "/config":
		go handleConfig(message)

	// MISC commands
	case "/ping":
		go handlePing(message)
	case "/id":
		go handleID(message)
	case "/isDM":
		go handleIsDM(message)

	case "/roll", "/r":
		if len(tokens) < 2 {
			message.SendMessageToChannel(message.ChannelID, "To roll dice just tell me how many I should roll and what Modifiers I shall apply.\nI can also roll custom dice like this: /roll 3d12")
		} else {
			rollHelper(s, m)
		}

	// Diagnostic Commands
	case "/diag":
		switch tokens[1] {
		case "dice":
			diceDiagnosticHelper(s, m)
		default:
			message.SendMessageToChannel(message.ChannelID, "Unknown Command!")
		}
	// Help commands
	case "/gm":
		switch tokens[1] {
		case "help":
			message.SendMessageToChannel(message.ChannelID, "Available Commands:\n - /gm rollinit COUNT INIT")
		case "rollinit":
			rollCount, err := strconv.Atoi(tokens[2])
			rollInit, err := strconv.Atoi(tokens[3])
			if err != nil {
				message.SendMessageToChannel(message.ChannelID, "There was an error in your command!")
			}
			message.SendMessageToChannel(message.ChannelID, rollMassInit(rollCount, rollInit))
		}
	}

	if message.GetContext(message.AuthorID, message.ChannelID) != "" {
		go message.SetContext(message.AuthorID, message.ChannelID, "", time.Second)
		go handleGenericError(message)
	}
	return ErrNoCommandMatched
}

func handleHelp(message MessageData) {
	if message.SupportsMarkdown {
		message.SendMessageToChannel(message.ChannelID, "# Available Command Categories:\n - Uni Passau - /unip help\n - PnP Tools - /pnp help")
	} else {
		message.SendMessageToChannel(message.ChannelID, "Available Command Categories:\n - Uni Passau - /unip help\n - PnP Tools - /pnp help")
	}
}

func handleUnip(message MessageData) {
	message.SendMessageToChannel(message.ChannelID, "Available Commands:\n/food - Food for today\n/food tomorrow - Food for tomorrow")
}

func handlePnPHelp(message MessageData) {
	message.SendMessageToChannel(message.ChannelID, "Available Commands:\n - /roll - Roll Dice after construct rules\n - /config initmod - Save your init modifier\n - /gm help - Get help for using the gm tools")
}

func handleFoodToday(message MessageData) {
	message.SendMessageToChannel(message.ChannelID, UniPassauBot.FoodToday())
}

func handleFoodTomorrow(message MessageData) {
	message.SendMessageToChannel(message.ChannelID, UniPassauBot.FoodTomorrow())
}

func handlePing(message MessageData) {
	message.SendMessageToChannel(message.ChannelID, "Pong!")
}

func handleID(message MessageData) {
	message.SendMessageToChannel(message.ChannelID, "Your user-id is: "+message.AuthorID)
}

func handleIsDM(message MessageData) {
	var output string
	if message.IsDM {
		output = "This **is** a DM!"
	} else {
		output = "This is **not** a DM!"
	}
	message.SendMessageToChannel(message.ChannelID, output)
}

func handleConfig(message MessageData) {
	tokens := strings.Split(message.Content, " ")
	if len(tokens) < 2 {
		message.SendMessageToChannel(message.ChannelID, "Save Data to the Bot. Currently available:\n - /save initmod x - Save you Init Modifier")
	} else {
		switch tokens[1] {
		case "initmod":
			if len(tokens) < 3 {
				message.SetUserData(message.AuthorID, "initmod", "")
				message.SendMessageToChannel(message.ChannelID, "Your init modifier was reset.")
			} else if len(tokens) == 3 {
				initMod, err := strconv.Atoi(tokens[2])
				if err != nil {
					message.SendMessageToChannel(message.ChannelID, "There was an error in your command!")
				} else {
					message.SetUserData(message.AuthorID, "initmod", strconv.Itoa(initMod))
					message.SendMessageToChannel(message.ChannelID, "Your init modifier was set to "+strconv.Itoa(initMod)+".")
				}
			} else {
				var output strings.Builder
				limit := len(tokens)
				for i := 2; i < limit; i++ {
					_, err := strconv.Atoi(tokens[i])
					if err != nil {
						message.SendMessageToChannel(message.ChannelID, "There was an error while parsing your command")
						return
					}
					if i == limit-1 {
						output.WriteString(tokens[i])
					} else {
						output.WriteString(tokens[i] + "|")
					}
				}
				initModString := output.String()
				message.SetUserData(message.AuthorID, "initmod", initModString)
				//inputString = inputString[:2]
				//Save("glyph/discord:"+m.Author.ID+"/initmod", inputString)
				message.SendMessageToChannel(message.ChannelID, "Your init modifier was set to following values: "+initModString+".")
			}
		default:
			message.SendMessageToChannel(message.ChannelID, "Sorry, I don't know what to save here!")
		}
	}
}

func handleGenericError(message MessageData) {
	message.SendMessageToChannel(message.ChannelID, "Sorry, an internal error occurred. Please try again or contact the bot administrator.")
}

// To be imported

// Roll init rollCount times with given initmod and return a string for user
func rollMassInit(rollCount, initMod int) string {
	var output strings.Builder
	for i := 1; i <= rollCount; i++ {
		output.WriteString(strconv.Itoa(i) + ": " + strconv.Itoa(rollXSidedDie(1, 10)[0]+initMod) + "\n")
	}
	return output.String()
}

// Parse Dice diagnostics
func diceDiagnosticHelper(s *discordgo.Session, m *discordgo.MessageCreate) {
	inputString := strings.Split(m.Content, " ")
	count := 0
	if len(inputString) < 3 {
		count = 100000
	} else {
		var err error
		count, err = strconv.Atoi(inputString[2])
		if err != nil {
			s.ChannelMessageSend(m.ChannelID, "There was an error parsing your command!")
			return
		}
	}
	if count > 1000000 {
		s.ChannelMessageSend(m.ChannelID, "Please choose a valid range!")
		return
	}
	sidesCount := make([]int, 10)
	result := rollXSidedDie(count, 10)
	for i := 0; i < len(result); i++ {
		sidesCount[result[i]-1]++
	}
	var output strings.Builder
	countFloat := float64(count)
	output.WriteString("Diagnostics show following percentages:\n")
	for i := 0; i < len(sidesCount); i++ {
		percentString := floatToString((float64(sidesCount[i])/countFloat)*100) + "%"
		output.WriteString("Side " + strconv.Itoa(i+1) + ": " + percentString + "\n")
	}
	s.ChannelMessageSend(m.ChannelID, output.String())
}

// convert a Float64 into a string
func floatToString(inputNum float64) string {
	// to convert a float number to a string
	return strconv.FormatFloat(inputNum, 'f', 6, 64)
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
			_, _ = s.ChannelMessageSend(m.ChannelID, "**Epic Fail!** Your Chance Die failed spectacularly!")
		default:
			_, _ = s.ChannelMessageSend(m.ChannelID, "**Fail!** You rolled a **"+strconv.Itoa(diceRollResult)+"** on your Chance die!")
		}
		return
	case "init":
		/*var initModSliceObject interface{}
		  err := Load("glyph/discord:"+m.Author.ID+"/initmod", &initModSliceObject)
		  if err != nil || reflect.TypeOf(initModSliceObject) != reflect.TypeOf("") {
		      _, _ = s.ChannelMessageSend(m.ChannelID, "There was an internal error, please try again!")
		      del("glyph/discord:" + m.Author.ID + "/initmod")
		      glyphDiscordLog.Warning("Error while getting init from data with type of "+reflect.TypeOf(initModSliceObject).String()+" and error: ", err.Error())
		      return
		  }
		  initModSlice := initModSliceObject.([]string)*/
		initModSlice := strings.Split(data.GetTmp("glyph", "dg:"+m.Author.ID+"|initmod"), "|")
		number := 1
		if len(inputString) > 2 {
			var err error
			number, err = strconv.Atoi(inputString[2])
			if err != nil {
				_, _ = s.ChannelMessageSend(m.ChannelID, "There was an error parsing your command!")
				return
			}
		}
		if number < 1 || number > len(initModSlice) {
			_, _ = s.ChannelMessageSend(m.ChannelID, "Please specify a valid number!")
			return
		}
		initModString := initModSlice[number-1]
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
		// Count Successes and CritFails while parsing the return String
		// Parse Slice here
		var successes, critfails int
		var output strings.Builder
		output.WriteString("Results for " + m.Author.Mention() + ": ")
		for i := range retSlice {
			output.WriteString("[")
			for j := range retSlice[i] {
				output.WriteString(strconv.Itoa(retSlice[i][j]))
				if j != len(retSlice[i])-1 {
					output.WriteString(" ❯ ")
				}
				switch retSlice[i][j] {
				case 8, 9, 10:
					successes++
				case 1, 2:
					critfails++
				}
			}
			output.WriteString("] ")
			output.WriteString(" ")
		}
		critfailTreshold := int(math.Round(float64(throwCount) / 2))
		if critfails >= critfailTreshold && successes == 0 {
			output.WriteString("\nWell that's a **critical failure!**")
		} else {
			if successes > 0 {
				if successes >= 5 {
					output.WriteString("\nThat were **" + strconv.Itoa(successes) + "** Successes!\n" + "That was **exceptional**!")
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
