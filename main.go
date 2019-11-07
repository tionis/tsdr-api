package main

import (
	"log"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v7"
	_ "github.com/heroku/x/hmetrics/onload"
	"github.com/robfig/cron"
)

// TODO
// Check Alpha-telegram-bot
// Move to postgresql
// Check for more memory leaks (maybe reduce dependencies?)

var redclient *redis.Client

// Main and Init
func main() {
	// Start Uni-Passau-Bot
	go uniPassauBot()

	// Start Alpha Discord Bot
	go alphaDiscordBot()

	// Start Alpha Telegram Bot
	go alphaTelegramBot()

	// Init redis
	redisS1 := strings.Split(strings.TrimPrefix(os.Getenv("REDIS_URL"), "redis://"), "@")
	redisS2 := strings.Split(redisS1[0], ":")
	redclient = redis.NewClient(&redis.Options{
		Addr:     redisS1[1],
		Password: redisS2[1],
		DB:       0, // use default DB
	})
	if _, err := redclient.Ping().Result(); err != nil {
		log.Println("[FATAL] - Error connecting to redis database! err: ", err)
	}

	// Cron Job Definitions
	c := cron.New()
	c.AddFunc("*/15 * * * *", func() { updateAuth() })
	//c.AddFunc("*/5 * * * *", func() { updateStatus() })
	c.Start()

	// Creates a gin router with default middleware:
	// logger and recovery (crash-free) middleware
	// passes it to routes for setting of the routes
	router := gin.Default()
	routes(router) // Setup standard Routes and WA API
	//awsProxy(router)        // Setup AWS Proxy Routes
	log.Fatal(router.Run()) // Start WebServer
	c.Stop()
}

func handleError(err error) {
	log.Println("[TasdarApi] General Error: ", err)
}
