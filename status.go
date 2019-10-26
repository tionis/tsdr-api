package main

import "net/http"

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

	// Ping nextcloud over api

	// access dokuwiki api glyph

	// access tasadar wiki api

	// access shiori api

	// test search (maybe replace with custom javascript?)
	resp, _ = http.Get("https://search.tasadar.net")
	status.Grav = resp.StatusCode == 401

	// test dev
	resp, _ = http.Get("https://dev.tasadar.net")
	status.Grav = resp.StatusCode == 401

	// test books

	// test monica api

	// test matrix
	resp, _ = http.Get("https://matrix.tasadar.net:8448")
	status.Grav = resp.StatusCode == 200

	// check turnserver

	//check auth and API-server (currently this server, so always true) may change in future
	status.APIServer, status.AuthServer = true, true
}
