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
	// Start and init Webhook Component
	router.GET("/favicon.ico", favicon)
	router.GET("/", index)
	router.GET("/echo", httpecho)
	router.POST("/whatsapp", whatsapp)
	router.GET("/mensa/today", retFoodToday)
	router.GET("/mensa/tommorow", retFoodTommorow)
}

func retFoodToday(c *gin.Context) {
	c.String(200, foodtoday())
}

func retFoodTommorow(c *gin.Context) {
	c.String(200, foodtomorrow())
}

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
