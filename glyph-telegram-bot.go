package main

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	_ "github.com/heroku/x/hmetrics/onload"
	"github.com/keybase/go-logging"
	UniPassauBot "github.com/tionis/uni-passau-bot/api"
	tb "gopkg.in/tucnak/telebot.v2"
)

var glyphTelegramLog = logging.MustGetLogger("glyphTelegram")

const glyphTelegramContextDelay = time.Hour * 24

var msgGlyph chan string

// GlyphTelegramBot handles all the legacy Glyph-Telegram-Bot code for telegram
func glyphTelegramBot() {

	botquit := make(chan bool) // channel for quitting of bot

	// catch os signals like sigterm and interrupt
	signalChannel := make(chan os.Signal, 2)
	signal.Notify(signalChannel, os.Interrupt, syscall.SIGTERM)
	go func() {
		sig := <-signalChannel
		switch sig {
		case os.Interrupt:
			glyphTelegramLog.Info("Interruption Signal received, shutting down...")
			exit(botquit)
		case syscall.SIGTERM:
			botquit <- true
		}
	}()

	// Define Keyboards
	// Define Keyboards for Quotator
	replyBtnLanguageTopLeft := tb.ReplyButton{Text: "English"}
	replyBtnLanguageTopRight := tb.ReplyButton{Text: "German"}
	replyBtnLanguageBottomLeft := tb.ReplyButton{Text: "Latin"}
	replyBtnLanguageBottomRight := tb.ReplyButton{Text: "Spanish"}
	replyKeysLanguage := [][]tb.ReplyButton{
		{replyBtnLanguageTopLeft, replyBtnLanguageTopRight}, {replyBtnLanguageBottomLeft, replyBtnLanguageBottomRight}}

	// check for and read config variable, then create bot object
	glyphToken := os.Getenv("TELEGRAM_TOKEN")
	glyph, err := tb.NewBot(tb.Settings{
		Token:  glyphToken,
		Poller: &tb.LongPoller{Timeout: 10 * time.Second},
	})
	if err != nil {
		glyphTelegramLog.Fatal(err)
		return
	}

	// Command Handlers
	// handle general standard text commands
	glyph.Handle("/hello", func(m *tb.Message) {
		_, _ = glyph.Send(m.Sender, "What do you want?", &tb.ReplyMarkup{ReplyKeyboardRemove: true})
		printInfoGlyph(m)
	})
	glyph.Handle("/start", func(m *tb.Message) {
		_, _ = glyph.Send(m.Sender, "Hello.", &tb.ReplyMarkup{ReplyKeyboardRemove: true})
		printInfoGlyph(m)
	})
	glyph.Handle("Thanks", func(m *tb.Message) {
		_, _ = glyph.Send(m.Sender, "_It's a pleasure!_", tb.ModeMarkdown)
		printInfoGlyph(m)
	})
	glyph.Handle("/ping", func(m *tb.Message) {
		_, _ = glyph.Send(m.Sender, "_pong_", tb.ModeMarkdown)
		printInfoGlyph(m)
	})
	glyph.Handle("/help", func(m *tb.Message) {
		sendString := ""
		if isTasadarTGAdmin(m.Sender.ID) {
			sendString = `**Following Commands are Available:**
**UniPassau-Commands:**
  - /foodtoday - Food for today
  - /foodtomorrow - Food for tomorrow
  - /foodweek - Food for week
**Quotator-Commands**
  - /getquote - Get a random quote. You can also specify parameters by saying for example:  /getquote language german author "Emanuel Kant" 
  - /addquote - add a quote to the database
  - /quoteoftheday - Get your personal quote of the day`
		} else {
			sendString = "There is no help!"
		}
		_, _ = glyph.Send(m.Sender, sendString, tb.ModeMarkdown, &tb.ReplyMarkup{ReplyKeyboardRemove: true})
		printInfoGlyph(m)
	})

	// Handle Uni Passau Commands
	glyph.Handle("/food", func(m *tb.Message) {
		if !m.Private() {
			_, _ = glyph.Send(m.Chat, UniPassauBot.FoodToday())
			glyphTelegramLog.Info("Group Message:")
		} else {
			_, _ = glyph.Send(m.Sender, UniPassauBot.FoodToday(), tb.ModeMarkdown)
		}
		printInfoGlyph(m)
	})
	glyph.Handle("food", func(m *tb.Message) {
		if !m.Private() {
			_, _ = glyph.Send(m.Chat, UniPassauBot.FoodToday())
			glyphTelegramLog.Info("Group Message:")
		} else {
			_, _ = glyph.Send(m.Sender, UniPassauBot.FoodToday(), tb.ModeMarkdown)
		}
		printInfoGlyph(m)
	})
	glyph.Handle("/foodtoday", func(m *tb.Message) {
		if !m.Private() {
			_, _ = glyph.Send(m.Chat, UniPassauBot.FoodToday(), tb.ModeMarkdown)
			glyphTelegramLog.Info("Group Message:")
		} else {
			_, _ = glyph.Send(m.Sender, UniPassauBot.FoodToday(), tb.ModeMarkdown)
		}
		printInfoGlyph(m)
	})
	glyph.Handle("/foodtomorrow", func(m *tb.Message) {
		if !m.Private() {
			_, _ = glyph.Send(m.Chat, UniPassauBot.FoodTomorrow(), tb.ModeMarkdown)
			glyphTelegramLog.Info("Group Message:")
		} else {
			_, _ = glyph.Send(m.Sender, UniPassauBot.FoodTomorrow(), tb.ModeMarkdown)
		}
		printInfoGlyph(m)
	})
	glyph.Handle("food tomorrow", func(m *tb.Message) {
		if !m.Private() {
			_, _ = glyph.Send(m.Chat, UniPassauBot.FoodTomorrow(), tb.ModeMarkdown)
			glyphTelegramLog.Info("Group Message:")
		} else {
			_, _ = glyph.Send(m.Sender, UniPassauBot.FoodTomorrow(), tb.ModeMarkdown)
		}
		printInfoGlyph(m)
	})
	glyph.Handle("/foodweek", func(m *tb.Message) {
		if !m.Private() {
			_, _ = glyph.Send(m.Chat, UniPassauBot.FoodWeek())
			glyphTelegramLog.Info("Group Message:")
		} else {
			_, _ = glyph.Send(m.Sender, UniPassauBot.FoodWeek(), tb.ModeMarkdown)
		}
		printInfoGlyph(m)
	})
	glyph.Handle("food week", func(m *tb.Message) {
		if !m.Private() {
			_, _ = glyph.Send(m.Chat, UniPassauBot.FoodWeek())
			glyphTelegramLog.Info("Group Message:")
		} else {
			_, _ = glyph.Send(m.Sender, UniPassauBot.FoodWeek(), tb.ModeMarkdown)
		}
		printInfoGlyph(m)
	})

	// Handle Quotator Commands
	glyph.Handle("/getquote", func(m *tb.Message) {
		delTmp("glyph", "telegram:"+strconv.Itoa(m.Sender.ID)+"|context")
		author, language, universe, err := parseGetQuote(strings.TrimPrefix(m.Text, "/getquote "))
		if err != nil {
			glyphTelegramLog.Error("[glyph] Error parsing getQuote: ", err)
			_, _ = glyph.Send(m.Chat, "There was an error please check your command and try again later.", &tb.ReplyMarkup{ReplyKeyboardRemove: true})
		} else {
			_, _ = glyph.Send(m.Chat, getRandomQuote(author, language, universe), &tb.ReplyMarkup{ReplyKeyboardRemove: true})
		}
	})
	glyph.Handle("/setquote", func(m *tb.Message) {
		setTmp("glyph", "telegram:"+strconv.Itoa(m.Sender.ID)+"|context", "quoteRequired", glyphTelegramContextDelay)
		_, _ = glyph.Send(m.Chat, "Please write me your Quote.", &tb.ReplyMarkup{ReplyKeyboardRemove: true})
	})
	glyph.Handle("/addquote", func(m *tb.Message) {
		setTmp("glyph", "telegram:"+strconv.Itoa(m.Sender.ID)+"|context", "quoteRequired", glyphTelegramContextDelay)
		_, _ = glyph.Send(m.Chat, "Please write me your Quote.", &tb.ReplyMarkup{ReplyKeyboardRemove: true})
	})
	glyph.Handle("/quoteoftheday", func(m *tb.Message) {
		quote := getTmp("glyph", "telegram:"+strconv.Itoa(m.Sender.ID)+"|dayquote")
		if quote != "" {
			_, _ = glyph.Send(m.Chat, quote)
		} else {
			quote = getRandomQuote("", "", "")
			now := time.Now()
			year, month, day := now.Date()
			midnight := time.Date(year, month, day+1, 0, 0, 0, 0, now.Location())
			setTmp("glyph", "telegram:"+strconv.Itoa(m.Sender.ID)+"|dayquote", quote, time.Until(midnight))
			_, _ = glyph.Send(m.Chat, quote)
		}
		printInfoGlyph(m)
	})

	// Handle non command text
	glyph.Handle(tb.OnText, func(m *tb.Message) {
		context := getTmp("glyph", "telegram:"+strconv.Itoa(m.Sender.ID)+"|context")
		if context == "" {
			if !m.Private() {
				printInfoGlyph(m)
			} else {
				_, _ = glyph.Send(m.Chat, "Unknown Command - use help to get a list of available commands", &tb.ReplyMarkup{ReplyKeyboardRemove: true})
				printInfoGlyph(m)
			}
		} else {
			switch context {
			case "quoteRequired":
				setTmp("glyph", "telegram:"+strconv.Itoa(m.Sender.ID)+"|currentQuote", m.Text, glyphTelegramContextDelay)
				setTmp("glyph", "telegram:"+strconv.Itoa(m.Sender.ID)+"|context", "authorRequired", glyphTelegramContextDelay)
				_, _ = glyph.Send(m.Sender, "Thanks, now the author please.")
				printInfoGlyph(m)
			case "authorRequired":
				setTmp("glyph", "telegram:"+strconv.Itoa(m.Sender.ID)+"|currentAuthor", m.Text, glyphTelegramContextDelay)
				setTmp("glyph", "telegram:"+strconv.Itoa(m.Sender.ID)+"|context", "languageRequired", glyphTelegramContextDelay)
				_, _ = glyph.Send(m.Sender, "Thanks, now the language please.", &tb.ReplyMarkup{ReplyKeyboard: replyKeysLanguage})
				printInfoGlyph(m)
			case "languageRequired":
				setTmp("glyph", "telegram:"+strconv.Itoa(m.Sender.ID)+"|currentLanguage", m.Text, glyphTelegramContextDelay)
				setTmp("glyph", "telegram:"+strconv.Itoa(m.Sender.ID)+"|context", "universeRequired", glyphTelegramContextDelay)
				_, _ = glyph.Send(m.Sender, "And now the universe it comes from please:", &tb.ReplyMarkup{ReplyKeyboardRemove: true})
				printInfoGlyph(m)
			case "universeRequired":
				setTmp("glyph", "telegram:"+strconv.Itoa(m.Sender.ID)+"|currentUniverse", m.Text, glyphTelegramContextDelay)
				delTmp("glyph", "telegram:"+strconv.Itoa(m.Sender.ID)+"|context")
				_, _ = glyph.Send(m.Sender, addQuote(m))
				printInfoGlyph(m)
			default:
				_, _ = glyph.Send(m.Sender, "Unknown Command - use help to get a list of available commands")
			}

		}
	})

	// Graceful Shutdown (botquit)
	go func() {
		<-botquit
		glyph.Stop()
		glyphTelegramLog.Info("Glyph Telegram Bot was stopped")
		os.Exit(3)
	}()

	// Channel for sending messages
	go func(glyph *tb.Bot) {
		msgGlyph = make(chan string)
		for {
			toSend := <-msgGlyph
			tionis := tb.Chat{ID: 248533143}
			_, _ = glyph.Send(&tionis, toSend, tb.ModeMarkdown)
		}
	}(glyph)

	// print startup message
	glyphTelegramLog.Info("Glyph Telegram Bot was started.")
	glyph.Start()
}

