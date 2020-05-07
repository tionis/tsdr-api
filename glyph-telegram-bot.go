package main

import (
	"log"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	_ "github.com/heroku/x/hmetrics/onload"
	tb "gopkg.in/tucnak/telebot.v2"
)

const glyphTelegramContextDelay = time.Hour * 24

var msgGlyph chan string
var glyphToken string

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
		sendstring := ""
		if isTasadarTGAdmin(m.Sender.ID) {
			sendstring = `**Following Commands are Available:**
**UniPassauBot-Commands:**
  - /foodtoday - Food for todayco
  - /foodtomorrow - Food for tomorrow
  - /foodweek - Food for week
**Redis-Commands:**
  - /redisGet x - Get Key x from Redis
  - /redisSet x y - Set Key x to y from Redis
  - /redisPing - Ping redis Server
  - /redisBcryptSet x y - Set passw y for user x
  - /redisBcryptGet x y - Check if pass y is valid for user x
**TOTP-Commands:**
  - /addTOTP x y - Add key y for account x
  - /gen x - Get TOTP-Code for account x 
**MC-Commands:**
  - /mc x - Forward command x to MC-Server 
  - /mcStop n - Shutdown server in n minute
  - /mcCancel - Cancel Server shutdown`
		} else {
			sendstring = "There is no help!"
		}
		_, _ = glyph.Send(m.Sender, sendstring, tb.ModeMarkdown)
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
	glyph.Handle("/redis", func(m *tb.Message) {
		if isTasadarTGAdmin(m.Sender.ID) {
			_, _ = glyph.Send(m.Sender, "Available Commands:\n/redisSet - Set Redis Record like this key value\n/redisGet - Get value for key\n/redisPing - Ping/Pong\n/redisBcryptSet - Same as set but with bcrypt\n/redisBcryptGet - Same as Get but with Verify")
		} else {
			_, _ = glyph.Send(m.Sender, "You are not authorized to execute this command!")
		}
		printInfo(m)
	})
	glyph.Handle("/redisSet", func(m *tb.Message) {
		if isTasadarTGAdmin(m.Sender.ID) {
			if !strings.Contains(m.Text, " ") {
				_, _ = glyph.Send(m.Sender, "Error in Syntax!")
			} else {
				s1 := strings.TrimPrefix(m.Text, "/redisSet ")
				s := strings.Split(s1, " ")
				val := strings.TrimPrefix(m.Text, "/redisSet "+s[0]+" ")
				err := set(s[0], val)
				if err != nil {
					log.Println("[GlyphTelegramBot] Error while executing redis command: ", err)
					_, _ = glyph.Send(m.Sender, "There was an error! Check the logs!")
				} else {
					_, _ = glyph.Send(m.Sender, s[0]+" was set to:\n"+val)
				}
			}
		} else {
			_, _ = glyph.Send(m.Sender, "You are not authorized to execute this command!")
		}
		printInfo(m)
	})
	glyph.Handle("/redisGet", func(m *tb.Message) {
		if isTasadarTGAdmin(m.Sender.ID) {
			s1 := strings.TrimPrefix(m.Text, "/redisGet ")
			val, err := getError(s1)
			if err != nil {
				log.Println("[GlyphTelegramBot] Error while executing redis command: ", err)
				_, _ = glyph.Send(m.Sender, "Error! Maybe the value does not exist?")
			} else {
				_, _ = glyph.Send(m.Sender, "Value "+s1+" is set to:\n\n"+val)
			}
		} else {
			_, _ = glyph.Send(m.Sender, "You are not authorized to execute this command!")
		}
		printInfo(m)
	})
	glyph.Handle("/mcCancel", func(m *tb.Message) {
		if isTasadarTGAdmin(m.Sender.ID) {
			if mcStopping {
				mcStopping = false
				_, _ = glyph.Send(m.Sender, "Shutdown cancelled!")
			} else {
				_, _ = glyph.Send(m.Sender, "No Shutdown scheduled!")
			}
		} else {
			_, _ = glyph.Send(m.Sender, "You are not authorized to execute this command!")
		}
	})
	glyph.Handle("/addReminder", func(m *tb.Message) {
		setWithTimer("glyph|telegram:"+strconv.Itoa(m.Sender.ID)+"|context", "TimeRequired", glyphTelegramContextDelay)

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
				del("glyph|telegram:" + strconv.Itoa(m.Sender.ID) + "|context")
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
