package stripeapi

import (
	"errors"

	"github.com/stripe/stripe-go/v76"
	"github.com/stripe/stripe-go/v76/price"
	"gorm.io/gorm"

	DB "stripe-handler/v2/database"
	"stripe-handler/v2/models"
)

func NewPriceIfNotExist(Currency string, UnitAmount int64, Product string) (models.Price, error) {
	dbPrice := &models.Price{}
	dbResult := DB.Conn.Where(&models.Price{Currency: Currency, UnitAmount: UnitAmount, Product: Product}).First(&dbPrice)
	if errors.Is(dbResult.Error, gorm.ErrRecordNotFound) {
		item, err := price.New(&stripe.PriceParams{
			Currency:   stripe.String(Currency),
			UnitAmount: stripe.Int64(UnitAmount),
			Product:    stripe.String(Product),
		})
		if err != nil {
			return models.Price{}, err
		}

		return models.Price{
			Currency:   Currency,
			UnitAmount: UnitAmount,
			Product:    Product,
			PriceID:    item.ID,
		}, err
	}

	return *dbPrice, nil
}
