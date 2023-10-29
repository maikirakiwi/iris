package models

import "gorm.io/gorm"

type Settings struct {
	gorm.Model
	ApiKey                string
	WebhookEndpointSecret string
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
