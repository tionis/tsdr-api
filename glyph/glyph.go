package glyph

import (
	"errors"
	"strconv"
	"strings"
	"time"

	_ "github.com/heroku/x/hmetrics/onload" // Heroku advanced go metrics
	"github.com/keybase/go-logging"

	UniPassauBot "github.com/tionis/uni-passau-bot/api"
)

// ErrNoCommandMatched represents the state in which no command could be matched
var ErrNoCommandMatched = errors.New("no command was matched")

// ErrUserNotFound is thrown when the searched user could not be found
var ErrUserNotFound = errors.New("user not found")

var standardContextDelay = time.Minute * 5

// MessageData represents an message the bot can act on with callback functions
type MessageData struct {
	Content          string `json:"content,required"`
	AuthorID         string `json:"authorID,omitempty"`
	IsDM             bool   `json:"isDM,omitempty"`
	SupportsMarkdown bool   `json:"supportsMarkdown,omitempty"`
	ChannelID        string `json:"channelID,omitempty"`
	IsCommand        bool   `json:"isCommand,omitempty"`
}

// QuoteDB contains all functions to interface with an Database containing quotes
type QuoteDB struct {
	GetRandomQuote func(byAuthor, inLanguage, inUniverse string) (Quote, error)
	AddQuote       func(Quote) error
}

type stringWithTTL struct {
	Content    string
	ValidUntil time.Time
}

// Quote represents a Quote
type Quote struct {
	Content  string
	Author   string
	Language string
	Universe string
}

// Bot represents a glyph bot instance with configuration
type Bot struct {
	QuoteDBHandler       *QuoteDBHandler
	GetMention           func(userID string) (string, error)                                 // A function that when passed an userID returns an string mentioning the user
	GetContext           func(userID, channelID, key string) (string, error)                 // A function that when passed an channelID and UserID returns the current chat context for the specified key
	SetContext           func(userID, channelID, key, value string, ttl time.Duration) error // A function that allows setting the current channelID+UserID context with a specific key
	SetUserData          func(userID, key string, value interface{}) error                   // A function that saves data to a specific user by key
	GetUserData          func(userID, key string) (interface{}, error)                       // A function that gets data to a specific user by key
	SendMessageToChannel func(channelID, message string) error                               // A function that sends a simple text message to specified channel
	Prefix               string                                                              // The command prefix used
	Logger               *logging.Logger
}

// QuoteDBHandler exposes functions to interact with a Quote DB
type QuoteDBHandler struct {
	AddQuote       func(quote Quote) error                                      // A function that saves a given Quote
	GetRandomQuote func(byAuthor, inLanguage, inUniverse string) (Quote, error) // A function that gets a random quote based on parameters
}

// HandleAll takes a MessageData object and parses it for the glyph bot, calling callback functions as needed
func (g Bot) HandleAll(message MessageData) {
	if !message.IsCommand {
		go g.handleNonCommandMessage(message)
	}

	tokens := strings.Split(message.Content, " ")
	switch strings.ToLower(tokens[0]) {
	// Help Commands
	case "help":
		go g.handleHelp(message)
	case "unip":
		go g.handleUnip(message)
	case "pnp":
		go g.handlePnPHelp(message)

	// Food commands
	case "food":
		go g.handleFoodToday(message)
	case "food tomorrow":
		go g.handleFoodTomorrow(message)

	// Config commands
	case "config":
		go g.handleConfig(message)

	// MISC commands
	case "ping":
		go g.handlePing(message)
	case "id":
		go g.handleID(message)
	case "isDM":
		go g.handleIsDM(message)
	case "roll", "r":
		go g.handleRoll(message, tokens)
	case "cancel":
		go g.handleCancelContext(message)

	// Quotator Commands
	case "getquote":
		go g.handleGetQuote(message)
	case "addquote":
		go g.handleAddQuoteInit(message)
	case "quoteoftheday":
		go g.handleQuoteOfTheDay(message)

	// Diagnostic Commands
	case "diag":
		switch tokens[1] {
		case "dice":
			g.diceDiagnosticHelper(message)
		default:
			g.SendMessageToChannel(message.ChannelID, "Unknown Command!")
		}
	// Help commands
	case "gm":
		go g.handleGM(message, tokens)
	default:
		go g.handleInvalidCommand(message)
	}
}

func (g Bot) handleNonCommandMessage(message MessageData) {
	context, err := g.GetContext(message.AuthorID, message.ChannelID, "ctx")
	if err != nil {
		g.Logger.Error("could not get context for %v in Channel %v: %v", message.AuthorID, message.ChannelID, err)
		return
	}
	switch context {
	// Quotator Contexts
	case "quoteRequired":
		g.handleAddQuoteContent(message)
	case "authorRequired":
		g.handleAddQuoteAuthor(message)
	case "languageRequired":
		g.handleAddQuoteLanguage(message)
	case "universeRequired":
		g.handleAddQuoteUniverse(message)
		g.handleAddQuoteFinished(message)
	default:
		g.SetContext(message.AuthorID, message.ChannelID, "ctx", "", time.Second)
		g.handleGenericError(message)
	}
}

