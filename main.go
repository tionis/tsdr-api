package main

import (
	"log"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v7"
	_ "github.com/heroku/x/hmetrics/onload"
)

var redclient *redis.Client

// Main and Init
func main() {
	//resp, err := Query("tasadar.net", 25565, time.Second*5)
	//if err != nil {
	//	log.Fatal(err)
	//}
	//fmt.Printf("%d/%d players are online.", resp.PlayerCount, resp.MaxPlayers)

	// Start Uni-Passau-Bot
	go uniPassauBot()

	// Start Alpha Discord Bot
	go alphaDiscordBot()

	// Start Alpha Telegram Bot
	go alphaTelegramBot()

	// Init redis
	redclient = redis.NewClient(&redis.Options{
		Addr:     os.Getenv("REDIS_URL"),
		Password: "", // no password set
		DB:       0,  // use default DB
	})

	if _, err := redclient.Ping().Result(); err != nil {
		log.Println("[FATAL] - Error connecting to redis database!")
	}

	// Creates a gin router with default middleware:
	// logger and recovery (crash-free) middleware
	// passes it to routes for setting of the routes
	router := gin.Default()
	routes(router) // Setup standard Routes and WA API
	//awsProxy(router)        // Setup AWS Proxy Routes
	log.Fatal(router.Run()) // Start WebServer
}
