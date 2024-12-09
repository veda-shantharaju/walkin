package controllers

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
	"walkin/config"
	"walkin/models"

	// For decoding JWT token
	"github.com/gin-gonic/gin"
	"gorm.io/datatypes"
)

// CreateRecord handles the creation of a new record
func CreateRecord(c *gin.Context) {
	// Input struct for receiving JSON data
	var input struct {
		Name  string `json:"name"`
		Email []struct {
			Email   string `json:"email"`
			Primary bool   `json:"primary"`
		} `json:"email"`
		Number []struct {
			Number  string `json:"number"`
			Primary bool   `json:"primary"`
		} `json:"number"`
	}

	// Bind the input body to the struct
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Prepare the details for the new record (without "details")
	newDetails := map[string]interface{}{
		"name":   input.Name,
		"email":  input.Email,
		"number": input.Number,
	}

	// Convert the newDetails map to JSON format
	newDetailsJSON, _ := json.Marshal(newDetails)

	// Create a new record with the provided data
	newRecord := models.Record{
		Details: datatypes.JSON(newDetailsJSON),
	}

	// Save the new record to the database
	if err := config.DB.Create(&newRecord).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create record"})
		return
	}

	// Return the newly created record
	c.JSON(http.StatusCreated, gin.H{"message": "Record created successfully", "record": newRecord})
}

// ListRecords handles fetching all the records
func ListRecords(c *gin.Context) {
	var records []models.Record

	// Fetch all records from the database
	if err := config.DB.Find(&records).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch records"})
		return
	}

	// Response with all records
	c.JSON(http.StatusOK, gin.H{"data": records})
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

// UpdateRecordData handles the updating of a record's data
func UpdateRecordData(c *gin.Context) {
	// Get the ID from the query parameters
	recordID := c.DefaultQuery("id", "0")

	// Convert the ID to uint
	var id uint
	_, err := fmt.Sscanf(recordID, "%d", &id)
	if err != nil || id == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid ID"})
		return
	}

	// Retrieve the existing record from the database
	var record models.Record
	if err := config.DB.First(&record, id).Error; err != nil {
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

	// Handle file upload (if any)
	file, _ := c.FormFile("record")
	if file != nil {
		// Define the path to save the file
		savePath := "media/" + file.Filename

		// Save the file to the media directory
		if err := c.SaveUploadedFile(file, savePath); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save file"})
			return
		}

		// Create the new data to be appended
		newRecordData := map[string]interface{}{
			"verified": "true",
			"record":   savePath, // Save the file path in the record data
			"comment":  "Updated comment",
			"created":  time.Now().Format(time.RFC3339),
			"author":   author,
		}

		// Append to the record's record_data
		var currentData []map[string]interface{}
		if err := json.Unmarshal(record.Record_data, &currentData); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to parse existing record_data"})
			return
		}
		currentData = append(currentData, newRecordData)

		// Update the record_data in the record
		updatedRecordData, err := json.Marshal(currentData)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to marshal updated record_data"})
			return
		}
		record.Record_data = updatedRecordData
	} else {
		// If no file is uploaded, update without file data
		newRecordData := map[string]interface{}{
			"verified": "true",
			"comment":  "Updated comment",
			"created":  time.Now().Format(time.RFC3339),
			"author":   author,
		}

		// Append to the record's record_data
		var currentData []map[string]interface{}
		if err := json.Unmarshal(record.Record_data, &currentData); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to parse existing record_data"})
			return
		}
		currentData = append(currentData, newRecordData)

		// Update the record_data in the record
		updatedRecordData, err := json.Marshal(currentData)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to marshal updated record_data"})
			return
		}
		record.Record_data = updatedRecordData
	}

	// Save the updated record
	if err := config.DB.Save(&record).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save record"})
		return
	}

	// Return the updated record
	c.JSON(http.StatusOK, gin.H{"message": "Record updated successfully", "record": record})
}
