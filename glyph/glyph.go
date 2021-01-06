package glyph

import (
	"encoding/json"
	"math/rand"
	"strconv"
	"strings"
	"time"

	_ "github.com/heroku/x/hmetrics/onload" // Heroku advanced go metrics
	"github.com/keybase/go-logging"

	UniPassauBot "github.com/tionis/uni-passau-bot/api" // This provides the uni passau food functionality
)

// Valid letters for an auth session token
var letters = []rune("abcdefghijklmnopqrstuvwxyz1234567890")

// AuthSessionDelay describes how long a session stays valid
const AuthSessionDelay = time.Hour

// MessageData represents an message the bot can act on with callback functions
type MessageData struct {
	Content          string
	AuthorID         string
	IsDM             bool
	SupportsMarkdown bool
	ChannelID        string
	IsCommand        bool
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
	SetUserData           func(userID, key string, value string) error                                 // SetUserData saves data to a specific user by key
	GetUserData           func(userID, key string) (string, error)                                     // GetUserData gets data to a specific user by key
	DeleteUserData        func(userID, key string) error                                               // DeleteUserData deletes user data with a given key
	GetMatrixUserID       func(userID string) (string, error)                                          // GetMatrixUserID gets the MatrixID from the implementation specific userID
	DoesMatrixUserIDExist func(matrixUserID string) (bool, error)                                      // DoesMatrixUserIDExist checks if a user with given matrixID is already registered (this is used with a hardcoded transformation from tasadar user to matrix id)
	AddAuthSession        func(key, value, userID string) (string, error)                              // AddAuthSession adds an auth session with an key and value which will be set in the userdb when the login was successful
	AuthenticateSession   func(matrixUserID, authSessionID string) error                               // AuthenticateSession sets the session with given ID as authenticated
	DeleteSession         func(authSessionID string) error                                             // DeleteSession deletes the session with given ID
	GetAuthSessions       func(matrixID string) ([]string, error)                                      // GetAuthSessions return the state of all sessions registered to the user
	RegisterNewUser       func(matrixID, email string, isAdmin bool, preferredAdapters []string) error // RegisterNewUser add a new user to the database
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
	QuoteDBHandler        *QuoteDB                                                            // QuoteDBHandler implements functions to interact with the quote database
	UserDBHandler         *UserDB                                                             // UserDBHandler implements functions to interact with user-specific data
	GetMention            func(userID string) (string, error)                                 // GetMention when passed an userID returns an string mentioning the user
	GetContext            func(userID, channelID, key string) (string, error)                 // GetContext when passed an channelID and UserID returns the current chat context for the specified key
	SetContext            func(userID, channelID, key, value string, ttl time.Duration) error // SetContext allows setting the current channelID+UserID context with a specific key
	SendMessageToChannel  func(channelID, message string) error                               // SendMessageToChannel sends a simple text message to specified channel
	Prefix                string                                                              // The command prefix used
	SendMessageViaAdapter func(matrixUserID, message string) error                            // Sends a message to a users favored adapter
	CurrentAdapter        string                                                              // Specifies the id of the currently used adapter
	Logger                *logging.Logger                                                     // A Logger implementation to send logs to
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
	case "user":
		go g.handleUser(message, tokens)
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
	case "registerIDRequired":
		// TODO get id, transform it to a valid one and save it then ask if they want to add an email to the account
	case "registerEmailWanted":
		// TODO get answer and parse it, if (yes) set context to below, if not set finish register user process and
	case "registerEmailRequired":
		// TODO check if mail is valid, then finish register user process
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
		case ErrNoSuchSession: // No auth session found
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

func (g Bot) handleUser(message MessageData, tokens []string) {
	if len(tokens) < 2 {
		g.sendMessageDefault(message, "Interact with your user account. Currently available:\n - /user login {userID} - Log into your account\n - /user register - Register an new account\n - /user send-token - Configure tokens allowing sending of messages to you over the API.")
	} else {
		switch tokens[1] {
		case "login":
			if len(tokens) < 3 {
				g.sendMessageDefault(message, "Please specify your userID to log in.")
			} else {
				var userID string
				if IsValidMatrixID.MatchString(tokens[2]) {
					userID = tokens[2]
				} else {
					userID = "@" + tokens[2] + "_virtual:tasadar.net"
				}

				doesExist, err := g.UserDBHandler.DoesMatrixUserIDExist(userID)
				if err != nil {
					g.Logger.Warning("error checking if user ID exists: ", err)
					g.handleGenericError(message)
				}
				authCode := GenerateAuthSessionID()
				if doesExist {
					authCode, err = g.UserDBHandler.AddAuthSession(g.CurrentAdapter, message.AuthorID, userID)
					if err != nil {
						g.Logger.Warning("could not add auth session: ", err)
						g.handleGenericError(message)
					}
				}
				g.sendMessageDefault(message, "Login process started. To login please write me following command on platform on which you are logged in.\n/auth "+authCode)
			}
		case "register":
			g.SetContext(message.AuthorID, message.ChannelID, "ctx", "registerIDRequired", standardContextDelay)
			g.sendMessageDefault(message, "The register process was started. To cancel it write me /cancel.\nPlease specify your desired username (or use your existing matrixID if you have one.)")
		case "send-token":
			if len(tokens) < 3 {
				g.sendMessageDefault(message, "Manage your send-tokens. Currently available:\n - /user send-token generate - Generate a new send token\n - /user send-token list - List all your current send-tokens\n - /user send-token delete {send-token} - Delete specified send-token\n - /user send-token help - Get more info about send-tokens")
			} else {
				switch tokens[2] {
				case "generate", "new":
					// TODO generate a new UUID-V4 token and save it to db. Also output it to user
				case "list":
					// TODO list all tokens and when they were last used
				case "delete":
					// TODO search in list for token with UID and delete it.
				default:
					g.sendMessageDefault(message, "Unknown Command! Please chek your spelling!")
				}
			}
		default:
			g.sendMessageDefault(message, "Unknown Command! Please chek your spelling!")
		}
	}
}

func (g Bot) handleConfig(message MessageData, tokens []string) {
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
				g.Logger.Warningf("error setting userData: %v", err)
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

func (g Bot) sendMessageDefault(messageToParse MessageData, textToSend string) {
	if messageToParse.IsDM {
		err := g.SendMessageToChannel(messageToParse.ChannelID, textToSend)
		if err != nil {
			g.Logger.Warningf("error sending message in Channel %v: %v", messageToParse.ChannelID, err)
			return
		}
	} else {
		mention, err := g.GetMention(messageToParse.AuthorID)
		if err != nil {
			g.Logger.Warningf("Could not get mention for %v: %v", messageToParse.AuthorID, err)
		}
		err = g.SendMessageToChannel(messageToParse.ChannelID, mention+"\n"+textToSend)
		if err != nil {
			g.Logger.Warningf("error sending message in Channel %v: %v", messageToParse.ChannelID, err)
			return
		}
	}
}

// GenerateAuthSessionID generates a random ID for an auth session
func GenerateAuthSessionID() string {
	// This small number of characters leads to ~50% probability of collision when generating 54562 tokens.
	// -> checking of duplicate tokens may be necessary, leaving it as is to make user facing tokens simpler
	return randSeq(6)
}

func randSeq(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}
