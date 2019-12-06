package main

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/smtp"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/heroku/x/hmetrics/onload"
)

type alphaMsgStruct struct {
	Message string `form:"message" json:"message" binding:"required"`
	Token   string `form:"token" json:"token" binding:"required"`
}

type mcWhitelistStruct struct {
	User     string `form:"user" json:"user" binding:"required"`
	Password string `form:"password" json:"password" binding:"required"`
	MCUser   string `form:"mcuser" json:"mcuser" binding:"required"`
}

type tokenStruct struct {
	Token string `json:"token"`
}

func routes(router *gin.Engine) {
	// Default Stuff
	router.GET("/favicon.svg", favicon)
	router.GET("/", index)
	router.NoRoute(notFound)
	router.GET("/echo", httpecho)

	// Handle Status Watch
	router.GET("/onlinecheck", func(c *gin.Context) {
		c.String(418, "I'm online")
	})

	// WhatsApp Bot
	router.POST("/twilio/uni-passau-bot/whatsapp", whatsapp)

	// CURL API
	router.GET("/mensa/today", retFoodToday)
	router.GET("/mensa/tomorrow", retFoodTomorow)
	router.GET("/mensa/week", retFoodWeek)

	// Auth API
	router.GET("/auth/basic", authGin)
	router.POST("/auth/basic", authGin)
	router.PUT("/auth/basic", authGin)
	router.GET("/auth/group/:group", authGinGroup)

	// Minecraft API
	router.GET("/mc/stopped/:token", func(c *gin.Context) {
		getAuthorization, err := redclient.Get("mc|token|" + c.Param("token")).Result()
		if err != nil {
			c.String(500, "uncategorized fuckery")
		}
		if getAuthorization == "true" {
			mcRunning = false
			c.String(200, "OK")
		}
	})

	// Google Assitant API - WIP
	router.POST("/dialogflow/alpha", retFoodToday)

	// Google Assitant IFTTT API - tokenization
	router.POST("/iot/assistant/order/:number", assistantOrderHandler)

	// Receive Message from contact form
	router.POST("/contact/tasadar", contactTasadar)
	router.GET("/contact/tasadar", contactTasadar)

	// MC API
	router.GET("/mc/whitelist", mcWhitelist)
	router.POST("/mc/whitelist", mcWhitelist)

	// IoT Handling
	router.GET("/iot/:home/:service/:command", func(c *gin.Context) {
		iotWebhookHandler(c.Param("home"), c.Param("service"), c.Param("command"), c)
	})
	router.POST("/phonetrack/geofence/:device/:location/:movement/:coordinates", func(c *gin.Context) {
		iotGeofenceHandler(c.Param("device"), c.Param("location"), c.Param("coordinates"), c)
	})
	router.GET("/phonetrack/geofence/:device/:location/:movement/:coordinates", func(c *gin.Context) {
		iotGeofenceHandler(c.Param("device"), c.Param("location"), c.Param("coordinates"), c)
	})

	// Send Alpha Message to configured Admin
	// TODÖ: Change this to full blown REST API (JSON TOKENS WITH SSO SOLUTION?)
	/*router.GET("/tg/:message", func(c *gin.Context) {
		message := c.Param("message")
		msgAlpha <- message
		c.String(200, message)
	})*/
	router.POST("/alpha/msg", func(c *gin.Context) {
		var json alphaMsgStruct
		if c.BindJSON(&json) == nil {
			if authenticateAlphaToken(json.Token) {
				msgAlpha <- json.Message
				c.String(200, "OK")
			} else {
				c.String(401, "Unauthorized!")
			}
		} else {
			c.String(400, "Error parsing your packet")
		}
	})
	router.POST("/alpha/msg-discord", func(c *gin.Context) {
		var json alphaMsgStruct
		if c.BindJSON(&json) == nil {
			if authenticateAlphaToken(json.Token) {
				msgDiscordMC <- json.Message
				c.String(200, "OK")
			} else {
				c.String(401, "Unauthorized!")
			}
		} else {
			c.String(400, "Error parsing your packet")
		}
	})
}

