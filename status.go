package main

import (
	"log"
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
	resp, err := http.Get("https://grav.tasadar.net")
	if err != nil {
		log.Println("[Fatal] Error getting Status: ", err)
		return
	}
	status.Grav = resp.StatusCode == 200

	// Ping nextcloud over api - use status.php in future!
	resp, err = http.Get("https://cloud.tasadar.net")
	if err != nil {
		log.Println("[Fatal] Error getting Status: ", err)
		return
	}
	status.Nextcloud = resp.StatusCode == 200

	// access dokuwiki api glyph
	resp, err = http.Get("https://glyph.tasadar.net/lib/exe/xmlrpc.php")
	if err != nil {
		log.Println("[Fatal] Error getting Status: ", err)
		return
	}
	status.Glyph = resp.StatusCode == 200

	// access tasadar wiki api
	resp, err = http.Get("https://wiki.tasadar.net/lib/exe/xmlrpc.php")
	if err != nil {
		log.Println("[Fatal] Error getting Status: ", err)
		return
	}
	status.Wiki = resp.StatusCode == 200

	// access shiori api
	resp, err = http.Get("https://shiori.tasadar.net")
	if err != nil {
		log.Println("[Fatal] Error getting Status: ", err)
		return
	}
	status.Shiori = resp.StatusCode == 200

	// test search (maybe replace with custom javascript?)
	resp, err = http.Get("https://search.tasadar.net")
	if err != nil {
		log.Println("[Fatal] Error getting Status: ", err)
		return
	}
	status.Golinks = resp.StatusCode == 200

	// test dev
	resp, err = http.Get("https://dev.tasadar.net")
	if err != nil {
		log.Println("[Fatal] Error getting Status: ", err)
		return
	}
	status.Dev = resp.StatusCode == 401

	// test books
	resp, err = http.Get("https://books.tasadar.net")
	if err != nil {
		log.Println("[Fatal] Error getting Status: ", err)
		return
	}
	status.Books = resp.StatusCode == 200

	// test monica api
	resp, err = http.Get("https://monica.tasadar.net")
	if err != nil {
		log.Println("[Fatal] Error getting Status: ", err)
		return
	}
	status.Monica = resp.StatusCode == 200

	// test matrix
	resp, err = http.Get("https://matrix.tasadar.net:8448")
	if err != nil {
		log.Println("[Fatal] Error getting Status: ", err)
		return
	}
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
