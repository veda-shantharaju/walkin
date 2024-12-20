package controllers

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
	"walkin/config"
	"walkin/models"

	// For decoding JWT token
	"github.com/gin-gonic/gin"
)

// CreateRecordData handles the creation of a new record with student and author info
func CreateRecordData(c *gin.Context) {
	// Get the JWT token from the Authorization header
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Authorization header is missing"})
		return
	}

	// Extract the token from the "Bearer <token>" format
	tokenString := strings.TrimPrefix(authHeader, "Bearer ")

	// Decode the token and get the author
	author, err := getAuthorFromToken(tokenString)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	// Define the payload struct for student data
	type Student struct {
		Name  string `json:"name"`
		Email []struct {
			Email string `json:"email"`
		} `json:"email"`
		Number []struct {
			Number      string      `json:"number"`
			CountryCode string      `json:"country_code"`
			Verified    interface{} `json:"verified"` // Keep null by default
		} `json:"number"`
	}

	// Define the incoming request body structure
	type CreateRecordRequest struct {
		Student Student `json:"student"`
	}

	// Initialize the incoming request data
	var request CreateRecordRequest

	// Parse the JSON request body
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON input"})
		return
	}

	// Ensure the `verified` field defaults to `null` if not provided
	for i, number := range request.Student.Number {
		if number.Verified == nil {
			request.Student.Number[i].Verified = nil
		}
	}

	// Marshal the student data into JSON
	studentJSON, err := json.Marshal(request.Student)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to marshal student data"})
		return
	}

	// Marshal the author data into JSON
	authorJSON, err := json.Marshal(author)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to marshal author data"})
		return
	}

	// Create the record in the database
	record := models.Record{
		Student:   studentJSON, // Store the marshaled student data as JSON in the "Student" column
		Author:    authorJSON,  // Store the marshaled author data as JSON in the "Author" column
		CreatedAt: time.Now(),  // Set the creation timestamp
		UpdatedAt: time.Now(),  // Set the updated timestamp
	}

	// Save the new record in the database
	if err := config.DB.Create(&record).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save the record"})
		return
	}

	// Return the created record response
	c.JSON(http.StatusOK, gin.H{
		"message": "Record created successfully",
		"record":  record,
	})
}

// ListRecords handles fetching records associated with the authenticated author with pagination and filtering by number
func ListRecords(c *gin.Context) {
	var records []models.Record
	var limitInt int
	var pageInt int
	var numberFilter string

	// Get the JWT token from the Authorization header
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Authorization header is missing"})
		return
	}

	// Extract the token from the "Bearer <token>" format
	tokenString := strings.TrimPrefix(authHeader, "Bearer ")

	// Decode the token and get the author
	author, err := getAuthorFromToken(tokenString)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	// Default pagination values
	pageInt = 1
	limitInt = 10 // Default limit

	// Handle pagination (page and limit)
	if page, ok := c.GetQuery("page"); ok {
		if pageInt, err = strconv.Atoi(page); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid page"})
			return
		}
	}

	if limit, ok := c.GetQuery("limit"); ok {
		if limitInt, err = strconv.Atoi(limit); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid limit"})
			return
		}
	}

	// Get the number filter from query parameters (optional)
	numberFilter = c.DefaultQuery("number", "") // This will be an empty string if no number is provided

	// Build the query to fetch records associated with the author
	query := config.DB.Where("author ->> 'uid' = ?", author["uid"])

	// If a number filter is provided, apply it to the query
	if numberFilter != "" {
		// Use JSON query to search for records where student number matches the given number
		query = query.Where("student -> 'number' @> ?", fmt.Sprintf(`[{"number":"%s"}]`, numberFilter))
	}

	// Apply pagination to the query (LIMIT and OFFSET)
	query = query.Limit(limitInt).Offset((pageInt - 1) * limitInt)

	// Execute the query
	if err := query.Find(&records).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch records"})
		return
	}

	// If no records are found for the author, return an empty array
	if len(records) == 0 {
		c.JSON(http.StatusOK, gin.H{"data": []interface{}{}})
		return
	}

	// Response with the records associated with the author
	c.JSON(http.StatusOK, gin.H{
		"data":  records,
		"page":  pageInt,
		"limit": limitInt,
		"total": len(records), // You can adjust this to total records in DB if needed
	})
}

