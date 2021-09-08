package main

import (
	"fmt"
	"log"
	"math"
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
	UniPassauBot "github.com/tionis/uni-passau-bot/api"
)

// Discord ID of admin
//var discordAdminID string = "259076782408335360"
var discordServerID string = "695330213953011733"

// Needed for onlyonce execution of random source
var onlyOnce sync.Once

// Logger
var glyphDiscordLog = logging.MustGetLogger("glyphDiscord")

// Define discord commands
var (
	commands = []*discordgo.ApplicationCommand{
		{
			Name: "basic-command",
			// All commands and options must have a description
			// Commands/options without description will fail the registration
			// of the command.
			Description: "Basic command",
		},
		{
			Name:        "options",
			Description: "Command for demonstrating options",
			Options: []*discordgo.ApplicationCommandOption{

				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "string-option",
					Description: "String option",
					Required:    true,
				},
				{
					Type:        discordgo.ApplicationCommandOptionInteger,
					Name:        "integer-option",
					Description: "Integer option",
					Required:    true,
				},
				{
					Type:        discordgo.ApplicationCommandOptionBoolean,
					Name:        "bool-option",
					Description: "Boolean option",
					Required:    true,
				},

				// Required options must be listed first since optional parameters
				// always come after when they're used.
				// The same concept applies to Discord's Slash-commands API

				{
					Type:        discordgo.ApplicationCommandOptionChannel,
					Name:        "channel-option",
					Description: "Channel option",
					Required:    false,
				},
				{
					Type:        discordgo.ApplicationCommandOptionUser,
					Name:        "user-option",
					Description: "User option",
					Required:    false,
				},
				{
					Type:        discordgo.ApplicationCommandOptionRole,
					Name:        "role-option",
					Description: "Role option",
					Required:    false,
				},
			},
		},
		{
			Name:        "subcommands",
			Description: "Subcommands and command groups example",
			Options: []*discordgo.ApplicationCommandOption{
				// When a command has subcommands/subcommand groups
				// It must not have top-level options, they aren't accesible in the UI
				// in this case (at least not yet), so if a command has
				// subcommands/subcommand any groups registering top-level options
				// will cause the registration of the command to fail

				{
					Name:        "scmd-grp",
					Description: "Subcommands group",
					Options: []*discordgo.ApplicationCommandOption{
						// Also, subcommand groups aren't capable of
						// containing options, by the name of them, you can see
						// they can only contain subcommands
						{
							Name:        "nst-subcmd",
							Description: "Nested subcommand",
							Type:        discordgo.ApplicationCommandOptionSubCommand,
						},
					},
					Type: discordgo.ApplicationCommandOptionSubCommandGroup,
				},
				// Also, you can create both subcommand groups and subcommands
				// in the command at the same time. But, there's some limits to
				// nesting, count of subcommands (top level and nested) and options.
				// Read the intro of slash-commands docs on Discord dev portal
				// to get more information
				{
					Name:        "subcmd",
					Description: "Top-level subcommand",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
				},
			},
		},
		{
			Name:        "responses",
			Description: "Interaction responses testing initiative",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Name:        "resp-type",
					Description: "Response type",
					Type:        discordgo.ApplicationCommandOptionInteger,
					Choices: []*discordgo.ApplicationCommandOptionChoice{
						{
							Name:  "Acknowledge",
							Value: 2,
						},
						{
							Name:  "Channel message",
							Value: 3,
						},
						{
							Name:  "Channel message with source",
							Value: 4,
						},
						{
							Name:  "Acknowledge with source",
							Value: 5,
						},
					},
					Required: true,
				},
			},
		},
		{
			Name:        "followups",
			Description: "Followup messages",
		},
	}
	commandHandlers = map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate){
		"basic-command": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionApplicationCommandResponseData{
					Content: "Hey there! Congratulations, you just executed your first slash command",
				},
			})
		},
		"options": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			margs := []interface{}{
				// Here we need to convert raw interface{} value to wanted type.
				// Also, as you can see, here is used utility functions to convert the value
				// to particular type. Yeah, you can use just switch type,
				// but this is much simpler
				i.Data.Options[0].StringValue(),
				i.Data.Options[1].IntValue(),
				i.Data.Options[2].BoolValue(),
			}
			msgformat :=
				` Now you just learned how to use command options. Take a look to the value of which you've just entered:
				> string_option: %s
				> integer_option: %d
				> bool_option: %v
`
			if len(i.Data.Options) >= 4 {
				margs = append(margs, i.Data.Options[3].ChannelValue(nil).ID)
				msgformat += "> channel-option: <#%s>\n"
			}
			if len(i.Data.Options) >= 5 {
				margs = append(margs, i.Data.Options[4].UserValue(nil).ID)
				msgformat += "> user-option: <@%s>\n"
			}
			if len(i.Data.Options) >= 6 {
				margs = append(margs, i.Data.Options[5].RoleValue(nil, "").ID)
				msgformat += "> role-option: <@&%s>\n"
			}
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				// Ignore type for now, we'll discuss them in "responses" part
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionApplicationCommandResponseData{
					Content: fmt.Sprintf(
						msgformat,
						margs...,
					),
				},
			})
		},
		"subcommands": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			content := ""

			// As you can see, the name of subcommand (nested, top-level) or subcommand group
			// is provided through arguments.
			switch i.Data.Options[0].Name {
			case "subcmd":
				content =
					"The top-level subcommand is executed. Now try to execute the nested one."
			default:
				if i.Data.Options[0].Name != "scmd-grp" {
					return
				}
				switch i.Data.Options[0].Options[0].Name {
				case "nst-subcmd":
					content = "Nice, now you know how to execute nested commands too"
				default:
					// I added this in the case something might go wrong
					content = "Oops, something gone wrong.\n" +
						"Hol' up, you aren't supposed to see this message."
				}
			}
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionApplicationCommandResponseData{
					Content: content,
				},
			})
		},
		"responses": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			// Responses to a command are very important.
			// First of all, because you need to react to the interaction
			// by sending the response in 3 seconds after receiving, otherwise
			// interaction will be considered invalid and you can no longer
			// use the interaction token and ID for responding to the user's request

			content := ""
			// As you can see, the response type names used here are pretty self-explanatory,
			// but for those who want more information see the official documentation
			switch i.Data.Options[0].IntValue() {
			case int64(discordgo.InteractionResponseChannelMessage):
				content =
					"Well, you just responded to an interaction, and sent a message.\n" +
						"That's all what I wanted to say, yeah."
				content +=
					"\nAlso... you can edit your response, wait 5 seconds and this message will be changed"
			case int64(discordgo.InteractionResponseChannelMessageWithSource):
				content =
					"You just responded to an interaction, sent a message and showed the original one. " +
						"Congratulations!"
				content +=
					"\nAlso... you can edit your response, wait 5 seconds and this message will be changed"
			default:
				err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseType(i.Data.Options[0].IntValue()),
				})
				if err != nil {
					s.FollowupMessageCreate(s.State.User.ID, i.Interaction, true, &discordgo.WebhookParams{
						Content: "Something went wrong",
					})
				}
				return
			}

			err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseType(i.Data.Options[0].IntValue()),
				Data: &discordgo.InteractionApplicationCommandResponseData{
					Content: content,
				},
			})
			if err != nil {
				s.FollowupMessageCreate(s.State.User.ID, i.Interaction, true, &discordgo.WebhookParams{
					Content: "Something went wrong",
				})
				return
			}
			time.AfterFunc(time.Second*5, func() {
				err = s.InteractionResponseEdit(s.State.User.ID, i.Interaction, &discordgo.WebhookEdit{
					Content: content + "\n\nWell, now you know how to create and edit responses. " +
						"But you still don't know how to delete them... so... wait 10 seconds and this " +
						"message will be deleted.",
				})
				if err != nil {
					s.FollowupMessageCreate(s.State.User.ID, i.Interaction, true, &discordgo.WebhookParams{
						Content: "Something went wrong",
					})
					return
				}
				time.Sleep(time.Second * 10)
				s.InteractionResponseDelete(s.State.User.ID, i.Interaction)
			})
		},
		"followups": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			// Followup messages are basically regular messages (you can create as many of them as you wish)
			// but work as they are created by webhooks and their functionality
			// is for handling additional messages after sending a response.

			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionApplicationCommandResponseData{
					// Note: this isn't documented, but you can use that if you want to.
					// This flag just allows you to create messages visible only for the caller of the command
					// (user who triggered the command)
					Flags:   1 << 6,
					Content: "Surprise!",
				},
			})
			msg, err := s.FollowupMessageCreate(s.State.User.ID, i.Interaction, true, &discordgo.WebhookParams{
				Content: "Followup message has been created, after 5 seconds it will be edited",
			})
			if err != nil {
				s.FollowupMessageCreate(s.State.User.ID, i.Interaction, true, &discordgo.WebhookParams{
					Content: "Something went wrong",
				})
				return
			}
			time.Sleep(time.Second * 5)

			s.FollowupMessageEdit(s.State.User.ID, i.Interaction, msg.ID, &discordgo.WebhookEdit{
				Content: "Now the original message is gone and after 10 seconds this message will ~~self-destruct~~ be deleted.",
			})

			time.Sleep(time.Second * 10)

			s.FollowupMessageDelete(s.State.User.ID, i.Interaction, msg.ID)

			s.FollowupMessageCreate(s.State.User.ID, i.Interaction, true, &discordgo.WebhookParams{
				Content: "For those, who didn't skip anything and followed tutorial along fairly, " +
					"take a unicorn :unicorn: as reward!\n" +
					"Also, as bonus... look at the original interaction response :D",
			})
		},
	}
)

