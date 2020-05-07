package main

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	tb "gopkg.in/tucnak/telebot.v2"
)

const quotatorContextDelay = time.Hour * 24

func quotatorTelegramBot() {
	token := os.Getenv("QUOTATOR_TOKEN")
	if token == "" {
		log.Fatal("[Quotator] No token given!")
	}
	botquit := make(chan bool) // channel for quitting of bot

	//Define Keyboards
	replyBtnLanguageTopLeft := tb.ReplyButton{Text: "English"}
	replyBtnLanguageTopRight := tb.ReplyButton{Text: "German"}
	replyBtnLanguageBottomLeft := tb.ReplyButton{Text: "Latin"}
	replyBtnLanguageBottomRight := tb.ReplyButton{Text: "Spanish"}
	replyKeysLanguage := [][]tb.ReplyButton{
		{replyBtnLanguageTopLeft, replyBtnLanguageTopRight}, {replyBtnLanguageBottomLeft, replyBtnLanguageBottomRight}}

	// catch os signals like sigterm and interrupt
	signalChannel := make(chan os.Signal, 2)
	signal.Notify(signalChannel, os.Interrupt, syscall.SIGTERM)
	go func() {
		sig := <-signalChannel
		switch sig {
		case os.Interrupt:
			log.Println("[GlyphTelegramBot] " + "Interruption Signal received, shutting down...")
			exit(botquit)
		case syscall.SIGTERM:
			botquit <- true
		}
	}()

	// check for and read config variable, then create bot object
	quotator, err := tb.NewBot(tb.Settings{
		Token:  token,
		Poller: &tb.LongPoller{Timeout: 10 * time.Second},
	})
	if err != nil {
		log.Fatal(err)
		return
	}

	// Command Handlers
	// handle standard text commands
	quotator.Handle("/hello", func(m *tb.Message) {
		del("quotator|telegram:" + strconv.Itoa(m.Sender.ID) + "|context")
		_, _ = quotator.Send(m.Chat, "What do you want?", &tb.ReplyMarkup{ReplyKeyboardRemove: true})
		printInfoQuotator(m)
	})
	quotator.Handle("/start", func(m *tb.Message) {
		del("quotator|telegram:" + strconv.Itoa(m.Sender.ID) + "|context")
		_, _ = quotator.Send(m.Chat, "Hello.", &tb.ReplyMarkup{ReplyKeyboardRemove: true})
		printInfoQuotator(m)
	})
	quotator.Handle("/help", func(m *tb.Message) {
		del("quotator|telegram:" + strconv.Itoa(m.Sender.ID) + "|context")
		_, _ = quotator.Send(m.Chat, "Following Commands are available:\n/help - Show this message\n/getquote - Get a random quote. You can also specify parameters by saying for example:  /getquote language german author \"Emanuel Kant\" \n/addquote - add a quote to the database\n/quoteoftheday - Get your personal quote of the day", &tb.ReplyMarkup{ReplyKeyboardRemove: true})
		printInfoQuotator(m)
	})
	quotator.Handle("/getquote", func(m *tb.Message) {
		del("quotator|telegram:" + strconv.Itoa(m.Sender.ID) + "|context")
		author, language, universe, err := parseGetQuote(strings.TrimPrefix(m.Text, "/getquote "))
		if err != nil {
			log.Println("[Quotator] Error parsing getQuote: ", err)
			_, _ = quotator.Send(m.Chat, "There was an error please check your command and try again later.", &tb.ReplyMarkup{ReplyKeyboardRemove: true})
		} else {
			_, _ = quotator.Send(m.Chat, getRandomQuote(author, language, universe), &tb.ReplyMarkup{ReplyKeyboardRemove: true})
		}
	})
	quotator.Handle("/setquote", func(m *tb.Message) {
		setWithTimer("quotator|telegram:"+strconv.Itoa(m.Sender.ID)+"|context", "quoteRequired", quotatorContextDelay)
		_, _ = quotator.Send(m.Chat, "Please write me your Quote.", &tb.ReplyMarkup{ReplyKeyboardRemove: true})
	})
	quotator.Handle("/addquote", func(m *tb.Message) {
		setWithTimer("quotator|telegram:"+strconv.Itoa(m.Sender.ID)+"|context", "quoteRequired", quotatorContextDelay)
		_, _ = quotator.Send(m.Chat, "Please write me your Quote.", &tb.ReplyMarkup{ReplyKeyboardRemove: true})
	})
	quotator.Handle("/quoteoftheday", func(m *tb.Message) {
		quote := get("quotator|telegram:" + strconv.Itoa(m.Sender.ID) + "|dayquote")
		if quote != "" {
			_, _ = quotator.Send(m.Chat, quote)
		} else {
			quote = getRandomQuote("", "", "")
			now := time.Now()
			year, month, day := now.Date()
			midnight := time.Date(year, month, day+1, 0, 0, 0, 0, now.Location())
			redclient.Set("quotator|telegram:"+strconv.Itoa(m.Sender.ID)+"|dayquote", quote, time.Until(midnight))
			_, _ = quotator.Send(m.Chat, quote)
		}
		printInfoQuotator(m)
	})
	quotator.Handle(tb.OnText, func(m *tb.Message) {
		context := get("quotator|telegram:" + strconv.Itoa(m.Sender.ID) + "|context")
		if context == "" {
			if !m.Private() {
				printInfoGlyph(m)
			} else {
				_, _ = quotator.Send(m.Chat, "Unknown Command - use help to get a list of available commands", &tb.ReplyMarkup{ReplyKeyboardRemove: true})
				printInfoGlyph(m)
			}
		} else {
			switch context {
			case "quoteRequired":
				setWithTimer("quotator|telegram:"+strconv.Itoa(m.Sender.ID)+"|currentQuote", m.Text, quotatorContextDelay)
				setWithTimer("quotator|telegram:"+strconv.Itoa(m.Sender.ID)+"|context", "authorRequired", quotatorContextDelay)
				_, _ = quotator.Send(m.Sender, "Thanks, now the author please.")
				printInfoGlyph(m)
			case "authorRequired":
				setWithTimer("quotator|telegram:"+strconv.Itoa(m.Sender.ID)+"|currentAuthor", m.Text, quotatorContextDelay)
				setWithTimer("quotator|telegram:"+strconv.Itoa(m.Sender.ID)+"|context", "languageRequired", quotatorContextDelay)
				_, _ = quotator.Send(m.Sender, "Thanks, now the language please.", &tb.ReplyMarkup{ReplyKeyboard: replyKeysLanguage})
				printInfoGlyph(m)
			case "languageRequired":
				setWithTimer("quotator|telegram:"+strconv.Itoa(m.Sender.ID)+"|currentLanguage", m.Text, quotatorContextDelay)
				setWithTimer("quotator|telegram:"+strconv.Itoa(m.Sender.ID)+"|context", "universeRequired", quotatorContextDelay)
				_, _ = quotator.Send(m.Sender, "And now the universe it comes from please:", &tb.ReplyMarkup{ReplyKeyboardRemove: true})
				printInfoQuotator(m)
			case "universeRequired":
				setWithTimer("quotator|telegram:"+strconv.Itoa(m.Sender.ID)+"|currentUniverse", m.Text, quotatorContextDelay)
				del("quotator|telegram:" + strconv.Itoa(m.Sender.ID) + "|context")
				quotator.Send(m.Sender, addQuote(m))
				printInfoQuotator(m)
			}

		}
	})

	// Graceful Shutdown (botquit)
	go func() {
		<-botquit
		quotator.Stop()
		log.Println("[Quotator] " + "Glyph Telegram Bot was stopped")
		os.Exit(3)
	}()

	// Start the bot
	log.Println("[Quotator] " + "Telegram Bot was started.")
	quotator.Start()
}

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
				log.Println(current)
				return "", "", "", errors.New("Invalid quote selector")
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
		return []string{}, fmt.Errorf("Unclosed quote in command line: %s", command)
	}

	if current != "" {
		args = append(args, current)
	}

	return args, nil
}

