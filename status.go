package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type tasadarStatus struct {
	Grav      bool
	Minecraft bool
	Glyph     bool
	Wiki      bool
	Nextcloud bool
	Shiori    bool
	Golinks   bool
	Dev       bool
	Books     bool
	Monica    bool
	Matrix    bool
	//Collabora  bool No Check curretly known
	TurnServer bool
	AuthServer bool
	APIServer  bool
}

var status tasadarStatus

func updateStatus() {
	// Define all services and upchecks for them here
	// Maybe Cross-reference manual data from alpha-tg-bot
	// Handle Minecraft Server
	pingMC()
	status.Minecraft = mcRunning

	// Handle Grav
	resp, _ := http.Get("https://grav.tasadar.net")
	status.Grav = resp.StatusCode == 200

	// Ping nextcloud over api - use status.php in future!
	resp, _ = http.Get("https://cloud.tasadar.net")
	status.Nextcloud = resp.StatusCode == 200

	// access dokuwiki api glyph
	resp, _ = http.Get("https://glyph.tasadar.net/lib/exe/xmlrpc.php")
	status.Glyph = resp.StatusCode == 200

	// access tasadar wiki api
	resp, _ = http.Get("https://wiki.tasadar.net/lib/exe/xmlrpc.php")
	status.Wiki = resp.StatusCode == 200

	// access shiori api
	resp, _ = http.Get("https://shiori.tasadar.net")
	status.Shiori = resp.StatusCode == 200

	// test search (maybe replace with custom javascript?)
	resp, _ = http.Get("https://search.tasadar.net")
	status.Golinks = resp.StatusCode == 200

	// test dev
	resp, _ = http.Get("https://dev.tasadar.net")
	status.Dev = resp.StatusCode == 401

	// test books
	resp, _ = http.Get("https://books.tasadar.net")
	status.Books = resp.StatusCode == 200

	// test monica api
	resp, _ = http.Get("https://monica.tasadar.net")
	status.Monica = resp.StatusCode == 200

	// test matrix
	resp, _ = http.Get("https://matrix.tasadar.net:8448")
	status.Matrix = resp.StatusCode == 200

	// check turnserver
	status.TurnServer = true

	//check auth and API-server (currently this server, so always true) may change in future
	status.APIServer, status.AuthServer = true, true
}

func statusHandler(c *gin.Context) {
	//updateStatus()
	c.JSON(200, status)
}
