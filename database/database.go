package database

import (
	"log/slog"
	"slices"

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
	Conn.Statement.RaiseErrorOnNotFound = false
	if err != nil {
		panic("failed to connect to database")
	}

	Conn.AutoMigrate(
		&models.Settings{},
		&models.Price{},
		&models.CustomFields{},
		&models.DropdownOption{},
		&models.InvoicePDF{},
		&models.Inventory{},
		&models.SessionLink{},
		&models.ActiveSession{},
	)

}

func GetShippingCountries() []string {
	settings := models.Settings{}
	db_res := Conn.FirstOrCreate(&settings)
	if db_res.Error != nil {
		println("Error: %v\n", db_res.Error.Error())
		return []string{}
	}

	defaultCountries := []string{
		"AC", "AD", "AE", "AF", "AG", "AI", "AL", "AM", "AO", "AQ", "AR", "AT", "AU", "AW", "AX", "AZ", "BA", "BB", "BD", "BE", "BF", "BG", "BH", "BI", "BJ", "BL", "BM", "BN", "BO", "BQ", "BR", "BS", "BT", "BV", "BW", "BY", "BZ", "CA", "CD", "CF", "CG", "CH", "CI", "CK", "CL", "CM", "CN", "CO", "CR", "CV", "CW", "CY", "CZ", "DE", "DJ", "DK", "DM", "DO", "DZ", "EC", "EE", "EG", "EH", "ER", "ES", "ET", "FI", "FJ", "FK", "FO", "FR", "GA", "GB", "GD", "GE", "GF", "GG", "GH", "GI", "GL", "GM", "GN", "GP", "GQ", "GR", "GS", "GT", "GU", "GW", "GY", "HK", "HN", "HR", "HT", "HU", "ID", "IE", "IL", "IM", "IN", "IO", "IQ", "IS", "IT", "JE", "JM", "JO", "JP", "KE", "KG", "KH", "KI", "KM", "KN", "KR", "KW", "KY", "KZ", "LA", "LB", "LC", "LI", "LK", "LR", "LS", "LT", "LU", "LV", "LY", "MA", "MC", "MD", "ME", "MF", "MG", "MK", "ML", "MM", "MN", "MO", "MQ", "MR", "MS", "MT", "MU", "MV", "MW", "MX", "MY", "MZ", "NA", "NC", "NE", "NG", "NI", "NL", "NO", "NP", "NR", "NU", "NZ", "OM", "PA", "PE", "PF", "PG", "PH", "PK", "PL", "PM", "PN", "PR", "PS", "PT", "PY", "QA", "RE", "RO", "RS", "RU", "RW", "SA", "SB", "SC", "SE", "SG", "SH", "SI", "SJ", "SK", "SL", "SM", "SN", "SO", "SR", "SS", "ST", "SV", "SX", "SZ", "TA", "TC", "TD", "TF", "TG", "TH", "TJ", "TK", "TL", "TM", "TN", "TO", "TR", "TT", "TV", "TW", "TZ", "UA", "UG", "US", "UY", "UZ", "VA", "VC", "VE", "VG", "VN", "VU", "WF", "WS", "XK", "YE", "YT", "ZA", "ZM", "ZW", "ZZ",
	}

	if len(settings.ShippingBannedCountries) > 0 {
		filteredCountries := []string{}
		for _, country := range defaultCountries {
			if slices.Contains(settings.ShippingBannedCountries, country) {
				continue
			}
			filteredCountries = append(filteredCountries, country)
		}

		return filteredCountries
	}

	return defaultCountries
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
