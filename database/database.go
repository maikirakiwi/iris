package database

import (
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"

	"stripe-handler/v2/models"
)

var Conn *gorm.DB

func Init() {
	var err error

	Conn, err = gorm.Open(sqlite.Open(
		"file:data.db?&_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)&_pragma=synchronous(1)"), &gorm.Config{
		PrepareStmt: true,
	})
	if err != nil {
		panic("failed to connect to database")
	}
	Conn.AutoMigrate(&models.Settings{})
	Conn.AutoMigrate(&models.PaymentLink{})
}
