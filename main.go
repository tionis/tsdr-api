package main

import (
	"fmt"
	"log"
	"os"
	"strconv"
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
var psqlInfo string

// Main and Init
func main() {
	// Init postgres
	postgresString1 := strings.Split(strings.TrimPrefix(os.Getenv("DATABASE_URL"), "postgres://"), "@")
	postgresString2 := strings.Split(postgresString1[0], ":")
	postgresString3 := strings.Split(postgresString1[1], ":")
	postgresString4 := strings.Split(postgresString3[1], "/")
	host := postgresString3[0]
	port, err := strconv.Atoi(postgresString4[0])
	if err != nil {
		log.Fatal("Could not read Postgres Port")
	}
	user := postgresString2[0]
	password := postgresString2[1]
	dbname := postgresString4[1]
	psqlInfo = fmt.Sprintf("host=%s port=%d user=%s "+
		"password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname)

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

	// Cronjob Definitions
	c := cron.New()
	_ = c.AddFunc("@every 15m", func() { updateAuth() })
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
