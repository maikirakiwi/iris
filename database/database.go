package database

import (
	"log/slog"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"

	"iris/v2/models"
)

var Conn *gorm.DB

// Non exported
var apiKey string
var webhookSecret string
var defaultCurrency string

func Init() {
	var err error

	Conn, err = gorm.Open(sqlite.Open(
		"file:data.db?&_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)&_pragma=synchronous(1)"), &gorm.Config{
		PrepareStmt: true,
	})
	if err != nil {
		panic("failed to connect to database")
	}

	Conn.AutoMigrate(
		&models.Settings{},
		&models.PaymentLink{},
		&models.Price{},
		&models.CustomFields{},
		&models.DropdownOption{},
	)
}

func GetSettings() *models.Settings {
	settings := &models.Settings{}
	db_res := Conn.First(&settings)

	// Return global cache if db is overwhelmed
	if db_res.Error != nil {
		slog.Warn("Error reading database settings: %v\n", db_res.Error)

		return &models.Settings{
			ApiKey:                apiKey,
			WebhookEndpointSecret: webhookSecret,
			DefaultCurrency:       defaultCurrency,
		}
	}

	// Update cache
	apiKey = settings.ApiKey
	webhookSecret = settings.WebhookEndpointSecret
	defaultCurrency = settings.DefaultCurrency

	return settings
}