// Function to decode JWT token and get the entire claims as author data
func getAuthorFromToken(tokenString string) (map[string]interface{}, error) {
	// Log the token string to ensure it's being passed correctly
	fmt.Println("Received token:", tokenString)

	// Split the JWT into its components (Header, Payload, Signature)
	parts := strings.Split(tokenString, ".")
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid token format")
	}

	// Decode the payload (second part of the JWT)
	decodedPayload, err := base64.URLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, fmt.Errorf("failed to decode payload: %v", err)
	}

	// Parse the decoded payload into a map (JSON format)
	var claims map[string]interface{}
	if err := json.Unmarshal(decodedPayload, &claims); err != nil {
		return nil, fmt.Errorf("failed to unmarshal payload: %v", err)
	}

	// Log the decoded claims for debugging
	fmt.Println("Decoded claims:", claims)

	// Check for token expiration
	if claims["exp"] != nil {
		expTime := int64(claims["exp"].(float64))
		if expTime < time.Now().Unix() {
			return nil, fmt.Errorf("token is expired")
		}
	}

	// Return the entire claims map as the "author" data
	return claims, nil
}

// UpdateRecordData handles the updating of a record's details and file
func UpdateRecordData(c *gin.Context) {
	// Handle form data for verified, comment, and file (record)
	comment := c.DefaultPostForm("comment", "")
	file, _ := c.FormFile("record")                  // Get file from form-data
	verifiedRaw := c.DefaultPostForm("verified", "") // Get verified data from form-data

	// Log the received verified data for debugging
	log.Println("Received verified data:", verifiedRaw)

	// Check if verified data is provided and properly formatted
	var verifiedData []map[string]interface{}
	if verifiedRaw != "" {
		if err := json.Unmarshal([]byte(verifiedRaw), &verifiedData); err != nil {
			log.Printf("Failed to unmarshal verified data. Input: %s, Error: %v", verifiedRaw, err)
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "Invalid verified data format. Ensure it is a valid JSON array.",
				"details": err.Error(),
			})
			return
		}
	}

	// If no verified data is provided, return an error
	if len(verifiedData) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Verified data must be provided"})
		return
	}

	// Build the query to find the latest record based on verified number(s)
	var record models.Record
	var latestRecord models.Record

	for _, item := range verifiedData {
		if number, exists := item["number"].(string); exists {
			// Fetch the latest record for the number (sorting by timestamp or ID if available)
			if err := config.DB.Where("student -> 'number' @> ?", fmt.Sprintf(`[{"number":"%s"}]`, number)).
				Order("created_at DESC").First(&latestRecord).Error; err != nil {
				log.Printf("Record not found for number: %s", number)
				continue
			}

			// If a record is found for this number, assign it to the record to be updated
			record = latestRecord
			break
		}
	}

	// Check if record is found, otherwise return an error
	if record.ID == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Record not found"})
		return
	}

	// Get the JWT token from the Authorization header
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Authorization header is missing"})
		return
	}

	// Extract the token from the "Bearer <token>" format
	tokenString := strings.TrimPrefix(authHeader, "Bearer ")

	// Decode the token and get the author
	author, err := getAuthorFromToken(tokenString)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	// Unmarshal the author from the record's details and compare with the token's author
	var recordAuthor map[string]interface{}
	if err := json.Unmarshal(record.Author, &recordAuthor); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse record author"})
		return
	}

	// Check if the author UID from the token matches the UID in the record's author
	if recordAuthor["uid"] != author["uid"] {
		c.JSON(http.StatusForbidden, gin.H{"error": "You are not authorized to update this record"})
		return
	}

	// Prepare the details map for the update
	details := map[string]interface{}{
		"comment": comment,
		"type":    "walkin", // Example, this could be dynamic if needed
	}

	// If verified data is provided, update the corresponding numbers in the student data
	if len(verifiedData) > 0 {
		var student map[string]interface{}
		if err := json.Unmarshal(record.Student, &student); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse student data"})
			return
		}

		// Iterate over the numbers in the verified data and update their status
		for _, item := range verifiedData {
			if number, exists := item["number"].(string); exists {
				for i, num := range student["number"].([]interface{}) {
					numberObj := num.(map[string]interface{})
					if numberObj["number"] == number {
						// Update the verified status for this number
						numberObj["verified"] = item["verified"]
						student["number"].([]interface{})[i] = numberObj
					}
				}
			}
		}

		// Marshal the updated student data
		updatedStudentJSON, err := json.Marshal(student)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to marshal updated student data"})
			return
		}

		// Update the student field with the new data
		record.Student = updatedStudentJSON
	}

	// If a file is uploaded, save it and update the record data field
	if file != nil {
		// Define the path to save the file
		savePath := "media/" + file.Filename

		// Save the file to the media directory (or wherever you prefer)
		if err := c.SaveUploadedFile(file, savePath); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save file"})
			return
		}

		// Update the record's record_data with the file name
		record.Record = file.Filename
	}

	// Update the details field with the new values (verified, comment, type)
	record.Details, _ = json.Marshal(details)

	// Save the updated record to the database
	if err := config.DB.Save(&record).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update record"})
		return
	}

	// Return the updated record
	c.JSON(http.StatusOK, gin.H{
		"message": "Record updated successfully",
		"record":  record,
	})
}
