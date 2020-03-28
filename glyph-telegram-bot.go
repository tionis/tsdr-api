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

//TODO
// Export and Import Function for Database to text file (modeled after dokuwikis users.auth.php)
// Should contain dokuwiki information + telegram links + [TO BE EXTENDED]
// Change database from redis to postgresql for advanced functionality
// Maybe keep redis for Tokens? - Look into advanced functionality

// Global Variables
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
			fmt.Println("[GlyphTelegramBot] " + "Interruption Signal received, shutting down...")
			exit(botquit)
		case syscall.SIGTERM:
			botquit <- true
		}
	}()

	// check for and read config variable, then create bot object
	token := os.Getenv("TELEGRAM_TOKEN")
	glyph, err := tb.NewBot(tb.Settings{
		Token:  token,
		Poller: &tb.LongPoller{Timeout: 10 * time.Second},
	})
	if err != nil {
		log.Fatal(err)
		return
	}

	// Command Handlers
	// handle standard text commands
	glyph.Handle("/hello", func(m *tb.Message) {
		_, _ = glyph.Send(m.Sender, "What do you want?", tb.ModeMarkdown)
		printInfoGlyph(m)
	})
	glyph.Handle("/start", func(m *tb.Message) {
		_, _ = glyph.Send(m.Sender, "Hello.", &tb.ReplyMarkup{ReplyKeyboardRemove: true})
		printInfoGlyph(m)
	})
	glyph.Handle("/help", func(m *tb.Message) {
		sendstring := ""
		if isTasadarTGAdmin(m.Sender.ID) {
			sendstring = `==Following Commands are Available:==
=== UniPassauBot-Commands:
  - /foodtoday - Food for todayco
  - /foodtomorrow - Food for tomorrow
  - /foodweek - Food for week
=== Redis-Commands:
  - /redisGet x - Get Key x from Redis
  - /redisSet x y - Set Key x to y from Redis
  - /redisPing - Ping redis Server
  - /redisBcryptSet x y - Set passw y for user x
  - /redisBcryptGet x y - Check if pass y is valid for user x
=== TOTP-Commands:
  - /addTOTP x y - Add key y for account x
  - /gen x - Get TOTP-Code for account x 
=== MC-Commands:
  - /mc x - Forward command x to MC-Server 
  - /mcStop n - Shutdown server in n minute
  - /mcCancel - Cancel Server shutdown`
		} else {
			sendstring = "There is no help!"
		}
		_, _ = glyph.Send(m.Sender, sendstring)
		printInfoGlyph(m)
	})
	glyph.Handle("/food", func(m *tb.Message) {
		if !m.Private() {
			_, _ = glyph.Send(m.Chat, foodtoday())
			fmt.Println("[GlyphTelegramBot] " + "Group Message:")
		} else {
			_, _ = glyph.Send(m.Sender, foodtoday(), tb.ModeMarkdown)
		}
		printInfoGlyph(m)
	})
	glyph.Handle("food", func(m *tb.Message) {
		if !m.Private() {
			_, _ = glyph.Send(m.Chat, foodtoday())
			fmt.Println("[GlyphTelegramBot] " + "Group Message:")
		} else {
			_, _ = glyph.Send(m.Sender, foodtoday(), tb.ModeMarkdown)
		}
		printInfoGlyph(m)
	})
	glyph.Handle("/foodtomorrow", func(m *tb.Message) {
		if !m.Private() {
			_, _ = glyph.Send(m.Chat, foodtomorrow(), tb.ModeMarkdown)
			fmt.Println("[GlyphTelegramBot] " + "Group Message:")
		} else {
			_, _ = glyph.Send(m.Sender, foodtomorrow(), tb.ModeMarkdown)
		}
		printInfoGlyph(m)
	})
	glyph.Handle("food tomorrow", func(m *tb.Message) {
		if !m.Private() {
			_, _ = glyph.Send(m.Chat, foodtomorrow(), tb.ModeMarkdown)
			fmt.Println("[GlyphTelegramBot] " + "Group Message:")
		} else {
			_, _ = glyph.Send(m.Sender, foodtomorrow(), tb.ModeMarkdown)
		}
		printInfoGlyph(m)
	})
	glyph.Handle("/foodweek", func(m *tb.Message) {
		if !m.Private() {
			_, _ = glyph.Send(m.Chat, foodweek())
			fmt.Println("[GlyphTelegramBot] " + "Group Message:")
		} else {
			_, _ = glyph.Send(m.Sender, foodweek(), tb.ModeMarkdown)
		}
		printInfoGlyph(m)
	})
	glyph.Handle("food week", func(m *tb.Message) {
		if !m.Private() {
			_, _ = glyph.Send(m.Chat, foodweek())
			fmt.Println("[GlyphTelegramBot] " + "Group Message:")
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
			val, err := getResult(s1)
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
	glyph.Handle("/redisPing", func(m *tb.Message) {
		if isTasadarTGAdmin(m.Sender.ID) {
			pong, err := redclient.Ping().Result()
			if err != nil {
				_, _ = glyph.Send(m.Sender, "An Error occurred!")
			} else {
				_, _ = glyph.Send(m.Sender, "Everything normal: "+pong)
			}
		} else {
			_, _ = glyph.Send(m.Sender, "You are not authorized to execute this command!")
		}
		printInfo(m)
	})
	glyph.Handle("/mc", func(m *tb.Message) {
		if isTasadarTGAdmin(m.Sender.ID) {
			s1 := strings.TrimPrefix(m.Text, "/mc ")
			client, err := newClient(rconAddress, 25575, rconPassword)
			if err != nil {
				log.Println("[GlyphTelegramBot] Error occured while building client for connection: ", err)
				_, _ = glyph.Send(m.Sender, "Error occurred while trying to build a connection")
			} else {
				response, err := client.sendCommand(s1)
				if err != nil {
					log.Println("[GlyphTelegramBot] Error occured while making connection: ", err)
					_, _ = glyph.Send(m.Sender, "Error occurred while trying to connect")
				} else {
					if response != "" {
						_, _ = glyph.Send(m.Sender, response)
					} else {
						_, _ = glyph.Send(m.Sender, "Empty Response received")
					}
				}
			}
		} else {
			_, _ = glyph.Send(m.Sender, "You are not authorized to execute this command!")
		}
	})
	glyph.Handle("/mcStop", func(m *tb.Message) {
		if isTasadarTGAdmin(m.Sender.ID) {
			s1 := strings.TrimPrefix(m.Text, "/mcStop ")
			if s1 == "" {
				_, _ = glyph.Send(m.Sender, "Please specify a minute count!")
				return
			}
			minutes, err := strconv.Atoi(s1)
			if err != nil {
				_, _ = glyph.Send(m.Sender, "Error converting minutes, check your input")
			}
			mcShutdownTelegram(glyph, m, minutes)
		} else {
			_, _ = glyph.Send(m.Sender, "You are not authorized to execute this command!")
		}
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
	/*glyph.Handle("/psqlPing", func(m *tb.Message) {
		db, err := sql.Open("postgres", psqlInfo)
		if err != nil {
			log.Println("[PostgreSQL] Server Connection failed: ", err)
			_, _ = glyph.Send(m.Sender, "An error occurred!\nCheck the logs!")
			return
		}
		err = db.Ping()
		if err != nil {
			log.Println("[PostgreSQL] Server Ping failed: ", err)
			_, _ = glyph.Send(m.Sender, "An error occurred!\nCheck the logs!")
			err = db.Close()
			if err != nil {
				log.Println("[PostgreSQL] Error closing Postgres Session")
			}
			return
		}
		_, _ = glyph.Send(m.Sender, "Ping successfull!")
		err = db.Close()
		if err != nil {
			log.Println("[PostgreSQL] Error closing Postgres Session")
		}
	})*/
	glyph.Handle(tb.OnAddedToGroup, func(m *tb.Message) {
		fmt.Println("[GlyphTelegramBot] " + "Group Message:")
		printInfoGlyph(m)
	})
	glyph.Handle(tb.OnText, func(m *tb.Message) {
		if !m.Private() {
			fmt.Println("[GlyphTelegramBot] " + "Message from Group:")
			printInfoGlyph(m)
		} else {
			_, _ = glyph.Send(m.Sender, "Unknown Command - use help to get a list of available commands")
			printInfoGlyph(m)
		}
	})

	// Graceful Shutdown (botquit)
	go func() {
		<-botquit
		glyph.Stop()
		fmt.Println("[GlyphTelegramBot] " + "Bot was stopped")
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
	fmt.Println("[GlyphTelegramBot] " + "Starting Glyph-Telegram-Bot...")
	glyph.Start()
}

func printInfoGlyph(m *tb.Message) {
	loc, _ := time.LoadLocation("Europe/Berlin")
	fmt.Println("[GlyphTelegramBot] " + "[" + time.Now().In(loc).Format("02 Jan 06 15:04") + "]")
	fmt.Println("[GlyphTelegramBot] " + m.Sender.Username + " - " + m.Sender.FirstName + " " + m.Sender.LastName + " - ID: " + strconv.Itoa(m.Sender.ID))
	fmt.Println("[GlyphTelegramBot] " + "Message: " + m.Text + "\n")
}

func isTasadarTGAdmin(ID int) bool {
	if ID == 248533143 {
		return true
	}
	username := get("tg|" + strconv.Itoa(ID) + "|username")
	groups := get("auth|" + username + "|groups")
	// Should transform into array and then check through it
	if strings.Contains(groups, "admin,") || strings.Contains(groups, ",admin") /*|| strings.Contains(groups, "admin")*/ {
		return true
	}
	return false
}

func mcShutdownTelegram(glyph *tb.Bot, m *tb.Message, minutes int) {
	minutesString := strconv.Itoa(minutes)
	client, err := newClient(rconAddress, 25575, rconPassword)
	if err != nil {
		_, _ = glyph.Send(m.Sender, "Error creating RCON Client Object - Check the logs!")
		return
	}
	if !mcRunning {
		_, _ = glyph.Send(m.Sender, "The Server is currently not running!")
		return
	}
	msgDiscordMC <- "Server shutdown commencing in " + minutesString + " Minutes!\nYou can cancel it with /mc cancel"
	mcStopping = true
	_, err = client.sendCommand("tellraw @a [{\"text\":\"Server shutdown commencing in \",\"bold\":false,\"italic\":true,\"underlined\":false,\"striketrough\":false,\"obfuscated\":false,\"color\":\"gray\"},{\"text\":\"" + minutesString + " Minutes!\",\"bold\":false,\"italic\":true,\"underlined\":false,\"striketrough\":false,\"obfuscated\":false,\"color\":\"dark_aqua\"}]")
	if err != nil {
		log.Println("[GlyphDiscordBot] RCON server command connection failed: ", err)
	}
	_, err = client.sendCommand("tellraw @a [{\"text\":\"Type /mc cancel in the Discord Chat to cancel the shutdown! \",\"bold\":false,\"italic\":true,\"underlined\":false,\"striketrough\":false,\"obfuscated\":false,\"color\":\"gray\"}]")
	if err != nil {
		log.Println("[GlyphDiscordBot] RCON server command connection failed: ", err)
	}
	_, _ = glyph.Send(m.Sender, "If you don't say /mcCancel in the next "+minutesString+" Minutes I will shut down the server!")
	time.Sleep(time.Duration(minutes) * time.Minute)
	if mcStopping {
		err = client.reconnect()
		if err != nil {
			log.Println("[GlyphDiscordBot] RCON server reconnect failed: ", err)
		}
		_, _ = glyph.Send(m.Sender, "Shutting down Server...")
		msgDiscordMC <- "Shutting down Server..."
		if err != nil {
			log.Println("[GlyphDiscordBot] RCON server connection failed")
		}
		_, err = client.sendCommand("title @a title {\"text\":\"Warning!\",\"bold\":false,\"italic\":false,\"underlined\":false,\"striketrough\":false,\"obfuscated\":false,\"color\":\"red\"}")
		if err != nil {
			log.Println("[GlyphDiscordBot] RCON server command connection failed: ", err)
		}
		_, err = client.sendCommand("tellraw @a [{\"text\":\"Server shutdown commencing in \",\"bold\":false,\"italic\":true,\"underlined\":false,\"striketrough\":false,\"obfuscated\":false,\"color\":\"gray\"},{\"text\":\"10 Seconds!\",\"bold\":false,\"italic\":true,\"underlined\":false,\"striketrough\":false,\"obfuscated\":false,\"color\":\"dark_aqua\"}]")
		if err != nil {
			log.Println("[GlyphDiscordBot] RCON server command connection failed: ", err)
		}
		time.Sleep(3 * time.Second)
		for i := 10; i >= 0; i-- {
			time.Sleep(1 * time.Second)
			_, err = client.sendCommand("title @a title {\"text\":\"" + strconv.Itoa(i) + "\",\"bold\":false,\"italic\":false,\"underlined\":false,\"striketrough\":false,\"obfuscated\":false,\"color\":\"red\"}")
			if err != nil {
				log.Println("[GlyphDiscordBot] RCON server command connection failed:", err)
			}
		}
		_, err = client.sendCommand("stop")
		if err != nil {
			log.Println("[GlyphDiscordBot] RCON server command connection failed - trying again: ", err)
			_ = client.reconnect()
			_, err = client.sendCommand("stop")
			if err != nil {
				log.Println("[GlyphDiscordBot] RCON server reconnect failed finally: ", err)
			}
		}
	}
}
