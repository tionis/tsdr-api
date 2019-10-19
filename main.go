package main

import (
	"github.com/gin-gonic/gin"
	_ "github.com/heroku/x/hmetrics/onload"
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
	router.Run()
}
