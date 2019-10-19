package main

import (
	"github.com/gin-gonic/gin"
	_ "github.com/heroku/x/hmetrics/onload"
	"log"
)

// Main and Init
func main() {
	// Start Uni-Passau-Bot
	go uniPassauBot()

	// Creates a gin router with default middleware:
	// logger and recovery (crash-free) middleware
	// passes it to routes for setting of the routes
	router := gin.Default()
	routes(router)
	log.Fatal(router.Run())
}
