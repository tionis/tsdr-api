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

	"github.com/dgrijalva/jwt-go"
	"github.com/gin-gonic/gin"
	_ "github.com/heroku/x/hmetrics/onload"
)

const smtpHost = "smtp.eu.mailgun.org"
const smtpPort = "25"
const smtpUser = "postmaster@mail.tasadar.net"
const smtpFrom = "do-no-reply@mail.tasadar.net"

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
	router.GET("/auth/basic/group/:group", authGinGroup)
	router.GET("/auth/group/:group", authGinGroup)
	router.GET("/auth/resetpw/:token", resetPWToken)
	router.GET("/auth/reset-password", resetPW)
	router.GET("/auth/change-password", changePW)
	router.POST("/auth/new-password/:token", newPWFormHandlerMail)
	router.GET("/auth/new-password/:token", newPWFormHandlerMail)
	router.POST("/auth/new-password", newPWFormHandler)
	router.GET("/auth/new-password", newPWFormHandler)
	router.POST("/login/execute", tasadarLoginHandler)
	router.GET("/login/verify", tasadarLoginVerify)

	//3rd Party verify links
	router.GET("/auth/verify/mail/:token", emailVerifyHandler)

	// Minecraft API
	router.GET("/mc/stopped/:token", func(c *gin.Context) {
		getAuthorization, err := redclient.Get("mc|token|" + c.Param("token")).Result()
		if err != nil {
			c.File("static/error-pages/500.html")
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

// Auth System
func resetPW(c *gin.Context) {
	// Insert Form for Password Reset here
	// Generate Token and insert into redis
	// Send Email
	c.File("static/auth/reset-password.html")
}

func resetPWToken(c *gin.Context) {
	// Validate Token here
	if false {
		// Read first part
		// Read second part
		// Create custom Form with link
		// Merge them
		// IDEA as follows: Form always stays the same but link that its send to changes

	} else {
		c.File("static/error-pages/401.html")
	}
	// Send new form back with
}

func changePW(c *gin.Context) {
	c.File("static/auth/change-password.html")
}

func newPWFormHandlerMail(c *gin.Context) {
	// validate token
	if false {
		// Set new password
		c.File("static/auth/password-set.html")
	} else {
		c.File("static/error-pages/401.html")
	}
}

func newPWFormHandler(c *gin.Context) {
	var newPWFormData pwChangeForm
	err := c.Bind(&newPWFormData) // This will infer what binder to use depending on the content-type header.
	if err != nil {
		log.Println("[TasadarAPI] Error in contact form handling at c.Bind(&newPWFormData): ", err)
		c.String(401, "Error in your request")
		return
	}
	username := c.PostForm("username")
	oldPassword := c.PostForm("oldPassword")
	newPassword := c.PostForm("newPassword")
	newPasswordAgain := c.PostForm("newPasswordAgain")
	if newPassword != newPasswordAgain {
		c.File("static/auth/change-password-wrong.html")
	}
	if authUser(username, oldPassword) {
		err = authSetPassword(username, newPassword)
		if err != nil {
			log.Println("[TasadarAPI] Error while trying to set Password for User "+username+" with error code:", err)
			c.File("static/error-pages/500.html")
		}
		c.File("static/auth/new-password-set.html")
	}
}

func tasadarLoginHandler(c *gin.Context) {
	var loginFormData loginForm
	err := c.Bind(&loginFormData) // This will infer what binder to use depending on the content-type header.
	if err != nil {
		log.Println("[TasadarAPI] Error in contact form handling at c.Bind(&newPWFormData): ", err)
		c.Redirect(301, "https://tasadar.net/login/error")
		return
	}
	username := c.PostForm("username")
	password := c.PostForm("password")
	if authUser(username, password) {
		// Create a new token object, specifying signing method and the claims
		// you would like it to contain.
		groups, err := authGetGroupsString(username)
		if err != nil {
			log.Println("[TasadarAPI] Error getting groups string: ", err)
			c.Redirect(301, "https://tasadar.net/login/error")
			return
		}
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
			"sub":    username,
			"groups": groups,
			// ToDo expire token
		})
		// Sign and get the complete encoded token as a string using the secret
		hmacSampleSecret := []byte("v09AoteRzfUEDbxqjDFFyWaSPrNeDqOj")
		value, err := token.SignedString(hmacSampleSecret)
		if err != nil {
			log.Println("[TasadarAPI] Error while creating signed JWT Token String: ", err)
			c.Redirect(301, "https://tasadar.net/login/error")
			return
		}
		c.SetCookie("tasadar-token", value, 2678400, "", ".tasadar.net", true, true)
		c.Redirect(301, "https://tasadar.net/login/success")
	} else {
		c.Redirect(301, "https://tasadar.net/login/wrong")
	}
}

func tasadarLoginVerify(c *gin.Context) {
	// sample token string taken from the New example
	tokenString, err := c.Cookie("tasadar-token")
	if err != nil {
		c.String(200, "Error")
	}
	// Parse takes the token string and a function for looking up the key. The latter is especially
	// useful if you use multiple keys for your application.  The standard is to use 'kid' in the
	// head of the token to identify which key to use, but the parsed token (head and claims) is provided
	// to the callback, providing flexibility.
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// Don't forget to validate the alg is what you expect:
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("Unexpected signing method: %v", token.Header["alg"])
		}
		hmacSampleSecret := []byte("v09AoteRzfUEDbxqjDFFyWaSPrNeDqOj")
		// hmacSampleSecret is a []byte containing your secret, e.g. []byte("my_secret_key")
		return hmacSampleSecret, nil
	})

	if _, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		c.String(200, "OK: ")
	} else {
		c.String(200, "Traitor!")
	}
}

func emailVerifyHandler(c *gin.Context) {
	// Verify token and getuser
	// send correct file back
	// set email of user
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
		client, err := newClient(rconAddress, 25575, rconPassword)
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
			err = client.reconnect()
			if err != nil {
				log.Println("[TasadarAPI] Error occured rebuilding connection, aborting... err: ", err)
				return
			}
			_, _ = client.sendCommand("whitelist add " + oldName)
			return
		}
		if !strings.Contains(response, "Added") {
			log.Println("[TasadarAPI] Error adding MCuser "+mcuser+" from whitelist: ", response)
			c.Redirect(302, "https://mc.tasadar.net/error")
			_, _ = client.sendCommand("whitelist add " + oldName)
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
