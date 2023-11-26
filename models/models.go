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
	ApiKey                     string
	WebhookEndpointSecret      string
	DefaultCurrency            string `gorm:"default:usd"`
	PaymentConfirmationMessage string `gorm:"default:Thanks!"`
	Domain                     string `gorm:"default:localhost"`
}

type SessionParams struct {
	stripe.CheckoutSessionParams `gorm:"embedded"`
}

type Sessions struct {
	Relation map[string]string `gorm:"embedded"`
}

func (sp *Sessions) Scan(src interface{}) error {
	return json.Unmarshal([]byte(src.(string)), &sp)
}

func (sp Sessions) Value() (driver.Value, error) {
	val, err := json.Marshal(sp)
	return string(val), err
}

type ActiveSession struct {
	gorm.Model
	Sessions Sessions `gorm:"embedded"`
}

type SessionLink struct {
	gorm.Model
	Active               bool
	LinkID               string `gorm:"unique"`
	Nickname             string
	Used                 int
	MaxUses              int
	Params               SessionParams `gorm:"embedded"`
	TrackingInventoryIDs IDs           `gorm:"embedded"`
}

func (sp *SessionParams) Scan(src interface{}) error {
	return json.Unmarshal([]byte(src.(string)), &sp)
}

func (sp SessionParams) Value() (driver.Value, error) {
	val, err := json.Marshal(sp)
	return string(val), err
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
	RecValue   string `gorm:"unique"` // Internal value for reconciliation
}

type InvoicePDF struct {
	gorm.Model       `json:"-"`
	Nickname         string
	TaxID            string
	CustomFieldName  string
	CustomFieldValue string
	Description      string
	Footer           string
}

type Inventory struct {
	gorm.Model
	DisplayName    string
	Product        string
	Quantity       int64
	SessionLinkIDs IDs `gorm:"embedded"`
}

type IDs []string

func (i *IDs) Scan(src interface{}) error {
	return json.Unmarshal([]byte(src.(string)), &i)
}

func (i IDs) Value() (driver.Value, error) {
	val, err := json.Marshal(i)
	return string(val), err
}

func (cf *CustomFields) ToStripe() *stripe.CheckoutSessionCustomFieldParams {
	var dOpts []*stripe.CheckoutSessionCustomFieldDropdownOptionParams
	if cf.Type == "dropdown" {
		for _, opt := range cf.Dropdown {
			dOpts = append(dOpts, &stripe.CheckoutSessionCustomFieldDropdownOptionParams{
				Label: stripe.String(opt.Label),
				Value: stripe.String(opt.RecValue),
			})
		}
	}
	return &stripe.CheckoutSessionCustomFieldParams{
		Key: stripe.String(strconv.Itoa(int(cf.ID))),
		Dropdown: &stripe.CheckoutSessionCustomFieldDropdownParams{
			Options: dOpts,
		},
		Label: &stripe.CheckoutSessionCustomFieldLabelParams{
			Type:   stripe.String("custom"), //the only type supported by Stripe
			Custom: &cf.Label,
		},
		Type: stripe.String(cf.Type),
	}

}