// Main and Init
func glyphDiscordBot() {
	dg, err := discordgo.New("Bot " + os.Getenv("DISCORD_TOKEN"))
	if err != nil {
		glyphDiscordLog.Error("Error creating Discord session,", err)
	}

	// Register the messageCreate func as a callback for MessageCreate events.
	dg.AddHandler(messageCreate)
	dg.AddHandler(func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		if h, ok := commandHandlers[i.Data.Name]; ok {
			h(s, i)
		}
	})
	dg.AddHandler(func(s *discordgo.Session, r *discordgo.Ready) {
		log.Println("Bot is up!")
	})

	// Open a websocket connection to Discord and begin listening.
	err = dg.Open()
	if err != nil {
		glyphDiscordLog.Error("Error opening connection,", err)
		return
	}

	// Set some StartUp Stuff
	/*dgStatus, err := getError("dgStatus")
	  if err != nil {
	      glyphDiscordLog.Warning("Error getting dgStatus from redis: ", err)
	      dgStatus = "/help for help"
	  }*/
	_ = dg.UpdateGameStatus(0, "/help for help")

	// Wait here until CTRL-C or other term signal is received.
	glyphDiscordLog.Info("Glyph Discord Bot was started.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, syscall.SIGQUIT, syscall.SIGHUP)
	<-sc

	// Cleanly close down the Discord session.
	_ = dg.Close()
}

