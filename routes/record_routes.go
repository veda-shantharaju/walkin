package routes

import (
	"walkin/controllers"

	"github.com/gin-gonic/gin"
)

// RecordRoutes registers all record-related API endpoints
func RecordRoutes(router *gin.Engine) {
	api := router.Group("/records") // This creates the /records route group
	{
		// Route for creating a record
		api.POST("/", controllers.CreateRecord)

		api.POST("/records-data", controllers.UpdateRecordData)

		// Route for listing all records
		api.GET("/", controllers.ListRecords)
	}
}
