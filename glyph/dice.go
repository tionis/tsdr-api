package glyph

import (
	"math"
	"math/rand"
	"strconv"
	"strings"
	"sync"
	"time"

	_ "github.com/heroku/x/hmetrics/onload" // Heroku advanced go metrics
)

// randomSeedOnce ensures that the random number generator is only seeded once
var randomSeedOnce sync.Once

// handleRoll handles the roll command, taking a message struct and the input tokens
func (g Bot) handleRoll(message MessageData, tokens []string) {
	if len(tokens) < 2 {
		g.sendMessageDefault(message, "To roll dice just tell me how many I should roll and what Modifiers I shall apply.\nI can also roll custom dice like this: /roll 3d12")
	} else {
		g.rollHelper(message)
	}
}

// handleGM handles GM subcommands
func (g Bot) handleGM(message MessageData, tokens []string) {
	if len(tokens) == 1 {
		g.sendMessageDefault(message, "# Available Commands:\n - /gm rollinit COUNT INIT")
	} else {
		switch tokens[1] {
		case "help":
			g.sendMessageDefault(message, "# Available Commands:\n - /gm rollinit COUNT INIT")
		case "rollinit":
			rollCount, err := strconv.Atoi(tokens[2])
			if err != nil {
				g.sendMessageDefault(message, "There was an error in your command!")
			}
			rollInit, err := strconv.Atoi(tokens[3])
			if err != nil {
				g.sendMessageDefault(message, "There was an error in your command!")
			}
			g.SendMessageToChannel(message.ChannelID, g.rollMassInit(rollCount, rollInit))
		}
	}
}

// Roll init rollCount times with given initmod and return a string for user
func (g Bot) rollMassInit(rollCount, initMod int) string {
	var output strings.Builder
	for i := 1; i <= rollCount; i++ {
		output.WriteString(strconv.Itoa(i) + ": " + strconv.Itoa(rollXSidedDie(1, 10)[0]+initMod) + "\n")
	}
	return output.String()
}

