package glyph

import (
	"encoding/json"
	"errors"
	"regexp"
	"strconv"
	"strings"
	"time"

	_ "github.com/heroku/x/hmetrics/onload" // Heroku advanced go metrics
	"github.com/keybase/go-logging"

	UniPassauBot "github.com/tionis/uni-passau-bot/api"
)

// Define Errors needed for consistent interaction with foreign and own functions

// ErrNoCommandMatched represents the state in which no command could be matched
var ErrNoCommandMatched = errors.New("no command was matched")

// ErrNoUserDataFound is thrown if now data for the user with the specified key could be found
var ErrNoUserDataFound = errors.New("no userdata found")

// ErrUserNotFound is thrown when the searched user could not be found
var ErrUserNotFound = errors.New("user not found")

// ErrNoMappingFound is thrown if no valid mapping from a 3PID to an userID could be found
var ErrNoMappingFound = errors.New("no mapping between 3PID and userID found")

// ErrNoSuchSession is thrown if no auth session with the given ID could be founc
var ErrNoSuchSession = errors.New("no session with given ID could be found")

// ErrSessionNotOfUser is thrown if a given session exists but does not belong to a given user
var ErrSessionNotOfUser = errors.New("session exists but does not belong to user")

// standardContextDelay is the standard ttl of chat contexts
var standardContextDelay = time.Minute * 5

// Define regex used to check validity of usernames and addresses

// isValidUserName checks if the string is a valid username (after matrix ID and thus tasadar.net specification)
var isValidUserName = regexp.MustCompile(`(?m)^[a-z\-_]+$`)

// isValidMatrixID checks if the string is a valid matrix id (but ignores the case in which the domain starts or ends with an dash)
var isValidMatrixID = regexp.MustCompile(`(?m)^@[a-z\-_]+:([A-Za-z0-9-]{1,63}\.)+[A-Za-z]{2,6}$`)

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
	GetRandomQuote   func(byAuthor, inLanguage, inUniverse string) (Quote, error) // GetRandomQuote gets a random quote based on parameters
	AddQuote         func(Quote) error                                            // AddQuote saves a given Quote
	GetQuoteOfTheDay func(userID string) (QuoteOfTheDay, error)                   // GetQuoteOfTheDay gets the quote of the day for a given userID
	SetQuoteOfTheDay func(userID string, quoteOfTheDay QuoteOfTheDay) error       // SetQuoteOfTheDay sets the quote of the day for a given userID
}

// UserDB contains all function to interface with an Database holding the user data
type UserDB struct {
	SetUserData           func(userID, key string, value string) error                 // SetUserData saves data to a specific user by key
	GetUserData           func(userID, key string) (string, error)                     // GetUserData gets data to a specific user by key
	DeleteUserData        func(userID, key string) error                               // DeleteUserData deletes user data with a given key
	MigrateUserToNewID    func(oldMatrixUserID, newMatrixUserID string) error          // MigrateUserToNewID migrates userdata to a new ID
	GetMatrixUserID       func(userID string) (string, error)                          // GetMatrixUserID gets the MatrixID from the implementation specific userID
	DoesMatrixUserIDExist func(matrixUserID string) (bool, error)                      // DoesMatrixUserIDExist checks if a user with given matrixID is already registered (this is used with a hardcoded transformation from tasadar user to matrix id)
	AddAuthSession        func(authWorker func() error, userID string) (string, error) // AddAuthSession adds an auth session with an authWorker that is executed when the session is authenticated. The functions returns an error and the ID of the auth session
	GetAuthSessionStatus  func(authSessionID string) (string, error)                   // GetAuthSessionStatus is used to get the status of an auth session with the ID
	AuthenticateSession   func(matrixUserID, authSessionID string) error               // AuthenticateSession sets the session with given ID as authenticated
	DeleteSession         func(authSessionID string) error                             // DeleteSession deletes the session with given ID
	GetAuthSessions       func(matrixID string) ([]string, error)                      // GetAuthSessions return the state of all sessions registered to the user
}

