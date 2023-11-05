package models

import (
	"database/sql/driver"
	"encoding/json"
	"strconv"

	"github.com/stripe/stripe-go/v76"
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
	LinkID   string `gorm:"unique"`
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

type CustomFields struct {
	gorm.Model // custom_fields.key
	Label      string
	Type       string    // custom_fields.type
	Dropdown   Dropdowns `gorm:"embedded"` // custom_fields.dropdown
}

func (do *Dropdowns) Scan(src interface{}) error {
	return json.Unmarshal([]byte(src.(string)), &do)
}

func (do Dropdowns) Value() (driver.Value, error) {
	val, err := json.Marshal(do)
	return string(val), err
}

type Dropdowns []DropdownOption
type DropdownOption struct {
	gorm.Model `json:"-"`
	Label      string
	RecValue   string `gorm:"unique"`
}

func (cf *CustomFields) ToStripe() *stripe.PaymentLinkCustomFieldParams {
	var dOpts []*stripe.PaymentLinkCustomFieldDropdownOptionParams
	if cf.Type == "dropdown" {
		for _, opt := range cf.Dropdown {
			dOpts = append(dOpts, &stripe.PaymentLinkCustomFieldDropdownOptionParams{
				Label: stripe.String(opt.Label),
				Value: stripe.String(opt.RecValue),
			})
		}
	}
	return &stripe.PaymentLinkCustomFieldParams{
		Key: stripe.String(strconv.Itoa(int(cf.ID))),
		Dropdown: &stripe.PaymentLinkCustomFieldDropdownParams{
			Options: dOpts,
		},
		Label: &stripe.PaymentLinkCustomFieldLabelParams{
			Type:   stripe.String("custom"), //the only type supported by Stripe
			Custom: &cf.Label,
		},
		Type: stripe.String(cf.Type),
	}

}
