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

var addQuoteContextDelay = time.Minute * 15

func (g Bot) handleGetQuote(message MessageData) {
	quoteSel, err := parseGetQuote(message.Content)
	if err != nil {
		g.handleGenericError(message)
	}
	quote, err := g.GetRandomQuote(quoteSel.author, quoteSel.language, quoteSel.universe)
	if err != nil {
		g.handleGenericError(message)
	}
	g.sendMessageDefault(message, quote.Content)
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
	if qotd := g.GetUserData(message.AuthorID, "QuoteOfTheDay"); qotd != nil {
		switch qotd.(type) {
		case stringWithTTL:
			if qotd.(stringWithTTL).isValid() {
				g.sendMessageDefault(message, qotd.(stringWithTTL).Content)
			} else {
				g.handleNewQuoteOfTheDay(message)
			}
		default:
			g.handleGenericError(message)
		}
	}
}

func (g Bot) handleNewQuoteOfTheDay(message MessageData) {
	quote, err := g.GetRandomQuote("", "", "")
	if err != nil {
		g.handleGenericError(message)
	}
	now := time.Now()
	year, month, day := now.Date()
	midnight := time.Date(year, month, day+1, 0, 0, 0, 0, now.Location())
	qotd := stringWithTTL{
		Content:    quote.Content,
		ValidUntil: midnight,
	}
	g.SetUserData(message.AuthorID, "QuoteOfTheDay", qotd)
	g.sendMessageDefault(message, qotd.Content)
}

func (g Bot) addQuoteToDB(message MessageData) error {
	return g.AddQuote(Quote{
		Content:  g.GetContext(message.AuthorID, message.ChannelID, "currentQuote"),
		Author:   g.GetContext(message.AuthorID, message.ChannelID, "currentAuthor"),
		Language: g.GetContext(message.AuthorID, message.ChannelID, "currentLanguage"),
		Universe: g.GetContext(message.AuthorID, message.ChannelID, "currentUniverse"),
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

func parseGetQuote(message string) (quoteSelector, error) {
	// Do not run code if no additional parameters were given
	if message == "getquote" {
		return quoteSelector{}, nil
	}
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
				return quoteSelector{}, errors.New("invalid quote selector")
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