// Quote represents a Quote
type Quote struct {
	ID       string // ID is an implementation specific ID of the Quote used to identify it between the quoteData functions
	Content  string // Content contains the Quote itself
	Author   string // Author is the name of the author of the quote
	Language string // Language is the language the quote is written in
	Universe string // Universe represent in which (fictional) work/universe the Quote was said in (may be the real world)
	ByUser   string // ByUser contains the matrixID of the user who added the Quote to the bot database
}

// QuoteOfTheDay represent an quote of the day with an saved Quote plus expiry time
type QuoteOfTheDay struct {
	Quote      Quote     // Quote represent the Quote
	ValidUntil time.Time // ValidUntil represents until when the Quote is the current QuoteOfTheDay
}

// Bot represents a glyph bot instance with configuration
type Bot struct {
	QuoteDBHandler       *QuoteDB                                                            // QuoteDBHandler implements functions to interact with the quote database
	UserDBHandler        *UserDB                                                             // UserDBHandler implements functions to interact with user-specific data
	GetMention           func(userID string) (string, error)                                 // GetMention when passed an userID returns an string mentioning the user
	GetContext           func(userID, channelID, key string) (string, error)                 // GetContext when passed an channelID and UserID returns the current chat context for the specified key
	SetContext           func(userID, channelID, key, value string, ttl time.Duration) error // SetContext allows setting the current channelID+UserID context with a specific key
	SendMessageToChannel func(channelID, message string) error                               // SendMessageToChannel sends a simple text message to specified channel
	Prefix               string                                                              // The command prefix used
	Logger               *logging.Logger                                                     // A Logger implementation to send logs to
}

// HandleAll takes a MessageData object and parses it for the glyph bot, calling callback functions as needed
func (g Bot) HandleAll(message MessageData) {
	if !message.IsCommand {
		go g.handleNonCommandMessage(message)
		return
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
	case "uid":
		go g.handleUID(message)
	case "gm":
		go g.handleGM(message, tokens)

	// Food commands
	case "food":
		go g.handleFoodToday(message)
	case "food tomorrow":
		go g.handleFoodTomorrow(message)

	// User data commands
	case "config":
		go g.handleConfig(message, tokens)
	case "auth":
		go g.handleAuth(message, tokens)

	// MISC & META commands
	case "ping":
		go g.handlePing(message)
	case "id":
		go g.handleID(message)
	case "isDM":
		go g.handleIsDM(message)
	case "cancel":
		go g.handleCancelContext(message)

	// Quotator Commands
	case "getquote":
		go g.handleGetQuote(message)
	case "addquote":
		go g.handleAddQuoteInit(message)
	case "quoteoftheday":
		go g.handleQuoteOfTheDay(message)

		// Dice Commands
	case "roll", "r":
		go g.handleRoll(message, tokens)
	case "diag":
		switch tokens[1] {
		case "dice":
			g.diceDiagnosticHelper(message)
		default:
			g.sendMessageDefault(message, "Unknown Command!")
		}

	default:
		go g.handleInvalidCommand(message)
	}
}

func (g Bot) handleNonCommandMessage(message MessageData) {
	context, err := g.GetContext(message.AuthorID, message.ChannelID, "ctx")
	if err != nil {
		g.Logger.Error("could not get context for %v in Channel %v: %v", message.AuthorID, message.ChannelID, err)
		g.handleGenericError(message)
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
	default: // If the context is set to something unknown reset the context and print a message that the command is not known
		g.SetContext(message.AuthorID, message.ChannelID, "ctx", "", time.Second)
		g.handleInvalidCommand(message)
	}
}