// This function will be called (due to AddHandler above) every time a new
// message is created on any channel that the authenticated bot has access to.
func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {

	// Ignore all messages created by the bot itself
	if m.Author.ID == s.State.User.ID {
		return
	}

	// Check if message was sent in an DM
	AuthorDMChannelID := getTmp("glyph", "dg:"+m.Author.ID+"|DM-Channel")
	if AuthorDMChannelID == "" {
		AuthorDMChannel, err := s.UserChannelCreate(m.Author.ID)
		if err != nil {
			return
		}
		AuthorDMChannelID = AuthorDMChannel.ID
		setTmp("glyph", "dg:"+m.Author.ID+"|DM-Channel", AuthorDMChannelID, time.Hour*24)
	}
	isDM := m.ChannelID == AuthorDMChannelID

	// Handle Server Messages
	// Check if glyph is currently in a conversation with user
	/*messageContext := get("glyph|discord:" + m.Author.ID + "|messageContext")
	  if messageContext != "" {
	      switch messageContext {
	      default:
	          glyphDiscordLog.Info("New Command by " + m.Author.Username + ": " + m.Content)
	          _, _ = s.ChannelMessageSend(m.ChannelID, "I encountered an internal error, please contact the administrator.")
	          return
	      }
	  }*/

	// Check if a known command was written
	inputString := strings.Split(m.Content, " ")
	if strings.Contains(inputString[0], "/") {
		glyphDiscordLog.Info(m.Author.Username + ": " + m.Content)
	}
	switch inputString[0] {
	// Dice commands
	case "/roll", "/r":
		if len(inputString) < 2 {
			_, _ = s.ChannelMessageSend(m.ChannelID, "To roll dice just tell me how many I should roll and what Modifiers I shall apply.\nI can also roll custom dice like this: /roll 3d12")
		} else {
			rollHelper(s, m)
		}

	// Diagnostic Commands
	case "/diag":
		switch inputString[1] {
		case "dice":
			diceDiagnosticHelper(s, m)
		default:
			_, _ = s.ChannelMessageSend(m.ChannelID, "Unknown Command!")
		}
	// Help commands
	case "/help":
		_, _ = s.ChannelMessageSend(m.ChannelID, "Available Command Categories:\n - Uni Passau - /unip help\n - PnP Tools - /pnp help")
	case "/unip":
		_, _ = s.ChannelMessageSend(m.ChannelID, "Available Commands:\n/food - Food for today\n/food tomorrow - Food for tomorrow")
	case "/pnp":
		_, _ = s.ChannelMessageSend(m.ChannelID, "Available Commands:\n - /roll - Roll Dice after construct rules\n - /save initmod - Save your init modifier\n - /gm help - Get help for using the gm tools")
	case "/gm", "/GM":
		switch inputString[1] {
		case "help":
			_, _ = s.ChannelMessageSend(m.ChannelID, "Available Commands:\n - /gm rollinit COUNT INIT")
		case "rollinit":
			rollCount, err := strconv.Atoi(inputString[2])
			rollInit, err := strconv.Atoi(inputString[3])
			if err != nil {
				_, _ = s.ChannelMessageSend(m.ChannelID, "There was an error in your command!")
			}
			_, _ = s.ChannelMessageSend(m.ChannelID, rollMassInit(rollCount, rollInit))
		}
	// Food commands
	case "/food":
		_, _ = s.ChannelMessageSend(m.ChannelID, UniPassauBot.FoodToday())
	case "/food tomorrow":
		_, _ = s.ChannelMessageSend(m.ChannelID, UniPassauBot.FoodTomorrow())

	// Config commands
	case "/save":
		if len(inputString) < 2 {
			_, _ = s.ChannelMessageSend(m.ChannelID, "Save Data to the Bot. Currently available:\n - /save initmod x - Save you Init Modifier")
		} else {
			saveHandler(s, m, inputString)
		}

	// MISC commands
	case "/ping":
		_, _ = s.ChannelMessageSend(m.ChannelID, "Pong!")
	case "/id":
		_, _ = s.ChannelMessageSend(m.ChannelID, "Your ID is:\n"+m.Author.ID)
	/*case "/updateStatus":
	  if m.Author.ID == discordAdminID {
	      newStatus := strings.TrimPrefix(m.Content, "/updateStatus ")
	      err := set("dgStatus", newStatus)
	      if err != nil {
	          glyphDiscordLog.Warning("Error setting dgStatus on Redis: ", err)
	          _, _ = s.ChannelMessageSend(m.ChannelID, "Error sending Status to Safe!")
	      }
	      _ = s.UpdateStatus(0, newStatus)
	      _, _ = s.ChannelMessageSend(m.ChannelID, "New Status set!")
	  } else {
	      _, _ = s.ChannelMessageSend(m.ChannelID, "You are not authorized to execute this command!\nThis incident will be reported.\nhttps://imgs.xkcd.com/comics/incident.png")
	  }*/
	case "/whoami":
		_, _ = s.ChannelMessageSend(m.ChannelID, m.Author.String())
	case "/todo":
		_, _ = s.ChannelMessageSend(m.ChannelID, "Feature still in Development")
	case "/isDM":
		if isDM {
			_, _ = s.ChannelMessageSend(m.ChannelID, "This **is** a DM!")
		} else {
			_, _ = s.ChannelMessageSend(m.ChannelID, "This is **not** a DM!")
		}
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

// Handle Save Command
func saveHandler(s *discordgo.Session, m *discordgo.MessageCreate, inputString []string) {
	switch inputString[1] {
	case "initmod":
		if len(inputString) < 3 {
			delTmp("glyph", "dg:"+m.Author.ID+"|initmod")
			_, _ = s.ChannelMessageSend(m.ChannelID, "Your init modifier was reset.")
		} else if len(inputString) == 3 {
			initMod, err := strconv.Atoi(inputString[2])
			if err != nil {
				_, _ = s.ChannelMessageSend(m.ChannelID, "There was an error in your command!")
			} else {
				setTmp("glyph", "dg:"+m.Author.ID+"|initmod", strconv.Itoa(initMod), time.Hour*24)
				_, _ = s.ChannelMessageSend(m.ChannelID, "Your init modifier was set to "+strconv.Itoa(initMod)+".")
			}
		} else {
			var output strings.Builder
			limit := len(inputString)
			for i := 2; i < limit; i++ {
				_, err := strconv.Atoi(inputString[i])
				if err != nil {
					_, _ = s.ChannelMessageSend(m.ChannelID, "There was an error while parsing your command")
					return
				}
				if i == limit-1 {
					output.WriteString(inputString[i])
				} else {
					output.WriteString(inputString[i] + "|")
				}
			}
			initModString := output.String()
			setTmp("glyph", "dg:"+m.Author.ID+"|initmod", initModString, time.Hour*24)
			//inputString = inputString[:2]
			//Save("glyph/discord:"+m.Author.ID+"/initmod", inputString)
			_, _ = s.ChannelMessageSend(m.ChannelID, "Your init modifier was set to following values: "+initModString+".")
		}
	default:
		_, _ = s.ChannelMessageSend(m.ChannelID, "Sorry, I don't know what to save here!")
	}
}

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
		initModSlice := strings.Split(getTmp("glyph", "dg:"+m.Author.ID+"|initmod"), "|")
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