type contactForm struct {
	Name    string `form:"name" binding:"required"`
	Mail    string `form:"mail" binding:"required"`
	Message string `form:"message" binding:"required"`
}

// Google Assistant IFTTT Bindings
func authenticateIFTTTToken(token string) bool {
	val, err := redclient.Get("token|" + token + "|ifttt").Result()
	if err != nil {
		return false
	}
	return val == "true"
}

func assistantOrderHandler(c *gin.Context) {
	var tokenJSON tokenStruct
	if c.BindJSON(&tokenJSON) == nil {
		if authenticateIFTTTToken(tokenJSON.Token) {
			err := assistantOrder(c.Param("number"))
			if err != nil {
				c.String(500, "Uncategorized Fuckery")
				log.Println("[TasadarAPI] Error executing order"+c.Param("number")+" : ", err)
			} else {
				c.String(200, "Order executed")
			}
		} else {
			c.String(401, "Unauthorized!")
		}
	} else {
		c.String(400, "Error parsing your packet")
	}
}

func assistantOrder(orderNumber string) error {
	num, err := strconv.Atoi(orderNumber)
	if err != nil {
		return err
	}
	switch num {
	case 31:
		_, err := http.Get("https://maker.ifttt.com/trigger/node_on/with/key/cxGr-6apUjU9_cwUQMCGQ5")
		if err != nil {
			return err
		}
	default:
		return errors.New("Tasadar-Assistant: Unknown command")
	}
	return nil
}

// mc
func mcWhitelist(c *gin.Context) {
	var mcData mcWhitelistStruct
	err := c.Bind(&mcData) // This will infer what binder to use depending on the content-type header.
	if err != nil {
		log.Println("[TasadarAPI] Error in contact form handling at c.Bind(&mcData): ", err)
		c.String(401, "Error in your request")
		return
	}
	user := c.PostForm("user")
	password := c.PostForm("password")
	mcuser := c.PostForm("mcuser")
	if authUser(user, password) {
		oldName, errNoOldName := redclient.Get("auth|" + user + "|mc").Result()
		if oldName == mcuser {
			c.Redirect(302, "https://mc.tasadar.net/nochange")
			return
		}
		// Blacklist old username and whitelist new username
		client, err := newClient(os.Getenv("RCON_ADDRESS"), 25575, os.Getenv("RCON_PASS"))
		if err != nil {
			log.Println("[TasadarAPI] Error occured while building client for connection: ", err)
			if mcStart() {
				c.Redirect(302, "https://mc.tasadar.net/offline")
				return
			}
			c.Redirect(302, "https://mc.tasadar.net/error")
			return
		}
		if errNoOldName == nil {
			response, err := client.sendCommand("whitelist remove " + oldName)
			if err != nil {
				log.Println("[TasadarAPI] Error occured while making connection: ", err)
				c.Redirect(302, "https://mc.tasadar.net/error")
				return
			}
			if !strings.Contains(response, "Removed") {
				log.Println("[TasadarAPI] Error removing MCuser "+oldName+" from whitelist: ", response)
				c.Redirect(302, "https://mc.tasadar.net/error")
				return
			}
		}
		response, err := client.sendCommand("whitelist add " + mcuser)
		if err != nil {
			log.Println("[TasadarAPI] Error occured while making connection: ", err)
			c.Redirect(302, "https://mc.tasadar.net/error")
			return
		}
		if strings.Contains(response, "Added") {
			log.Println("[TasadarAPI] Error adding MCuser "+mcuser+" from whitelist: ", response)
			c.Redirect(302, "https://mc.tasadar.net/error")
			return
		}
		err = redclient.Set("auth|"+user+"|mc", mcuser, 0).Err()
		if err != nil {
			log.Println("[TasadarAPI] Error saving new mc username "+mcuser+" to database for user "+user+" : ", err)
		}
		c.Redirect(302, "https://mc.tasadar.net/success")
		return
	}
	c.Redirect(302, "https://mc.tasadar.net/unauthorized")
	return
}