func (g Bot) handleHelp(message MessageData) {
	if message.SupportsMarkdown {
		g.sendMessageDefault(message, "# Available Command Categories:\n - Uni Passau - /unip help\n - PnP Tools - /pnp help")
	} else {
		g.sendMessageDefault(message, "Available Command Categories:\n - Uni Passau - /unip help\n - PnP Tools - /pnp help")
	}
}

func (g Bot) handleUnip(message MessageData) {
	g.sendMessageDefault(message, "Available Commands:\n/food - Food for today\n/food tomorrow - Food for tomorrow")
}

func (g Bot) handlePnPHelp(message MessageData) {
	sendString := "Available Commands:\n " +
		"- /roll - Roll Dice after construct rules\n " +
		"- /config initmod - Save your init modifier\n " +
		"- /gm help - Get help for using the gm tools\n "
	g.sendMessageDefault(message, sendString)
}

func (g Bot) handleFoodToday(message MessageData) {
	g.sendMessageDefault(message, UniPassauBot.FoodToday())
}

func (g Bot) handleFoodTomorrow(message MessageData) {
	g.sendMessageDefault(message, UniPassauBot.FoodTomorrow())
}

func (g Bot) handlePing(message MessageData) {
	g.sendMessageDefault(message, "Pong!")
}

func (g Bot) handleID(message MessageData) {
	g.sendMessageDefault(message, "Your user-id is: "+message.AuthorID)
}

func (g Bot) handleUID(message MessageData) {
	sendString := "To use some of the services of this bot you have to " +
		"connect your platform chat account to a userID spanning all chat platforms." +
		"\nThis userid has the form of an matrix id, thus you can use an existing " +
		"matrix id if you have one or just let the bot create a virtual one for you. " +
		"Just write your username (all lowercase with dashes, numbers and underscores) " +
		"to the bot using\n" + g.Prefix + "config uid YOUR_CHOICE. The same also works for " +
		"matrix ids but will then need confirmation of the matrix address."
	g.sendMessageDefault(message, sendString)
}

func (g Bot) handleIsDM(message MessageData) {
	var output string
	if message.IsDM {
		output = "This **is** a DM!"
	} else {
		output = "This is **not** a DM!"
	}
	g.sendMessageDefault(message, output)

}

func (g Bot) handleAuth(message MessageData, tokens []string) {
	if len(tokens) < 2 {
		g.sendMessageDefault(message, "Authenticate to the bot. Use this command with an Auth-ID to authorize a login.")
	} else {
		authID := tokens[1]
		matrixID, err := g.UserDBHandler.GetMatrixUserID(message.AuthorID)
		if err != nil {
			g.Logger.Warning("error getting matrixID from userID: ", err)
			g.handleGenericError(message)
			return
		}
		err = g.UserDBHandler.AuthenticateSession(matrixID, authID)
		switch err {
		case ErrNoMappingFound: // No auth session found
			g.sendMessageDefault(message, "Auth ID invalid!")
		case ErrSessionNotOfUser: // Auth session does not belong to user
			g.sendMessageDefault(message, "Auth ID invalid!")
		case nil: // Auth was successfull
			g.sendMessageDefault(message, "Auth Session "+authID+" was authenticated!")
		default: // Another error occurred
			g.Logger.Warningf("error authenticating %v with %v: %v", matrixID, authID, err)
			g.handleGenericError(message)
			return
		}
	}
}

func (g Bot) handleConfig(message MessageData, tokens []string) {
	// TODO handle setting of userID
	// TODO improve high cyclo
	// it should support migrating to a new if (/config uid idofuser)
	// logging into existing id (use /auth authCode to authorize a login to an existing account) TODO add this
	// checking if userid is available and warning if its not
	// also the whole application should handle the error of not having a userID mapping with a warning message
	// also if a matrix id is given the matrix bot should also allow the /auth directive this will need an matrixBot object in the bot config
	// i will need to think about that -> login flow where? -> solved by using callback functions
	if len(tokens) < 2 {
		g.sendMessageDefault(message, "Save Data to the Bot. Currently available:\n - /config initmod x - Save you Init Modifier")
	} else {
		switch tokens[1] {
		case "initmod":
			g.configInitmodHandler(message, tokens)
		case "uid":
			g.sendMessageDefault(message, "not implemented yet!")
		default:
			g.sendMessageDefault(message, "Sorry, I don't know what to save here!")
		}
	}
}

