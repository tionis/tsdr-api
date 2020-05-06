package main

import (
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	_ "github.com/heroku/x/hmetrics/onload"
	"github.com/robfig/cron/v3"
)

const defaultPort = "8081"

type hostSwitch map[string]http.Handler

var isProduction bool

// Initialize Main Functions
func main() {
	// Initialize basic requirements
	dbInit()

	// Detect Development Mode
	switch strings.ToUpper(os.Getenv("MODE")) {
	case "PRODUCTION":
		log.Println("[Tasadar] Detected Production Mode")
		gin.SetMode(gin.ReleaseMode)
		isProduction = true
	case "DEBUG":
		log.SetFlags(log.LstdFlags | log.Lshortfile)
		log.Println("[Tasadar] Detected Debug Mode")
		gin.SetMode(gin.DebugMode)
		isProduction = false
	default:
		log.SetFlags(log.LstdFlags | log.Lshortfile)
		log.Println("[Tasadar] No Operation Mode set, switching to Default: Debug Mode")
		gin.SetMode(gin.DebugMode)
		isProduction = false
	}

	// Start Uni-Passau-Bot
	go uniPassauBot()

	// Start Alpha Discord Bot
	go glyphDiscordBot()

	// Start Alpha Telegram Bot
	go glyphTelegramBot()

	// Start Quotator Telegram Bot
	go quotatorTelegramBot()

	// Start Telegram Trello Bot
	go trelloTelegramBot()

	// Cronjob Definitions
	// MC Cronjobs
	c := cron.New()
	c.AddFunc("@every 5m", func() { pingMC() })
	c.AddFunc("@every 5m", func() { updateMC() })
	loadTrelloBotJobs(c)
	c.Start()
	defer c.Stop()

	// Create Default gin router
	port := os.Getenv("PORT")
	if port == "" {
		log.Println("[Tasadar] Failed to detect Port Variable, switching to default :8081")
		port = defaultPort
	}
	apiRouter := gin.Default()
	apiRoutes(apiRouter) // Initialize API Routes

	// Create HostSwitch Handling for Virtual Hosts support
	hs := make(hostSwitch)
	if isProduction {
		hs["api.tasadar.net"] = apiRouter
		//hs["auth.tasadar.net"] = authRouter
	} else {
		hs["api.localhost:"+os.Getenv("PORT")] = apiRouter
		hs["api.localhost"] = apiRouter
		//hs["auth.localhost:8082"] = authRouter
	}

	// Start Webserver
	log.Fatal(http.ListenAndServe(":"+os.Getenv("PORT"), hs))
}

// Functions implementing helper methods for main initialization:

// Hostswitch HTTP Handler that enables the use in a standard lib way
func (hs hostSwitch) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if handler := hs[r.Host]; handler != nil {
		handler.ServeHTTP(w, r)
	} else {
		// Handle host names for which no handler is registered
		http.Error(w, "Forbidden", http.StatusForbidden)
	}
}