// contactTasadar
func contactTasadar(c *gin.Context) {
	auth := smtp.PlainAuth("", os.Getenv("SMTP_USERNAME"), os.Getenv("SMTP_PASSWORD"), os.Getenv("SMTP_HOST"))
	var contact contactForm
	err := c.Bind(&contact) // This will infer what binder to use depending on the content-type header.
	if err != nil {
		log.Println("[TasadarAPI] Error in contact form handling at c.Bind(&contact): ", err)
		c.String(401, "Error in your request")
		return
	}
	name := c.PostForm("name")
	email := c.PostForm("email")
	message := c.PostForm("message")
	to := []string{"support@tasadar.net"}
	msg := []byte("To: support@tasadar.net\r\n" +
		"Subject: New Message over Contact Form\r\n" +
		"\r\nNew Message from " + name + "\r\n Email: " + email + "\r\n---\r\n" +
		message + "\r\n")
	err = smtp.SendMail(os.Getenv("SMTP_HOST")+":"+os.Getenv("SMTP_PORT"), auth, "postmaster@mail.tasadar.net", to, msg)
	if err != nil {
		log.Println("[TasadarAPI] Error sending mail: ", err)
		c.String(500, "Error sending mail, please send an email to support@tasadar.net")
	} else {
		c.Redirect(302, "https://contact.tasadar.net/success")
	}

}

// iot Webhook handler
func iotWebhookHandler(home string, service string, command string, c *gin.Context) {
	switch home {
	case "passau":
		switch service {
		case "node":
			switch command {
			case "now_on":
				err := redclient.Set("iot|"+home+"|"+service, "on", 0).Err()
				if err != nil {
					log.Println("[TasadarIoT] Error while executing redis command: ", err)
					c.String(500, "Error executing command")
					return
				}
				c.String(200, "OK")
			case "now_off":
				err := redclient.Set("iot|"+home+"|"+service, "off", 0).Err()
				if err != nil {
					log.Println("[TasadarIoT] Error while executing redis command: ", err)
					c.String(500, "Error executing command")
					return
				}
				c.String(200, "OK")
			case "on":
				_, err := http.Get("https://maker.ifttt.com/trigger/" + service + "_on/with/key/cxGr-6apUjU9_cwUQMCGQ5")
				if err != nil {
					log.Println("[TasadarIoT] Error while sending HTTP request: ", err)
					c.String(500, "Error executing command")
					return
				}
				c.String(200, "OK")
			case "off":
				_, err := http.Get("https://maker.ifttt.com/trigger/" + service + "_off/with/key/cxGr-6apUjU9_cwUQMCGQ5")
				if err != nil {
					log.Println("[TasadarIoT] Error while sending HTTP request: ", err)
					c.String(500, "Error executing command")
					return
				}
				c.String(200, "OK")
			default:
				c.String(400, "Error in request")
				return
			}
		case "led":
			switch command {
			case "now_on":
				err := redclient.Set("iot|"+home+"|"+service, "on", 0).Err()
				if err != nil {
					log.Println("[TasadarIoT] Error while executing redis command: ", err)
					c.String(500, "Error executing command")
					return
				}
				c.String(200, "OK")
			case "now_off":
				err := redclient.Set("iot|"+home+"|"+service, "off", 0).Err()
				if err != nil {
					log.Println("[TasadarIoT] Error while executing redis command: ", err)
					c.String(500, "Error executing command")
					return
				}
				c.String(200, "OK")
			case "on":
				_, err := http.Get("https://maker.ifttt.com/trigger/" + service + "_on/with/key/cxGr-6apUjU9_cwUQMCGQ5")
				if err != nil {
					log.Println("[TasadarIoT] Error while sending HTTP request: ", err)
					c.String(500, "Error executing command")
					return
				}
				c.String(200, "OK")
			case "off":
				_, err := http.Get("https://maker.ifttt.com/trigger/" + service + "_off/with/key/cxGr-6apUjU9_cwUQMCGQ5")
				if err != nil {
					log.Println("[TasadarIoT] Error while sending HTTP request: ", err)
					c.String(500, "Error executing command")
					return
				}
				c.String(200, "OK")
			default:
				c.String(400, "Error in request")
				return
			}
		default:
			c.String(400, "Error in request")
			return
		}
	case "utting":
		c.String(400, "No devices in Utting!")
		return
	default:
		c.String(400, "Unknown location")
		return
	}
}

