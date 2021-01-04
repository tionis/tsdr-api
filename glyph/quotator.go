package glyph

import (
	"errors"
	"fmt"
	"strings"
	"time"

	_ "github.com/heroku/x/hmetrics/onload" // Heroku advanced go metrics
)

type quoteSelector struct {
	author   string
	language string
	universe string
}

// ErrNoQuotesFound is thrown if now quotes could be found
var ErrNoQuotesFound = errors.New("no quote found")

// ErrInvalidQuoteSelector is thrown if the given string could not be parsed into
// an quote selector due to errors in the string itself
var ErrInvalidQuoteSelector = errors.New("invalid quote selector")

var addQuoteContextDelay = time.Minute * 15

func (g Bot) handleGetQuote(message MessageData) {
	quoteSel, err := g.parseGetQuote(message.Content)
	if err != nil {
		if err == ErrInvalidQuoteSelector {
			g.sendMessageDefault(message, "There was an error in your command. Please double check it!")
			return
		}
		g.Logger.Warningf("could not parse getQuote: %v", err)
		g.handleGenericError(message)
		return
	}
	quote, err := g.QuoteDBHandler.GetRandomQuote(quoteSel.author, quoteSel.language, quoteSel.universe)
	if err != nil {
		if err == ErrNoQuotesFound {
			g.sendMessageDefault(message, "Sorry, no quote could be found.")
			return
		}
		g.Logger.Warningf("could not get random quote: %v", err)
		g.handleGenericError(message)
		return
	}
	textToSend := quote.Content + "\n- " + quote.Author + " (" + quote.Universe + ")"
	g.sendMessageDefault(message, textToSend)
}

func (g Bot) handleAddQuoteInit(message MessageData) {
	g.SetContext(message.AuthorID, message.ChannelID, "ctx", "quoteRequired", standardContextDelay)
	g.sendMessageDefault(message, "Ok, now please send me your Quote.")
}

func (g Bot) handleAddQuoteContent(message MessageData) {
	g.SetContext(message.AuthorID, message.ChannelID, "currentQuote", message.Content, addQuoteContextDelay)
	g.SetContext(message.AuthorID, message.ChannelID, "ctx", "authorRequired", standardContextDelay)
	g.sendMessageDefault(message, "Ok, now please send me the author of the quote.")
}

func (g Bot) handleAddQuoteAuthor(message MessageData) {
	g.SetContext(message.AuthorID, message.ChannelID, "currentAuthor", message.Content, addQuoteContextDelay)
	g.SetContext(message.AuthorID, message.ChannelID, "ctx", "languageRequired", standardContextDelay)
	g.sendMessageDefault(message, "Ok, now please send me the language the Quote is in.")
}

func (g Bot) handleAddQuoteLanguage(message MessageData) {
	g.SetContext(message.AuthorID, message.ChannelID, "currentLanguage", message.Content, addQuoteContextDelay)
	g.SetContext(message.AuthorID, message.ChannelID, "ctx", "universeRequired", standardContextDelay)
	g.sendMessageDefault(message, "Ok, now please send me the universe the Quote is from.")
}

func (g Bot) handleAddQuoteUniverse(message MessageData) {
	g.SetContext(message.AuthorID, message.ChannelID, "currentUniverse", message.Content, addQuoteContextDelay)
	g.SetContext(message.AuthorID, message.ChannelID, "ctx", "", time.Second)
	g.sendMessageDefault(message, "Ok, saving your Quote now...")
}

func (g Bot) handleAddQuoteFinished(message MessageData) {
	err := g.addQuoteToDB(message)
	if err != nil {
		g.sendMessageDefault(message, "There was an error saving your Quote.\nPlease try again later.")
	} else {
		g.sendMessageDefault(message, "Quote saved!")
	}
}

func (g Bot) handleQuoteOfTheDay(message MessageData) {
	if qotd, err := g.QuoteDBHandler.GetQuoteOfTheDay(message.AuthorID); err == nil {
		if qotd.isValid() {
			quote := qotd.Quote
			g.sendMessageDefault(message, quote.Content+"\n- "+quote.Author+" ("+quote.Universe+")")
		} else {
			g.handleNewQuoteOfTheDay(message)
		}
	} else {
		if err == ErrNoUserDataFound {
			g.handleNewQuoteOfTheDay(message)
		} else if err == ErrNoMappingFound {
			g.sendMessageDefault(message, "You are not logged in, please login via the /user login command.")
		} else {
			g.Logger.Warningf("error handling qotd: %v", err)
			g.handleGenericError(message)
		}
	}
}