func (g Bot) configInitmodHandler(message MessageData, tokens []string) {
	if len(tokens) < 3 {
		g.UserDBHandler.DeleteUserData(message.AuthorID, "initmod")
		g.sendMessageDefault(message, "Your init modifier was reset.")

	} else if len(tokens) == 3 {
		initMod, err := strconv.Atoi(tokens[2])
		if err != nil {
			g.sendMessageDefault(message, "There was an error in your command!")
		} else {
			jsonBytes, err := json.Marshal([]int{initMod})
			if err != nil {
				g.Logger.Warning("could not marshall initmod: %v", err)
				g.handleGenericError(message)
				return
			}
			err = g.UserDBHandler.SetUserData(message.AuthorID, "initmod", string(jsonBytes))
			if err != nil {
				if err == ErrNoMappingFound {
					g.handleNoMappingFound(message)
					return
				}
				g.handleGenericError(message)
			}
			g.sendMessageDefault(message, "Your init modifier was set to "+strconv.Itoa(initMod)+".")
		}
	} else {
		var initMod []int
		for i := 2; i < len(tokens); i++ {
			currentValue, err := strconv.Atoi(tokens[i])
			if err != nil {
				g.sendMessageDefault(message, "There was an error while parsing your command")
				return
			}
			initMod = append(initMod, currentValue)
		}
		jsonBytes, err := json.Marshal(initMod)
		if err != nil {
			g.Logger.Warning("could not marshall initmod: %v", err)
			g.handleGenericError(message)
			return
		}
		g.UserDBHandler.SetUserData(message.AuthorID, "initmod", string(jsonBytes))
		g.sendMessageDefault(message, "Your init modifier was set to following values: ["+string(jsonBytes)+"].")
	}
}

func (g Bot) handleGenericError(message MessageData) {
	g.sendMessageDefault(message, "Sorry, an internal error occurred. Please try again or contact the bot administrator.")
}

func (g Bot) handleNoMappingFound(message MessageData) {
	sendString := "This account is not connected to an valid matrix or tasadar user id.\nUse the " + g.Prefix +
		"config uid YOUR_ID option to set it or use the " + g.Prefix + "uid command to get more information"
	g.sendMessageDefault(message, sendString)
}

func (g Bot) handleInvalidCommand(message MessageData) {
	g.sendMessageDefault(message, "Unknown Command, to get a list of available command use the "+g.Prefix+"help command")
}

func (g Bot) handleCancelContext(message MessageData) {
	g.SetContext(message.AuthorID, message.ChannelID, "ctx", "", 1*time.Second)
	g.sendMessageDefault(message, "I canceled the process!")
}

func (q QuoteOfTheDay) isValid() bool {
	return q.ValidUntil.After(time.Now())
}

func (g Bot) sendMessageDefault(messageToParse MessageData, messageToSend string) {
	if messageToParse.IsDM {
		err := g.SendMessageToChannel(messageToParse.ChannelID, messageToSend)
		if err != nil {
			g.Logger.Warningf("error sending message in Channel %v: %v", messageToParse.ChannelID, err)
			return
		}
	} else {
		mention, err := g.GetMention(messageToParse.AuthorID)
		if err != nil {
			g.Logger.Warningf("Could not get mention for %v: %v", messageToParse.AuthorID, err)
		}
		err = g.SendMessageToChannel(messageToParse.ChannelID, mention+"\n"+messageToSend)
		if err != nil {
			g.Logger.Warningf("error sending message in Channel %v: %v", messageToParse.ChannelID, err)
			return
		}
	}
}
