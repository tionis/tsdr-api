package glyph

import (
	"errors"
	"fmt"
	"strings"
)

type quoteSelector struct {
	author   string
	language string
	universe string
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
	// TODO what the hell did I mean to do here????
	if message == "/getquote" {
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
