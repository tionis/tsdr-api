package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"strconv"
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
	replyBtn3 := tb.ReplyButton{Text: "Food for the week"}
	replyKeys := [][]tb.ReplyButton{
		{replyBtn, replyBtn2}, {replyBtn3}}

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
	b.Handle(&replyBtn3, func(m *tb.Message) {
		_, _ = b.Send(m.Sender, foodweek(), &tb.ReplyMarkup{ReplyKeyboard: replyKeys}, tb.ModeMarkdown)
		printInfoAlpha(m)
	})
	// handle standard text commands
	b.Handle("/hello", func(m *tb.Message) {
		_, _ = b.Send(m.Sender, "What do you want?", tb.ModeMarkdown)
		printInfoAlpha(m)
	})
	b.Handle("/start", func(m *tb.Message) {
		_, _ = b.Send(m.Sender, "Hello.", &tb.ReplyMarkup{ReplyKeyboard: replyKeys})
		printInfoAlpha(m)
	})
	b.Handle("/help", func(m *tb.Message) {
		_, _ = b.Send(m.Sender, "There is no help!", tb.ModeMarkdown)
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
			_, _ = b.Send(m.Chat, foodweek())
			fmt.Println("[AlphaTelegramBot] " + "Group Message:")
		} else {
			_, _ = b.Send(m.Sender, foodweek(), &tb.ReplyMarkup{ReplyKeyboard: replyKeys}, tb.ModeMarkdown)
		}
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
