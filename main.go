package main

import (
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	_ "github.com/heroku/x/hmetrics/onload"
	"github.com/robfig/cron"
)

const defaultPort = "8081"

type hostSwitch map[string]http.Handler

var isProduction bool

// Implement the ServeHTTP method on our new type
func (hs hostSwitch) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Check if a http.Handler is registered for the given host.
	// If yes, use it to handle the request.
	if handler := hs[r.Host]; handler != nil {
		handler.ServeHTTP(w, r)
	} else {
		// Handle host names for which no handler is registered
		http.Error(w, "Forbidden", 403)
	}
}

// Main and Init
func main() {
	// Init very important bits
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

	// Cronjob Definitions
	// MC Cronjobs
	c := cron.New()
	_ = c.AddFunc("@every 5m", func() { pingMC() })
	_ = c.AddFunc("@every 5m", func() { updateMC() })
	c.Start()
	defer c.Stop()

	// Creates a gin router with default middleware:
	// logger and recovery (crash-free) middleware
	// passes it to routes for setting of the routes
	port := os.Getenv("PORT")
	if port == "" {
		log.Println("[Tasadar] Failed to detect Port Variable, switching to default :8081")
		port = defaultPort
	}
	apiRouter := gin.Default()
	apiRoutes(apiRouter) // Initialize API Routes

	hs := make(hostSwitch)
	if isProduction {
		hs["api.tasadar.net"] = apiRouter
		//hs["auth.tasadar.net"] = authRouter
	} else {
		hs["api.localhost:"+os.Getenv("PORT")] = apiRouter
		//hs["auth.localhost:8082"] = authRouter
	}

	log.Fatal(http.ListenAndServe(":"+os.Getenv("PORT"), hs))
}
