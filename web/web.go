package web

import (
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/keybase/go-logging"
)

type hostSwitch map[string]http.Handler

// Server represents the web server
type Server struct {
	logger     *logging.Logger
	apiRouter  *gin.Engine
	corsRouter *gin.Engine
	hs         hostSwitch
}

// Init initializes the web server and returns a Server that can be started
func Init(isProduction bool) *Server {
	s := Server{logging.MustGetLogger("web"), nil, nil, make(hostSwitch)}

	s.apiRouter = gin.Default()
	s.apiRouter.Use(gin.Recovery())
	s.apiRoutes() // Initialize API Routes
	s.corsRouter = gin.Default()
	s.corsRouter.Use(gin.Recovery())
	s.corsRoutes() // Initialize CORS Routes

	// Create HostSwitch Handling for Virtual Hosts support
	if isProduction {
		s.hs["api.tasadar.net"] = s.apiRouter
		s.hs["cors.tasadar.net"] = s.corsRouter
	} else {
		s.hs["api.localhost:"+os.Getenv("PORT")] = s.apiRouter
		s.hs["api.localhost"] = s.apiRouter
		s.hs["cors.localhost:"+os.Getenv("PORT")] = s.corsRouter
		s.hs["cors.localhost"] = s.corsRouter
	}

	return &s
}

// Start starts the WebServer in a blocking operation
func (s *Server) Start() {
	// Start WebServer
	s.logger.Fatal(http.ListenAndServe(":"+os.Getenv("PORT"), s.hs))
}

// Hostswitch HTTP Handler that enables the use in a standard lib way
func (hs hostSwitch) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if handler := hs[r.Host]; handler != nil {
		handler.ServeHTTP(w, r)
	} else {
		// Handle host names for which no handler is registered
		http.Error(w, "Forbidden", http.StatusForbidden)
	}
}

// Use following with s.apiRouter.Use(gin.LoggerWithFormatter(ginLogFormatter)) // Better logging
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
