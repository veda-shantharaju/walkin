package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Database instance
var DB *gorm.DB

// Twilio credentials
var (
	TwilioAccountSID  string
	TwilioAuthToken   string
	TwilioPhoneNumber string
)

// LoadEnv loads environment variables from .env file
func LoadEnv() {
	err := godotenv.Load(".env")
	if err != nil {
		log.Fatalf("Error loading .env file")
	}
}

// In your config.go where you connect to DB
func ConnectDatabase() {
	var err error
	dsn := os.Getenv("DB")
	DB, err = gorm.Open(postgres.New(postgres.Config{
		DSN:                  dsn,
		PreferSimpleProtocol: true,
	}), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})

	if err != nil {
		panic("Failed to connect to DB")
	}

	// Log to confirm successful DB connection
	log.Println("Connected to database successfully")
}
