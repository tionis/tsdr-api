package main

import (
	"fmt"
	"log"
	"net/http/httputil"
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"
	_ "github.com/heroku/x/hmetrics/onload"
)

// Main and Init
func routes(router *gin.Engine) {
	// Default Stuff
	router.GET("/favicon.ico", favicon)
	router.GET("/", index)
	router.GET("/echo", httpecho)

	// WhatsApp Bot
	router.POST("/twilio/uni-passau-bot/whatsapp", whatsapp)

	// CURL API
	router.GET("/mensa/today", retFoodToday)
	router.GET("/mensa/tommorow", retFoodTomorow)

	// Google Assitant API - WIP
	router.POST("/dialogflow/alpha", retFoodToday)

	// MC Handling
	/*router.POST("/mc/started", mcSetIP)
	router.POST("/mc/stop", mcStop)*/

	router.GET("/tg/:message", func(c *gin.Context) {
		message := c.Param("message")
		msgAlpha <- message
		c.String(200, message)
	})
}

/*func mcSetIP(c *gin.Context) {
	// Set IP from request and check it with api
}

func mcStop(c *gin.Context) {
	go mcStopChecker()
	c.JSON(200, gin.H{"message": "Check will be executed in 2 Minutes"})
}

func mcStopChecker() {
	time.Sleep(2 * time.Minute)
	// check api for vms
	// Stop VM, after 2 min check for active VMs (ignore those specified in VMsToIgnore) if still exists write message on alpha to "TG_ADMIN"
	CheckError := true
	if CheckError {
		tionis := tb.Chat{ID: 248533143}
		alpha.Send(&tionis, "There are more VMs than there should be!")
	} else {
		log.Println("[MC] Shutdown successfull")
		mcOnline = false
		mcIP = "0.0.0.0"
	}
}*/

// handle simple GET requests
func retFoodToday(c *gin.Context) {
	c.String(200, foodtoday())
}

func retFoodTomorow(c *gin.Context) {
	c.String(200, foodtomorrow())
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
	c.File("static/favicon.ico")
}

func index(c *gin.Context) {
	c.File("static/index.html")
}

// Handle WhatsApp Twilio Webhook
func whatsapp(c *gin.Context) {
	buf := make([]byte, 1024)
	num, _ := c.Request.Body.Read(buf)
	params, err := url.ParseQuery(string(buf[0:num]))
	if err != nil {
		log.Print(c.Error(err))
		return
	}
	text := strings.Join(params["Body"], " ")
	if strings.Contains(text, "tommorow") || strings.Contains(text, "morgen") || strings.Contains(text, "Tommorow") || strings.Contains(text, "Morgen") {
		c.String(200, foodtomorrow())
	} else if strings.Contains(text, "food") || strings.Contains(text, "essen") || strings.Contains(text, "Food") || strings.Contains(text, "Essen") {
		c.String(200, foodtoday())
	} else if strings.Contains(text, "Hallo") || strings.Contains(text, "hallo") {
		c.String(200, "Hallo wie gehts? - Schreibe mir food oder essen morgen um loszulegen!")
	} else if strings.Contains(text, "danke") || strings.Contains(text, "Danke") {
		c.String(200, "Gern geschehen!")
	} else {
		c.String(200, "Befehl nicht erkannt - versuche es mal mit einem Hallo!")
	}
}
