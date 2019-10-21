package main

import (
	"log"

	"github.com/gin-gonic/gin"
	_ "github.com/heroku/x/hmetrics/onload"
	"github.com/hetznercloud/hcloud-go/hcloud"
)

// Global Variable
var hetznerClient *hcloud.Client

// Main and Init
func main() {
	// Init APIs
	//hetznerClient := hcloud.NewClient(hcloud.WithToken(os.Getenv("HetznerApiToken")))
	//center, _ := hetznerClient

	// Start Uni-Passau-Bot
	go uniPassauBot()

	// Start Alpha Discord Bot
	go alphaDiscordBot()

	// Start Alpha Telegram Bot
	go alphaTelegramBot()

	// Creates a gin router with default middleware:
	// logger and recovery (crash-free) middleware
	// passes it to routes for setting of the routes
	router := gin.Default()
	routes(router)
	log.Fatal(router.Run())
}
