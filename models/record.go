package models

import (
	"time"

	"gorm.io/datatypes"
)

// Record struct with the desired fields
type Record struct {
	ID        uint           `json:"id" gorm:"primaryKey"`
	Student   datatypes.JSON `json:"student"`    // JSON field for student information
	Record    string         `json:"record"`     // File path or reference to the uploaded file
	Details   datatypes.JSON `json:"details"`    // JSON field to store record details
	Author    datatypes.JSON `json:"author"`     // JSON field for author information
	CreatedAt time.Time      `json:"created_at"` // Timestamp for creation
	UpdatedAt time.Time      `json:"updated_at"` // Timestamp for last update
}