func iotGeofenceHandler(device string, location string, coordinates string, c *gin.Context) {
	switch device {
	case "note":
		switch location {
		case "home":
			switch "movement" {
			case "enter":
				msgAlpha <- "Entered with " + coordinates
				c.String(200, "OK")
			case "leave":
				msgAlpha <- "Left with " + coordinates
				c.String(200, "OK")
			default:
				c.String(400, "Error in request")
				return
			}
		default:
			c.String(400, "Error in request")
			return
		}
	default:
		c.String(400, "Error in request")
		return
	}
}

// authenticate token
func authenticateAlphaToken(token string) bool {
	val, err := redclient.Get("token|" + token + "|alpha").Result()
	if err != nil {
		return false
	}
	return val == "true"
}

// handle test case
func httpecho(c *gin.Context) {
	// Test Code
	requestDump, err := httputil.DumpRequest(c.Request, true)
	if err != nil {
		fmt.Println(err)
	}
	c.String(200, string(requestDump))
}

// Handle both root thingies
func favicon(c *gin.Context) {
	c.File("static/favicon.svg")
}

func index(c *gin.Context) {
	c.File("static/index.html")
}

func notFound(c *gin.Context) {
	c.File("static/404.html")
}

// handle simple GET requests for food
func retFoodToday(c *gin.Context) {
	c.String(200, foodtoday())
}
func retFoodTomorow(c *gin.Context) {
	c.String(200, foodtomorrow())
}
func retFoodWeek(c *gin.Context) {
	c.String(200, foodweek())
}

// Handle WhatsApp Twilio Webhook
func whatsapp(c *gin.Context) {
	buf := make([]byte, 1024)
	num, _ := c.Request.Body.Read(buf)
	params, err := url.ParseQuery(string(buf[0:num]))
	if err != nil {
		log.Println("[UniPassauBot-WA] ", c.Error(err))
		return
	}
	text := strings.Join(params["Body"], " ")
	from := strings.Join(params["From"], " ")
	messageID := strings.Join(params["MessageSid"], " ")

	loc, _ := time.LoadLocation("Europe/Berlin")
	log.Println("[UniPassauBot-WA] " + "[" + time.Now().In(loc).Format("02 Jan 06 15:04") + "]")
	log.Println("[UniPassauBot-WA] Number: " + from + " - MessageID: " + messageID)
	log.Println("[UniPassauBot-WA] " + "Message: " + text)

	if strings.Contains(text, "tommorow") || strings.Contains(text, "morgen") || strings.Contains(text, "Tommorow") || strings.Contains(text, "Morgen") {
		c.String(200, foodtomorrow())
	} else if strings.Contains(text, "food") || strings.Contains(text, "essen") || strings.Contains(text, "Food") || strings.Contains(text, "Essen") {
		c.String(200, foodtoday())
	} else if strings.Contains(text, "Hallo") || strings.Contains(text, "hallo") {
		c.String(200, "Hallo wie gehts? - Schreibe mir food oder essen morgen um loszulegen!\nMit hilfe kannst du alle Befehle sehen!")
	} else if strings.Contains(text, "danke") || strings.Contains(text, "Danke") {
		c.String(200, "Gern geschehen!")
	} else if strings.Contains(text, "hilfe") || strings.Contains(text, "Hilfe") || strings.Contains(text, "help") || strings.Contains(text, "Help") {
		c.String(200, "Verfügbare Befehle:\nEssen - Essen heute\nEssen morgen - Essen für morgen\nEssen Woche - Essen für die Woche\nAlle Befehle funktionieren auch auf Englisch!")
	} else if strings.Contains(text, "woche") || strings.Contains(text, "Woche") || strings.Contains(text, "week") || strings.Contains(text, "Week") {
		c.String(200, foodweek())
	} else {
		c.String(200, "Befehl nicht erkannt - versuche es mal mit einem Hallo!")
	}
}
