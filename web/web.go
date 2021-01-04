package web

import (
	"context"
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"              // This provides the web framework
	_ "github.com/heroku/x/hmetrics/onload" // Heroku advanced go metrics
	"github.com/keybase/go-logging"         // This unifies logging across the application
)

// This map contains the data for the switch enabling virtual host integration
type hostSwitch map[string]http.Handler

// Server represents a web server configutaion
type Server struct {
	logger     *logging.Logger
	apiRouter  *gin.Engine
	corsRouter *gin.Engine
	hs         hostSwitch
	port       string
}

// Init initializes the web server and returns a Server that can be started
func Init(isProduction bool, port string) *Server {
	s := Server{logging.MustGetLogger("web"), nil, nil, make(hostSwitch), port}

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
		s.hs["api.localhost:"+s.port] = s.apiRouter
		s.hs["api.localhost"] = s.apiRouter
		s.hs["cors.localhost:"+s.port] = s.corsRouter
		s.hs["cors.localhost"] = s.corsRouter
	}

	return &s
}

// Start starts the WebServer in a blocking operation
func (s *Server) Start(stop chan bool, wg *sync.WaitGroup) {
	wg.Add(1)
	srv := &http.Server{Addr: ":" + s.port}

	// Start WebServer in go routine
	go func() {
		defer wg.Done() // let the app now we are done cleaning up

		// always returns error. ErrServerClosed on graceful close
		if err := srv.ListenAndServe(); err != http.ErrServerClosed {
			// unexpected error. port in use?
			s.logger.Fatalf("ListenAndServe(): %v", err)
		}
	}()

	<-stop
	// Shutdown after receiving stop signal
	if err := srv.Shutdown(context.TODO()); err != nil {
		s.logger.Fatal(err) // failure/timeout shutting down the server gracefully
	}
}

// Hostswitch HTTP Handler that enables the use in a standard lib way.
// It is needed to enable redirection of http request to the correct virtual host router/handler
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
