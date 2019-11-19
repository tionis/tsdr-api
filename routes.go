package main

import (
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/smtp"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/heroku/x/hmetrics/onload"
)

type alphaMsgStruct struct {
	Message string `form:"message" json:"message" binding:"required"`
	Token   string `form:"token" json:"token" binding:"required"`
}

func routes(router *gin.Engine) {
	// Default Stuff
	router.GET("/favicon.svg", favicon)
	router.GET("/", index)
	router.NoRoute(notFound)
	router.GET("/echo", httpecho)

	// Handle Status Watch
	router.GET("/status", statusHandler)
	go updateStatus()

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

	// Google Assitant API - WIP
	router.POST("/dialogflow/alpha", retFoodToday)

	// Receive Message from contact form
	router.POST("/contact/tasadar", contactTasadar)
	router.GET("/contact/tasadar", contactTasadar)

	// IoT Handling
	router.GET("/iot/:home/:service/:command", func(c *gin.Context) {
		iotWebhookHandler(c.Param("home"), c.Param("service"), c.Param("command"), c)
	})
	router.POST("/phonetrack/geofence/:device/:location/:movement/:coordinates", func(c *gin.Context) {
		iotGeofenceHandler(c.Param("device"), c.Param("location"), c.Param("movement"), c.Param("coordinates"), c)
	})
	router.GET("/phonetrack/geofence/:device/:location/:movement/:coordinates", func(c *gin.Context) {
		iotGeofenceHandler(c.Param("device"), c.Param("location"), c.Param("movement"), c.Param("coordinates"), c)
	})

	// Send Alpha Message to configured Admin
	// TODO: Change this to full blown REST API (JSON TOKENS WITH SSO SOLUTION?)
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
}

type contactForm struct {
	name    string `form:"name" binding:"required"`
	mail    string `form:"mail" binding:"required"`
	message string `form:"message" binding:"required"`
}

// contactTasadar
func contactTasadar(c *gin.Context) {
	auth := smtp.PlainAuth("", os.Getenv("SMTP_USERNAME"), os.Getenv("SMTP_PASSWORD"), os.Getenv("SMTP_HOST"))
	var contact contactForm
	c.Bind(&contact) // This will infer what binder to use depending on the content-type header.
	name := c.PostForm("name")
	email := c.PostForm("email")
	message := c.PostForm("message")
	to := []string{"support@tasadar.net"}
	msg := []byte("To: support@tasadar.net\r\n" +
		"Subject: New Message over Contact Form\r\n" +
		"\r\nNew Message from " + name + "\r\n Email: " + email + "\r\n---\r\n" +
		message + "\r\n")
	err := smtp.SendMail(os.Getenv("SMTP_HOST")+":"+os.Getenv("SMTP_PORT"), auth, "postmaster@mail.tasadar.net", to, msg)
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

func iotGeofenceHandler(device string, location string, movement string, coordinates string, c *gin.Context) {
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
