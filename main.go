package main

import (
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	_ "github.com/heroku/x/hmetrics/onload" // Heroku advanced go metrics
	"github.com/keybase/go-logging"
	"github.com/tionis/tsdr-api/adapters/discord"
	"github.com/tionis/tsdr-api/adapters/matrix"
	"github.com/tionis/tsdr-api/adapters/telegram"
	"github.com/tionis/tsdr-api/data"
	"github.com/tionis/tsdr-api/web"
	UniPassauBot "github.com/tionis/uni-passau-bot/api"
)

const defaultPort = "8081"

var mainLog = logging.MustGetLogger("main")

var logFormat = logging.MustStringFormatter(
	`%{color}%{module} ▶ %{level:.4s}%{color:reset} %{message}`,
)

// Initialize Main Functions
func main() {
	logging.SetFormatter(logFormat)
	// Initialize basic requirements
	dataBackend := data.DBInit()

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

	// Start Uni-Passau-Bot
	go UniPassauBot.UniPassauBot(os.Getenv("UNIPASSAUBOT_TOKEN"))

	// Start Glyph Discord Bot
	go discord.InitBot(dataBackend)

	// Start Glyph Telegram Bot
	//go glyphTelegramBot(!isProduction)
	go telegram.InitBot(dataBackend, false)

	// Start Glyph Matrix Bot
	go matrix.InitBot(dataBackend)

	// Cronjob Definitions
	// MC Cronjobs
	//loc, err := time.LoadLocation("Europe/Berlin")
	//if err != nil {
	//	mainLog.Warning("[Tasadar] Error loading correct time zone!")
	//}
	//c := cron.New(cron.WithLocation(loc))
	//c.AddFunc("@every 5m", func() { pingMC() })
	//c.AddFunc("@every 5m", func() { updateMC() })
	//c.AddFunc("@every 1m", func() { remindChecker() })
	//c.Start()
	//defer c.Stop()

	// Create Default gin router
	port := os.Getenv("PORT")
	if port == "" {
		mainLog.Warning("Failed to detect Port Variable, switching to default :8081")
		port = defaultPort
	}

	// Start WebServer - this is concurrent blocking operation
	web.Init(isProduction).Start()
}
