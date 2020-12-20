package main

import (
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	_ "github.com/heroku/x/hmetrics/onload"
	"github.com/keybase/go-logging"
	"github.com/tionis/tsdr-api/data"
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

	// Handle Quotator Commands
	glyph.Handle("/getquote", func(m *tb.Message) {
		data.DelTmp("glyph", "telegram:"+strconv.Itoa(m.Sender.ID)+"|context")
		author, language, universe, err := parseGetQuote(strings.TrimPrefix(m.Text, "/getquote "))
		if err != nil {
			glyphTelegramLog.Error("[glyph] Error parsing getQuote: ", err)
			_, _ = glyph.Send(m.Chat, "There was an error please check your command and try again later.", &tb.ReplyMarkup{ReplyKeyboardRemove: true})
		} else {
			_, _ = glyph.Send(m.Chat, getRandomQuote(author, language, universe), &tb.ReplyMarkup{ReplyKeyboardRemove: true})
		}
	})
	glyph.Handle("/addquote", func(m *tb.Message) {
		data.SetTmp("glyph", "telegram:"+strconv.Itoa(m.Sender.ID)+"|context", "quoteRequired", glyphTelegramContextDelay)
		_, _ = glyph.Send(m.Chat, "Please write me your Quote.", &tb.ReplyMarkup{ReplyKeyboardRemove: true})
	})
	glyph.Handle("/quoteoftheday", func(m *tb.Message) {
		quote := data.GetTmp("glyph", "telegram:"+strconv.Itoa(m.Sender.ID)+"|dayquote")
		if quote != "" {
			_, _ = glyph.Send(m.Chat, quote)
		} else {
			quote = getRandomQuote("", "", "")
			now := time.Now()
			year, month, day := now.Date()
			midnight := time.Date(year, month, day+1, 0, 0, 0, 0, now.Location())
			data.SetTmp("glyph", "telegram:"+strconv.Itoa(m.Sender.ID)+"|dayquote", quote, time.Until(midnight))
			_, _ = glyph.Send(m.Chat, quote)
		}
		printInfoGlyph(m)
	})

	// Handle non command text
	glyph.Handle(tb.OnText, func(m *tb.Message) {
		context := data.GetTmp("glyph", "telegram:"+strconv.Itoa(m.Sender.ID)+"|context")
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
				data.SetTmp("glyph", "telegram:"+strconv.Itoa(m.Sender.ID)+"|currentQuote", m.Text, glyphTelegramContextDelay)
				data.SetTmp("glyph", "telegram:"+strconv.Itoa(m.Sender.ID)+"|context", "authorRequired", glyphTelegramContextDelay)
				_, _ = glyph.Send(m.Sender, "Thanks, now the author please.")
				printInfoGlyph(m)
			case "authorRequired":
				data.SetTmp("glyph", "telegram:"+strconv.Itoa(m.Sender.ID)+"|currentAuthor", m.Text, glyphTelegramContextDelay)
				data.SetTmp("glyph", "telegram:"+strconv.Itoa(m.Sender.ID)+"|context", "languageRequired", glyphTelegramContextDelay)
				_, _ = glyph.Send(m.Sender, "Thanks, now the language please.", &tb.ReplyMarkup{ReplyKeyboard: replyKeysLanguage})
				printInfoGlyph(m)
			case "languageRequired":
				data.SetTmp("glyph", "telegram:"+strconv.Itoa(m.Sender.ID)+"|currentLanguage", m.Text, glyphTelegramContextDelay)
				data.SetTmp("glyph", "telegram:"+strconv.Itoa(m.Sender.ID)+"|context", "universeRequired", glyphTelegramContextDelay)
				_, _ = glyph.Send(m.Sender, "And now the universe it comes from please:", &tb.ReplyMarkup{ReplyKeyboardRemove: true})
				printInfoGlyph(m)
			case "universeRequired":
				data.SetTmp("glyph", "telegram:"+strconv.Itoa(m.Sender.ID)+"|currentUniverse", m.Text, glyphTelegramContextDelay)
				data.DelTmp("glyph", "telegram:"+strconv.Itoa(m.Sender.ID)+"|context")
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
