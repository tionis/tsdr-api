package main

import (
	"database/sql"
	"log"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	tb "gopkg.in/tucnak/telebot.v2"
)

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
		_, _ = quotator.Send(m.Sender, "What do you want?", &tb.ReplyMarkup{ReplyKeyboardRemove: true})
		printInfoQuotator(m)
	})
	quotator.Handle("/start", func(m *tb.Message) {
		del("quotator|telegram:" + strconv.Itoa(m.Sender.ID) + "|context")
		_, _ = quotator.Send(m.Sender, "Hello.", &tb.ReplyMarkup{ReplyKeyboardRemove: true})
		printInfoQuotator(m)
	})
	quotator.Handle("/help", func(m *tb.Message) {
		del("quotator|telegram:" + strconv.Itoa(m.Sender.ID) + "|context")
		_, _ = quotator.Send(m.Sender, "Following Commands are available:\n/help - Show this message\n/getquote - Get a random quote. You can also specify parameters by saying for example:  /getquote language:german\n/setquote - add a quote to the database", &tb.ReplyMarkup{ReplyKeyboardRemove: true})
		printInfoQuotator(m)
	})
	quotator.Handle("/getquote", func(m *tb.Message) {
		del("quotator|telegram:" + strconv.Itoa(m.Sender.ID) + "|context")
		_, _ = quotator.Send(m.Sender, getRandomQuote("", "", ""), &tb.ReplyMarkup{ReplyKeyboardRemove: true})

	})
	quotator.Handle("/setquote", func(m *tb.Message) {
		set("quotator|telegram:"+strconv.Itoa(m.Sender.ID)+"|context", "quoteRequired")
		_, _ = quotator.Send(m.Sender, "Please write me your Quote.", &tb.ReplyMarkup{ReplyKeyboardRemove: true})
	})
	quotator.Handle(tb.OnText, func(m *tb.Message) {
		context := get("quotator|telegram:" + strconv.Itoa(m.Sender.ID) + "|context")
		if context == "" {
			_, _ = quotator.Send(m.Sender, "Unknown Command - use help to get a list of available commands", &tb.ReplyMarkup{ReplyKeyboardRemove: true})
			printInfoGlyph(m)
		} else {
			switch context {
			case "quoteRequired":
				set("quotator|telegram:"+strconv.Itoa(m.Sender.ID)+"|currentQuote", m.Text)
				set("quotator|telegram:"+strconv.Itoa(m.Sender.ID)+"|context", "authorRequired")
				_, _ = quotator.Send(m.Sender, "Thanks, now the author please.")
				printInfoGlyph(m)
			case "authorRequired":
				set("quotator|telegram:"+strconv.Itoa(m.Sender.ID)+"|currentAuthor", m.Text)
				set("quotator|telegram:"+strconv.Itoa(m.Sender.ID)+"|context", "languageRequired")
				_, _ = quotator.Send(m.Sender, "Thanks, now the language please.", &tb.ReplyMarkup{ReplyKeyboard: replyKeysLanguage})
				printInfoGlyph(m)
			case "languageRequired":
				set("quotator|telegram:"+strconv.Itoa(m.Sender.ID)+"|currentLanguage", m.Text)
				set("quotator|telegram:"+strconv.Itoa(m.Sender.ID)+"|context", "universeRequired")
				_, _ = quotator.Send(m.Sender, "And now the universe it comes from please:", &tb.ReplyMarkup{ReplyKeyboardRemove: true})
				printInfoQuotator(m)
			case "universeRequired":
				set("quotator|telegram:"+strconv.Itoa(m.Sender.ID)+"|currentUniverse", m.Text)
				del("quotator|telegram:" + strconv.Itoa(m.Sender.ID) + "|context")
				quotator.Send(m.Sender, addQuote(m))
			}

		}
	})

	// Graceful Shutdown (botquit)
	go func() {
		<-botquit
		quotator.Stop()
		log.Println("[GlyphTelegramBot] " + "Glyph Telegram Bot was stopped")
		os.Exit(3)
	}()

	// Start the bot
	log.Println("[Quotator] " + "Telegram Bot was started.")
	quotator.Start()
}

func getRandomQuote(byAuthor, inLanguage, inUniverse string) string {
	db, err := sql.Open("postgres", databaseURL)
	if err != nil {
		log.Println("[Quotator] Couldn't connect to database: ", err)
		return "Sorry, an internal error occurred!"
	}
	row := db.QueryRow("SELECT quote, author FROM quotes ORDER BY RANDOM() LIMIT 1")
	if err != nil {
		log.Println("[Quotator] Couldn't get random quote from database: ", err)
		return "Sorry, an internal error occurred!"
	}
	var quote, author string
	err = row.Scan(&quote, &author)
	if err != nil {
		if err == sql.ErrNoRows {
			return "No quote found!"
		}
		log.Println("[Quotator] Error getting random quote from database: ", err)
		return "There was an internal error!"
	}
	db.Close()
	return quote + "\n- " + author
}

func addQuote(m *tb.Message) string {
	db, err := sql.Open("postgres", databaseURL)
	if err != nil {
		log.Println("[Quotator] Couldn't connect to database: ", err)
		return "Sorry, an internal error occurred!"
	}
	quote := get("quotator|telegram:" + strconv.Itoa(m.Sender.ID) + "|currentQuote")
	author := get("quotator|telegram:" + strconv.Itoa(m.Sender.ID) + "|currentAuthor")
	language := get("quotator|telegram:" + strconv.Itoa(m.Sender.ID) + "|currentLanguage")
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
	db.Close()
	return "Added quote from " + author + " to database"
}

func printInfoQuotator(m *tb.Message) {
	log.Println("[Quotator] Telegram: " + m.Sender.Username + " - " + m.Sender.FirstName + " " + m.Sender.LastName + " - ID: " + strconv.Itoa(m.Sender.ID) + "Message: " + m.Text)
}