// Parse Dice diagnostics
func (g Bot) diceDiagnosticHelper(message MessageData) {
	inputString := strings.Split(message.Content, " ")
	count := 0
	if len(inputString) < 3 {
		count = 100000
	} else {
		var err error
		count, err = strconv.Atoi(inputString[2])
		if err != nil {
			g.sendMessageDefault(message, "There was an error parsing your command!")
			return
		}
	}
	if count > 1000000 {
		g.sendMessageDefault(message, "Please choose a valid range!")

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
	g.sendMessageDefault(message, output.String())
}

// convert a Float64 into a string
func floatToString(inputNum float64) string {
	// to convert a float number to a string
	return strconv.FormatFloat(inputNum, 'f', 6, 64)
}

// Parse roll command
func (g Bot) rollHelper(message MessageData) {
	// Catch errors in command
	inputString := strings.Split(message.Content, " ")
	if len(inputString) < 2 {
		g.sendMessageDefault(message, "There was an error in your command!")
		return
	}

	// Catch simple commands
	switch inputString[1] {
	case "one":
		g.sendMessageDefault(message, "Simple 1D10 = "+strconv.Itoa(rollXSidedDie(1, 10)[0]))

		return
	case "chance":
		diceRollResult := rollXSidedDie(1, 10)[0]
		switch diceRollResult {
		case 10:
			g.sendMessageDefault(message, "**Success!** Your Chance Die showed a 10!")
		case 1:
			g.sendMessageDefault(message, "**Epic Fail!** Your Chance Die failed spectacularly!")
		default:
			g.sendMessageDefault(message, "**Fail!** You rolled a **"+strconv.Itoa(diceRollResult)+"** on your Chance die!")
		}
		return
	case "init":
		/*var initModSliceObject interface{}
		  err := Load("glyph/discord:"+m.Author.ID+"/initmod", &initModSliceObject)
		  if err != nil || reflect.TypeOf(initModSliceObject) != reflect.TypeOf("") {
		      g.sendMessageDefault(message, "There was an internal error, please try again!")
		      del("glyph/discord:" + m.Author.ID + "/initmod")
		      glyphDiscordLog.Warning("Error while getting init from data with type of "+reflect.TypeOf(initModSliceObject).String()+" and error: ", err.Error())
		      return
		  }
		  initModSlice := initModSliceObject.([]string)*/
		initMod, err := g.GetUserData(message.AuthorID, "initmod")
		if err != nil {
			g.Logger.Error("could not get user data for %v: %v", message.AuthorID, err)
			g.handleGenericError(message)
			return
		}
		switch initMod.(type) {
		case []int:
		default:
			g.sendMessageDefault(message, "There was an error loading your init modifier")
			return
		}
		initModSlice := initMod.([]int)
		number := 1
		if len(inputString) > 2 {
			var err error
			number, err = strconv.Atoi(inputString[2])
			if err != nil {
				g.sendMessageDefault(message, "There was an error parsing your command!")
				return
			}
		}
		if number < 1 || number > len(initModSlice) {
			g.sendMessageDefault(message, "Please specify a valid number!")
			return
		}
		if len(initModSlice) == 0 {
			g.sendMessageDefault(message, "No init modifier saved, here's a simple D10 throw:\n1D10 = "+strconv.Itoa(rollXSidedDie(1, 10)[0]))
		} else {
			diceResult := rollXSidedDie(1, 10)[0]
			endResult := diceResult + initModSlice[number-1]
			g.sendMessageDefault(message, "Your Initiative is: **"+strconv.Itoa(endResult)+"**\n"+strconv.Itoa(diceResult)+" + "+strconv.Itoa(initModSlice[number-1])+" = "+strconv.Itoa(endResult))
		}
		return
	}

	// Check which dice designation is used
	if strings.Contains(inputString[1], "d") {
		// Catch error in dice designation [/roll 1*s*10 ]
		diceIndex := strings.Split(inputString[1], "d")
		if len(diceIndex) < 2 {
			g.sendMessageDefault(message, "There was an error in your command!")
			return
		}
		sides, err := strconv.Atoi(diceIndex[1])
		if err != nil {
			g.sendMessageDefault(message, "There was an error in your command!")
			return
		}
		amount, err := strconv.Atoi(diceIndex[0])
		if err != nil {
			g.sendMessageDefault(message, "There was an error in your command!")
			return
		}

		// Catch d-notation and read modifiers
		if amount < 1 {
			g.sendMessageDefault(message, "Nice try!")
			return
		}
		switch {
		case sides < 1:
			g.sendMessageDefault(message, "Nice try!")
			return
		case sides == 1:
			g.sendMessageDefault(message, "Really? That's one times "+diceIndex[0]+". I think you can do the math yourself!")
			return
		default:
			if amount > 1000 {
				g.sendMessageDefault(message, "Maybe try a few less dice. We're not playing Warhammer Ultimate here.")
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
			g.sendMessageDefault(message, retString.String())
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
		g.sendMessageDefault(message, retString)
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
			g.sendMessageDefault(message, "There was an error in your command!")
			return
		}
		if throwCount > 1000 {
			g.sendMessageDefault(message, "Don't you think that are a few to many dice to throw?")
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
					g.sendMessageDefault(message, "There was an error while parsing your input!")
					return
				}
			}
		}
		// Count Successes and CritFails while parsing the return String
		// Parse Slice here
		var successes, critfails int
		var output strings.Builder
		mention, err := g.GetMention(message.AuthorID)
		if err != nil {
			g.Logger.Warningf("Could not get mention for %v: %v", message.AuthorID, err)
		}
		output.WriteString("Results for " + mention + ": ")
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
		g.sendMessageDefault(message, output.String())
	}
}

// Role x dice with given amount of sides
func rollXSidedDie(throwCount, sides int) []int {
	randomSeedOnce.Do(func() {
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
