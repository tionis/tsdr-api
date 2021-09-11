package main

import (
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	_ "github.com/heroku/x/hmetrics/onload"
	"github.com/keybase/go-logging"
	UniPassauBot "github.com/tionis/uni-passau-bot/api"
)

const defaultPort = "8081"

type hostSwitch map[string]http.Handler

var mainLog = logging.MustGetLogger("main")

var logFormat = logging.MustStringFormatter(
	`%{color}%{module} ▶ %{level:.4s}%{color:reset} %{message}`,
)

var isProduction bool

// Initialize Main Functions
func main() {
	logging.SetFormatter(logFormat)
	// Initialize basic requirements
	dbInit()

	// Detect Development Mode
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

	// Start Glyph Discord Bot // deactivated in favor of github.com/tionis/glyph
	// go glyphDiscordBot()

	// Start Glyph Telegram Bot
	go glyphTelegramBot()

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
	apiRouter := gin.Default()
	//apiRouter.Use(gin.LoggerWithFormatter(ginLogFormatter))
	apiRouter.Use(gin.Recovery())
	apiRoutes(apiRouter) // Initialize API Routes
	corsRouter := gin.Default()
	corsRoutes(corsRouter)

	// Create HostSwitch Handling for Virtual Hosts support
	hs := make(hostSwitch)
	if isProduction {
		hs["api.tasadar.net"] = apiRouter
		hs["cors.tasadar.net"] = corsRouter
	} else {
		hs["api.localhost:"+os.Getenv("PORT")] = apiRouter
		hs["api.localhost"] = apiRouter
		hs["cors.localhost:"+os.Getenv("PORT")] = corsRouter
		hs["cors.localhost"] = corsRouter
	}

	// Start WebServer
	mainLog.Fatal(http.ListenAndServe(":"+os.Getenv("PORT"), hs))
}

/*func ginLogFormatter(param gin.LogFormatterParams) string {
	return fmt.Sprintf("%s - [%s] \"%s %s %s %d %s \"%s\" %s\"\n",
		param.ClientIP,
		param.TimeStamp.Format(time.RFC1123),
		param.Method,
		param.Path,
		param.Request.Proto,
		param.StatusCode,
		param.Latency,
		param.Request.UserAgent(),
		param.ErrorMessage,
	)
}*/

// Hostswitch HTTP Handler that enables the use in a standard lib way
func (hs hostSwitch) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if handler := hs[r.Host]; handler != nil {
		handler.ServeHTTP(w, r)
	} else {
		// Handle host names for which no handler is registered
		http.Error(w, "Forbidden", http.StatusForbidden)
	}
}
