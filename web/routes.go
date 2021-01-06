package web

import (
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"              // This provides the web framework
	_ "github.com/heroku/x/hmetrics/onload" // Heroku advanced go metrics
	"github.com/keybase/go-logging"         // This unifies logging across components of the application
	"github.com/tionis/tsdr-api/data"
	UniPassauBot "github.com/tionis/uni-passau-bot/api" // This provides logic to get the current food of the uni passau
)

var apiLog = logging.MustGetLogger("API")

/*type tasadarToken struct {
    jwt.Payload
    Groups string `json:"groups,omitempty"`
}*/

type glyphMsgAPIObject struct {
	Message string `form:"message" json:"message" binding:"required"`
	Token   string `form:"token" json:"token" binding:"required"`
}

func (s *Server) apiRoutes() {
	// Default Stuff
	s.apiRouter.GET("/favicon.svg", favicon)
	s.apiRouter.GET("/", index)
	s.apiRouter.GET("/glyph", glyphRedirect)
	s.apiRouter.NoRoute(notFound)
	s.apiRouter.GET("/echo", httpecho)

	// Handle short links
	s.apiRouter.GET("/discord", discordinvite)
	s.apiRouter.GET("/log/today", logTodayRedirect)

	// Handle Status Watch
	s.apiRouter.GET("/onlinecheck", func(c *gin.Context) {
		c.String(418, "I'm online")
	})

	// CURL API
	s.apiRouter.GET("/mensa/today", retFoodToday)
	s.apiRouter.GET("/mensa/tomorrow", retFoodTomorow)
	s.apiRouter.GET("/mensa/week", retFoodWeek)

	// Glyph Communication API
	s.apiRouter.POST("/glyph/send", s.glyphMessageSendHandler)
}

func (s *Server) glyphMessageSendHandler(c *gin.Context) {
	var messageData glyphMsgAPIObject
	err := c.Bind(messageData) // This will infer what binder to use depending on the content-type header.
	if err != nil {
		apiLog.Error("Error while trying to bind glyph discord message: ", err)
		c.JSON(400, err)
		return
	}
	tokenData, err := s.dataBackend.GetSendTokenByID(messageData.Token)
	if err != nil {
		c.JSON(500, err)
		return
	}
	for _, item := range tokenData.Adapters {
		adapterChannel, err := s.dataBackend.GetAdapterChannel(item)
		if err != nil {
			apiLog.Errorf("Error while getting adapter channel for %v: %v", item, err)
		} else {
			adapterChannel <- data.AdapterMessage{UserID: tokenData.UserID, Message: messageData.Message}
		}
	}
	c.String(200, `{status: "ok"}`)
}

// handle test case
func httpecho(c *gin.Context) {
	// Test Code
	requestDump, err := httputil.DumpRequest(c.Request, true)
	if err != nil {
		apiLog.Error("Error in echo: ", err)
	}
	c.String(200, string(requestDump))
}

func logTodayRedirect(c *gin.Context) {
	currentTime := time.Now()
	link := "https://wiki.tasadar.net/en/notes/log/" + currentTime.Format("2006/01/02")
	c.Redirect(http.StatusTemporaryRedirect, link)
}

// Handle both root thingies
func favicon(c *gin.Context) {
	c.File("web/static/icons/favicon.svg")
}

func discordinvite(c *gin.Context) {
	c.Redirect(302, "https://discord.gg/CSZyd87")
}

func index(c *gin.Context) {
	c.File("web/static/index.html")
}

func glyphRedirect(c *gin.Context) {
	c.Redirect(302, "https://discordapp.com/oauth2/authorize?client_id=635860503041802253&scope=bot&permissions=8")
}

func notFound(c *gin.Context) {
	c.File("web/static/error-pages/404.html")
}

// handle simple GET requests for food
func retFoodToday(c *gin.Context) {
	c.String(200, UniPassauBot.FoodToday())
}
func retFoodTomorow(c *gin.Context) {
	c.String(200, UniPassauBot.FoodTomorrow())
}
func retFoodWeek(c *gin.Context) {
	c.String(200, UniPassauBot.FoodWeek())
}

// Handle Cors Proxy
func (s *Server) corsRoutes() {
	s.corsRouter.Any("/*proxyPath", corsProxy)
}

type corsTransport http.Header

func (t corsTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	resp, err := http.DefaultTransport.RoundTrip(r)
	if err != nil {
		return nil, err
	}
	resp.Header.Set("Access-Control-Allow-Origin", "*")
	resp.Header.Set("Access-Control-Allow-Methods", "POST, GET")
	resp.Header.Set("Access-Control-Allow-Headers", "Content-Type")
	return resp, nil
}

func corsProxy(c *gin.Context) {
	if c.Param("proxyPath") == "/" {
		c.String(200, "Just append the url(including protocol) you want to call to the domain.\nAttention: For legal reasons requests are logged!")
	}
	remote, err := url.Parse(strings.TrimPrefix(c.Param("proxyPath"), "/"))
	if err != nil {
		panic(err)
	}

	proxy := httputil.ReverseProxy{Director: func(req *http.Request) {
		req.Header = c.Request.Header
		req.Host = remote.Host
		req.URL.Scheme = remote.Scheme
		req.URL.Host = remote.Host
		req.URL.Path = remote.Path
	}, Transport: corsTransport(http.Header{}),
	}
	proxy.ServeHTTP(c.Writer, c.Request)
}
