package stripeapi

import (
	"github.com/stripe/stripe-go/v76"
	"github.com/stripe/stripe-go/v76/price"

	DB "iris/v2/database"
	"iris/v2/models"
)

func NewPriceIfNotExist(Currency string, UnitAmount int64, Product string) (string, error) {
	dbPrice := models.Price{}
	dbErr := DB.Conn.Where(&models.Price{Currency: Currency, UnitAmount: UnitAmount, Product: Product}).FirstOrInit(&dbPrice).Error
	if dbErr != nil {
		return "", dbErr
	}
	// If price has to be created in the DB, create it on Stripe
	if dbPrice.PriceID == "" {
		item, err := price.New(&stripe.PriceParams{
			Currency:   stripe.String(Currency),
			UnitAmount: stripe.Int64(UnitAmount),
			Product:    stripe.String(Product),
		})
		if err != nil {
			return "", err
		}

		dbPrice.PriceID = item.ID
		DB.Conn.Save(&dbPrice)
		return item.ID, nil
	}
	return dbPrice.PriceID, nil
}