func getRandomQuote(byAuthor, inLanguage, inUniverse string) string {
	/*db, err := sql.Open("postgres", databaseURL)
	if err != nil {
		log.Println("[Quotator] Couldn't connect to database: ", err)
		return "Sorry, an internal error occurred!"
	}*/

	stmt, err := db.Prepare(`SELECT quote, author FROM quotes WHERE (length($1)=0 OR author=$1) AND (length($2)=0 OR language=$2) AND (length($3)=0 OR universe=$3) ORDER BY RANDOM() LIMIT 1`)
	if err != nil {
		log.Println("[Quotator] Couldn't prepare statement: ", err)
		return "Sorry, an internal error occurred!"
	}
	row := stmt.QueryRow(byAuthor, inLanguage, inUniverse)

	var quote, author string
	err = row.Scan(&quote, &author)
	if err != nil {
		if err == sql.ErrNoRows {
			return "Sorry, no quote found."
		}
		log.Println("[Quotator] Error getting random quote from database: ", err)
		return "There was an internal error!"
	}
	//db.Close()
	return quote + "\n- " + author
}

func addQuote(m *tb.Message) string {
	/*db, err := sql.Open("postgres", databaseURL)
	if err != nil {
		log.Println("[Quotator] Couldn't connect to database: ", err)
		return "Sorry, an internal error occurred!"
	}*/
	quote := get("quotator|telegram:" + strconv.Itoa(m.Sender.ID) + "|currentQuote")
	author := get("quotator|telegram:" + strconv.Itoa(m.Sender.ID) + "|currentAuthor")
	language := strings.ToLower(get("quotator|telegram:" + strconv.Itoa(m.Sender.ID) + "|currentLanguage"))
	universe := get("quotator|telegram:" + strconv.Itoa(m.Sender.ID) + "|currentUniverse")
	stmt, err := db.Prepare(`INSERT INTO quotes (quote, author, language, universe) VALUES ($1, $2, $3, $4)`)
	if err != nil {
		log.Println("[Quotator] Error preparing database statement: ", err)
		return "Sorry, there was an internal error!"
	}
	_, err = stmt.Query(quote, author, language, universe)
	if err != nil {
		log.Println("[Quotator] Error executing database statement: ", err)
		return "Sorry, there was an internal error!"
	}
	//db.Close()
	return "Added quote from " + author + " to database"
}

func printInfoQuotator(m *tb.Message) {
	log.Println("[Quotator] Telegram: " + m.Sender.Username + " - " + m.Sender.FirstName + " " + m.Sender.LastName + " - ID: " + strconv.Itoa(m.Sender.ID) + " Message: " + m.Text)
}
