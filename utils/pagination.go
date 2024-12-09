package utils

import (
	"fmt"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// Paginate function returns paginated data and pagination metadata
func Paginate(c *gin.Context, db *gorm.DB, model interface{}) ([]interface{}, int64, int64, int64, error) {
	page := c.DefaultQuery("page", "1")         // Default page is 1
	perPage := c.DefaultQuery("per_page", "25") // Default per_page is 25

	// Convert to integer
	pageNum, err := strconv.Atoi(page)
	if err != nil || pageNum < 1 {
		return nil, 0, 0, 0, fmt.Errorf("Invalid page number")
	}

	perPageNum, err := strconv.Atoi(perPage)
	if err != nil || perPageNum < 1 {
		return nil, 0, 0, 0, fmt.Errorf("Invalid per_page value")
	}

	// Fetch the total count of records for pagination calculation
	var totalRecords int64
	if err := db.Model(model).Count(&totalRecords).Error; err != nil {
		// Log the error message here
		fmt.Printf("Error counting records: %v\n", err)
		return nil, 0, 0, 0, fmt.Errorf("Failed to count records")
	}

	// Calculate the offset and last page number
	offset := (pageNum - 1) * perPageNum
	lastPage := int64((totalRecords + int64(perPageNum) - 1) / int64(perPageNum))

	// Fetch the records for the current page with limit and offset
	var records []interface{}
	if err := db.Offset(offset).Limit(perPageNum).Find(&records).Error; err != nil {
		// Log the error message here
		fmt.Printf("Error fetching records: %v\n", err)
		return nil, 0, 0, 0, fmt.Errorf("Failed to fetch records")
	}

	return records, totalRecords, int64(pageNum), lastPage, nil
}
