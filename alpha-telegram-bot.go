package main

import (
	"fmt"
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

// Global Variables
var msgAlpha chan string

// AlphaTelegramBot handles all the legacy Alpha-Telegram-Bot code for telegram
func alphaTelegramBot() {

	botquit := make(chan bool) // channel for quitting of bot

	// catch os signals like sigterm and interrupt
	signalChannel := make(chan os.Signal, 2)
	signal.Notify(signalChannel, os.Interrupt, syscall.SIGTERM)
	go func() {
		sig := <-signalChannel
		switch sig {
		case os.Interrupt:
			fmt.Println("[AlphaTelegramBot] " + "Interruption Signal received, shutting down...")
			exit(botquit)
		case syscall.SIGTERM:
			botquit <- true
		}
	}()

	// check for and read config variable, then create bot object
	token := os.Getenv("AlphaTelegramBot")
	alpha, err := tb.NewBot(tb.Settings{
		Token:  token,
		Poller: &tb.LongPoller{Timeout: 10 * time.Second},
	})
	if err != nil {
		log.Fatal(err)
		return
	}

	// init reply keyboard
	replyBtn := tb.ReplyButton{Text: "Food for today"}
	replyBtn2 := tb.ReplyButton{Text: "Food for tomorrow"}
	replyBtn3 := tb.ReplyButton{Text: "Food for the week"}
	replyKeys := [][]tb.ReplyButton{
		{replyBtn, replyBtn2}, {replyBtn3}}

	// Command Handlers
	// handle special keyboard commands
	alpha.Handle(&replyBtn, func(m *tb.Message) {
		_, _ = alpha.Send(m.Sender, foodtoday(), &tb.ReplyMarkup{ReplyKeyboard: replyKeys}, tb.ModeMarkdown)
		printInfoAlpha(m)
	})
	alpha.Handle(&replyBtn2, func(m *tb.Message) {
		_, _ = alpha.Send(m.Sender, foodtomorrow(), &tb.ReplyMarkup{ReplyKeyboard: replyKeys}, tb.ModeMarkdown)
		printInfoAlpha(m)
	})
	alpha.Handle(&replyBtn3, func(m *tb.Message) {
		_, _ = alpha.Send(m.Sender, foodweek(), &tb.ReplyMarkup{ReplyKeyboard: replyKeys}, tb.ModeMarkdown)
		printInfoAlpha(m)
	})
	// handle standard text commands
	alpha.Handle("/hello", func(m *tb.Message) {
		_, _ = alpha.Send(m.Sender, "What do you want?", tb.ModeMarkdown)
		printInfoAlpha(m)
	})
	alpha.Handle("/start", func(m *tb.Message) {
		_, _ = alpha.Send(m.Sender, "Hello.", &tb.ReplyMarkup{ReplyKeyboard: replyKeys})
		printInfoAlpha(m)
	})
	alpha.Handle("/help", func(m *tb.Message) {
		_, _ = alpha.Send(m.Sender, "There is no help!", tb.ModeMarkdown)
		printInfoAlpha(m)
	})
	alpha.Handle("/food", func(m *tb.Message) {
		if !m.Private() {
			_, _ = alpha.Send(m.Chat, foodtoday())
			fmt.Println("[AlphaTelegramBot] " + "Group Message:")
		} else {
			_, _ = alpha.Send(m.Sender, foodtoday(), &tb.ReplyMarkup{ReplyKeyboard: replyKeys}, tb.ModeMarkdown)
		}
		printInfoAlpha(m)
	})
	alpha.Handle("/foodtomorrow", func(m *tb.Message) {
		if !m.Private() {
			_, _ = alpha.Send(m.Chat, foodtomorrow())
			fmt.Println("[AlphaTelegramBot] " + "Group Message:")
		} else {
			_, _ = alpha.Send(m.Sender, foodtomorrow(), &tb.ReplyMarkup{ReplyKeyboard: replyKeys}, tb.ModeMarkdown)
		}
		printInfoAlpha(m)
	})
	alpha.Handle("/foodweek", func(m *tb.Message) {
		if !m.Private() {
			_, _ = alpha.Send(m.Chat, foodweek())
			fmt.Println("[AlphaTelegramBot] " + "Group Message:")
		} else {
			_, _ = alpha.Send(m.Sender, foodweek(), &tb.ReplyMarkup{ReplyKeyboard: replyKeys}, tb.ModeMarkdown)
		}
		printInfoAlpha(m)
	})
	alpha.Handle("Thanks", func(m *tb.Message) {
		_, _ = alpha.Send(m.Sender, "_It's a pleasure!_", tb.ModeMarkdown)
		printInfoAlpha(m)
	})
	alpha.Handle("/ping", func(m *tb.Message) {
		_, _ = alpha.Send(m.Sender, "_pong_", tb.ModeMarkdown)
		printInfoAlpha(m)
	})
	alpha.Handle("/redis", func(m *tb.Message) {
		if m.Sender.ID == 248533143 {
			alpha.Send(m.Sender, "Available Commands:\n/redisSet - Set Redis Record like this key value\n/redisGet - Get value for key\n/redisPing - Ping/Pong\n/redisBcryptSet - Same as set but with bcrypt\n/redisBcryptGet - Same as Get but with Verify")
		} else {
			_, _ = alpha.Send(m.Sender, "You are not authorized to execute this command!")
		}
		printInfo(m)
	})
	alpha.Handle("/redisSet", func(m *tb.Message) {
		if m.Sender.ID == 248533143 {
			if !strings.Contains(m.Text, " ") {
				alpha.Send(m.Sender, "Error in Syntax!")
			} else {
				s1 := strings.TrimPrefix(m.Text, "/redisSet ")
				s := strings.Split(s1, " ")
				err := redclient.Set(s[0], s[1], 0).Err()
				if err != nil {
					log.Println("[AlphaTelegramBot] Error while executing redis command: ", err)
					_, _ = alpha.Send(m.Sender, "There was an error! Check the logs!")
				} else {
					alpha.Send(m.Sender, s[0]+" was set to "+s[1])
				}
			}
		} else {
			_, _ = alpha.Send(m.Sender, "You are not authorized to execute this command!")
		}
		printInfo(m)
	})
	alpha.Handle("/redisGet", func(m *tb.Message) {
		if m.Sender.ID == 248533143 {
			s1 := strings.TrimPrefix(m.Text, "/redisGet ")
			val, err := redclient.Get(s1).Result()
			if err != nil {
				log.Println("[AlphaTelegramBot] Error while executing redis command: ", err)
				_, _ = alpha.Send(m.Sender, "Error! Maybe the value does not exist?")
			} else {
				_, _ = alpha.Send(m.Sender, "Value "+s1+" is set to "+val)
			}
		} else {
			_, _ = alpha.Send(m.Sender, "You are not authorized to execute this command!")
		}
		printInfo(m)
	})
	alpha.Handle("/redisPing", func(m *tb.Message) {
		if m.Sender.ID == 248533143 {
			pong, err := redclient.Ping().Result()
			if err != nil {
				alpha.Send(m.Sender, "An Error occurred!")
			} else {
				alpha.Send(m.Sender, "Everything normal: "+pong)
			}
		} else {
			_, _ = alpha.Send(m.Sender, "You are not authorized to execute this command!")
		}
		printInfo(m)
	})
	alpha.Handle("/redisBcryptSet", func(m *tb.Message) {
		if m.Sender.ID == 248533143 {
			if !strings.Contains(m.Text, " ") {
				alpha.Send(m.Sender, "Error in Syntax!")
			} else {
				s1 := strings.TrimPrefix(m.Text, "/redisBcryptSet ")
				s := strings.Split(s1, " ")
				hash, err := hashPassword(s[1])
				err = redclient.Set(s[0], hash, 0).Err()
				if err != nil {
					log.Println("[AlphaTelegramBot] Error while executing redis command: ", err)
					_, _ = alpha.Send(m.Sender, "There was an error! Check the logs!")
				} else {
					alpha.Send(m.Sender, s[0]+" now has the hash "+hash)
				}
			}
		} else {
			_, _ = alpha.Send(m.Sender, "You are not authorized to execute this command!")
		}
		printInfo(m)
	})
	alpha.Handle("/redisBcryptGet", func(m *tb.Message) {
		if m.Sender.ID == 248533143 {
			if !strings.Contains(m.Text, " ") {
				alpha.Send(m.Sender, "Error in Syntax!")
			} else {
				s1 := strings.TrimPrefix(m.Text, "/redisBcryptGet ")
				s := strings.Split(s1, " ")
				val, err := redclient.Get("auth|" + s[0] + "|hash").Result()
				if err != nil {
					log.Println("[AlphaTelegramBot] Error while executing redis command: ", err)
					_, _ = alpha.Send(m.Sender, "Error! Maybe the value does not exist?")
				} else {
					if checkPasswordHash(s[1], val) {
						alpha.Send(m.Sender, "Password matches!")
					} else {
						alpha.Send(m.Sender, "Password doesn't match!")
						alpha.Send(m.Sender, "Just to bes sure, I checked:\n"+s[0]+"\n"+s[1])
					}
				}
			}
		} else {
			_, _ = alpha.Send(m.Sender, "You are not authorized to execute this command!")
		}
		printInfoAlpha(m)
	})
	alpha.Handle("/bcryptVerify", func(m *tb.Message) {
		s1 := strings.TrimPrefix(m.Text, "/bcryptVerify ")
		s := strings.Split(s1, " ")
		if checkPasswordHash(s[0], s[1]) {
			alpha.Send(m.Sender, "Hash"+s[1]+" matches the password!")
		} else {
			alpha.Send(m.Sender, "Error: Hash"+s[1]+"doesn't match the password!")
		}
		printInfoAlpha(m)
	})
	alpha.Handle("/updateAuth", func(m *tb.Message) {
		_, _ = alpha.Send(m.Sender, "_Updating Auth Database ...._", tb.ModeMarkdown)
		updateAuth()
		_, _ = alpha.Send(m.Sender, "_Updated the Auth Database_", tb.ModeMarkdown)
		printInfoAlpha(m)
	})
	alpha.Handle(tb.OnAddedToGroup, func(m *tb.Message) {
		fmt.Println("[AlphaTelegramBot] " + "Group Message:")
		printInfoAlpha(m)
	})
	alpha.Handle(tb.OnText, func(m *tb.Message) {
		if !m.Private() {
			fmt.Println("[AlphaTelegramBot] " + "Message from Group:")
			printInfoAlpha(m)
		} else {
			_, _ = alpha.Send(m.Sender, "Unknown Command - use help to get a list of available commands")
			printInfoAlpha(m)
		}
	})

	// Graceful Shutdown (botquit)
	go func() {
		<-botquit
		alpha.Stop()
		fmt.Println("[AlphaTelegramBot] " + "Bot was stopped")
		os.Exit(3)
	}()

	// Channel for sending messages
	go func(alpha *tb.Bot) {
		msgAlpha = make(chan string)
		for {
			toSend := <-msgAlpha
			tionis := tb.Chat{ID: 248533143}
			alpha.Send(&tionis, toSend)
		}
	}(alpha)

	// print startup message
	fmt.Println("[AlphaTelegramBot] " + "Starting Alpha-Telegram-Bot...")
	alpha.Start()
}

func printInfoAlpha(m *tb.Message) {
	loc, _ := time.LoadLocation("Europe/Berlin")
	fmt.Println("[AlphaTelegramBot] " + "[" + time.Now().In(loc).Format("02 Jan 06 15:04") + "]")
	fmt.Println("[AlphaTelegramBot] " + m.Sender.Username + " - " + m.Sender.FirstName + " " + m.Sender.LastName + " - ID: " + strconv.Itoa(m.Sender.ID))
	fmt.Println("[AlphaTelegramBot] " + "Message: " + m.Text + "\n")
}