// General Telegram Glyph Logic

func printInfoGlyph(m *tb.Message) {
	glyphTelegramLog.Info(m.Sender.Username + " - " + m.Sender.FirstName + " " + m.Sender.LastName + " - ID: " + strconv.Itoa(m.Sender.ID) + "Message: " + m.Text)
}

func isTasadarTGAdmin(ID int) bool {
	return ID == 248533143
	/*if ID == 248533143 {
		return true
	}
	username := kvget("tg|" + strconv.Itoa(ID) + "|username")
	groups := kvget("auth|" + username + "|groups")
	// Should transform into array and then check through it
	if strings.Contains(groups, "admin,") || strings.Contains(groups, ",admin") //|| strings.Contains(groups, "admin")// {
		return true
	}
	return false*/
}

// Quotator Logic

func parseGetQuote(message string) (string, string, string, error) {
	if message == "/getquote" {
		return "", "", "", nil
	}
	args, err := parseString(message)
	if err != nil {
		return "", "", "", err
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
				glyphTelegramLog.Warning(current)
				return "", "", "", errors.New("invalid quote selector")
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
	return author, language, universe, nil
}

func parseString(command string) ([]string, error) {
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

func getRandomQuote(byAuthor, inLanguage, inUniverse string) string {
	stmt, err := db.Prepare(`SELECT quote, author FROM quotes WHERE (length($1)=0 OR author=$1) AND (length($2)=0 OR language=$2) AND (length($3)=0 OR universe=$3) ORDER BY RANDOM() LIMIT 1`)
	if err != nil {
		glyphTelegramLog.Error("Couldn't prepare statement: ", err)
		return "Sorry, an internal error occurred!"
	}
	row := stmt.QueryRow(byAuthor, inLanguage, inUniverse)

	var quote, author string
	err = row.Scan(&quote, &author)
	if err != nil {
		if err == sql.ErrNoRows {
			return "Sorry, no quote found."
		}
		glyphTelegramLog.Error("Error getting random quote from database: ", err)
		return "There was an internal error!"
	}
	return quote + "\n- " + author
}

func addQuote(m *tb.Message) string {
	quote := getTmp("glyph", "telegram:"+strconv.Itoa(m.Sender.ID)+"|currentQuote")
	author := getTmp("glyph", "telegram:"+strconv.Itoa(m.Sender.ID)+"|currentAuthor")
	language := strings.ToLower(getTmp("glyph", "telegram:"+strconv.Itoa(m.Sender.ID)+"|currentLanguage"))
	universe := getTmp("glyph", "telegram:"+strconv.Itoa(m.Sender.ID)+"|currentUniverse")
	stmt, err := db.Prepare(`INSERT INTO quotes (quote, author, language, universe) VALUES ($1, $2, $3, $4)`)
	if err != nil {
		glyphTelegramLog.Error("Error preparing database statement: ", err)
		return "Sorry, there was an internal error!"
	}
	_, err = stmt.Query(quote, author, language, universe)
	if err != nil {
		glyphTelegramLog.Error("Error executing database statement: ", err)
		return "Sorry, there was an internal error!"
	}
	return "Added quote from " + author + " to database"
}

// Stop the program and kill hanging routines
func exit(quit chan bool) {
	// function for normal exit
	quit <- true
	simpleExit()
}

// Exit while ignoring running routines
func simpleExit() {
	// Exit without using graceful shutdown channels
	glyphTelegramLog.Info("Shutting down...")
	os.Exit(0)
}
