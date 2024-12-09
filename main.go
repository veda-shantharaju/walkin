package main

import (
	"log"
	"walkin/config"
	"walkin/models"
	"walkin/routes" // This imports the routes package

	"github.com/gin-gonic/gin"
)

func main() {
	// Load environment variables
	config.LoadEnv()

	// Connect to the database
	config.ConnectDatabase()

	// Migrate the database schema
	config.DB.AutoMigrate(&models.Record{})

	// Initialize Gin router
	router := gin.Default()

	// Register API routes
	routes.RecordRoutes(router)

	// Start the server
	log.Fatal(router.Run(":8080"))
}
