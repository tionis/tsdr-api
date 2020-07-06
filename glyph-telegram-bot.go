package main

import (
	"log"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	_ "github.com/heroku/x/hmetrics/onload"
	tb "gopkg.in/tucnak/telebot.v2"
)

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
			log.Println("[GlyphTelegramBot] " + "Interruption Signal received, shutting down...")
			exit(botquit)
		case syscall.SIGTERM:
			botquit <- true
		}
	}()

	// check for and read config variable, then create bot object
	glyphToken := os.Getenv("TELEGRAM_TOKEN")
	glyph, err := tb.NewBot(tb.Settings{
		Token:  glyphToken,
		Poller: &tb.LongPoller{Timeout: 10 * time.Second},
	})
	if err != nil {
		log.Fatal(err)
		return
	}

	// Command Handlers
	// handle standard text commands
	glyph.Handle("/hello", func(m *tb.Message) {
		_, _ = glyph.Send(m.Sender, "What do you want?", &tb.ReplyMarkup{ReplyKeyboardRemove: true})
		printInfoGlyph(m)
	})
	glyph.Handle("/start", func(m *tb.Message) {
		_, _ = glyph.Send(m.Sender, "Hello.", &tb.ReplyMarkup{ReplyKeyboardRemove: true})
		printInfoGlyph(m)
	})
	glyph.Handle("/help", func(m *tb.Message) {
		sendString := ""
		if isTasadarTGAdmin(m.Sender.ID) {
			sendString = `**Following Commands are Available:**
**UniPassauBot-Commands:**
  - /foodtoday - Food for today
  - /foodtomorrow - Food for tomorrow
  - /foodweek - Food for week
**Redis-Commands:**
  - /redisGet x - Get Key x from Redis
  - /redisSet x y - Set Key x to y from Redis
  - /redisPing - Ping redis Server
  - /redisBcryptSet x y - Set password y for user x
  - /redisBcryptGet x y - Check if pass y is valid for user x
**TOTP-Commands:**
  - /addTOTP x y - Add key y for account x
  - /gen x - Get TOTP-Code for account x `
		} else {
			sendString = "There is no help!"
		}
		_, _ = glyph.Send(m.Sender, sendString, tb.ModeMarkdown)
		printInfoGlyph(m)
	})
	glyph.Handle("/food", func(m *tb.Message) {
		if !m.Private() {
			_, _ = glyph.Send(m.Chat, foodtoday())
			log.Println("[GlyphTelegramBot] " + "Group Message:")
		} else {
			_, _ = glyph.Send(m.Sender, foodtoday(), tb.ModeMarkdown)
		}
		printInfoGlyph(m)
	})
	glyph.Handle("food", func(m *tb.Message) {
		if !m.Private() {
			_, _ = glyph.Send(m.Chat, foodtoday())
			log.Println("[GlyphTelegramBot] " + "Group Message:")
		} else {
			_, _ = glyph.Send(m.Sender, foodtoday(), tb.ModeMarkdown)
		}
		printInfoGlyph(m)
	})
	glyph.Handle("/foodtoday", func(m *tb.Message) {
		if !m.Private() {
			_, _ = glyph.Send(m.Chat, foodtoday(), tb.ModeMarkdown)
			log.Println("[GlyphTelegramBot] " + "Group Message:")
		} else {
			_, _ = glyph.Send(m.Sender, foodtoday(), tb.ModeMarkdown)
		}
		printInfoGlyph(m)
	})
	glyph.Handle("/foodtomorrow", func(m *tb.Message) {
		if !m.Private() {
			_, _ = glyph.Send(m.Chat, foodtomorrow(), tb.ModeMarkdown)
			log.Println("[GlyphTelegramBot] " + "Group Message:")
		} else {
			_, _ = glyph.Send(m.Sender, foodtomorrow(), tb.ModeMarkdown)
		}
		printInfoGlyph(m)
	})
	glyph.Handle("food tomorrow", func(m *tb.Message) {
		if !m.Private() {
			_, _ = glyph.Send(m.Chat, foodtomorrow(), tb.ModeMarkdown)
			log.Println("[GlyphTelegramBot] " + "Group Message:")
		} else {
			_, _ = glyph.Send(m.Sender, foodtomorrow(), tb.ModeMarkdown)
		}
		printInfoGlyph(m)
	})
	glyph.Handle("/foodweek", func(m *tb.Message) {
		if !m.Private() {
			_, _ = glyph.Send(m.Chat, foodweek())
			log.Println("[GlyphTelegramBot] " + "Group Message:")
		} else {
			_, _ = glyph.Send(m.Sender, foodweek(), tb.ModeMarkdown)
		}
		printInfoGlyph(m)
	})
	glyph.Handle("food week", func(m *tb.Message) {
		if !m.Private() {
			_, _ = glyph.Send(m.Chat, foodweek())
			log.Println("[GlyphTelegramBot] " + "Group Message:")
		} else {
			_, _ = glyph.Send(m.Sender, foodweek(), tb.ModeMarkdown)
		}
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
	glyph.Handle("/addReminder", func(m *tb.Message) {
		_ = setWithTimer("glyph|telegram:"+strconv.Itoa(m.Sender.ID)+"|context", "TimeRequired", glyphTelegramContextDelay)

	})
	glyph.Handle(tb.OnAddedToGroup, func(m *tb.Message) {
		log.Println("[GlyphTelegramBot] " + "Group Message:")
		printInfoGlyph(m)
	})
	glyph.Handle(tb.OnText, func(m *tb.Message) {
		if !m.Private() {
			log.Println("[GlyphTelegramBot] " + "Message from Group:")
			printInfoGlyph(m)
		} else {
			context := get("glyph|telegram:" + strconv.Itoa(m.Sender.ID) + "|context")
			switch context {
			case "TimeRequired":
				_ = del("glyph|telegram:" + strconv.Itoa(m.Sender.ID) + "|context")
				// TODO Parse Message and add result to queue: Add to relevant minute
				// Add reminder to user data
				_, _ = glyph.Send(m.Sender, "Whaaat!")
			default:
				_, _ = glyph.Send(m.Sender, "Unknown Command - use help to get a list of available commands")
			}
			printInfoGlyph(m)
		}
	})

	// Graceful Shutdown (botquit)
	go func() {
		<-botquit
		glyph.Stop()
		log.Println("[GlyphTelegramBot] " + "Glyph Telegram Bot was stopped")
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
	log.Println("[GlyphTelegramBot] " + "Glyph Telegram Bot was started.")
	glyph.Start()
}

func printInfoGlyph(m *tb.Message) {
	log.Println("[GlyphTelegramBot] " + m.Sender.Username + " - " + m.Sender.FirstName + " " + m.Sender.LastName + " - ID: " + strconv.Itoa(m.Sender.ID) + "Message: " + m.Text)
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
