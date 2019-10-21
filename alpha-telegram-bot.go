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
	b, err := tb.NewBot(tb.Settings{
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
	//replyBtn3 := tb.ReplyButton{Text: "Food for the week"}
	replyKeys := [][]tb.ReplyButton{
		{replyBtn, replyBtn2} /*, {replyBtn3}*/}

	// Command Handlers
	// handle special keyboard commands
	b.Handle(&replyBtn, func(m *tb.Message) {
		_, _ = b.Send(m.Sender, foodtoday(), &tb.ReplyMarkup{ReplyKeyboard: replyKeys}, tb.ModeMarkdown)
		printInfoAlpha(m)
	})
	b.Handle(&replyBtn2, func(m *tb.Message) {
		_, _ = b.Send(m.Sender, foodtomorrow(), &tb.ReplyMarkup{ReplyKeyboard: replyKeys}, tb.ModeMarkdown)
		printInfoAlpha(m)
	})
	/*b.Handle(&replyBtn3, func(m *tb.Message) {
		_, _ = b.Send(m.Sender, foodweek(), &tb.ReplyMarkup{ReplyKeyboard: replyKeys}, tb.ModeMarkdown)
		printInfoAlpha(m)
	})*/
	// handle standard text commands
	b.Handle("/hello", func(m *tb.Message) {
		_, _ = b.Send(m.Sender, "Hi! How are you?", tb.ModeMarkdown)
		printInfoAlpha(m)
	})
	b.Handle("/start", func(m *tb.Message) {
		_, _ = b.Send(m.Sender, "Hallo! Ich bin der inoffizielle ChatBot der Uni Passau! Was kann ich dir Gutes tun?\nWenn du Hilfe benötigst benutze einfach /help!\nSolltest du den Mensa- und Stundenplan in einer App wollen, schreibe /app für mehr Informationen", &tb.ReplyMarkup{ReplyKeyboard: replyKeys})
		printInfoAlpha(m)
	})
	b.Handle("/help", func(m *tb.Message) {
		_, _ = b.Send(m.Sender, "Information about the Bot is in the Description\nAvailable Commands are:\n*/help* - Show this help\n*/food* - Get Information for the food TODAY in the Uni Passau\n*/foodtomorrow* - Get Information for the food TOMORROW in the Uni Passau\n*/foodweek* - Get Information for the wood this WEEK in the Uni Passau\n*/contact* - Contact the bot maintainer for requests and bug reports\n*/app* - More Information for an useful Android-App for studip", tb.ModeMarkdown)
		printInfoAlpha(m)
	})
	b.Handle("/food", func(m *tb.Message) {
		if !m.Private() {
			_, _ = b.Send(m.Chat, foodtoday())
			fmt.Println("[AlphaTelegramBot] " + "Group Message:")
		} else {
			_, _ = b.Send(m.Sender, foodtoday(), &tb.ReplyMarkup{ReplyKeyboard: replyKeys}, tb.ModeMarkdown)
		}
		printInfoAlpha(m)
	})
	b.Handle("/foodtomorrow", func(m *tb.Message) {
		if !m.Private() {
			_, _ = b.Send(m.Chat, foodtomorrow())
			fmt.Println("[AlphaTelegramBot] " + "Group Message:")
		} else {
			_, _ = b.Send(m.Sender, foodtomorrow(), &tb.ReplyMarkup{ReplyKeyboard: replyKeys}, tb.ModeMarkdown)
		}
		printInfoAlpha(m)
	})
	b.Handle("/foodweek", func(m *tb.Message) {
		if !m.Private() {
			//_, _ = b.Send(m.Chat, foodweek())
			_, _ = b.Send(m.Chat, "This command is temporarily disabled.")
			fmt.Println("[AlphaTelegramBot] " + "Group Message:")
		} else {
			//_, _ = b.Send(m.Sender, foodweek(), &tb.ReplyMarkup{ReplyKeyboard: replyKeys}, tb.ModeMarkdown)
			_, _ = b.Send(m.Sender, "This command is temporarily disabled.")
		}
		printInfoAlpha(m)
	})
	b.Handle("/contact", func(m *tb.Message) {
		if m.Text == "/contact" {
			_, _ = b.Send(m.Sender, "For requests and bug reports just add your message to the _/contact_ command.", tb.ModeMarkdown)
		} else {
			_, _ = b.Send(m.Sender, "Sending Message to the Bot Maintainer...")
			tionis := tb.Chat{ID: 248533143}
			sendstring := "Message by " + m.Sender.FirstName + " " + m.Sender.LastName + "\nID: " + strconv.Itoa(m.Sender.ID) + " Username: " + m.Sender.Username + "\n- - - - -\n" + strings.TrimPrefix(m.Text, "/contact ")
			_, _ = b.Send(&tionis, sendstring)
		}
		printInfoAlpha(m)
	})
	b.Handle("/send", func(m *tb.Message) {
		if m.Sender.ID == 248533143 {
			s1 := strings.TrimPrefix(m.Text, "/send ")
			s := strings.Split(s1, "$")
			recID, _ := strconv.ParseInt(s[0], 10, 64)
			rec := tb.Chat{ID: recID}
			_, _ = b.Send(&rec, s[1])
		} else {
			_, _ = b.Send(m.Sender, "You are not authorized to execute this command!")
			printInfoAlpha(m)
		}
	})
	b.Handle("Danke", func(m *tb.Message) {
		_, _ = b.Send(m.Sender, "_Gern geschehen!_", tb.ModeMarkdown)
		printInfoAlpha(m)
	})
	b.Handle("Thanks", func(m *tb.Message) {
		_, _ = b.Send(m.Sender, "_It's a pleasure!_", tb.ModeMarkdown)
		printInfoAlpha(m)
	})
	b.Handle("/ping", func(m *tb.Message) {
		_, _ = b.Send(m.Sender, "_pong_", tb.ModeMarkdown)
		printInfoAlpha(m)
	})
	b.Handle(tb.OnAddedToGroup, func(m *tb.Message) {
		fmt.Println("[AlphaTelegramBot] " + "Group Message:")
		printInfoAlpha(m)
	})
	b.Handle(tb.OnText, func(m *tb.Message) {
		if !m.Private() {
			// No Code here for privacy purposes
			// Added test logging - to be removed later
			fmt.Println("[AlphaTelegramBot] " + "Message from Group:")
			printInfoAlpha(m)
		} else {
			_, _ = b.Send(m.Sender, "Unknown Command - use help to get a list of available commands")
			printInfoAlpha(m)
		}
	})

	// Graceful Shutdown (botquit)
	go func() {
		<-botquit
		b.Stop()
		fmt.Println("[AlphaTelegramBot] " + "Bot was stopped")
		os.Exit(3)
	}()

	// print startup message
	fmt.Println("[AlphaTelegramBot] " + "Starting Alpha-Telegram-Bot...")
	b.Start()
}

func printInfoAlpha(m *tb.Message) {
	loc, _ := time.LoadLocation("Europe/Berlin")
	fmt.Println("[AlphaTelegramBot] " + "[" + time.Now().In(loc).Format("02 Jan 06 15:04") + "]")
	fmt.Println("[AlphaTelegramBot] " + m.Sender.Username + " - " + m.Sender.FirstName + " " + m.Sender.LastName + " - ID: " + strconv.Itoa(m.Sender.ID))
	fmt.Println("[AlphaTelegramBot] " + "Message: " + m.Text + "\n")
}