func (g Bot) handleNewQuoteOfTheDay(message MessageData) {
	quote, err := g.QuoteDBHandler.GetRandomQuote("", "", "")
	if err != nil {
		g.Logger.Warningf("error handling new qotd: %v", err)
		g.handleGenericError(message)
	}
	now := time.Now()
	year, month, day := now.Date()
	midnight := time.Date(year, month, day+1, 0, 0, 0, 0, now.Location())
	qotd := QuoteOfTheDay{
		Quote:      quote,
		ValidUntil: midnight,
	}
	g.QuoteDBHandler.SetQuoteOfTheDay(message.AuthorID, qotd)
	g.sendMessageDefault(message, quote.Content+"\n- "+quote.Author+" ("+quote.Universe+")")
}

func (g Bot) addQuoteToDB(message MessageData) error {
	quote, err := g.GetContext(message.AuthorID, message.ChannelID, "currentQuote")
	if err != nil {
		return err
	}
	author, err := g.GetContext(message.AuthorID, message.ChannelID, "currentAuthor")
	if err != nil {
		return err
	}
	language, err := g.GetContext(message.AuthorID, message.ChannelID, "currentLanguage")
	if err != nil {
		return err
	}
	universe, err := g.GetContext(message.AuthorID, message.ChannelID, "currentUniverse")
	if err != nil {
		return err
	}
	return g.QuoteDBHandler.AddQuote(Quote{
		Content:  quote,
		Author:   author,
		Language: language,
		Universe: universe,
	})
}

// parseCommandString takes a string and parses it as string slice using "" and spaces as seperators
func parseCommandString(command string) ([]string, error) {
	var args []string
	state := "start"
	current := ""
	quote := "\""
	escapeNext := true
	for i := 0; i < len(command); i++ {
		c := command[i]

		if state == "quotes" {
			if string(c) != quote {
				current += string(c)
			} else {
				args = append(args, current)
				current = ""
				state = "start"
			}
			continue
		}

		if escapeNext {
			current += string(c)
			escapeNext = false
			continue
		}

		if c == '\\' {
			escapeNext = true
			continue
		}

		if c == '"' || c == '\'' {
			state = "quotes"
			quote = string(c)
			continue
		}

		if state == "arg" {
			if c == ' ' || c == '\t' {
				args = append(args, current)
				current = ""
				state = "start"
			} else {
				current += string(c)
			}
			continue
		}

		if c != ' ' && c != '\t' {
			state = "arg"
			current += string(c)
		}
	}

	if state == "quotes" {
		return []string{}, fmt.Errorf("unclosed quote in command line: %s", command)
	}

	if current != "" {
		args = append(args, current)
	}

	return args, nil
}

func (g Bot) parseGetQuote(message string) (quoteSelector, error) {
	// Do not run code if no additional parameters were given

	if strings.ToLower(message) == "getquote" {
		return quoteSelector{}, nil
	}
	message = strings.TrimPrefix(message, "getQuote")
	message = strings.TrimPrefix(message, "getquote")
	message = strings.TrimLeft(message, "\t \r \n \v \f")
	args, err := parseCommandString(message)
	if err != nil {
		return quoteSelector{}, err
	}
	var author, language, universe string // These shall be handles below as 1, 2, 3 respectivly
	var variableToSet int
	for i, current := range args {
		if i%2 == 0 {
			switch strings.ToLower(current) {
			case "author":
				variableToSet = 1
			case "language":
				variableToSet = 2
			case "universe":
				variableToSet = 3
			default:
				g.Logger.Debugf("|%v|", strings.ToLower(current))
				return quoteSelector{}, ErrInvalidQuoteSelector
			}
		} else {
			switch variableToSet {
			case 1:
				author = current
			case 2:
				language = strings.ToLower(current)
			case 3:
				universe = current
			}
		}
	}

	switch language {
	case "deutsch", "aleman", "alemán":
		language = "german"
	case "englisch", "ingles", "inglés":
		language = "english"
	case "latein":
		language = "latin"
	case "spanisch", "espanol", "español":
		language = "spanish"
	}
	return quoteSelector{
		author:   author,
		language: language,
		universe: universe,
	}, nil
}
