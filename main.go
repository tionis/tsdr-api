package main

import (
	"log"

	"github.com/gin-gonic/gin"
	_ "github.com/heroku/x/hmetrics/onload"
	"github.com/robfig/cron"
)

var psqlInfo string

// Main and Init
func main() {
	// Init very important bits
	dbInit()

	// Start Uni-Passau-Bot
	go uniPassauBot()

	// Start Alpha Discord Bot
	go glyphDiscordBot()

	// Start Alpha Telegram Bot
	go glyphTelegramBot()

	// Cronjob Definitions
	c := cron.New()
	_ = c.AddFunc("@every 5m", func() { pingMC() })
	_ = c.AddFunc("@every 5m", func() { updateMC() })
	c.Start()

	// Creates a gin router with default middleware:
	// logger and recovery (crash-free) middleware
	// passes it to routes for setting of the routes
	router := gin.Default()
	routes(router) // Setup standard Routes and WA API
	//awsProxy(router)        // Setup AWS Proxy Routes
	log.Fatal(router.Run()) // Start WebServer
	//c.Stop()
}
