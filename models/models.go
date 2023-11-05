package models

import (
	"gorm.io/gorm"
)

type Settings struct {
	gorm.Model
	ApiKey                string
	WebhookEndpointSecret string
	DefaultCurrency       string `gorm:"default:usd"`
}

type PaymentLink struct {
	gorm.Model
	Active   bool
	Nickname string
	LinkID   string
	URL      string
	Used     int
	MaxUses  int
}

type Price struct {
	gorm.Model
	PriceID    string
	Currency   string
	UnitAmount int64
	Product    string
}
