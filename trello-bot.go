package main

import (
	"log"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	trello "github.com/VojtechVitek/go-trello"
	"github.com/robfig/cron/v3"
	tb "gopkg.in/tucnak/telebot.v2"
)

type telegramMessage struct {
	Text        string `json:"text"`
	RecipientID int    `json:"recipientid"`
}

var appKey string
var msgTrello chan telegramMessage

func loadTrelloBotJobs(c *cron.Cron) {
	c.AddFunc("0 22 * * *", func() { trelloReminderHandler(248533143) })
}

func trelloTelegramBot() {
	appKey = os.Getenv("TRELLO_APP_KEY")
	token := os.Getenv("TELEGRAM_TRELLO_TOKEN")
	if token == "" {
		log.Fatal("[TrelloBot] No token given!")
	}
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
	trelloBot, err := tb.NewBot(tb.Settings{
		Token:  token,
		Poller: &tb.LongPoller{Timeout: 10 * time.Second},
	})
	if err != nil {
		log.Fatal(err)
		return
	}

	// go routine message backend
	msgTrello = make(chan telegramMessage)
	go func() {
		for {
			messageToSend := <-msgTrello
			recID := int64(messageToSend.RecipientID)
			rec := tb.Chat{ID: recID}
			trelloBot.Send(&rec, messageToSend.Text)
		}
	}()

	// Command Handlers
	// handle standard text commands
	trelloBot.Handle("/start", func(m *tb.Message) {
		del("trellobot|telegram:" + strconv.Itoa(m.Sender.ID) + "|context")
		_, _ = trelloBot.Send(m.Chat, "Please set the [Token](https://trello.com/1/authorize?expiration=never&scope=read,write,account&response_type=token&name=Tasadar%20Trello%20Bot&key="+appKey+") and List now!", tb.ModeMarkdown)
		printInfoTrelloBot(m)
	})
	trelloBot.Handle("/settoken", func(m *tb.Message) {
		trelloToken := strings.TrimPrefix(m.Text, "/settoken ")
		_, err := trello.NewAuthClient(appKey, &trelloToken)
		if err != nil {
			_, _ = trelloBot.Send(m.Chat, "Could not validate Token!")
		} else {
			set("trellobot|telegram:"+strconv.Itoa(m.Sender.ID)+"|token", trelloToken)
			_, _ = trelloBot.Send(m.Chat, "Token set!")
		}
		printInfoTrelloBot(m)
	})
	trelloBot.Handle("/setlist", func(m *tb.Message) {
		listName := strings.TrimPrefix(m.Text, "/setlist ")
		/*trelloToken := get("trellobot|telegram:" + strconv.Itoa(m.Sender.ID) + "|token")
		if token != "" {
			trelloclient, err := trello.NewAuthClient(appKey, &trelloToken)
			if err != nil {
				log.Println("[Trellobot] Error creating New Trello auth client: ", err)
			}

			boards, err := trelloclient.Boards()
			if err != nil {
				log.Println("[Trellobot] Error getting trello boards: ", err)
			}
			for _, board := range boards {
				lists, err := board.Lists()
				if err != nil {
					log.Println("[Trellobot] Error getting trello lists: ", err)
				}
				for _, list := range lists {
					if list.Name == listName {
						listName = list.Id
						break
					}
				}
			}
		}*/
		set("trellobot|telegram:"+strconv.Itoa(m.Sender.ID)+"|list", listName)
		_, _ = trelloBot.Send(m.Chat, "List-ID set!")
		printInfoTrelloBot(m)
	})
	/*trelloBot.Handle(tb.OnDocument, func(m *tb.Message) {
		if m.Private() {
			set("trellobot|telegram:"+strconv.Itoa(m.Sender.ID)+"|context", "photoTitleRequired")
			set("trellobot|telegram:"+strconv.Itoa(m.Sender.ID)+"|url", m.Document.FileURL)
			trelloBot.Send(m.Chat, "Please write what note to add to that")
		}
	})
	trelloBot.Handle(tb.OnPhoto, func(m *tb.Message) {
		if m.Private() {
			set("trellobot|telegram:"+strconv.Itoa(m.Sender.ID)+"|context", "photoTitleRequired")
			set("trellobot|telegram:"+strconv.Itoa(m.Sender.ID)+"|url", m.Photo.FileURL)
			trelloBot.Send(m.Chat, "Please write what note to add to that")
		}
	})*/
	trelloBot.Handle(tb.OnText, func(m *tb.Message) {
		if m.Private() {
			//context := get("trellobot|telegram:" + strconv.Itoa(m.Sender.ID) + "|context")
			trelloToken := get("trellobot|telegram:" + strconv.Itoa(m.Sender.ID) + "|token")
			listID := get("trellobot|telegram:" + strconv.Itoa(m.Sender.ID) + "|list")
			if listID == "" {
				trelloBot.Send(m.Chat, "Please give me the list to send to first!")
			}
			if trelloToken == "" {
				trelloBot.Send(m.Chat, "Please give me a token to use!")
			}
			trelloBot.Send(m.Chat, addTrelloCard(trelloToken, listID, m.Text))
		}
		printInfoTrelloBot(m)
	})

	// Graceful Shutdown (botquit)
	go func() {
		<-botquit
		trelloBot.Stop()
		log.Println("[TrelloBot] " + "Glyph Telegram Bot was stopped")
		os.Exit(3)
	}()

	// Start the bot
	log.Println("[TrelloBot] " + "Telegram Bot was started.")
	trelloBot.Start()
}

func trelloReminderHandler(telegramID int) {
	ListID := get("trellobot|telegram:" + strconv.Itoa(telegramID) + "|list")
	token := get("trellobot|telegram:" + strconv.Itoa(telegramID) + "|token")
	trelloclient, err := trello.NewAuthClient(appKey, &token)
	if err != nil {
		log.Println("[Trellobot] Error creating New Trello auth client: ", err)
		return
	}
	list, err := trelloclient.List(ListID)
	if err != nil {
		log.Println("[Trellobot] Error getting trello list: ", err)
		return
	}
	cards, err := list.Cards()
	if err != nil {
		log.Println("[Trellobot] Error getting trello list: ", err)
		return
	}
	if len(cards) > 0 {
		var message telegramMessage
		message.Text = "You have Cards in your Inbox!"
		message.RecipientID = telegramID
		msgTrello <- message
	}
}

func addTrelloCard(token, ListID, name string) string {
	trelloclient, err := trello.NewAuthClient(appKey, &token)
	if err != nil {
		log.Println("[Trellobot] Error creating New Trello auth client: ", err)
		return "There was an internal Error!"
	}
	list, err := trelloclient.List(ListID)
	if err != nil {
		log.Println("[Trellobot] Error getting trello list: ", err)
		return "There was an internal Error!"
	}
	var cardToAdd trello.Card
	cardToAdd.Name = name
	_, err = list.AddCard(cardToAdd)
	if err != nil {
		log.Println("[Trellobot] Error adding card to trello list: ", err)
		return "There was an internal Error!"
	}
	return "Added Card to Trello List " + list.Name // + "on board " + list.IdBoard
}

func printInfoTrelloBot(m *tb.Message) {
	log.Println("[TelegramTrelloBot] Telegram: " + m.Sender.Username + " - " + m.Sender.FirstName + " " + m.Sender.LastName + " - ID: " + strconv.Itoa(m.Sender.ID) + " Message: " + m.Text)
}