func (g Bot) handleHelp(message MessageData) {
	if message.SupportsMarkdown {
		g.SendMessageToChannel(message.ChannelID, "# Available Command Categories:\n - Uni Passau - /unip help\n - PnP Tools - /pnp help")
	} else {
		g.SendMessageToChannel(message.ChannelID, "Available Command Categories:\n - Uni Passau - /unip help\n - PnP Tools - /pnp help")
	}
}

func (g Bot) handleUnip(message MessageData) {
	g.SendMessageToChannel(message.ChannelID, "Available Commands:\n/food - Food for today\n/food tomorrow - Food for tomorrow")
}

func (g Bot) handlePnPHelp(message MessageData) {
	g.SendMessageToChannel(message.ChannelID, "Available Commands:\n - /roll - Roll Dice after construct rules\n - /config initmod - Save your init modifier\n - /gm help - Get help for using the gm tools")
}

func (g Bot) handleFoodToday(message MessageData) {
	g.SendMessageToChannel(message.ChannelID, UniPassauBot.FoodToday())
}

func (g Bot) handleFoodTomorrow(message MessageData) {
	g.SendMessageToChannel(message.ChannelID, UniPassauBot.FoodTomorrow())
}

func (g Bot) handlePing(message MessageData) {
	g.SendMessageToChannel(message.ChannelID, "Pong!")
}

func (g Bot) handleID(message MessageData) {
	g.SendMessageToChannel(message.ChannelID, "Your user-id is: "+message.AuthorID)
}

func (g Bot) handleIsDM(message MessageData) {
	var output string
	if message.IsDM {
		output = "This **is** a DM!"
	} else {
		output = "This is **not** a DM!"
	}
	g.SendMessageToChannel(message.ChannelID, output)
}

func (g Bot) handleConfig(message MessageData) {
	tokens := strings.Split(message.Content, " ")
	if len(tokens) < 2 {
		g.SendMessageToChannel(message.ChannelID, "Save Data to the Bot. Currently available:\n - /save initmod x - Save you Init Modifier")
	} else {
		switch tokens[1] {
		case "initmod":
			if len(tokens) < 3 {
				g.SetUserData(message.AuthorID, "initmod", "")
				g.SendMessageToChannel(message.ChannelID, "Your init modifier was reset.")
			} else if len(tokens) == 3 {
				initMod, err := strconv.Atoi(tokens[2])
				if err != nil {
					g.SendMessageToChannel(message.ChannelID, "There was an error in your command!")
				} else {
					g.SetUserData(message.AuthorID, "initmod", strconv.Itoa(initMod))
					g.SendMessageToChannel(message.ChannelID, "Your init modifier was set to "+strconv.Itoa(initMod)+".")
				}
			} else {
				var output strings.Builder
				limit := len(tokens)
				for i := 2; i < limit; i++ {
					_, err := strconv.Atoi(tokens[i])
					if err != nil {
						g.SendMessageToChannel(message.ChannelID, "There was an error while parsing your command")
						return
					}
					if i == limit-1 {
						output.WriteString(tokens[i])
					} else {
						output.WriteString(tokens[i] + "|")
					}
				}
				initModString := output.String()
				g.SetUserData(message.AuthorID, "initmod", initModString)
				//inputString = inputString[:2]
				//Save("glyph/discord:"+m.Author.ID+"/initmod", inputString)
				g.SendMessageToChannel(message.ChannelID, "Your init modifier was set to following values: "+initModString+".")
			}
		default:
			g.SendMessageToChannel(message.ChannelID, "Sorry, I don't know what to save here!")
		}
	}
}

func (g Bot) handleGenericError(message MessageData) {
	g.SendMessageToChannel(message.ChannelID, "Sorry, an internal error occurred. Please try again or contact the bot administrator.")
}

func (g Bot) handleInvalidCommand(message MessageData) {
	g.SendMessageToChannel(message.ChannelID, "Unknown Command, to get a list of available command use the "+g.Prefix+"help command")
}

func (g Bot) handleCancelContext(message MessageData) {
	g.SetContext(message.AuthorID, message.ChannelID, "ctx", "", 1*time.Second)
	g.SendMessageToChannel(message.ChannelID, "I canceled the process!")
}

func (s stringWithTTL) isValid() bool {
	return s.ValidUntil.Before(time.Now())
}

func (g Bot) sendMessageDefault(messageToParse MessageData, messageToSend string) {
	if messageToParse.IsDM {
		g.SendMessageToChannel(messageToParse.ChannelID, messageToSend)
	} else {
		mention, err := g.GetMention(messageToParse.AuthorID)
		if err != nil {
			g.Logger.Warningf("Could not get mention for %v: %v", messageToParse.AuthorID, err)
		}
		g.SendMessageToChannel(messageToParse.ChannelID, mention+"\n"+messageToSend)
	}
}
