package main

import (
	"errors"
	"log"
	"net/http"
	"net/http/httputil"
	"net/smtp"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gbrlsnchs/jwt/v3"
	"github.com/gin-gonic/gin"
	_ "github.com/heroku/x/hmetrics/onload"
)

const smtpHost = "smtp.eu.mailgun.org"
const smtpPort = "25"
const smtpUser = "postmaster@mail.tasadar.net"
const smtpFrom = "do-no-reply@mail.tasadar.net"

var hs = jwt.NewHS256([]byte("v09AoteRzfUEDbxqjDFFyWaSPrNeDqOj"))

type tasadarToken struct {
	jwt.Payload
	Groups string `json:"groups,omitempty"`
}

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

type contactForm struct {
	Name    string `form:"name" binding:"required"`
	Mail    string `form:"mail" binding:"required"`
	Message string `form:"message" binding:"required"`
}

type pwChangeForm struct {
	Username         string `form:"username" binding:"required"`
	OldPassword      string `form:"oldPassword" binding:"required"`
	NewPassword      string `form:"newPassword" binding:"required"`
	NewPasswordAgain string `form:"newPasswordAgain" binding:"required"`
}

type loginForm struct {
	Username string `form:"username" binding:"required"`
	Password string `form:"password" binding:"required"`
}

func apiRoutes(router *gin.Engine) {
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

	// Minecraft API
	router.GET("/mc/stopped/:token", func(c *gin.Context) {
		getAuthorization, err := kvgetResult("mc|token|" + c.Param("token"))
		if err != nil {
			c.File("static/error-pages/500.html")
		}
		if getAuthorization == "true" {
			mcRunning = false
			c.String(200, "OK")
		}
	})

	// Google Assitant IFTTT API - tokenization
	router.POST("/iot/assistant/order/:number", assistantOrderHandler)

	// Receive Message from contact form
	router.POST("/contact/tasadar", contactTasadar)
}

// Google Assistant IFTTT Binding
func authenticateIFTTTToken(token string) bool {
	val, err := kvgetResult("token|" + token + "|ifttt")
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

// contactTasadar
func contactTasadar(c *gin.Context) {
	auth := smtp.PlainAuth("", smtpUser, os.Getenv("SMTP_PASSWORD"), smtpHost)
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
	err = smtp.SendMail(smtpHost+":"+smtpPort, auth, smtpFrom, to, msg)
	if err != nil {
		log.Println("[TasadarAPI] Error sending mail: ", err)
		c.String(500, "Error sending mail, please send an email to support@tasadar.net")
	} else {
		c.Redirect(302, "https://contact.tasadar.net/success")
	}

}

// handle test case
func httpecho(c *gin.Context) {
	// Test Code
	requestDump, err := httputil.DumpRequest(c.Request, true)
	if err != nil {
		log.Println(err)
	}
	c.String(200, string(requestDump))
}

// Handle both root thingies
func favicon(c *gin.Context) {
	c.File("static/icons/favicon.svg")
}

func index(c *gin.Context) {
	c.File("static/index.html")
}

func notFound(c *gin.Context) {
	c.File("static/error-pages/404.html")
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
