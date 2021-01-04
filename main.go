package main

import (
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"

	"github.com/gin-gonic/gin"                          // This provides needed directives to interface with the WebServer
	_ "github.com/heroku/x/hmetrics/onload"             // Heroku advanced go metrics
	"github.com/keybase/go-logging"                     // This unifies logging across the application
	"github.com/tionis/tsdr-api/adapters/discord"       // This provides the adapter to discord
	"github.com/tionis/tsdr-api/adapters/matrix"        // This provides the adapter to matrix
	"github.com/tionis/tsdr-api/adapters/telegram"      // This provides the adapter to telegram
	"github.com/tionis/tsdr-api/data"                   // This provides the application data layer
	"github.com/tionis/tsdr-api/web"                    // This provides the webServer
	UniPassauBot "github.com/tionis/uni-passau-bot/api" // This provides a simple LEGACY uni passau bot that can be started
)

const defaultPort = "8081"

var mainLog = logging.MustGetLogger("main")

var logFormat = logging.MustStringFormatter(
	`%{color}%{module} ▶ %{level:.4s}%{color:reset} %{message}`,
)

// Initialize Main Functions
func main() {
	logging.SetFormatter(logFormat)
	syncGroup := &sync.WaitGroup{}

	// Get environment variables
	port := os.Getenv("PORT")
	if port == "" {
		mainLog.Warning("Failed to detect Port Variable, switching to default :8081")
		port = defaultPort
	}
	sqlURL := os.Getenv("DATABASE_URL")
	if sqlURL == "" {
		mainLog.Info("Database: " + os.Getenv("DATABASE_URL"))
		mainLog.Fatal("Fatal Error getting Database Information!")
	}
	discordToken := os.Getenv("DISCORD_TOKEN")
	if discordToken == "" {
		mainLog.Fatal("No glyph discord token specified")
	}
	uniPassauBotToken := os.Getenv("UNIPASSAUBOT_TOKEN")
	if uniPassauBotToken == "" {
		mainLog.Fatal("No uni passau telegram token specified")
	}
	telegramToken := os.Getenv("TELEGRAM_TOKEN")
	if telegramToken == "" {
		mainLog.Fatal("No glyph telegram token specified")
	}
	matrixHomerServer := os.Getenv("MATRIX_HOMESERVER_URL")
	if matrixHomerServer == "" {
		mainLog.Fatal("No glyph matrix homeserver URL specified")
	}
	matrixUserName := os.Getenv("MATRIX_USERNAME")
	if matrixUserName == "" {
		mainLog.Fatal("No glyph matrix username specified")
	}
	matrixPassword := os.Getenv("MATRIX_PASSWORD")
	if matrixPassword == "" {
		mainLog.Fatal("No glyph matrix password specified")
	}

	// Initialize data layer
	dataBackend := data.DBInit(sqlURL)

	// Initialize stop channels
	var stopSignals = []chan bool{
		make(chan bool),
		make(chan bool),
		make(chan bool),
		make(chan bool),
	}
	var systemStopSignal chan bool
	go stopDetector(systemStopSignal)
	go stopMultiPlexer(systemStopSignal, stopSignals)

	// Detect Development Mode
	var isProduction bool
	switch strings.ToUpper(os.Getenv("MODE")) {
	case "PRODUCTION":
		mainLog.Info("Detected Production Mode")
		gin.SetMode(gin.ReleaseMode)
		isProduction = true
	case "DEBUG":
		logging.SetFormatter(logging.MustStringFormatter(
			`%{color}%{module}: %{shortfile} ▶ %{level:.4s}%{color:reset} %{message}`,
		))
		mainLog.Info("Detected Debug Mode")
		gin.SetMode(gin.DebugMode)
		isProduction = false
	default:
		mainLog.Warning("No Mode Config detected, switching to Production Mode")
		gin.SetMode(gin.ReleaseMode)
		isProduction = true
	}

	// Start Uni-Passau-Bot // LEGACY CODE THAT WILL BE REMOVED IN THE FUTURE
	go UniPassauBot.UniPassauBot(uniPassauBotToken)

	// Start Glyph Discord Bot
	go discord.Init(dataBackend, discordToken).Start(stopSignals[0], syncGroup)

	// Start Glyph Telegram Bot
	go telegram.Init(dataBackend, telegramToken).Start(stopSignals[1], syncGroup)

	// Start Glyph Matrix Bot
	go matrix.Init(dataBackend, matrixHomerServer, matrixUserName, matrixPassword).Start(stopSignals[2], syncGroup)

	// Start WebServer - this is a blocking operation
	web.Init(isProduction, port).Start(stopSignals[3], syncGroup)

	// Wait until all goroutines have stopped
	syncGroup.Wait()
}

func stopDetector(stopSignal chan bool) {
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, syscall.SIGQUIT, syscall.SIGHUP)
	<-sc
	mainLog.Info("Received stop signal...")
	stopSignal <- true
}

// stopMultiPlexer forwars the value of stopSignal to all channels in stopSubSignals when a value is received
func stopMultiPlexer(stopSignal chan bool, stopSubSignals []chan bool) {
	value := <-stopSignal
	length := len(stopSubSignals)
	for index, item := range stopSubSignals {
		item <- value
		mainLog.Debugf("Forwarded stop signal %v/%v...", index+1, length)
	}
}
