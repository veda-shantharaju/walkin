package models

import (
	"time"

	"gorm.io/datatypes"
)

type Record struct {
	ID          uint           `json:"id" gorm:"primaryKey"`
	Details     datatypes.JSON `json:"details"`                         // JSON field to store all record information
	Record_data datatypes.JSON `json:"record_data" gorm:"default:'[]'"` // JSON field with default value of empty array
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
}
