package stripeapi

import (
	"errors"

	"github.com/stripe/stripe-go/v76"
	"github.com/stripe/stripe-go/v76/price"
	"github.com/stripe/stripe-go/v76/product"

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

func GetAllProduct(active bool) ([]*stripe.Product, error) {
	stripe.Key = DB.GetSettings().ApiKey
	params := &stripe.ProductListParams{}
	params.Active = stripe.Bool(active)
	i := product.List(params)
	var products []*stripe.Product
	for i.Next() {
		products = append(products, i.Product())
	}
	if products == nil {
		return nil, errors.New("no products found")
	}
	return products, nil
}

func GetProductsInLinkReadable(LinkID string) []string {
	link := models.SessionLink{}
	err := DB.Conn.Where(&models.SessionLink{LinkID: LinkID}).First(&link)
	if err.Error != nil {
		return nil
	}
	var products []string
	for _, li := range link.Params.LineItems {
		products = append(products, *li.PriceData.ProductData.Name)
	}
	return products
}

func GetProductsInLink(LinkID string) []string {
	link := models.SessionLink{}
	err := DB.Conn.Where(&models.SessionLink{LinkID: LinkID}).First(&link)
	if err.Error != nil {
		return nil
	}
	var products []string
	for _, li := range link.Params.LineItems {
		products = append(products, *li.PriceData.Product)
	}
	return products
}

func ToggleProductActivity(Product string, Active bool) error {
	stripe.Key = DB.GetSettings().ApiKey
	_, err := product.Update(Product, &stripe.ProductParams{Active: stripe.Bool(Active)})
	if err != nil {
		return err
	}
	return nil
}
